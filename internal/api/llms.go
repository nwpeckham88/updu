package api

import (
	"net/http"
)

const llmsTxtContent = `# updu

> updu is a modern, single-binary uptime and service monitoring tool built in Go with an embedded SvelteKit frontend. Designed for homelabs and small-to-medium businesses.

## Overview
- Architecture: Single static Go binary, embedded SQLite database with WAL mode, embedded SvelteKit GUI.
- CLI: Run 'updu serve' for web server & monitor daemon, or 'updu mcp' for local Model Context Protocol (MCP) stdio server.
- API: REST API with OpenAPI specification available at '/api/v1/openapi.json'.
- Authentication: Bearer tokens via 'Authorization: Bearer <token>' or session cookies.

## Core API Endpoints

### Services & Network Topology
- GET /api/v1/services: List all registered services, their physical zones, and reachability scopes (lan, tailnet, public).
- GET /api/v1/services/{id}: Retrieve detailed service information, including probe endpoints and latest root-cause diagnostics.
- POST /api/v1/services/{id}/probe: Trigger an immediate ad-hoc probe across all service endpoints.
- GET /api/v1/topology: Retrieve comprehensive network topology graph (nodes, zones, scopes, services, matrix diagnostics).
- GET /api/v1/zones: List physical failure domains / locations.
- GET /api/v1/scopes: List network reachability fabrics.

### TLS Certificates
- GET /api/v1/certificates: Audit discovered TLS certificates, validity windows, and days until expiration.
- POST /api/v1/certificates/test: Perform an on-demand live TLS handshake audit against a host or endpoint.

### System & Health
- GET /api/v1/system/health or /healthz: System health check and uptime.
- GET /api/v1/metrics: Prometheus-compatible metrics endpoint.
- GET /api/v1/openapi.json: Complete OpenAPI 3.1 specification.

## AI Agent Integration: Model Context Protocol (MCP)
updu includes a built-in MCP server that communicates over stdio for LLM agents (e.g. Claude Desktop, Cursor, Antigravity):

Command:
  updu mcp --db /path/to/updu.db

Available MCP Tools:
- list_services: List all configured services and status.
- get_service_details: Get service metadata, probe endpoints, and root-cause diagnostic matrix.
- get_network_topology: Fetch complete network topology and zone/scope layout.
- get_tls_certificates: Audit TLS certificate expiration across all services.
- audit_tls_handshake: Perform a live TLS handshake inspection against any endpoint.
`

func (s *Server) handleLLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(llmsTxtContent))
}
