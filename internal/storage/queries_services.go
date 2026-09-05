package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/updu/updu/internal/models"
)

// --- Services ---

// CreateService inserts a new service and its endpoints.
func (db *DB) CreateService(ctx context.Context, s *models.Service) error {
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	if s.ZoneID == "" {
		s.ZoneID = "default"
	}
	if s.CreatedBy == "" {
		s.CreatedBy = "admin"
	}

	groupsJSON, err := json.Marshal(s.Groups)
	if err != nil {
		groupsJSON = []byte("[]")
	}
	tagsJSON, err := json.Marshal(s.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO services (id, name, type, zone_id, groups, tags, enabled, maintenance_id, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.Name, s.Type, s.ZoneID, string(groupsJSON), string(tagsJSON), s.Enabled, s.MaintenanceID, s.CreatedBy, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting service: %w", err)
	}

	for _, ep := range s.Endpoints {
		ep.ServiceID = s.ID
		if err := db.CreateServiceEndpoint(ctx, ep); err != nil {
			return fmt.Errorf("inserting service endpoint: %w", err)
		}
	}

	return nil
}

// GetService retrieves a service by ID, including its endpoints.
func (db *DB) GetService(ctx context.Context, id string) (*models.Service, error) {
	s := &models.Service{}
	var groupsJSON, tagsJSON string
	var maintID sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, name, type, zone_id, groups, tags, enabled, maintenance_id, created_by, created_at, updated_at
		FROM services WHERE id = ?
	`, id).Scan(&s.ID, &s.Name, &s.Type, &s.ZoneID, &groupsJSON, &tagsJSON, &s.Enabled, &maintID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting service: %w", err)
	}

	if maintID.Valid {
		s.MaintenanceID = &maintID.String
	}
	_ = json.Unmarshal([]byte(groupsJSON), &s.Groups)
	_ = json.Unmarshal([]byte(tagsJSON), &s.Tags)

	endpoints, err := db.ListServiceEndpoints(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Endpoints = endpoints

	return s, nil
}

// ListServices retrieves all services.
func (db *DB) ListServices(ctx context.Context) ([]*models.Service, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, type, zone_id, groups, tags, enabled, maintenance_id, created_by, created_at, updated_at
		FROM services ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		s := &models.Service{}
		var groupsJSON, tagsJSON string
		var maintID sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.ZoneID, &groupsJSON, &tagsJSON, &s.Enabled, &maintID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if maintID.Valid {
			s.MaintenanceID = &maintID.String
		}
		_ = json.Unmarshal([]byte(groupsJSON), &s.Groups)
		_ = json.Unmarshal([]byte(tagsJSON), &s.Tags)
		services = append(services, s)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	for _, s := range services {
		endpoints, err := db.ListServiceEndpoints(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		s.Endpoints = endpoints
	}
	return services, nil
}

// UpdateService updates a service's core properties.
func (db *DB) UpdateService(ctx context.Context, s *models.Service) error {
	groupsJSON, err := json.Marshal(s.Groups)
	if err != nil {
		groupsJSON = []byte("[]")
	}
	tagsJSON, err := json.Marshal(s.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	s.UpdatedAt = time.Now()
	_, err = db.ExecContext(ctx, `
		UPDATE services SET
			name = ?,
			type = ?,
			zone_id = ?,
			groups = ?,
			tags = ?,
			enabled = ?,
			maintenance_id = ?,
			updated_at = ?
		WHERE id = ?
	`, s.Name, s.Type, s.ZoneID, string(groupsJSON), string(tagsJSON), s.Enabled, s.MaintenanceID, s.UpdatedAt, s.ID)
	return err
}

// DeleteService deletes a service and its endpoints.
func (db *DB) DeleteService(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM services WHERE id = ?", id)
	return err
}

// --- Service Endpoints ---

// CreateServiceEndpoint inserts an endpoint for a service.
func (db *DB) CreateServiceEndpoint(ctx context.Context, ep *models.ServiceEndpoint) error {
	if ep.CreatedAt.IsZero() {
		ep.CreatedAt = time.Now()
	}
	if ep.ScopeID == "" {
		ep.ScopeID = models.ScopePublic
	}
	if ep.IntervalS <= 0 {
		ep.IntervalS = 30
	}
	if ep.TimeoutS <= 0 {
		ep.TimeoutS = 10
	}
	if ep.Retries <= 0 {
		ep.Retries = 2
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO service_endpoints (id, service_id, name, scope_id, target_type, config, interval_s, timeout_s, retries, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			scope_id = excluded.scope_id,
			target_type = excluded.target_type,
			config = excluded.config,
			interval_s = excluded.interval_s,
			timeout_s = excluded.timeout_s,
			retries = excluded.retries,
			is_primary = excluded.is_primary
	`, ep.ID, ep.ServiceID, ep.Name, ep.ScopeID, ep.TargetType, string(ep.Config), ep.IntervalS, ep.TimeoutS, ep.Retries, ep.IsPrimary, ep.CreatedAt)
	return err
}

