package api

import (
	"net/http"

	"github.com/updu/updu/internal/version"
)

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonOK(w, map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "updu API",
			"version":     version.Version,
			"description": "Single-binary uptime monitoring API for self-hosted deployments.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"sessionCookie": map[string]any{
					"type": "apiKey",
					"in":   "cookie",
					"name": "updu_session",
				},
				"bearerAuth": map[string]any{
					"type":   "http",
					"scheme": "bearer",
				},
			},
		},
		"paths": map[string]any{
			"/healthz": map[string]any{
				"get": map[string]any{
					"summary":   "Health check",
					"security":  []any{},
					"responses": map[string]any{"200": map[string]any{"description": "Service health"}},
				},
			},
			"/api/v1/openapi.json": map[string]any{
				"get": map[string]any{
					"summary":   "OpenAPI contract",
					"security":  []any{},
					"responses": map[string]any{"200": map[string]any{"description": "OpenAPI document"}},
				},
			},
			"/api/v1/monitors": map[string]any{
				"get": map[string]any{
					"summary":  "List monitors",
					"security": []any{map[string]any{"sessionCookie": []any{}}, map[string]any{"bearerAuth": []any{}}},
				},
				"post": map[string]any{
					"summary":  "Create monitor",
					"security": []any{map[string]any{"sessionCookie": []any{}}, map[string]any{"bearerAuth": []any{}}},
				},
			},
			"/api/v1/admin/api-tokens": map[string]any{
				"get": map[string]any{
					"summary":  "List API tokens",
					"security": []any{map[string]any{"sessionCookie": []any{}}},
				},
				"post": map[string]any{
					"summary":  "Create API token",
					"security": []any{map[string]any{"sessionCookie": []any{}}},
				},
			},
			"/api/v1/admin/api-tokens/{id}": map[string]any{
				"delete": map[string]any{
					"summary":  "Revoke API token",
					"security": []any{map[string]any{"sessionCookie": []any{}}},
				},
			},
			"/api/v1/audit-logs": map[string]any{
				"get": map[string]any{
					"summary":  "List audit logs",
					"security": []any{map[string]any{"sessionCookie": []any{}}},
				},
			},
		},
	})
}
