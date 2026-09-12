package api

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/checker"
	"github.com/updu/updu/internal/diagnostic"
	"github.com/updu/updu/internal/models"
)

// GET /api/v1/services
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services, err := s.db.ListServices(ctx)
	if err != nil {
		jsonError(w, "failed to list services: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, svc := range services {
		checksMap := make(map[string]*models.EndpointCheck)
		for _, ep := range svc.Endpoints {
			if ep.LastCheckedAt != nil {
				checksMap[ep.ID] = &models.EndpointCheck{
					ServiceID:  svc.ID,
					EndpointID: ep.ID,
					Status:     ep.Status,
					LatencyMs:  ep.LastLatency,
					StatusCode: ep.LastStatusCode,
					Message:    ep.LastMessage,
					CheckedAt:  *ep.LastCheckedAt,
				}
			}
			if ep.IsPrimary && ep.LastLatency != nil {
				svc.PrimaryLatency = ep.LastLatency
			}
		}

		diag := diagnostic.EvaluateServiceHealth(svc, checksMap)
		svc.Status = diag.Status
		svc.Diagnosis = diag.Summary
	}

	if services == nil {
		services = []*models.Service{}
	}

	jsonOK(w, services)
}

// POST /api/v1/services
func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var svc models.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if svc.Name == "" {
		jsonError(w, "service name is required", http.StatusBadRequest)
		return
	}
	if svc.Type == "" {
		svc.Type = models.ServiceTypeWeb
	}
	if svc.ZoneID == "" {
		svc.ZoneID = "default"
	}
	if svc.ID == "" {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", svc.Name, svc.Type, time.Now().UnixNano())))
		svc.ID = hex.EncodeToString(hash[:])[:16]
	}
	if user != nil {
		svc.CreatedBy = user.Username
	}

	for i, ep := range svc.Endpoints {
		if ep.ID == "" {
			epHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", svc.ID, ep.Name, i)))
			ep.ID = hex.EncodeToString(epHash[:])[:16]
		}
		if ep.ScopeID == "" {
			ep.ScopeID = models.ScopePublic
		}
		if ep.TargetType == "" {
			ep.TargetType = svc.Type
		}
	}

	if err := s.db.CreateService(r.Context(), &svc); err != nil {
		jsonError(w, "failed to create service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	primaryEp := svc.PrimaryEndpoint()
	if primaryEp != nil {
		mon, _ := s.db.GetMonitor(r.Context(), svc.ID)
		if mon == nil {
			interval := primaryEp.IntervalS
			if interval <= 0 {
				interval = 60
			}
			timeout := primaryEp.TimeoutS
			if timeout <= 0 {
				timeout = 10
			}
			m := &models.Monitor{
				ID:        svc.ID,
				Name:      svc.Name,
				Type:      primaryEp.TargetType,
				Config:    primaryEp.Config,
				IntervalS: interval,
				TimeoutS:  timeout,
				Retries:   primaryEp.Retries,
				Enabled:   svc.Enabled,
				Groups:    svc.Groups,
				Tags:      svc.Tags,
			}
			if err := s.db.CreateMonitor(r.Context(), m); err == nil && s.scheduler != nil {
				s.scheduler.AddMonitor(context.Background(), m)
			}
		}
	}

	s.recordAudit(r, "service.create", "service", svc.ID, "created service "+svc.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, svc)
}

// GET /api/v1/services/{id}
func (s *Server) handleGetService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc, err := s.db.GetService(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to get service: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if svc == nil {
		jsonError(w, "service not found", http.StatusNotFound)
		return
	}

	checksMap := make(map[string]*models.EndpointCheck)
	for _, ep := range svc.Endpoints {
		if ep.LastCheckedAt != nil {
			checksMap[ep.ID] = &models.EndpointCheck{
				ServiceID:  svc.ID,
				EndpointID: ep.ID,
				Status:     ep.Status,
				LatencyMs:  ep.LastLatency,
				StatusCode: ep.LastStatusCode,
				Message:    ep.LastMessage,
				CheckedAt:  *ep.LastCheckedAt,
			}
		}
		if ep.IsPrimary && ep.LastLatency != nil {
			svc.PrimaryLatency = ep.LastLatency
		}
	}

	diag := diagnostic.EvaluateServiceHealth(svc, checksMap)
	svc.Status = diag.Status
	svc.Diagnosis = diag.Summary

	jsonOK(w, map[string]any{
		"service":   svc,
		"diagnosis": diag,
	})
}

// PUT /api/v1/services/{id}
func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := s.db.GetService(r.Context(), id)
	if err != nil || existing == nil {
		jsonError(w, "service not found", http.StatusNotFound)
		return
	}

	var update models.Service
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	update.ID = id
	if update.Name == "" {
		update.Name = existing.Name
	}
	if update.Type == "" {
		update.Type = existing.Type
	}
	if update.ZoneID == "" {
		update.ZoneID = existing.ZoneID
	}

	if err := s.db.UpdateService(r.Context(), &update); err != nil {
		jsonError(w, "failed to update service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update endpoints if provided
	if len(update.Endpoints) > 0 {
		for i, ep := range update.Endpoints {
			ep.ServiceID = id
			if ep.ID == "" {
				epHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", id, ep.Name, i)))
				ep.ID = hex.EncodeToString(epHash[:])[:16]
			}
			_ = s.db.CreateServiceEndpoint(r.Context(), ep)
		}
	}

	s.recordAudit(r, "service.update", "service", id, "updated service "+update.Name)
	jsonOK(w, update)
}

// DELETE /api/v1/services/{id}
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.scheduler != nil {
		s.scheduler.RemoveMonitor(id)
	}
	_ = s.db.DeleteMonitor(r.Context(), id)
	if err := s.db.DeleteService(r.Context(), id); err != nil {
		jsonError(w, "failed to delete service: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.recordAudit(r, "service.delete", "service", id, "deleted service "+id)
	jsonOK(w, map[string]string{"status": "deleted"})
}

// POST /api/v1/services/{id}/endpoints
func (s *Server) handleCreateServiceEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc, err := s.db.GetService(r.Context(), id)
	if err != nil || svc == nil {
		jsonError(w, "service not found", http.StatusNotFound)
		return
	}

	var ep models.ServiceEndpoint
	if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if ep.Name == "" {
		jsonError(w, "endpoint name is required", http.StatusBadRequest)
		return
	}
	ep.ServiceID = id
	if ep.ID == "" {
		epHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", id, ep.Name, time.Now().UnixNano())))
		ep.ID = hex.EncodeToString(epHash[:])[:16]
	}
	if ep.ScopeID == "" {
		ep.ScopeID = models.ScopePublic
	}
	if ep.TargetType == "" {
		ep.TargetType = svc.Type
	}
	if ep.IntervalS <= 0 {
		ep.IntervalS = 30
	}
	if ep.TimeoutS <= 0 {
		ep.TimeoutS = 10
	}

	if err := s.db.CreateServiceEndpoint(r.Context(), &ep); err != nil {
		jsonError(w, "failed to create endpoint: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.recordAudit(r, "service.endpoint.create", "service", id, "created endpoint "+ep.Name)
	w.WriteHeader(http.StatusCreated)
	jsonOK(w, ep)
}

// DELETE /api/v1/services/{id}/endpoints/{epId}
func (s *Server) handleDeleteServiceEndpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	epID := r.PathValue("epId")
	if err := s.db.DeleteServiceEndpoint(r.Context(), id, epID); err != nil {
		jsonError(w, "failed to delete endpoint: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.recordAudit(r, "service.endpoint.delete", "service", id, "deleted endpoint "+epID)
	jsonOK(w, map[string]string{"status": "deleted"})
}

// POST /api/v1/services/{id}/probe
func (s *Server) handleProbeService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc, err := s.db.GetService(r.Context(), id)
	if err != nil || svc == nil {
		jsonError(w, "service not found", http.StatusNotFound)
		return
	}

	if len(svc.Endpoints) == 0 {
		jsonError(w, "service has no endpoints configured", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(s.withLocalhostContext(r.Context()), 15*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	probeResults := make(map[string]*models.EndpointCheck)

	for _, ep := range svc.Endpoints {
		wg.Add(1)
		go func(endpoint *models.ServiceEndpoint) {
			defer wg.Done()

			c := s.registry.Get(endpoint.TargetType)
			if c == nil {
				return
			}

			// Wrap endpoint into a transient monitor for the prober registry
			tempMon := &models.Monitor{
				ID:        endpoint.ID,
				Name:      endpoint.Name,
				Type:      endpoint.TargetType,
				Config:    endpoint.Config,
				TimeoutS:  endpoint.TimeoutS,
				IntervalS: endpoint.IntervalS,
			}

			start := time.Now()
			checkRes, err := c.Check(ctx, tempMon)
			duration := int(time.Since(start).Milliseconds())

			ec := &models.EndpointCheck{
				ServiceID:  svc.ID,
				EndpointID: endpoint.ID,
				NodeID:     "local",
				CheckedAt:  time.Now(),
				LatencyMs:  &duration,
			}

			if err != nil || checkRes == nil {
				ec.Status = models.StatusDown
				if err != nil {
					ec.Message = err.Error()
				}
			} else {
				ec.Status = checkRes.Status
				ec.LatencyMs = checkRes.LatencyMs
				ec.StatusCode = checkRes.StatusCode
				ec.Message = checkRes.Message
				ec.Metadata = checkRes.Metadata

				// If TLS metadata is available, record to tls_certificates table
				if len(checkRes.Metadata) > 0 {
					if cert := checker.ExtractTLSCertificate(checkRes.Metadata, &endpoint.ID, endpoint.Name); cert != nil {
						_ = s.db.UpsertTLSCertificate(context.Background(), cert)
					}
				}
			}

			_ = s.db.RecordEndpointCheck(context.Background(), ec)

			mu.Lock()
			probeResults[endpoint.ID] = ec
			mu.Unlock()
		}(ep)
	}

	wg.Wait()

	diag := diagnostic.EvaluateServiceHealth(svc, probeResults)
	jsonOK(w, map[string]any{
		"service_id": svc.ID,
		"diagnosis":  diag,
		"probes":     probeResults,
	})
}

// GET /api/v1/zones
func (s *Server) handleListZones(w http.ResponseWriter, r *http.Request) {
	zones, err := s.db.ListZones(r.Context())
	if err != nil {
		jsonError(w, "failed to list zones: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, zones)
}

// POST /api/v1/zones
func (s *Server) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	var z models.Zone
	if err := json.NewDecoder(r.Body).Decode(&z); err != nil || z.ID == "" || z.Name == "" {
		jsonError(w, "invalid request body: id and name required", http.StatusBadRequest)
		return
	}
	if err := s.db.CreateZone(r.Context(), &z); err != nil {
		jsonError(w, "failed to create zone: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, z)
}

// GET /api/v1/scopes
func (s *Server) handleListScopes(w http.ResponseWriter, r *http.Request) {
	scopes, err := s.db.ListScopes(r.Context())
	if err != nil {
		jsonError(w, "failed to list scopes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, scopes)
}

// GET /api/v1/topology
func (s *Server) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	nodeID := "local-node"
	nodeName, _ := os.Hostname()
	if nodeName == "" {
		nodeName = "Primary Node"
	}
	if s.p2p != nil && s.p2p.Identity() != nil {
		nodeID = s.p2p.Identity().NodeID
		nodeName = s.p2p.Identity().Name
	}

	topo, err := s.db.GetNetworkTopology(r.Context(), nodeID, nodeName)
	if err != nil {
		jsonError(w, "failed to get topology: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, topo)
}

// GET /api/v1/certificates
func (s *Server) handleListTLSCertificates(w http.ResponseWriter, r *http.Request) {
	certs, err := s.db.ListTLSCertificates(r.Context())
	if err != nil {
		jsonError(w, "failed to list certificates: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if certs == nil {
		certs = []*models.TLSCertificate{}
	}
	jsonOK(w, certs)
}

type testTLSCertificateRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// POST /api/v1/certificates/test
func (s *Server) handleTestTLSCertificate(w http.ResponseWriter, r *http.Request) {
	var req testTLSCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host == "" {
		jsonError(w, "host is required", http.StatusBadRequest)
		return
	}

	host := strings.TrimSpace(req.Host)
	port := req.Port

	// Clean hostname: strip scheme and paths if provided
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil {
			if u.Hostname() != "" {
				host = u.Hostname()
			}
			if u.Port() != "" {
				if p, err := strconv.Atoi(u.Port()); err == nil && p > 0 {
					port = p
				}
			}
		}
	} else if strings.Contains(host, "/") {
		host = strings.Split(host, "/")[0]
	}

	// If host still contains port as "host:port", split it
	if h, pStr, err := net.SplitHostPort(host); err == nil {
		host = h
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			port = p
		}
	}

	if port <= 0 {
		port = 443
	}

	if host == "" {
		jsonError(w, "invalid host", http.StatusBadRequest)
		return
	}

	tlsCtx := s.withLocalhostContext(r.Context())

	// SSRF protection
	if err := checker.CheckHostSSRF(tlsCtx, host); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: checker.SafeDialer(tlsCtx),
	}

	// InsecureSkipVerify: true allows connecting to inspect peer certificates even if expired or untrusted
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		jsonError(w, "TLS connection failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		jsonError(w, "no certificates presented", http.StatusBadRequest)
		return
	}

	leaf := state.PeerCertificates[0]
	now := time.Now()
	daysRemaining := int(time.Until(leaf.NotAfter).Hours() / 24)

	issuer := leaf.Issuer.CommonName
	if issuer == "" {
		if len(leaf.Issuer.Organization) > 0 {
			issuer = leaf.Issuer.Organization[0]
		} else {
			issuer = leaf.Issuer.String()
		}
	}

	subject := leaf.Subject.CommonName
	if subject == "" {
		if len(leaf.DNSNames) > 0 {
			subject = leaf.DNSNames[0]
		} else {
			subject = leaf.Subject.String()
		}
	}

	ocspStatus := "verified"
	verifyOpts := x509.VerifyOptions{
		Intermediates: x509.NewCertPool(),
	}
	if net.ParseIP(host) == nil {
		verifyOpts.DNSName = host
	}
	for _, c := range state.PeerCertificates[1:] {
		verifyOpts.Intermediates.AddCert(c)
	}
	if _, verifyErr := leaf.Verify(verifyOpts); verifyErr != nil {
		if now.After(leaf.NotAfter) {
			ocspStatus = "expired"
		} else if now.Before(leaf.NotBefore) {
			ocspStatus = "not yet valid"
		} else {
			ocspStatus = "untrusted: " + verifyErr.Error()
		}
	}

	cleanDomain := strings.TrimPrefix(host, "*.")
	cert := &models.TLSCertificate{
		ID:                 fmt.Sprintf("cert-%s", cleanDomain),
		Domain:             cleanDomain,
		Issuer:             issuer,
		Subject:            subject,
		SANs:               leaf.DNSNames,
		ValidFrom:          leaf.NotBefore,
		ValidUntil:         leaf.NotAfter,
		DaysRemaining:      daysRemaining,
		SerialNumber:       leaf.SerialNumber.String(),
		SignatureAlgorithm: leaf.SignatureAlgorithm.String(),
		OCSPStatus:         ocspStatus,
		LastVerifiedAt:     now,
	}

	// Persist the inspected certificate to the database
	_ = s.db.UpsertTLSCertificate(r.Context(), cert)

	jsonOK(w, cert)
}
