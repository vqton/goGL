package backup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/backup"
)

const defaultMaxBackups = 10

// Auditor records backup operations. Satisfied by the audit service.
type Auditor interface {
	Record(ctx context.Context, l *audit.AuditLog) error
}

type Service interface {
	// CreateBackup snapshots the live DB with VACUUM INTO, records the
	// artifact with its SHA-256, then rotates the oldest beyond MaxBackups.
	CreateBackup(ctx context.Context, by, tier, trigger string) (*backup.BackupArtifact, error)
	ListBackups(ctx context.Context) ([]*backup.BackupArtifact, error)
	DeleteBackup(ctx context.Context, by, id string) error
	LastBackupAt(ctx context.Context) (string, error)
	GetJob(ctx context.Context) (*backup.BackupJob, error)
	SetJob(ctx context.Context, by string, j *backup.BackupJob) (*backup.BackupJob, error)
	// StageRestore copies + verifies an artifact (SHA-256, integrity_check)
	// and records a RestorePlan awaiting approval.
	StageRestore(ctx context.Context, by, artifactID string) (*backup.RestorePlan, error)
	// ApproveRestore swaps the staged file over the live DB. A process
	// restart is required for the server to reopen the new file.
	ApproveRestore(ctx context.Context, by, planID string) error
}

type service struct {
	db         *sql.DB
	repo       backup.Repository
	auditor    Auditor
	targetDir  string
	liveDBPath string
	maxBackups int
	now        func() time.Time
}

func NewService(db *sql.DB, repo backup.Repository, auditor Auditor, targetDir, liveDBPath string, maxBackups int) Service {
	if maxBackups <= 0 {
		maxBackups = defaultMaxBackups
	}
	return &service{
		db: db, repo: repo, auditor: auditor,
		targetDir: targetDir, liveDBPath: liveDBPath,
		maxBackups: maxBackups, now: time.Now,
	}
}

func (s *service) CreateBackup(ctx context.Context, by, tier, trigger string) (*backup.BackupArtifact, error) {
	if err := os.MkdirAll(s.targetDir, 0o755); err != nil {
		return nil, err
	}
	if tier == "" {
		tier = "default"
	}
	if trigger == "" {
		trigger = "manual"
	}

	id, err := newBackupID()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.targetDir, "backup_"+s.now().Format("20060102_150405")+"_"+id+".db")

	// VACUUM INTO does not accept bound parameters; the path is server-side
	// generated, so escaping single quotes is the only guard needed.
	escaped := strings.ReplaceAll(path, "'", "''")
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO '"+escaped+"'"); err != nil {
		return nil, err
	}

	sum, size, err := hashFile(path)
	if err != nil {
		return nil, err
	}

	a := &backup.BackupArtifact{
		ID:        "bk_" + id,
		Path:      path,
		SizeBytes: size,
		SHA256:    sum,
		Tier:      tier,
		Trigger:   trigger,
		CreatedAt: s.now().UTC(),
		CreatedBy: by,
	}
	if err := s.repo.CreateArtifact(ctx, a); err != nil {
		return nil, err
	}

	if err := s.rotate(ctx); err != nil {
		return nil, err
	}

	if err := s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "backup", Action: "backup.create", TargetID: a.ID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	}); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *service) ListBackups(ctx context.Context) ([]*backup.BackupArtifact, error) {
	return s.repo.ListArtifacts(ctx)
}

