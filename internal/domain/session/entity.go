package session

import "context"

// Session is a server-side login session. The ID is the opaque token stored
// in the httpOnly session cookie; everything else lives in the database so a
// stolen cookie value can be revoked server-side.
type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	LastSeen  string `json:"last_seen"`
	ExpiresAt string `json:"expires_at"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

type Repository interface {
	Create(ctx context.Context, s *Session) error
	FindByID(ctx context.Context, id string) (*Session, error)
	Update(ctx context.Context, s *Session) error
	Delete(ctx context.Context, id string) error
	DeleteByUser(ctx context.Context, userID string) error
	CountActive(ctx context.Context) (int, error)
	CountByUser(ctx context.Context, userID string) (int, error)
	ListByUser(ctx context.Context, userID string) ([]*Session, error)
}
