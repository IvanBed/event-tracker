package auth

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	ServiceID uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
}

type RefreshTokenRepository struct {
	db *sql.DB
}

func InitRefreshTokenRepo(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (tokenRepo *RefreshTokenRepository) CreateRefreshToken(serviceID uuid.UUID, ttl time.Duration) (*RefreshToken, error) {

	tokenID := uuid.New()
	expiresAt := time.Now().Add(ttl)
	token := &RefreshToken{
		ID:        tokenID,
		ServiceID: serviceID,
		Token:     tokenID.String(), // Use the UUID as the token
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		Revoked:   false,
	}
	query := `
        INSERT INTO refresh_tokens (id, service_id, token, expires_at, created_at, revoked)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err := tokenRepo.db.Exec(query, token.ID, token.ServiceID, token.Token, token.ExpiresAt, token.CreatedAt, token.Revoked)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (tokenRepo *RefreshTokenRepository) GetRefreshToken(tokenString string) (*RefreshToken, error) {
	query := `
        SELECT id, user_id, token, expires_at, created_at, revoked
        FROM refresh_tokens
        WHERE token = $1
    `
	var token RefreshToken
	err := tokenRepo.db.QueryRow(query, tokenString).Scan(
		&token.ID,
		&token.ServiceID,
		&token.Token,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.Revoked,
	)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken marks a refresh token as revoked
func (r *RefreshTokenRepository) RevokeRefreshToken(tokenString string) error {
	query := `
        UPDATE refresh_tokens
        SET revoked = true
        WHERE token = $1
    `
	_, err := r.db.Exec(query, tokenString)
	return err
}