func (s *service) DeleteBackup(ctx context.Context, by, id string) error {
	a, err := s.repo.FindArtifact(ctx, id)
	if err != nil {
		return err
	}
	if a.Path != "" {
		if err := os.Remove(a.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := s.repo.DeleteArtifact(ctx, id); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "backup", Action: "backup.delete", TargetID: id,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) LastBackupAt(ctx context.Context) (string, error) {
	arts, err := s.repo.ListArtifacts(ctx)
	if err != nil {
		return "", err
	}
	if len(arts) == 0 {
		return "", nil
	}
	newest := arts[0]
	for _, a := range arts[1:] {
		if a.CreatedAt.After(newest.CreatedAt) {
			newest = a
		}
	}
	return newest.CreatedAt.UTC().Format(time.RFC3339), nil
}

func (s *service) GetJob(ctx context.Context) (*backup.BackupJob, error) {
	j, err := s.repo.FindJob(ctx, "default")
	if errors.Is(err, backup.ErrNotFound) {
		return &backup.BackupJob{ID: "default", Schedule: "0 2 * * *", TargetDir: s.targetDir, Enabled: false}, nil
	}
	return j, err
}

func (s *service) SetJob(ctx context.Context, by string, j *backup.BackupJob) (*backup.BackupJob, error) {
	j.ID = "default"
	if j.Schedule == "" {
		j.Schedule = "0 2 * * *"
	}
	if j.TargetDir == "" {
		j.TargetDir = s.targetDir
	}
	j.UpdatedAt = s.now().UTC()
	_, err := s.repo.FindJob(ctx, j.ID)
	if errors.Is(err, backup.ErrNotFound) {
		if err := s.repo.CreateJob(ctx, j); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if err := s.repo.UpdateJob(ctx, j); err != nil {
			return nil, err
		}
	}
	if err := s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "backup", Action: "backup.job", TargetID: j.ID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	}); err != nil {
		return nil, err
	}
	return j, nil
}

func (s *service) StageRestore(ctx context.Context, by, artifactID string) (*backup.RestorePlan, error) {
	a, err := s.repo.FindArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}
	if a.Path == "" {
		return nil, backup.ErrEmptyArtifact
	}
	if err := os.MkdirAll(s.targetDir, 0o755); err != nil {
		return nil, err
	}

	staged := filepath.Join(s.targetDir, "restore_"+artifactID+".db")
	if err := copyFile(a.Path, staged); err != nil {
		return nil, err
	}

	integrity := "ok"
	if err := checkIntegrity(staged); err != nil {
		integrity = err.Error()
	}
	if err := verifySHA256(staged, a.SHA256); err != nil {
		integrity = err.Error()
	}

	plan := &backup.RestorePlan{
		ID:         "rp_" + artifactID,
		ArtifactID: artifactID,
		StagedPath: staged,
		Status:     "staged",
		Integrity:  integrity,
		CreatedAt:  s.now().UTC(),
		CreatedBy:  by,
	}
	if integrity != "ok" {
		return plan, backup.ErrIntegrity
	}
	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}
	return plan, s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "backup", Action: "backup.restore_staged", TargetID: artifactID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) ApproveRestore(ctx context.Context, by, planID string) error {
	plan, err := s.repo.FindPlan(ctx, planID)
	if err != nil {
		return err
	}
	if plan.Status != "staged" {
		return backup.ErrNoActivePlan
	}
	if plan.Integrity != "ok" {
		return backup.ErrIntegrity
	}
	if err := copyFile(plan.StagedPath, s.liveDBPath); err != nil {
		return err
	}
	plan.Status = "approved"
	if err := s.repo.UpdatePlan(ctx, plan); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "backup", Action: "backup.restore_approved", TargetID: planID,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

// rotate removes files of artifacts beyond maxBackups (newest kept).
func (s *service) rotate(ctx context.Context) error {
	arts, err := s.repo.ListArtifacts(ctx)
	if err != nil {
		return err
	}
	if len(arts) <= s.maxBackups {
		return nil
	}
	for _, a := range arts[s.maxBackups:] {
		if a.Path != "" {
			if err := os.Remove(a.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if err := s.repo.DeleteArtifact(ctx, a.ID); err != nil {
			return err
		}
	}
	return nil
}

func newBackupID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func verifySHA256(path, want string) error {
	got, _, err := hashFile(path)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("%w: got %s want %s", backup.ErrIntegrity, got, want)
	}
	return nil
}

// checkIntegrity opens the DB file directly and runs PRAGMA integrity_check.
func checkIntegrity(path string) error {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer conn.Close()
	var result string
	if err := conn.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("%w: integrity_check=%s", backup.ErrIntegrity, result)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
