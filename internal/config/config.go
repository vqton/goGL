package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server        Server        `yaml:"server"`
	Database      Database      `yaml:"database"`
	Authorization Authorization `yaml:"authorization"`
	Session       Session       `yaml:"session"`
}

// Session configures server-side sessions and the password/account policy.
type Session struct {
	// CookieName names the httpOnly session cookie the auth handler sets.
	CookieName string `yaml:"cookie_name"`
	// MaxHours is the absolute session lifetime.
	MaxHours int `yaml:"max_hours"`
	// IdleMinutes is the rolling idle timeout; a session older than this
	// since last use is rejected.
	IdleMinutes int `yaml:"idle_minutes"`
	// MaxFailures locks a user out after this many consecutive failed logins.
	MaxFailures int `yaml:"max_failures"`
	// LockoutMinutes is how long a locked user stays locked.
	LockoutMinutes int `yaml:"lockout_minutes"`
	// MinPasswordLen is the minimum accepted password length.
	MinPasswordLen int `yaml:"min_password_len"`
}

type Server struct {
	HTTPAddr string `yaml:"http_addr"`
}

type Database struct {
	DSN string `yaml:"dsn"`
}

// Authorization configures casbin-based access control on the /api/v1 group.
type Authorization struct {
	// Enabled toggles the enforcement middleware. Leave on; it is the
	// authorization boundary for the whole API.
	Enabled bool `yaml:"enabled"`
	// IdentityHeader names the header the principal resolver reads the
	// subject (user id) from. Dev seam until real authentication lands.
	IdentityHeader string `yaml:"identity_header"`
	// DevMode routes principal resolution through the identity header
	// instead of the session cookie. The header is spoofable, so it must
	// only be used in development — production must keep DevMode false.
	DevMode bool `yaml:"dev_mode"`
}

func Load(path string) *Config {
	cfg := &Config{
		Server:   Server{HTTPAddr: ":8080"},
		Database: Database{DSN: "gogl.db"},
		Authorization: Authorization{
			Enabled:        true,
			IdentityHeader: "X-User-Id",
		},
		Session: Session{
			CookieName:     "gogl_session",
			MaxHours:       12,
			IdleMinutes:    30,
			MaxFailures:    5,
			LockoutMinutes: 15,
			MinPasswordLen: 8,
		},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(data, cfg)
	return cfg
}
