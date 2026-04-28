package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/updu/updu/internal/version"
)

type openAPISecurityProfile string

const (
	openAPIPublicSecurity       openAPISecurityProfile = "public"
	openAPIUserSecurity         openAPISecurityProfile = "user"
	openAPIAdminSecurity        openAPISecurityProfile = "admin"
	openAPISessionAdminSecurity openAPISecurityProfile = "session_admin"
)

type openAPIDocument struct {
	OpenAPI    string                    `json:"openapi"`
	Info       openAPIInfo               `json:"info"`
	Tags       []openAPITag              `json:"tags,omitempty"`
	Components map[string]any            `json:"components"`
	Paths      map[string]map[string]any `json:"paths"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type openAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type openAPIParameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Schema      any    `json:"schema"`
}

type openAPIRequestBodySpec struct {
	Description string
	Required    bool
	ContentType string
	Schema      any
}

type openAPIResponseSpec struct {
	Description string
	ContentType string
	Schema      any
}

type openAPIRoute struct {
	Method          string
	Path            string
	Tag             string
	Summary         string
	Description     string
	Security        openAPISecurityProfile
	Parameters      []openAPIParameter
	RequestBody     *openAPIRequestBodySpec
	SuccessStatus   string
	SuccessResponse openAPIResponseSpec
	ErrorStatuses   []string
	Responses       map[string]openAPIResponseSpec
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonOK(w, buildOpenAPIDocument())
}

func buildOpenAPIDocument() openAPIDocument {
	routes := append(baseOpenAPIRoutes(), openAPIOIDCRoutes()...)
	return openAPIDocument{
		OpenAPI: "3.1.0",
		Info: openAPIInfo{
			Title:       "updu API",
			Version:     version.Version,
			Description: "Single-binary uptime monitoring API for self-hosted deployments.",
		},
		Tags:       openAPITags(),
		Components: openAPIComponents(),
		Paths:      buildOpenAPIPaths(routes),
	}
}

func openAPITags() []openAPITag {
	return []openAPITag{
		{Name: "Auth", Description: "Authentication, setup, and session management routes."},
		{Name: "Heartbeat", Description: "Heartbeat ingest and token-based keepalive routes."},
		{Name: "System", Description: "Health, backup, metrics, update, and runtime administration routes."},
		{Name: "Documentation", Description: "Machine-readable API documentation and frontend assets."},
		{Name: "Events", Description: "Realtime and historical event feeds."},
		{Name: "Monitors", Description: "Monitor CRUD, diagnostics, and uptime data."},
		{Name: "Dashboard", Description: "Dashboard and analytics data surfaces."},
		{Name: "Status Pages", Description: "Public and authenticated status page management routes."},
		{Name: "Notifications", Description: "Notification channel management and test delivery."},
		{Name: "Incidents", Description: "Incident lifecycle routes."},
		{Name: "Maintenance", Description: "Maintenance window lifecycle routes."},
		{Name: "Groups", Description: "Monitor grouping routes."},
		{Name: "Admin", Description: "Admin-only user, API token, and audit operations."},
		{Name: "Settings", Description: "Instance settings and customization routes."},
	}
}

func openAPIComponents() map[string]any {
	return map[string]any{
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
		"schemas": map[string]any{
			"GenericObject": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
			"ErrorResponse": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"error": map[string]any{"type": "string"},
				},
				"required": []string{"error"},
			},
			"MessageResponse": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message": map[string]any{"type": "string"},
				},
				"required": []string{"message"},
			},
		},
	}
}

func buildOpenAPIPaths(routes []openAPIRoute) map[string]map[string]any {
	paths := make(map[string]map[string]any)
	for _, route := range routes {
		pathItem, ok := paths[route.Path]
		if !ok {
			pathItem = map[string]any{}
			paths[route.Path] = pathItem
		}
		pathItem[strings.ToLower(route.Method)] = route.operation()
	}
	return paths
}

func (route openAPIRoute) operation() map[string]any {
	operation := map[string]any{
		"tags":        []string{route.Tag},
		"summary":     route.Summary,
		"operationId": openAPIOperationID(route.Method, route.Path),
		"responses":   route.responses(),
	}
	if route.Description != "" {
		operation["description"] = route.Description
	}
	if len(route.Parameters) > 0 {
		operation["parameters"] = route.Parameters
	}
	if route.RequestBody != nil {
		operation["requestBody"] = route.RequestBody.document()
	}
	if security := openAPISecurityRequirements(route.Security); len(security) > 0 {
		operation["security"] = security
	}
	return operation
}

func (body openAPIRequestBodySpec) document() map[string]any {
	contentType := body.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	document := map[string]any{
		"required": body.Required,
		"content": map[string]any{
			contentType: map[string]any{
				"schema": body.schema(),
			},
		},
	}
	if body.Description != "" {
		document["description"] = body.Description
	}
	return document
}

func (body openAPIRequestBodySpec) schema() any {
	if body.Schema != nil {
		return body.Schema
	}
	return schemaRef("GenericObject")
}

func (route openAPIRoute) responses() map[string]any {
	if route.Responses != nil {
		responses := make(map[string]any, len(route.Responses))
		for status, response := range route.Responses {
			responses[status] = response.document()
		}
		return responses
	}

	responses := map[string]any{}
	status := route.SuccessStatus
	if status == "" {
		status = "200"
	}
	responses[status] = route.SuccessResponse.document()
	for _, errorStatus := range route.ErrorStatuses {
		responses[errorStatus] = openAPIErrorResponse(errorStatus).document()
	}
	return responses
}

func (response openAPIResponseSpec) document() map[string]any {
	document := map[string]any{
		"description": response.description(),
	}
	if response.Schema == nil {
		return document
	}
	contentType := response.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	document["content"] = map[string]any{
		contentType: map[string]any{
			"schema": response.Schema,
		},
	}
	return document
}

func (response openAPIResponseSpec) description() string {
	if response.Description != "" {
		return response.Description
	}
	return "Successful response"
}

func openAPIErrorResponse(status string) openAPIResponseSpec {
	descriptions := map[string]string{
		"400": "Bad request",
		"401": "Authentication required",
		"403": "Forbidden",
		"404": "Not found",
		"429": "Rate limited",
		"500": "Internal server error",
	}
	description := descriptions[status]
	if description == "" {
		description = "Error response"
	}
	return openAPIResponseSpec{
		Description: description,
		ContentType: "application/json",
		Schema:      schemaRef("ErrorResponse"),
	}
}

func openAPISecurityRequirements(profile openAPISecurityProfile) []map[string][]string {
	switch profile {
	case openAPIUserSecurity, openAPIAdminSecurity:
		return []map[string][]string{{"sessionCookie": {}}, {"bearerAuth": {}}}
	case openAPISessionAdminSecurity:
		return []map[string][]string{{"sessionCookie": {}}}
	default:
		return nil
	}
}

func openAPIOperationID(method, path string) string {
	cleaned := strings.Trim(path, "/")
	if cleaned == "" {
		cleaned = "root"
	}
	replacer := strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_")
	return strings.ToLower(method) + "_" + replacer.Replace(cleaned)
}

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func arraySchema(item any) map[string]any {
	return map[string]any{"type": "array", "items": item}
}

func genericJSONResponse(description string) openAPIResponseSpec {
	return openAPIResponseSpec{
		Description: description,
		ContentType: "application/json",
		Schema:      schemaRef("GenericObject"),
	}
}

func messageJSONResponse(description string) openAPIResponseSpec {
	return openAPIResponseSpec{
		Description: description,
		ContentType: "application/json",
		Schema:      schemaRef("MessageResponse"),
	}
}

func genericJSONRequest(description string) *openAPIRequestBodySpec {
	return &openAPIRequestBodySpec{
		Description: description,
		Required:    true,
		ContentType: "application/json",
		Schema:      schemaRef("GenericObject"),
	}
}

func pathParameter(name, description string) openAPIParameter {
	return openAPIParameter{
		Name:        name,
		In:          "path",
		Description: description,
		Required:    true,
		Schema:      map[string]any{"type": "string"},
	}
}

func queryParameter(name, description string) openAPIParameter {
	return openAPIParameter{
		Name:        name,
		In:          "query",
		Description: description,
		Required:    false,
		Schema:      map[string]any{"type": "string"},
	}
}

func integerQueryParameter(name, description string) openAPIParameter {
	return openAPIParameter{
		Name:        name,
		In:          "query",
		Description: description,
		Required:    false,
		Schema:      map[string]any{"type": "integer"},
	}
}

func sortOpenAPIRoutes(routes []openAPIRoute) []openAPIRoute {
	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes
}

func baseOpenAPIRoutes() []openAPIRoute {
	return sortOpenAPIRoutes([]openAPIRoute{
		{Method: "POST", Path: "/api/v1/auth/login", Tag: "Auth", Summary: "Log in with username and password", RequestBody: genericJSONRequest("Username and password credentials."), SuccessStatus: "200", SuccessResponse: messageJSONResponse("Authenticated successfully."), ErrorStatuses: []string{"400", "401", "429", "500"}},
		{Method: "POST", Path: "/api/v1/auth/register", Tag: "Auth", Summary: "Register a local user", RequestBody: genericJSONRequest("Registration payload."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Registered user."), ErrorStatuses: []string{"400", "403", "429", "500"}},
		{Method: "GET", Path: "/api/v1/auth/setup", Tag: "Auth", Summary: "Check whether first-run setup is required", SuccessStatus: "200", SuccessResponse: genericJSONResponse("Current setup requirement state."), ErrorStatuses: []string{"500"}},
		{Method: "GET", Path: "/api/v1/auth/providers", Tag: "Auth", Summary: "List enabled authentication providers", SuccessStatus: "200", SuccessResponse: genericJSONResponse("Available authentication providers."), ErrorStatuses: []string{"500"}},
		{Method: "POST", Path: "/api/v1/auth/logout", Tag: "Auth", Summary: "Log out the current session", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Logout succeeded."), ErrorStatuses: []string{"401", "500"}},
		{Method: "GET", Path: "/api/v1/auth/session", Tag: "Auth", Summary: "Get the authenticated session user", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Current session user."), ErrorStatuses: []string{"401", "500"}},
		{Method: "PUT", Path: "/api/v1/auth/password", Tag: "Auth", Summary: "Change the current user's password", Security: openAPIUserSecurity, RequestBody: genericJSONRequest("Current and new password payload."), SuccessStatus: "200", SuccessResponse: messageJSONResponse("Password updated."), ErrorStatuses: []string{"400", "401", "500"}},
		{Method: "POST", Path: "/api/v1/status-pages/{slug}/unlock", Tag: "Status Pages", Summary: "Unlock a password-protected public status page", Parameters: []openAPIParameter{pathParameter("slug", "Status page slug.")}, RequestBody: genericJSONRequest("Unlock request payload."), SuccessStatus: "200", SuccessResponse: messageJSONResponse("Status page unlocked."), ErrorStatuses: []string{"400", "404", "500"}},
		{Method: "GET", Path: "/api/v1/status-pages/{slug}", Tag: "Status Pages", Summary: "Fetch a public status page by slug", Parameters: []openAPIParameter{pathParameter("slug", "Status page slug.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Public status page payload."), ErrorStatuses: []string{"404", "500"}},
		{Method: "POST", Path: "/api/v1/heartbeat/{slug}", Tag: "Heartbeat", Summary: "Report a heartbeat ping by monitor slug using the legacy POST-only route", Parameters: []openAPIParameter{pathParameter("slug", "Heartbeat monitor slug.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Heartbeat accepted."), ErrorStatuses: []string{"404", "500"}},
		{Method: "GET", Path: "/heartbeat/{token}", Tag: "Heartbeat", Summary: "Trigger a heartbeat using the recommended token route", Parameters: []openAPIParameter{pathParameter("token", "Heartbeat token.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Heartbeat accepted."), ErrorStatuses: []string{"404", "500"}},
		{Method: "POST", Path: "/heartbeat/{token}", Tag: "Heartbeat", Summary: "Post a heartbeat using the recommended token route", Parameters: []openAPIParameter{pathParameter("token", "Heartbeat token.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Heartbeat accepted."), ErrorStatuses: []string{"404", "500"}},
		{Method: "PUT", Path: "/heartbeat/{token}", Tag: "Heartbeat", Summary: "Update a heartbeat using the recommended token route", Parameters: []openAPIParameter{pathParameter("token", "Heartbeat token.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Heartbeat accepted."), ErrorStatuses: []string{"404", "500"}},
		{Method: "GET", Path: "/api/v1/system/health", Tag: "System", Summary: "Run the JSON health check", SuccessStatus: "200", SuccessResponse: genericJSONResponse("Health status payload."), ErrorStatuses: []string{"500"}},
		{Method: "GET", Path: "/healthz", Tag: "System", Summary: "Run the lightweight health check", SuccessStatus: "200", SuccessResponse: genericJSONResponse("Health status payload."), ErrorStatuses: []string{"500"}},
		{Method: "GET", Path: "/api/v1/openapi.json", Tag: "Documentation", Summary: "Fetch the OpenAPI document", SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "OpenAPI document.", ContentType: "application/json", Schema: map[string]any{"type": "object"}}},
		{Method: "GET", Path: "/api/v1/metrics", Tag: "System", Summary: "Fetch Prometheus metrics", SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Prometheus text exposition.", ContentType: "text/plain", Schema: map[string]any{"type": "string"}}, ErrorStatuses: []string{"401", "500"}},
		{Method: "GET", Path: "/api/v1/custom.css", Tag: "Documentation", Summary: "Fetch custom CSS overrides", SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "CSS overrides.", ContentType: "text/css", Schema: map[string]any{"type": "string"}}, ErrorStatuses: []string{"500"}},
		{Method: "GET", Path: "/api/v1/events", Tag: "Events", Summary: "Subscribe to realtime server-sent events", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Server-sent event stream.", ContentType: "text/event-stream", Schema: map[string]any{"type": "string"}}, ErrorStatuses: []string{"401", "500"}},
		{Method: "GET", Path: "/api/v1/monitors", Tag: "Monitors", Summary: "List monitors", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Monitor list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "500"}},
		{Method: "POST", Path: "/api/v1/monitors", Tag: "Monitors", Summary: "Create a monitor", Security: openAPIAdminSecurity, RequestBody: genericJSONRequest("Monitor definition."), SuccessStatus: "201", SuccessResponse: genericJSONResponse("Created monitor."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "POST", Path: "/api/v1/monitors/test", Tag: "Monitors", Summary: "Test monitor configuration without saving", Security: openAPIAdminSecurity, RequestBody: genericJSONRequest("Monitor definition to test."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Test result payload."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/monitors/{id}", Tag: "Monitors", Summary: "Fetch a monitor by ID", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Monitor details."), ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "POST", Path: "/api/v1/monitors/{id}/investigate", Tag: "Monitors", Summary: "Set or clear an in-memory investigation marker", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, RequestBody: genericJSONRequest("Investigation marker payload."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Investigation marker state."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "PUT", Path: "/api/v1/monitors/{id}", Tag: "Monitors", Summary: "Update a monitor", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, RequestBody: genericJSONRequest("Monitor definition."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Updated monitor."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/monitors/{id}", Tag: "Monitors", Summary: "Delete a monitor", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/monitors/{id}/checks", Tag: "Monitors", Summary: "List recent checks for a monitor", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Recent checks.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "GET", Path: "/api/v1/monitors/{id}/events", Tag: "Monitors", Summary: "List events for a monitor", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Monitor event list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "GET", Path: "/api/v1/monitors/{id}/uptime", Tag: "Monitors", Summary: "Fetch uptime aggregates for a monitor", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Monitor ID.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Uptime summary."), ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "GET", Path: "/api/v1/dashboard", Tag: "Dashboard", Summary: "Fetch dashboard summary data", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Dashboard payload."), ErrorStatuses: []string{"401", "500"}},
		{Method: "GET", Path: "/api/v1/stats", Tag: "Dashboard", Summary: "Fetch aggregated statistics", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Statistics payload."), ErrorStatuses: []string{"401", "500"}},
		{Method: "GET", Path: "/api/v1/events/history", Tag: "Events", Summary: "List historical events", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Historical event list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "500"}},
		{Method: "GET", Path: "/api/v1/status-pages", Tag: "Status Pages", Summary: "List status pages", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Status page list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "500"}},
		{Method: "POST", Path: "/api/v1/status-pages", Tag: "Status Pages", Summary: "Create a status page", Security: openAPIAdminSecurity, RequestBody: genericJSONRequest("Status page definition."), SuccessStatus: "201", SuccessResponse: genericJSONResponse("Created status page."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/status-pages/{id}/detail", Tag: "Status Pages", Summary: "Fetch a status page by ID for editing", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Status page ID.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Status page details."), ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "PUT", Path: "/api/v1/status-pages/{id}", Tag: "Status Pages", Summary: "Update a status page", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Status page ID.")}, RequestBody: genericJSONRequest("Status page definition."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Updated status page."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/status-pages/{id}", Tag: "Status Pages", Summary: "Delete a status page", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Status page ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/notifications", Tag: "Notifications", Summary: "List notification channels", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Notification channels.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "POST", Path: "/api/v1/notifications", Tag: "Notifications", Summary: "Create a notification channel", Security: openAPIAdminSecurity, RequestBody: genericJSONRequest("Notification channel definition."), SuccessStatus: "201", SuccessResponse: genericJSONResponse("Created notification channel."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/notifications/{id}", Tag: "Notifications", Summary: "Fetch a notification channel", Security: openAPISessionAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Notification channel ID.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Notification channel details."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "PUT", Path: "/api/v1/notifications/{id}", Tag: "Notifications", Summary: "Update a notification channel", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Notification channel ID.")}, RequestBody: genericJSONRequest("Notification channel definition."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Updated notification channel."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/notifications/{id}", Tag: "Notifications", Summary: "Delete a notification channel", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Notification channel ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "POST", Path: "/api/v1/notifications/{id}/test", Tag: "Notifications", Summary: "Dispatch a test notification", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Notification channel ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Test notification dispatched."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/incidents", Tag: "Incidents", Summary: "List incidents", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Incident list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "500"}},
		{Method: "POST", Path: "/api/v1/incidents", Tag: "Incidents", Summary: "Create an incident", Security: openAPIAdminSecurity, RequestBody: genericJSONRequest("Incident payload."), SuccessStatus: "201", SuccessResponse: genericJSONResponse("Created incident."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/incidents/{id}", Tag: "Incidents", Summary: "Fetch an incident", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Incident ID.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Incident details."), ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "PUT", Path: "/api/v1/incidents/{id}", Tag: "Incidents", Summary: "Update an incident", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Incident ID.")}, RequestBody: genericJSONRequest("Incident payload."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Updated incident."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/incidents/{id}", Tag: "Incidents", Summary: "Delete an incident", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Incident ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/maintenance", Tag: "Maintenance", Summary: "List maintenance windows", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Maintenance window list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "500"}},
		{Method: "POST", Path: "/api/v1/maintenance", Tag: "Maintenance", Summary: "Create a maintenance window", Security: openAPIAdminSecurity, RequestBody: genericJSONRequest("Maintenance window payload."), SuccessStatus: "201", SuccessResponse: genericJSONResponse("Created maintenance window."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/maintenance/{id}", Tag: "Maintenance", Summary: "Fetch a maintenance window", Security: openAPIUserSecurity, Parameters: []openAPIParameter{pathParameter("id", "Maintenance window ID.")}, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Maintenance window details."), ErrorStatuses: []string{"401", "404", "500"}},
		{Method: "PUT", Path: "/api/v1/maintenance/{id}", Tag: "Maintenance", Summary: "Update a maintenance window", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Maintenance window ID.")}, RequestBody: genericJSONRequest("Maintenance window payload."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Updated maintenance window."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/maintenance/{id}", Tag: "Maintenance", Summary: "Delete a maintenance window", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "Maintenance window ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/groups", Tag: "Groups", Summary: "List monitor groups", Security: openAPIUserSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Group list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "500"}},
		{Method: "PUT", Path: "/api/v1/groups/{name}", Tag: "Groups", Summary: "Rename a monitor group", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("name", "Existing group name.")}, RequestBody: genericJSONRequest("Group update payload."), SuccessStatus: "200", SuccessResponse: messageJSONResponse("Group updated."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/groups/{name}", Tag: "Groups", Summary: "Delete a monitor group", Security: openAPIAdminSecurity, Parameters: []openAPIParameter{pathParameter("name", "Group name.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/admin/users", Tag: "Admin", Summary: "List users", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "User list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "PUT", Path: "/api/v1/admin/users/{id}/role", Tag: "Admin", Summary: "Update a user's role", Security: openAPISessionAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "User ID.")}, RequestBody: genericJSONRequest("Role update payload."), SuccessStatus: "200", SuccessResponse: messageJSONResponse("Role updated."), ErrorStatuses: []string{"400", "401", "403", "404", "500"}},
		{Method: "DELETE", Path: "/api/v1/admin/users/{id}", Tag: "Admin", Summary: "Delete a user", Security: openAPISessionAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "User ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Deletion confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/admin/api-tokens", Tag: "Admin", Summary: "List API tokens", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "API token list.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "POST", Path: "/api/v1/admin/api-tokens", Tag: "Admin", Summary: "Create an API token", Security: openAPISessionAdminSecurity, RequestBody: genericJSONRequest("API token creation payload."), SuccessStatus: "201", SuccessResponse: genericJSONResponse("Created API token secret."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "DELETE", Path: "/api/v1/admin/api-tokens/{id}", Tag: "Admin", Summary: "Revoke an API token", Security: openAPISessionAdminSecurity, Parameters: []openAPIParameter{pathParameter("id", "API token ID.")}, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Revocation confirmation."), ErrorStatuses: []string{"401", "403", "404", "500"}},
		{Method: "GET", Path: "/api/v1/audit-logs", Tag: "Admin", Summary: "List audit log entries", Security: openAPISessionAdminSecurity, Parameters: []openAPIParameter{integerQueryParameter("limit", "Maximum number of entries to return."), queryParameter("actor_id", "Filter by actor ID."), queryParameter("action", "Filter by action name."), queryParameter("resource_type", "Filter by resource type."), queryParameter("resource_id", "Filter by resource ID.")}, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "Audit log entries.", ContentType: "application/json", Schema: arraySchema(schemaRef("GenericObject"))}, ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/settings", Tag: "Settings", Summary: "Fetch instance settings", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Settings map."), ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "POST", Path: "/api/v1/settings", Tag: "Settings", Summary: "Update instance settings", Security: openAPISessionAdminSecurity, RequestBody: genericJSONRequest("Settings payload."), SuccessStatus: "200", SuccessResponse: messageJSONResponse("Settings updated."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/system/metrics", Tag: "System", Summary: "Fetch internal system metrics", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("System metrics payload."), ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/system/backup", Tag: "System", Summary: "Export a JSON backup", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Backup payload."), ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "POST", Path: "/api/v1/system/backup", Tag: "System", Summary: "Import a JSON backup", Security: openAPISessionAdminSecurity, RequestBody: genericJSONRequest("Backup payload."), SuccessStatus: "200", SuccessResponse: genericJSONResponse("Import result."), ErrorStatuses: []string{"400", "401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/system/export/yaml", Tag: "System", Summary: "Export the configuration as YAML", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: openAPIResponseSpec{Description: "YAML configuration.", ContentType: "application/yaml", Schema: map[string]any{"type": "string"}}, ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "GET", Path: "/api/v1/system/version", Tag: "System", Summary: "Check for available updates", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: genericJSONResponse("Update availability."), ErrorStatuses: []string{"401", "403", "500"}},
		{Method: "POST", Path: "/api/v1/system/update", Tag: "System", Summary: "Apply the latest available update", Security: openAPISessionAdminSecurity, SuccessStatus: "200", SuccessResponse: messageJSONResponse("Update apply result."), ErrorStatuses: []string{"401", "403", "500"}},
	})
}
