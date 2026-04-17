//go:build oidc

package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/updu/updu/internal/models"
	"github.com/updu/updu/internal/storage"
)

const (
	mockOIDCClientID     = "updu-oidc-test-client"
	mockOIDCClientSecret = "updu-oidc-test-secret"
	mockOIDCSubject      = "oidc-test-subject"
	mockOIDCUsername     = "admin"
	mockOIDCEmail        = "admin@example.test"
	mockOIDCName         = "OIDC Admin"
)

type mockOIDCProvider struct {
	testingT            *testing.T
	server              *httptest.Server
	privateKey          *rsa.PrivateKey
	clientID            string
	clientSecret        string
	expectedRedirectURI string
	keyID               string
	subject             string
	username            string
	email               string
	name                string

	mu    sync.Mutex
	codes map[string]mockOIDCAuthorization
}

type mockOIDCAuthorization struct {
	nonce       string
	redirectURI string
	subject     string
	username    string
	email       string
	name        string
}

func TestAPI_OIDC_LoginRedirectsToProvider(t *testing.T) {
	_, _, provider, app, client, cleanup := startOIDCFlowHarness(t, true)
	defer cleanup()

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Get(app.URL + "/api/v1/auth/oidc/login")
	if err != nil {
		t.Fatalf("failed to start OIDC login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected login redirect status 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, provider.issuer()+"/authorize") {
		t.Fatalf("expected redirect to mock issuer authorize endpoint, got %q", location)
	}

	redirectURL, err := url.Parse(location)
	if err != nil {
		t.Fatalf("failed to parse login redirect URL: %v", err)
	}

	query := redirectURL.Query()
	if query.Get("client_id") != mockOIDCClientID {
		t.Fatalf("expected client_id %q, got %q", mockOIDCClientID, query.Get("client_id"))
	}
	if query.Get("redirect_uri") != app.URL+"/api/v1/auth/oidc/callback" {
		t.Fatalf("expected redirect_uri %q, got %q", app.URL+"/api/v1/auth/oidc/callback", query.Get("redirect_uri"))
	}
	if query.Get("scope") != "openid profile email" {
		t.Fatalf("expected scope %q, got %q", "openid profile email", query.Get("scope"))
	}
	if query.Get("response_type") != "code" {
		t.Fatalf("expected response_type %q, got %q", "code", query.Get("response_type"))
	}

	stateCookie := findCookie(resp.Cookies(), oidcStateCookie)
	if stateCookie == nil {
		t.Fatal("expected oidc_state cookie to be set")
	}
	if stateCookie.Value == "" {
		t.Fatal("expected oidc_state cookie to contain a value")
	}
	if query.Get("state") != stateCookie.Value {
		t.Fatalf("expected state query parameter %q to match cookie value %q", query.Get("state"), stateCookie.Value)
	}

	nonceCookie := findCookie(resp.Cookies(), "oidc_nonce")
	if nonceCookie == nil {
		t.Fatal("expected oidc_nonce cookie to be set")
	}
	if nonceCookie.Value == "" {
		t.Fatal("expected oidc_nonce cookie to contain a value")
	}
	if query.Get("nonce") != nonceCookie.Value {
		t.Fatalf("expected nonce query parameter %q to match cookie value %q", query.Get("nonce"), nonceCookie.Value)
	}

	if stateCookie.HttpOnly != true {
		t.Fatal("expected oidc_state cookie to be HttpOnly")
	}
	if nonceCookie.HttpOnly != true {
		t.Fatal("expected oidc_nonce cookie to be HttpOnly")
	}
}

func TestAPI_OIDC_FullLoginFlowCreatesAdminSession(t *testing.T) {
	_, db, provider, app, client, cleanup := startOIDCFlowHarness(t, true)
	defer cleanup()

	resp := performOIDCLogin(t, client, app.URL)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected callback redirect status 302, got %d", resp.StatusCode)
	}
	if location := resp.Header.Get("Location"); location != "/" {
		t.Fatalf("expected callback redirect to %q, got %q", "/", location)
	}

	sessionCookie := findCookie(cookiesForURL(t, client, app.URL), "updu_session")
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected updu_session cookie to be created after successful OIDC login")
	}

	user, err := db.GetUserByOIDCSub(context.Background(), provider.subject, provider.issuer())
	if err != nil {
		t.Fatalf("failed to look up OIDC user: %v", err)
	}
	if user == nil {
		t.Fatal("expected OIDC user to be created")
	}
	if user.Role != models.RoleAdmin {
		t.Fatalf("expected first OIDC user role %q, got %q", models.RoleAdmin, user.Role)
	}
	if user.Username != provider.username {
		t.Fatalf("expected created OIDC username %q, got %q", provider.username, user.Username)
	}

	count, err := db.CountUsers(context.Background())
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 user after first OIDC login, got %d", count)
	}
}

