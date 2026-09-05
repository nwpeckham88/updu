package diagnostic_test

import (
	"testing"

	"github.com/updu/updu/internal/diagnostic"
	"github.com/updu/updu/internal/models"
)

func TestEvaluateServiceHealth_AllUp(t *testing.T) {
	service := &models.Service{
		ID:   "srv-1",
		Name: "Vaultwarden",
		Endpoints: []*models.ServiceEndpoint{
			{ID: "ep-lan", Name: "LAN", ScopeID: models.ScopeLAN},
			{ID: "ep-wan", Name: "WAN", ScopeID: models.ScopePublic},
		},
	}

	checks := map[string]*models.EndpointCheck{
		"ep-lan": {Status: models.StatusUp},
		"ep-wan": {Status: models.StatusUp},
	}

	diag := diagnostic.EvaluateServiceHealth(service, checks)
	if diag.Status != models.StatusUp {
		t.Fatalf("expected StatusUp, got %s", diag.Status)
	}
	if diag.HealthyCount != 2 {
		t.Errorf("expected 2 healthy, got %d", diag.HealthyCount)
	}
}

func TestEvaluateServiceHealth_TotalOutage(t *testing.T) {
	service := &models.Service{
		ID:   "srv-1",
		Name: "Vaultwarden",
		Endpoints: []*models.ServiceEndpoint{
			{ID: "ep-lan", Name: "LAN", ScopeID: models.ScopeLAN},
			{ID: "ep-wan", Name: "WAN", ScopeID: models.ScopePublic},
		},
	}

	checks := map[string]*models.EndpointCheck{
		"ep-lan": {Status: models.StatusDown, Message: "connection refused"},
		"ep-wan": {Status: models.StatusDown, Message: "timeout"},
	}

	diag := diagnostic.EvaluateServiceHealth(service, checks)
	if diag.Status != models.StatusDown {
		t.Fatalf("expected StatusDown, got %s", diag.Status)
	}
	if diag.HealthyCount != 0 {
		t.Errorf("expected 0 healthy, got %d", diag.HealthyCount)
	}
}

func TestEvaluateServiceHealth_ReverseProxyFailure(t *testing.T) {
	service := &models.Service{
		ID:   "srv-1",
		Name: "Nextcloud",
		Endpoints: []*models.ServiceEndpoint{
			{ID: "ep-lan", Name: "Direct Docker", ScopeID: models.ScopeLAN},
			{ID: "ep-wan", Name: "Public Domain", ScopeID: models.ScopePublic},
		},
	}

	// Backend container is up on LAN, but public reverse proxy / Cloudflare fails
	checks := map[string]*models.EndpointCheck{
		"ep-lan": {Status: models.StatusUp},
		"ep-wan": {Status: models.StatusDown, Message: "502 Bad Gateway"},
	}

	diag := diagnostic.EvaluateServiceHealth(service, checks)
	if diag.Status != models.StatusDegraded {
		t.Fatalf("expected StatusDegraded, got %s", diag.Status)
	}
	if diag.Summary != "Ingress / Reverse Proxy Failure" {
		t.Errorf("unexpected summary: %s", diag.Summary)
	}
	if diag.ActionHint == "" {
		t.Error("expected action hint for reverse proxy failure")
	}
}

func TestEvaluateServiceHealth_TailnetActivePublicDown(t *testing.T) {
	service := &models.Service{
		ID:   "srv-1",
		Name: "HomeAssistant",
		Endpoints: []*models.ServiceEndpoint{
			{ID: "ep-tailnet", Name: "Tailnet Mesh", ScopeID: models.ScopeTailnet},
			{ID: "ep-wan", Name: "WAN Domain", ScopeID: models.ScopePublic},
		},
	}

	checks := map[string]*models.EndpointCheck{
		"ep-tailnet": {Status: models.StatusUp},
		"ep-wan":     {Status: models.StatusDown, Message: "timeout"},
	}

	diag := diagnostic.EvaluateServiceHealth(service, checks)
	if diag.Status != models.StatusDegraded {
		t.Fatalf("expected StatusDegraded, got %s", diag.Status)
	}
	if diag.Summary != "Public Gateway Outage (Tailnet Active)" {
		t.Errorf("unexpected summary: %s", diag.Summary)
	}
}
