package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/updu/updu/internal/models"
)

func (db *DB) CreateAPIToken(ctx context.Context, token *models.APIToken, tokenHash string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO api_tokens (id, name, token_hash, prefix, scope, created_by, created_at, last_used_at, revoked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.Name, tokenHash, token.Prefix, token.Scope, token.CreatedBy, token.CreatedAt, token.LastUsedAt, token.RevokedAt,
	)
	return err
}

func (db *DB) GetAPITokenByHash(ctx context.Context, tokenHash string) (*models.APIToken, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, name, prefix, scope, created_by, created_at, last_used_at, revoked_at
		 FROM api_tokens WHERE token_hash = ? AND revoked_at IS NULL`,
		tokenHash,
	)

	var token models.APIToken
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	if err := row.Scan(&token.ID, &token.Name, &token.Prefix, &token.Scope, &token.CreatedBy, &token.CreatedAt, &lastUsedAt, &revokedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if lastUsedAt.Valid {
		timestamp := lastUsedAt.Time
		token.LastUsedAt = &timestamp
	}
	if revokedAt.Valid {
		timestamp := revokedAt.Time
		token.RevokedAt = &timestamp
	}
	return &token, nil
}

func (db *DB) ListAPITokens(ctx context.Context) ([]*models.APIToken, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, name, prefix, scope, created_by, created_at, last_used_at, revoked_at
		 FROM api_tokens ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.APIToken
	for rows.Next() {
		var token models.APIToken
		var lastUsedAt sql.NullTime
		var revokedAt sql.NullTime
		if err := rows.Scan(&token.ID, &token.Name, &token.Prefix, &token.Scope, &token.CreatedBy, &token.CreatedAt, &lastUsedAt, &revokedAt); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			timestamp := lastUsedAt.Time
			token.LastUsedAt = &timestamp
		}
		if revokedAt.Valid {
			timestamp := revokedAt.Time
			token.RevokedAt = &timestamp
		}
		tokens = append(tokens, &token)
	}
	return tokens, rows.Err()
}

func (db *DB) UpdateAPITokenLastUsed(ctx context.Context, id string, timestamp time.Time) error {
	_, err := db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, timestamp, id)
	return err
}

func (db *DB) RevokeAPIToken(ctx context.Context, id string, timestamp time.Time) error {
	_, err := db.ExecContext(ctx, `UPDATE api_tokens SET revoked_at = ? WHERE id = ?`, timestamp, id)
	return err
}