// ListServiceEndpoints returns all endpoints associated with a service, enriched with recent check state.
func (db *DB) ListServiceEndpoints(ctx context.Context, serviceID string) ([]*models.ServiceEndpoint, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, service_id, name, scope_id, target_type, config, interval_s, timeout_s, retries, is_primary, created_at
		FROM service_endpoints WHERE service_id = ? ORDER BY is_primary DESC, name ASC
	`, serviceID)
	if err != nil {
		return nil, fmt.Errorf("listing endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []*models.ServiceEndpoint
	for rows.Next() {
		ep := &models.ServiceEndpoint{}
		var cfgStr string
		if err := rows.Scan(&ep.ID, &ep.ServiceID, &ep.Name, &ep.ScopeID, &ep.TargetType, &cfgStr, &ep.IntervalS, &ep.TimeoutS, &ep.Retries, &ep.IsPrimary, &ep.CreatedAt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ep.Config = json.RawMessage(cfgStr)
		ep.Status = models.StatusPending
		endpoints = append(endpoints, ep)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	// Enrich with latest check result
	for _, ep := range endpoints {
		var status string
		var latency, statusCode sql.NullInt64
		var message, metadata sql.NullString
		var checkedAt time.Time
		if err := db.QueryRowContext(ctx, `
			SELECT status, latency_ms, status_code, message, metadata, checked_at
			FROM endpoint_checks WHERE endpoint_id = ? ORDER BY checked_at DESC LIMIT 1
		`, ep.ID).Scan(&status, &latency, &statusCode, &message, &metadata, &checkedAt); err == nil {
			ep.Status = models.MonitorStatus(status)
			if latency.Valid {
				l := int(latency.Int64)
				ep.LastLatency = &l
			}
			if statusCode.Valid {
				sc := int(statusCode.Int64)
				ep.LastStatusCode = &sc
			}
			if message.Valid {
				ep.LastMessage = message.String
			}
			if metadata.Valid {
				ep.LastMetadata = json.RawMessage(metadata.String)
			}
			ep.LastCheckedAt = &checkedAt
		}
	}
	return endpoints, nil
}

// RecordEndpointCheck records the check result for an endpoint.
func (db *DB) RecordEndpointCheck(ctx context.Context, check *models.EndpointCheck) error {
	if check.CheckedAt.IsZero() {
		check.CheckedAt = time.Now()
	}
	if check.NodeID == "" {
		check.NodeID = "local"
	}

	var metaStr sql.NullString
	if len(check.Metadata) > 0 {
		metaStr = sql.NullString{String: string(check.Metadata), Valid: true}
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO endpoint_checks (service_id, endpoint_id, node_id, status, latency_ms, status_code, message, metadata, checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, check.ServiceID, check.EndpointID, check.NodeID, check.Status, check.LatencyMs, check.StatusCode, check.Message, metaStr, check.CheckedAt)
	return err
}

// --- Zones & Scopes ---

// ListZones returns all configured zones.
func (db *DB) ListZones(ctx context.Context) ([]*models.Zone, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, description, created_at FROM zones ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("listing zones: %w", err)
	}
	defer rows.Close()

	var zones []*models.Zone
	for rows.Next() {
		z := &models.Zone{}
		var desc sql.NullString
		if err := rows.Scan(&z.ID, &z.Name, &desc, &z.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			z.Description = desc.String
		}
		zones = append(zones, z)
	}
	return zones, rows.Err()
}

// CreateZone inserts a new zone.
func (db *DB) CreateZone(ctx context.Context, z *models.Zone) error {
	if z.CreatedAt.IsZero() {
		z.CreatedAt = time.Now()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO zones (id, name, description, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description
	`, z.ID, z.Name, z.Description, z.CreatedAt)
	return err
}

// ListScopes returns all configured network reachability scopes.
func (db *DB) ListScopes(ctx context.Context) ([]*models.Scope, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, description, created_at FROM scopes ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("listing scopes: %w", err)
	}
	defer rows.Close()

	var scopes []*models.Scope
	for rows.Next() {
		sc := &models.Scope{}
		var desc sql.NullString
		if err := rows.Scan(&sc.ID, &sc.Name, &desc, &sc.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			sc.Description = desc.String
		}
		scopes = append(scopes, sc)
	}
	return scopes, rows.Err()
}

// --- TLS Certificates ---