func TestAPI_OIDC_FullLoginFlowReusesExistingOIDCUser(t *testing.T) {
	_, db, provider, app, client, cleanup := startOIDCFlowHarness(t, true)
	defer cleanup()

	issuer := provider.issuer()
	subject := provider.subject
	existing := &models.User{
		ID:         "existing-oidc-user",
		Username:   "admin",
		Password:   "!oidc-only",
		Role:       models.RoleAdmin,
		OIDCSub:    &subject,
		OIDCIssuer: &issuer,
		CreatedAt:  time.Now(),
	}
	if err := db.CreateUser(context.Background(), existing); err != nil {
		t.Fatalf("failed to seed existing OIDC user: %v", err)
	}

	resp := performOIDCLogin(t, client, app.URL)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected callback redirect status 302, got %d", resp.StatusCode)
	}

	sessionCookie := findCookie(cookiesForURL(t, client, app.URL), "updu_session")
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected session cookie for existing OIDC user")
	}

	count, err := db.CountUsers(context.Background())
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected user count to remain 1 when OIDC user exists, got %d", count)
	}

	user, err := db.GetUserByOIDCSub(context.Background(), provider.subject, provider.issuer())
	if err != nil {
		t.Fatalf("failed to look up existing OIDC user: %v", err)
	}
	if user == nil {
		t.Fatal("expected existing OIDC user to remain available")
	}
	if user.ID != existing.ID {
		t.Fatalf("expected existing OIDC user ID %q, got %q", existing.ID, user.ID)
	}
}

func TestAPI_OIDC_FullLoginFlowHonorsAutoRegisterSetting(t *testing.T) {
	srv, db, _, app, client, cleanup := startOIDCFlowHarness(t, false)
	defer cleanup()

	if _, err := srv.auth.Register(context.Background(), "local-admin", "password123"); err != nil {
		t.Fatalf("failed to seed local admin: %v", err)
	}

	resp := performOIDCLogin(t, client, app.URL)
	defer resp.Body.Close()
	body := readResponseBody(t, resp)

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected auto-register disabled callback status 403, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Auto-registration is disabled") {
		t.Fatalf("expected auto-register disabled error message, got %q", body)
	}

	count, err := db.CountUsers(context.Background())
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected user count to remain 1 when auto-register is disabled, got %d", count)
	}

	if sessionCookie := findCookie(cookiesForURL(t, client, app.URL), "updu_session"); sessionCookie != nil {
		t.Fatal("expected no session cookie when auto-register is disabled")
	}
}

func performOIDCLogin(t *testing.T, client *http.Client, appURL string) *http.Response {
	t.Helper()

	resp, err := client.Get(appURL + "/api/v1/auth/oidc/login")
	if err != nil {
		t.Fatalf("failed to complete OIDC login flow: %v", err)
	}

	return resp
}

func startOIDCFlowHarness(t *testing.T, autoRegister bool) (*Server, *storage.DB, *mockOIDCProvider, *httptest.Server, *http.Client, func()) {
	t.Helper()

	srv, db, cleanup := setupOIDCAuthTest(t, false)
	provider := newMockOIDCProvider(t)

	cfg := srv.auth.Config()
	cfg.OIDCIssuer = provider.issuer()
	cfg.OIDCClientID = provider.clientID
	cfg.OIDCClientSecret = provider.clientSecret
	cfg.OIDCAutoRegister = autoRegister

	app := httptest.NewServer(srv.Router())
	cfg.BaseURL = app.URL
	provider.expectedRedirectURI = app.URL + "/api/v1/auth/oidc/callback"
	cfg.OIDCRedirectURL = provider.expectedRedirectURI

	client := newOIDCTestClient(t, app.URL)

	return srv, db, provider, app, client, func() {
		app.Close()
		cleanup()
	}
}

func newOIDCTestClient(t *testing.T, appURL string) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	return &http.Client{
		Timeout: 5 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.String() == appURL+"/" {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

func cookiesForURL(t *testing.T, client *http.Client, rawURL string) []*http.Cookie {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", rawURL, err)
	}

	return client.Jar.Cookies(parsed)
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func readResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return string(body)
}

func newMockOIDCProvider(t *testing.T) *mockOIDCProvider {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate mock OIDC signing key: %v", err)
	}

	provider := &mockOIDCProvider{
		testingT:     t,
		privateKey:   privateKey,
		clientID:     mockOIDCClientID,
		clientSecret: mockOIDCClientSecret,
		keyID:        "updu-oidc-test-key",
		subject:      mockOIDCSubject,
		username:     mockOIDCUsername,
		email:        mockOIDCEmail,
		name:         mockOIDCName,
		codes:        make(map[string]mockOIDCAuthorization),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", provider.handleDiscovery)
	mux.HandleFunc("/authorize", provider.handleAuthorize)
	mux.HandleFunc("/token", provider.handleToken)
	mux.HandleFunc("/jwks", provider.handleJWKS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	provider.server = httptest.NewServer(mux)
	t.Cleanup(provider.server.Close)

	return provider
}

func (p *mockOIDCProvider) issuer() string {
	return p.server.URL
}

func (p *mockOIDCProvider) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                p.issuer(),
		"authorization_endpoint":                p.issuer() + "/authorize",
		"token_endpoint":                        p.issuer() + "/token",
		"jwks_uri":                              p.issuer() + "/jwks",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
	})
}

