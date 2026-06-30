package domain

import "time"

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
	TokenType    string
}

type RefreshToken struct {
	ID        string
	UserID    string
	TenantID  string
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func (t *RefreshToken) IsValid() bool {
	return !t.Revoked && time.Now().Before(t.ExpiresAt)
}
