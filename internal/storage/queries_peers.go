package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/updu/updu/internal/models"
)

// CreatePeer inserts a new peer into the database. If the peer ID exists, it updates it.
func (db *DB) CreatePeer(ctx context.Context, p *models.Peer) error {
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.Role == "" {
		p.Role = models.PeerRolePeer
	}
	if p.Status == "" {
		p.Status = models.PeerStatusPending
	}

	var metadataStr sql.NullString
	if len(p.Metadata) > 0 {
		metadataStr = sql.NullString{String: string(p.Metadata), Valid: true}
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO peers (id, name, address, public_key, role, status, last_seen, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name       = excluded.name,
			address    = excluded.address,
			public_key = excluded.public_key,
			role       = excluded.role,
			-- Never downgrade an approved peer back to pending.
			-- A re-sent pair-request cannot strip a human-granted approval.
			status     = CASE WHEN peers.status = 'approved' THEN 'approved' ELSE excluded.status END,
			last_seen  = COALESCE(excluded.last_seen, peers.last_seen),
			metadata   = COALESCE(excluded.metadata, peers.metadata)
	`, p.ID, p.Name, p.Address, p.PublicKey, p.Role, p.Status, p.LastSeen, metadataStr, p.CreatedAt)
	return err
}

// GetPeer retrieves a peer by its unique ID.
func (db *DB) GetPeer(ctx context.Context, id string) (*models.Peer, error) {
	p := &models.Peer{}
	var lastSeen sql.NullTime
	var metadataStr sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, name, address, public_key, role, status, last_seen, metadata, created_at
		FROM peers WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Address, &p.PublicKey, &p.Role, &p.Status, &lastSeen, &metadataStr, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastSeen.Valid {
		p.LastSeen = &lastSeen.Time
	}
	if metadataStr.Valid && metadataStr.String != "" {
		p.Metadata = json.RawMessage(metadataStr.String)
	}
	return p, nil
}

// GetPeerByPublicKey retrieves a peer by its public key.
func (db *DB) GetPeerByPublicKey(ctx context.Context, pubKey string) (*models.Peer, error) {
	p := &models.Peer{}
	var lastSeen sql.NullTime
	var metadataStr sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, name, address, public_key, role, status, last_seen, metadata, created_at
		FROM peers WHERE public_key = ?
	`, pubKey).Scan(&p.ID, &p.Name, &p.Address, &p.PublicKey, &p.Role, &p.Status, &lastSeen, &metadataStr, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastSeen.Valid {
		p.LastSeen = &lastSeen.Time
	}
	if metadataStr.Valid && metadataStr.String != "" {
		p.Metadata = json.RawMessage(metadataStr.String)
	}
	return p, nil
}

// ListPeers returns all configured peers.
func (db *DB) ListPeers(ctx context.Context) ([]*models.Peer, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, address, public_key, role, status, last_seen, metadata, created_at
		FROM peers ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		p := &models.Peer{}
		var lastSeen sql.NullTime
		var metadataStr sql.NullString

		if err := rows.Scan(&p.ID, &p.Name, &p.Address, &p.PublicKey, &p.Role, &p.Status, &lastSeen, &metadataStr, &p.CreatedAt); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			p.LastSeen = &lastSeen.Time
		}
		if metadataStr.Valid && metadataStr.String != "" {
			p.Metadata = json.RawMessage(metadataStr.String)
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

// ListApprovedPeers returns all approved peers.
func (db *DB) ListApprovedPeers(ctx context.Context) ([]*models.Peer, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, address, public_key, role, status, last_seen, metadata, created_at
		FROM peers WHERE status = 'approved' ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		p := &models.Peer{}
		var lastSeen sql.NullTime
		var metadataStr sql.NullString

		if err := rows.Scan(&p.ID, &p.Name, &p.Address, &p.PublicKey, &p.Role, &p.Status, &lastSeen, &metadataStr, &p.CreatedAt); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			p.LastSeen = &lastSeen.Time
		}
		if metadataStr.Valid && metadataStr.String != "" {
			p.Metadata = json.RawMessage(metadataStr.String)
		}
		peers = append(peers, p)
	}
	return peers, rows.Err()
}

// UpdatePeerStatus updates the status of a peer (e.g. approved, rejected).
func (db *DB) UpdatePeerStatus(ctx context.Context, id, status string) error {
	_, err := db.ExecContext(ctx, `UPDATE peers SET status = ? WHERE id = ?`, status, id)
	return err
}

// UpdatePeerLastSeen updates the last_seen timestamp and metadata for a peer.
func (db *DB) UpdatePeerLastSeen(ctx context.Context, id string, lastSeen time.Time, metadata json.RawMessage) error {
	var metadataStr sql.NullString
	if len(metadata) > 0 {
		metadataStr = sql.NullString{String: string(metadata), Valid: true}
	}
	_, err := db.ExecContext(ctx, `
		UPDATE peers SET last_seen = ?, metadata = COALESCE(?, metadata) WHERE id = ?
	`, lastSeen, metadataStr, id)
	return err
}

// UpdatePeer updates address, name, role and status for a peer.
func (db *DB) UpdatePeer(ctx context.Context, p *models.Peer) error {
	var metadataStr sql.NullString
	if len(p.Metadata) > 0 {
		metadataStr = sql.NullString{String: string(p.Metadata), Valid: true}
	}
	_, err := db.ExecContext(ctx, `
		UPDATE peers SET name = ?, address = ?, public_key = ?, role = ?, status = ?, metadata = ?
		WHERE id = ?
	`, p.Name, p.Address, p.PublicKey, p.Role, p.Status, metadataStr, p.ID)
	return err
}

// DeletePeer removes a peer by ID.
func (db *DB) DeletePeer(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM peers WHERE id = ?`, id)
	return err
}