func (p *mockOIDCProvider) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	if query.Get("client_id") != p.clientID {
		http.Error(w, "invalid client_id", http.StatusUnauthorized)
		return
	}
	if query.Get("response_type") != "code" {
		http.Error(w, "invalid response_type", http.StatusBadRequest)
		return
	}

	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	nonce := query.Get("nonce")
	if redirectURI == "" || state == "" || nonce == "" {
		http.Error(w, "missing required authorize parameters", http.StatusBadRequest)
		return
	}
	if p.expectedRedirectURI != "" && redirectURI != p.expectedRedirectURI {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	if !scopeSetMatches(query.Get("scope"), []string{"openid", "profile", "email"}) {
		http.Error(w, "invalid scope", http.StatusBadRequest)
		return
	}

	target, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	code := randomToken(p.testingT, 24)

	p.mu.Lock()
	p.codes[code] = mockOIDCAuthorization{
		nonce:       nonce,
		redirectURI: redirectURI,
		subject:     p.subject,
		username:    p.username,
		email:       p.email,
		name:        p.name,
	}
	p.mu.Unlock()

	values := target.Query()
	values.Set("code", code)
	values.Set("state", state)
	target.RawQuery = values.Encode()

	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (p *mockOIDCProvider) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		http.Error(w, "invalid grant_type", http.StatusBadRequest)
		return
	}

	clientID, clientSecret, _ := r.BasicAuth()
	if clientID == "" {
		clientID = r.Form.Get("client_id")
	}
	if clientSecret == "" {
		clientSecret = r.Form.Get("client_secret")
	}
	if clientID != p.clientID || clientSecret != p.clientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}

	code := r.Form.Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	p.mu.Lock()
	authorization, ok := p.codes[code]
	if ok {
		delete(p.codes, code)
	}
	p.mu.Unlock()
	if !ok {
		http.Error(w, "unknown authorization code", http.StatusBadRequest)
		return
	}
	if r.Form.Get("redirect_uri") != authorization.redirectURI {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	idToken, err := p.signIDToken(authorization)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign ID token: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"expires_in":   300,
		"id_token":     idToken,
	})
}

func (p *mockOIDCProvider) handleJWKS(w http.ResponseWriter, r *http.Request) {
	keySet := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{
			Key:       &p.privateKey.PublicKey,
			KeyID:     p.keyID,
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}},
	}
	writeJSON(w, http.StatusOK, keySet)
}

func (p *mockOIDCProvider) signIDToken(authorization mockOIDCAuthorization) (string, error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key: jose.JSONWebKey{
				Key:       p.privateKey,
				KeyID:     p.keyID,
				Algorithm: string(jose.RS256),
				Use:       "sig",
			},
		},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}

	now := time.Now()
	return josejwt.Signed(signer).
		Claims(josejwt.Claims{
			Issuer:   p.issuer(),
			Subject:  authorization.subject,
			Audience: josejwt.Audience{p.clientID},
			IssuedAt: josejwt.NewNumericDate(now),
			Expiry:   josejwt.NewNumericDate(now.Add(5 * time.Minute)),
		}).
		Claims(map[string]any{
			"email":              authorization.email,
			"preferred_username": authorization.username,
			"name":               authorization.name,
			"nonce":              authorization.nonce,
		}).
		Serialize()
}

func randomToken(t *testing.T, size int) string {
	t.Helper()

	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("failed to generate random token: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

func scopeContains(rawScopes string, expectedScope string) bool {
	for _, scope := range strings.Fields(rawScopes) {
		if scope == expectedScope {
			return true
		}
	}
	return false
}

func scopeSetMatches(rawScopes string, expectedScopes []string) bool {
	actualScopes := strings.Fields(rawScopes)
	if len(actualScopes) != len(expectedScopes) {
		return false
	}

	for _, expectedScope := range expectedScopes {
		if !scopeContains(rawScopes, expectedScope) {
			return false
		}
	}

	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