// UpsertTLSCertificate inserts or updates a discovered TLS certificate.
func (db *DB) UpsertTLSCertificate(ctx context.Context, cert *models.TLSCertificate) error {
	sansJSON, err := json.Marshal(cert.SANs)
	if err != nil {
		sansJSON = []byte("[]")
	}
	if cert.LastVerifiedAt.IsZero() {
		cert.LastVerifiedAt = time.Now()
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO tls_certificates (
			id, domain, issuer, subject, sans, valid_from, valid_until, 
			days_remaining, serial_number, signature_algorithm, ocsp_status, 
			last_verified_at, associated_endpoint_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			issuer = excluded.issuer,
			subject = excluded.subject,
			sans = excluded.sans,
			valid_from = excluded.valid_from,
			valid_until = excluded.valid_until,
			days_remaining = excluded.days_remaining,
			serial_number = excluded.serial_number,
			signature_algorithm = excluded.signature_algorithm,
			ocsp_status = excluded.ocsp_status,
			last_verified_at = excluded.last_verified_at,
			associated_endpoint_id = COALESCE(excluded.associated_endpoint_id, tls_certificates.associated_endpoint_id)
	`, cert.ID, cert.Domain, cert.Issuer, cert.Subject, string(sansJSON), cert.ValidFrom, cert.ValidUntil,
		cert.DaysRemaining, cert.SerialNumber, cert.SignatureAlgorithm, cert.OCSPStatus,
		cert.LastVerifiedAt, cert.AssociatedEndpointID)
	return err
}

// ListTLSCertificates retrieves all certificates, ordered by expiration date ascending.
func (db *DB) ListTLSCertificates(ctx context.Context) ([]*models.TLSCertificate, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT 
			c.id, c.domain, c.issuer, c.subject, c.sans, c.valid_from, c.valid_until,
			c.days_remaining, c.serial_number, c.signature_algorithm, c.ocsp_status,
			c.last_verified_at, c.associated_endpoint_id, COALESCE(s.name, '') as service_name
		FROM tls_certificates c
		LEFT JOIN service_endpoints ep ON c.associated_endpoint_id = ep.id
		LEFT JOIN services s ON ep.service_id = s.id
		ORDER BY c.valid_until ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing certificates: %w", err)
	}
	defer rows.Close()

	var certs []*models.TLSCertificate
	for rows.Next() {
		c := &models.TLSCertificate{}
		var sansJSON string
		var serial, sigAlg, ocsp, epID sql.NullString
		var sName string
		if err := rows.Scan(
			&c.ID, &c.Domain, &c.Issuer, &c.Subject, &sansJSON, &c.ValidFrom, &c.ValidUntil,
			&c.DaysRemaining, &serial, &sigAlg, &ocsp, &c.LastVerifiedAt, &epID, &sName,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(sansJSON), &c.SANs)
		if serial.Valid {
			c.SerialNumber = serial.String
		}
		if sigAlg.Valid {
			c.SignatureAlgorithm = sigAlg.String
		}
		if ocsp.Valid {
			c.OCSPStatus = ocsp.String
		}
		if epID.Valid {
			c.AssociatedEndpointID = &epID.String
		}
		c.AssociatedService = sName
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

// --- Topology ---

// GetNetworkTopology compiles the visual graph data for the Network Flow Map and Zone Swimlanes.
func (db *DB) GetNetworkTopology(ctx context.Context, localNodeID, localNodeName string) (*models.NetworkTopology, error) {
	zones, err := db.ListZones(ctx)
	if err != nil {
		return nil, err
	}
	scopes, err := db.ListScopes(ctx)
	if err != nil {
		return nil, err
	}
	services, err := db.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	// Build prober nodes: Local Node + Approved P2P Peers
	nodes := []*models.TopologyNode{
		{
			ID:       localNodeID,
			Name:     localNodeName,
			ZoneID:   "default",
			Scopes:   []string{models.ScopeLAN, models.ScopeTailnet, models.ScopePublic},
			Status:   "online",
			IsLocal:  true,
			LastSeen: time.Now(),
		},
	}

	peers, err := db.ListApprovedPeers(ctx)
	if err == nil {
		for _, p := range peers {
			pStatus := "online"
			var lastSeen time.Time
			if p.LastSeen != nil {
				lastSeen = *p.LastSeen
				if time.Since(lastSeen) > 2*time.Minute {
					pStatus = "offline"
				}
			}
			nodes = append(nodes, &models.TopologyNode{
				ID:       p.ID,
				Name:     p.Name,
				ZoneID:   "default", // Can be customized per peer metadata
				Scopes:   []string{models.ScopeTailnet, models.ScopePublic},
				Status:   pStatus,
				IsLocal:  false,
				LastSeen: lastSeen,
			})
		}
	}

	// Build edges connecting nodes to service endpoints
	var edges []*models.TopologyEdge
	for _, s := range services {
		for _, ep := range s.Endpoints {
			edgeStatus := ep.Status
			if edgeStatus == "" {
				edgeStatus = models.StatusPending
			}
			edges = append(edges, &models.TopologyEdge{
				ID:         fmt.Sprintf("edge-%s-%s", localNodeID, ep.ID),
				NodeID:     localNodeID,
				ServiceID:  s.ID,
				EndpointID: ep.ID,
				ScopeID:    ep.ScopeID,
				Status:     edgeStatus,
				LatencyMs:  ep.LastLatency,
				Message:    ep.LastMessage,
				CheckedAt:  time.Now(),
			})
		}
	}

	return &models.NetworkTopology{
		Nodes:    nodes,
		Zones:    zones,
		Scopes:   scopes,
		Services: services,
		Edges:    edges,
	}, nil
}
