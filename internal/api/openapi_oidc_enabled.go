//go:build oidc

package api

func openAPIOIDCRoutes() []openAPIRoute {
	return []openAPIRoute{
		{
			Method:        "GET",
			Path:          "/api/v1/auth/oidc/login",
			Tag:           "Auth",
			Summary:       "Start the OIDC login flow",
			SuccessStatus: "302",
			SuccessResponse: openAPIResponseSpec{
				Description: "Redirect to the configured OIDC provider.",
			},
			ErrorStatuses: []string{"404", "500"},
		},
		{
			Method:     "GET",
			Path:       "/api/v1/auth/oidc/callback",
			Tag:        "Auth",
			Summary:    "Handle the OIDC provider callback",
			Parameters: []openAPIParameter{queryParameter("state", "OIDC state parameter."), queryParameter("code", "OIDC authorization code.")},
			Responses: map[string]openAPIResponseSpec{
				"302": {Description: "Redirect back to the application after a successful login."},
				"400": openAPIErrorResponse("400"),
				"404": openAPIErrorResponse("404"),
				"500": openAPIErrorResponse("500"),
			},
		},
	}
}
