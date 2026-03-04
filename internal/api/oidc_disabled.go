//go:build !oidc

package api

import "net/http"

// registerOIDCRoutes is a no-op when the "oidc" build tag is not provided.
func registerOIDCRoutes(mux *http.ServeMux, s *Server) {
	// No OIDC routes registered
}
