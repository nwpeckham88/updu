package diagnostic

import (
	"fmt"
	"strings"

	"github.com/updu/updu/internal/models"
)

// ServiceDiagnosis represents the diagnostic analysis of a service across its vantage endpoints.
type ServiceDiagnosis struct {
	Status         models.MonitorStatus `json:"status"`
	Summary        string               `json:"summary"`
	ProbableCause  string               `json:"probable_cause"`
	ActionHint     string               `json:"action_hint"`
	HealthyCount   int                  `json:"healthy_count"`
	TotalCount     int                  `json:"total_count"`
	ProbeBreakdown []string             `json:"probe_breakdown"`
}

// EvaluateServiceHealth analyzes the collection of endpoint results for a service and diagnoses root causes.
func EvaluateServiceHealth(service *models.Service, endpointChecks map[string]*models.EndpointCheck) *ServiceDiagnosis {
	if len(service.Endpoints) == 0 {
		return &ServiceDiagnosis{
			Status:        models.StatusPending,
			Summary:       "No endpoints configured",
			ProbableCause: "Service has no active endpoints to monitor.",
			ActionHint:    "Add at least one endpoint to begin monitoring.",
		}
	}

	totalEndpoints := len(service.Endpoints)
	healthyEndpoints := 0

	var lanUp, lanDown bool
	var tailnetUp, tailnetDown bool
	var publicUp, publicDown bool
	var breakdowns []string

	for _, ep := range service.Endpoints {
		check, exists := endpointChecks[ep.ID]
		if !exists || check.Status == models.StatusPending {
			breakdowns = append(breakdowns, fmt.Sprintf("%s (%s): Pending", ep.Name, ep.ScopeID))
			continue
		}

		isUp := check.Status == models.StatusUp
		if isUp {
			healthyEndpoints++
		}

		statusStr := "UP"
		if !isUp {
			statusStr = "DOWN"
			if check.Message != "" {
				statusStr = fmt.Sprintf("DOWN (%s)", check.Message)
			}
		}
		breakdowns = append(breakdowns, fmt.Sprintf("%s [%s]: %s", ep.Name, ep.ScopeID, statusStr))

		switch ep.ScopeID {
		case models.ScopeLAN:
			if isUp {
				lanUp = true
			} else {
				lanDown = true
			}
		case models.ScopeTailnet:
			if isUp {
				tailnetUp = true
			} else {
				tailnetDown = true
			}
		case models.ScopePublic:
			if isUp {
				publicUp = true
			} else {
				publicDown = true
			}
		}
	}

	// 1. All Endpoints Up
	if healthyEndpoints == totalEndpoints {
		return &ServiceDiagnosis{
			Status:         models.StatusUp,
			Summary:        "All probe paths healthy",
			ProbableCause:  "Service operating nominally across all vantage points.",
			ActionHint:     "",
			HealthyCount:   healthyEndpoints,
			TotalCount:     totalEndpoints,
			ProbeBreakdown: breakdowns,
		}
	}

	// 2. Total Outage (All Endpoints Down)
	if healthyEndpoints == 0 {
		return &ServiceDiagnosis{
			Status:         models.StatusDown,
			Summary:        "Total Outage: All endpoints unreachable",
			ProbableCause:  "Target machine powered off, container crashed, or network interface down.",
			ActionHint:     "Verify host power state, container status, and local process availability.",
			HealthyCount:   0,
			TotalCount:     totalEndpoints,
			ProbeBreakdown: breakdowns,
		}
	}

	// 3. Multi-path Differential Diagnostics
	diag := &ServiceDiagnosis{
		Status:         models.StatusDegraded,
		HealthyCount:   healthyEndpoints,
		TotalCount:     totalEndpoints,
		ProbeBreakdown: breakdowns,
	}

	// Case A: LAN OK, but Public WAN Down
	if lanUp && publicDown && !publicUp {
		diag.Summary = "Ingress / Reverse Proxy Failure"
		diag.ProbableCause = "Backend service responds on local LAN, but public ingress/reverse proxy or port forward is down."
		diag.ActionHint = "Inspect reverse proxy (Caddy/Traefik/Nginx), Cloudflare tunnel, or router WAN port-forwarding."
		return diag
	}

	// Case B: Public WAN OK, but LAN Down
	if publicUp && lanDown && !lanUp {
		diag.Summary = "Internal LAN Routing Failure"
		diag.ProbableCause = "External ingress is serving traffic (possibly from CDN cache or alternate route), but direct local LAN probe failed."
		diag.ActionHint = "Check local RFC1918 IP binding, local firewall/subnet ACLs, or container port publishing."
		return diag
	}

	// Case C: Tailnet OK, but Public WAN Down
	if tailnetUp && publicDown && !publicUp {
		diag.Summary = "Public Gateway Outage (Tailnet Active)"
		diag.ProbableCause = "Encrypted overlay mesh connection is functional, but public internet ingress is dead."
		diag.ActionHint = "Check external ISP connection, public DNS records, or gateway router."
		return diag
	}

	// Case D: Tailnet OK, but LAN Down
	if tailnetUp && lanDown && !lanUp {
		diag.Summary = "Local LAN Interface Isolated"
		diag.ProbableCause = "Host is reachable over Tailscale/WireGuard overlay, but LAN interface is unreachable."
		diag.ActionHint = "Check physical Ethernet cable, local switch port, or DHCP lease on the LAN."
		return diag
	}

	// Case E: LAN OK, but Tailnet Down
	if lanUp && tailnetDown && !tailnetUp {
		diag.Summary = "Tailnet Mesh Disconnected"
		diag.ProbableCause = "Host is reachable on local LAN, but Tailscale/WireGuard daemon is disconnected."
		diag.ActionHint = "Check tailscaled service status or VPN key expiration."
		return diag
	}

	// Case E: Partial Failure across endpoints
	diag.Summary = fmt.Sprintf("Degraded (%d/%d endpoints operational)", healthyEndpoints, totalEndpoints)
	diag.ProbableCause = fmt.Sprintf("One or more probe paths failed: %s", strings.Join(breakdowns, "; "))
	diag.ActionHint = "Check the failing endpoint logs and latency metrics."
	return diag
}
