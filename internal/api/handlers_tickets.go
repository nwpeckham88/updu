package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/updu/updu/internal/auth"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/realtime"
)

// handleListTickets lists tickets. Non-admins only see their own tickets.
func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filterUser := ""
	if user.Role != models.RoleAdmin {
		filterUser = user.Username
	} else {
		// Admin can optionally filter by user query param
		filterUser = r.URL.Query().Get("user")
	}

	tickets, err := s.db.ListTickets(r.Context(), filterUser)
	if err != nil {
		jsonError(w, "failed to list tickets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by status if requested
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusFilter != "" {
		var filtered []*models.Ticket
		for _, t := range tickets {
			if strings.EqualFold(string(t.Status), statusFilter) {
				filtered = append(filtered, t)
			}
		}
		tickets = filtered
	}

	if tickets == nil {
		tickets = []*models.Ticket{}
	}

	jsonOK(w, tickets)
}

// handleCreateTicket creates a new ticket.
func (s *Server) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Severity    string  `json:"severity"`
		ServiceID   *string `json:"service_id,omitempty"`
		MonitorID   *string `json:"monitor_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" {
		jsonError(w, "title is required", http.StatusBadRequest)
		return
	}
	if req.Description == "" {
		jsonError(w, "description is required", http.StatusBadRequest)
		return
	}

	severity := strings.ToLower(strings.TrimSpace(req.Severity))
	switch severity {
	case "low", "medium", "high":
		// valid
	default:
		severity = "medium"
	}

	var serviceName string
	if req.ServiceID != nil && *req.ServiceID != "" {
		svc, err := s.db.GetService(r.Context(), *req.ServiceID)
		if err != nil || svc == nil {
			jsonError(w, "specified service not found", http.StatusBadRequest)
			return
		}
		if !canAccessService(user, svc) {
			jsonError(w, "forbidden: cannot access this service", http.StatusForbidden)
			return
		}
		serviceName = svc.Name
	}

	id, err := auth.GenerateID()
	if err != nil {
		id = "tkt_" + time.Now().Format("20060102150405")
	} else {
		id = "tkt_" + id
	}

	now := time.Now()
	ticket := &models.Ticket{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TicketStatusOpen,
		Severity:    severity,
		ServiceID:   req.ServiceID,
		ServiceName: serviceName,
		MonitorID:   req.MonitorID,
		CreatedBy:   user.Username,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.db.CreateTicket(r.Context(), ticket); err != nil {
		jsonError(w, "failed to create ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.sse != nil {
		s.sse.Broadcast(realtime.Event{
			Type: "ticket:create",
			Data: ticket,
		})
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, ticket)
}

// handleGetTicket retrieves a ticket by ID.
func (s *Server) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	ticket, err := s.db.GetTicket(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to get ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if ticket == nil {
		jsonError(w, "ticket not found", http.StatusNotFound)
		return
	}

	if user.Role != models.RoleAdmin && ticket.CreatedBy != user.Username {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	jsonOK(w, ticket)
}

// handleUpdateTicket updates an existing ticket.
func (s *Server) handleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	ticket, err := s.db.GetTicket(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to get ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if ticket == nil {
		jsonError(w, "ticket not found", http.StatusNotFound)
		return
	}

	if user.Role != models.RoleAdmin && ticket.CreatedBy != user.Username {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Title       *string `json:"title,omitempty"`
		Description *string `json:"description,omitempty"`
		Status      *string `json:"status,omitempty"`
		Severity    *string `json:"severity,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	now := time.Now()

	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		ticket.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil && strings.TrimSpace(*req.Description) != "" {
		ticket.Description = strings.TrimSpace(*req.Description)
	}
	if req.Severity != nil {
		sev := strings.ToLower(strings.TrimSpace(*req.Severity))
		if sev == "low" || sev == "medium" || sev == "high" {
			ticket.Severity = sev
		}
	}

	if req.Status != nil {
		targetStatus := models.TicketStatus(strings.ToLower(strings.TrimSpace(*req.Status)))
		// If non-admin viewer, they may only set status to closed
		if user.Role != models.RoleAdmin && targetStatus != models.TicketStatusClosed {
			jsonError(w, "viewers may only close their own tickets", http.StatusForbidden)
			return
		}

		switch targetStatus {
		case models.TicketStatusOpen, models.TicketStatusInProgress:
			ticket.Status = targetStatus
			ticket.ResolvedAt = nil
		case models.TicketStatusResolved, models.TicketStatusClosed:
			ticket.Status = targetStatus
			ticket.ResolvedAt = &now
		default:
			jsonError(w, "invalid status", http.StatusBadRequest)
			return
		}
	}

	ticket.UpdatedAt = now

	if err := s.db.UpdateTicket(r.Context(), ticket); err != nil {
		jsonError(w, "failed to update ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.sse != nil {
		s.sse.Broadcast(realtime.Event{
			Type: "ticket:update",
			Data: ticket,
		})
	}

	jsonOK(w, ticket)
}

// handleDeleteTicket deletes a ticket. Admin only.
func (s *Server) handleDeleteTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.db.DeleteTicket(r.Context(), id); err != nil {
		jsonError(w, "failed to delete ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.sse != nil {
		s.sse.Broadcast(realtime.Event{
			Type: "ticket:delete",
			Data: map[string]string{"id": id},
		})
	}

	w.WriteHeader(http.StatusNoContent)
}
