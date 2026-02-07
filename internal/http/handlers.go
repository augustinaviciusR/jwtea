package http

import (
	"crypto/rsa"
	"fmt"
	"jwtea/internal/keys"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"jwtea/internal/config"
	"jwtea/internal/core"
)

type oidcDiscovery struct {
	Issuer                           string   `json:"issuer"`
	JWKSURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported,omitempty"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported,omitempty"`
	ClaimsSupported                  []string `json:"claims_supported,omitempty"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint                    string   `json:"token_endpoint,omitempty"`
	TokenEndpointAuthMethods         []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
	IntrospectionEndpoint            string   `json:"introspection_endpoint,omitempty"`
	RevocationEndpoint               string   `json:"revocation_endpoint,omitempty"`
	RevocationEndpointAuthMethods    []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
}

type Dependencies struct {
	Store   *core.Store
	Config  *config.Config
	Chaos   *core.ChaosFlags
	Issuer  string
	PrivKey *rsa.PrivateKey
	Kid     string
}

func authenticateClient(s *core.Store, r *http.Request) (core.Client, bool) {
	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		clientID = r.Form.Get("client_id")
		clientSecret = r.Form.Get("client_secret")
	}
	cl, ok := s.GetClient(clientID)
	if !ok {
		return core.Client{}, false
	}
	if cl.Secret != "" && cl.Secret != clientSecret {
		return core.Client{}, false
	}
	return cl, true
}

// RootHandler handles / endpoint
type RootHandler struct{}

func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

func (h *RootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintln(w, "jwtea: ok")
	if err != nil {
		return
	}
}

// HealthHandler handles /healthz endpoint
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// JWKSHandler handles /jwks.json endpoint
type JWKSHandler struct {
	jwk keys.JwkRSA
}

func NewJWKSHandler(jwk keys.JwkRSA) *JWKSHandler {
	return &JWKSHandler{jwk: jwk}
}

func (h *JWKSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=30")
	writeJSON(w, struct {
		Keys []keys.JwkRSA `json:"keys"`
	}{Keys: []keys.JwkRSA{h.jwk}})
}

// DiscoveryHandler handles /.well-known/openid-configuration endpoint
type DiscoveryHandler struct {
	issuer string
	config *config.Config
}

func NewDiscoveryHandler(issuer string, cfg *config.Config) *DiscoveryHandler {
	return &DiscoveryHandler{
		issuer: issuer,
		config: cfg,
	}
}

func (h *DiscoveryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	conf := oidcDiscovery{
		Issuer:                           h.issuer,
		JWKSURI:                          h.issuer + "/jwks.json",
		ResponseTypesSupported:           []string{"code"},
		GrantTypesSupported:              h.config.OAuth.AllowedGrantTypes,
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{h.config.Tokens.Algorithm},
		ScopesSupported:                  h.config.OAuth.SupportedScopes,
		ClaimsSupported:                  []string{"iss", "sub", "aud", "exp", "iat"},
		AuthorizationEndpoint:            h.issuer + "/authorize",
		TokenEndpoint:                    h.issuer + "/oauth2/token",
		TokenEndpointAuthMethods:         []string{"client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:    []string{"plain", "S256"},
	}
	if h.config.Introspection.Enabled {
		conf.IntrospectionEndpoint = h.issuer + "/oauth2/introspect"
	}
	if h.config.Revocation.Enabled {
		conf.RevocationEndpoint = h.issuer + "/oauth2/revoke"
		conf.RevocationEndpointAuthMethods = []string{"client_secret_basic", "client_secret_post"}
	}
	writeJSON(w, conf)
}

// AuthorizeHandler handles /authorize endpoint
type AuthorizeHandler struct {
	deps *Dependencies
}

func NewAuthorizeHandler(deps *Dependencies) *AuthorizeHandler {
	return &AuthorizeHandler{deps: deps}
}

func (h *AuthorizeHandler) resolveUserID(loginHint string) string {
	if loginHint != "" {
		if _, ok := h.deps.Store.GetUser(loginHint); ok {
			return loginHint
		}
	}

	users := h.deps.Store.ListUsers()
	if len(users) > 0 {
		return users[0].Email
	}

	return "user@example.com"
}

func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	scope := q.Get("scope")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

	if responseType != "code" || clientID == "" || redirectURI == "" {
		OAuthErrorRedirect(w, r, redirectURI, state, "invalid_request", "missing or invalid parameters")
		return
	}

	if _, err := url.ParseRequestURI(redirectURI); err != nil {
		OAuthErrorRedirect(w, r, redirectURI, state, "invalid_request", "invalid redirect_uri")
		return
	}

	cl, ok := h.deps.Store.GetClient(clientID)
	if !ok || !RedirectAllowed(cl, redirectURI) {
		OAuthErrorRedirect(w, r, redirectURI, state, "unauthorized_client", "client or redirect_uri not allowed")
		return
	}

	isPublicClient := cl.Secret == ""
	pkceRequired := h.deps.Config.OAuth.PKCERequired || (h.deps.Config.OAuth.PKCERequiredForPublic && isPublicClient)
	if pkceRequired && codeChallenge == "" {
		OAuthErrorRedirect(w, r, redirectURI, state, "invalid_request", "code_challenge required")
		return
	}

	if codeChallenge != "" {
		if codeChallengeMethod == "" {
			codeChallengeMethod = "plain"
		}
		if codeChallengeMethod != "plain" && codeChallengeMethod != "S256" {
			OAuthErrorRedirect(w, r, redirectURI, state, "invalid_request", "unsupported code_challenge_method")
			return
		}
	}

	code, err := RandCode(32)
	if err != nil {
		OAuthErrorRedirect(w, r, redirectURI, state, "server_error", "code generation failed")
		return
	}

	userID := h.resolveUserID(q.Get("login_hint"))

	ac := core.AuthCode{
		Code:                code,
		ClientID:            cl.ID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		State:               state,
		UserID:              userID,
		ExpiresAt:           time.Now().Add(h.deps.Config.OAuth.AuthCodeExpiry.Duration),
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}
	h.deps.Store.SaveCode(ac)

	u, _ := url.Parse(redirectURI)
	params := u.Query()
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	u.RawQuery = params.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// TokenHandler handles /oauth2/token endpoint
type TokenHandler struct {
	deps *Dependencies
}

func NewTokenHandler(deps *Dependencies) *TokenHandler {
	return &TokenHandler{deps: deps}
}

func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}
	grantType := r.Form.Get("grant_type")

	switch grantType {
	case "authorization_code":
		h.handleAuthorizationCode(w, r)
	case "client_credentials":
		h.handleClientCredentials(w, r)
	case "refresh_token":
		h.handleRefreshToken(w, r)
	default:
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "unsupported_grant_type", "grant type not supported")
	}
}

func (h *TokenHandler) handleAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	cl, ok := authenticateClient(h.deps.Store, r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic realm=token")
		WriteOAuthErrorJSON(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	code := r.Form.Get("code")
	redirectURI := r.Form.Get("redirect_uri")
	codeVerifier := r.Form.Get("code_verifier")

	if code == "" || redirectURI == "" {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "code and redirect_uri required")
		return
	}

	ac, ok := h.deps.Store.ConsumeCode(code)
	if !ok || ac.ClientID != cl.ID || ac.RedirectURI != redirectURI {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_grant", "code invalid, expired, used, or mismatched")
		return
	}

	if ac.CodeChallenge != "" {
		if codeVerifier == "" {
			WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "code_verifier required")
			return
		}
		if !ValidatePKCE(codeVerifier, ac.CodeChallenge, ac.CodeChallengeMethod) {
			WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_grant", "code_verifier invalid")
			return
		}
	}

	gen := core.NewTokenGenerator(h.deps.PrivKey, h.deps.Kid, h.deps.Issuer)
	req := core.TokenRequest{
		Subject:               ac.UserID,
		Audience:              cl.ID,
		Scope:                 ac.Scope,
		ExpiresIn:             h.deps.Config.Tokens.AccessTokenExpiry.Duration,
		ChaosExpired:          h.deps.Chaos.ConsumeNextTokenExpired(),
		ChaosInvalidSignature: h.deps.Chaos.IsInvalidSignature(),
	}

	result, err := gen.Generate(req)
	if err != nil {
		WriteOAuthErrorJSON(w, http.StatusInternalServerError, "server_error", "token generation failed")
		return
	}

	resp := map[string]any{
		"access_token": result.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   result.ExpiresIn,
		"scope":        ac.Scope,
		"id_token":     result.IDToken,
	}

	if h.deps.Config.Tokens.IssueRefreshToken || HasScope(ac.Scope, "offline_access") {
		refreshToken, err := GenerateRefreshToken()
		if err != nil {
			WriteOAuthErrorJSON(w, http.StatusInternalServerError, "server_error", "refresh token generation failed")
			return
		}
		rt := core.RefreshToken{
			Token:     refreshToken,
			ClientID:  cl.ID,
			UserID:    ac.UserID,
			Scope:     ac.Scope,
			ExpiresAt: time.Now().Add(h.deps.Config.Tokens.RefreshTokenExpiry.Duration),
			IssuedAt:  time.Now(),
		}
		h.deps.Store.SaveRefreshToken(rt)
		resp["refresh_token"] = refreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, resp)
}

func (h *TokenHandler) handleClientCredentials(w http.ResponseWriter, r *http.Request) {
	cl, ok := authenticateClient(h.deps.Store, r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic realm=token")
		WriteOAuthErrorJSON(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	scope := r.Form.Get("scope")
	if scope == "" {
		scope = strings.Join(h.deps.Config.OAuth.DefaultScopes, " ")
	}

	gen := core.NewTokenGenerator(h.deps.PrivKey, h.deps.Kid, h.deps.Issuer)
	req := core.TokenRequest{
		Subject:               cl.ID,
		Audience:              cl.ID,
		Scope:                 scope,
		ExpiresIn:             h.deps.Config.Tokens.AccessTokenExpiry.Duration,
		ChaosExpired:          h.deps.Chaos.ConsumeNextTokenExpired(),
		ChaosInvalidSignature: h.deps.Chaos.IsInvalidSignature(),
	}

	result, err := gen.Generate(req)
	if err != nil {
		WriteOAuthErrorJSON(w, http.StatusInternalServerError, "server_error", "token generation failed")
		return
	}

	resp := map[string]any{
		"access_token": result.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   result.ExpiresIn,
		"scope":        scope,
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, resp)
}

func (h *TokenHandler) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	cl, ok := authenticateClient(h.deps.Store, r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic realm=token")
		WriteOAuthErrorJSON(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	refreshTokenStr := r.Form.Get("refresh_token")
	if refreshTokenStr == "" {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "refresh_token required")
		return
	}

	rt, ok := h.deps.Store.GetRefreshToken(refreshTokenStr)
	if !ok || rt.ClientID != cl.ID {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_grant", "refresh token invalid or expired")
		return
	}

	requestedScope := r.Form.Get("scope")
	scope := rt.Scope
	if requestedScope != "" {
		if !IsScopeSubset(requestedScope, rt.Scope) {
			WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds original scope")
			return
		}
		scope = requestedScope
	}

	gen := core.NewTokenGenerator(h.deps.PrivKey, h.deps.Kid, h.deps.Issuer)
	req := core.TokenRequest{
		Subject:               rt.UserID,
		Audience:              cl.ID,
		Scope:                 scope,
		ExpiresIn:             h.deps.Config.Tokens.AccessTokenExpiry.Duration,
		ChaosExpired:          h.deps.Chaos.ConsumeNextTokenExpired(),
		ChaosInvalidSignature: h.deps.Chaos.IsInvalidSignature(),
	}

	result, err := gen.Generate(req)
	if err != nil {
		WriteOAuthErrorJSON(w, http.StatusInternalServerError, "server_error", "token generation failed")
		return
	}

	resp := map[string]any{
		"access_token": result.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   result.ExpiresIn,
		"scope":        scope,
	}

	if h.deps.Config.Tokens.RefreshTokenRotation {
		h.deps.Store.RevokeRefreshToken(refreshTokenStr)
		newRefreshToken, err := GenerateRefreshToken()
		if err != nil {
			WriteOAuthErrorJSON(w, http.StatusInternalServerError, "server_error", "refresh token generation failed")
			return
		}
		newRT := core.RefreshToken{
			Token:     newRefreshToken,
			ClientID:  cl.ID,
			UserID:    rt.UserID,
			Scope:     rt.Scope,
			ExpiresAt: time.Now().Add(h.deps.Config.Tokens.RefreshTokenExpiry.Duration),
			IssuedAt:  time.Now(),
		}
		h.deps.Store.SaveRefreshToken(newRT)
		resp["refresh_token"] = newRefreshToken
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, resp)
}

// IntrospectionHandler handles /oauth2/introspect endpoint (RFC 7662)
type IntrospectionHandler struct {
	deps *Dependencies
}

func NewIntrospectionHandler(deps *Dependencies) *IntrospectionHandler {
	return &IntrospectionHandler{deps: deps}
}

func (h *IntrospectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}

	if h.deps.Config.Introspection.RequireClientAuth {
		cl, ok := authenticateClient(h.deps.Store, r)
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=introspect")
			WriteOAuthErrorJSON(w, http.StatusUnauthorized, "invalid_client", "client authentication required")
			return
		}
		if len(h.deps.Config.Introspection.AllowedClients) > 0 &&
			!slices.Contains(h.deps.Config.Introspection.AllowedClients, cl.ID) {
			WriteOAuthErrorJSON(w, http.StatusForbidden, "access_denied", "client not allowed to introspect")
			return
		}
	}

	tokenStr := r.Form.Get("token")
	if tokenStr == "" {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "token required")
		return
	}

	resp := h.introspectToken(tokenStr)
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, resp)
}

func (h *IntrospectionHandler) introspectToken(tokenStr string) map[string]any {
	claims, err := core.ParseAndValidateToken(tokenStr, &h.deps.PrivKey.PublicKey)
	if err != nil {
		return map[string]any{"active": false}
	}

	if jti, ok := claims["jti"].(string); ok {
		if h.deps.Store.IsAccessTokenRevoked(jti) {
			return map[string]any{"active": false}
		}
	}

	resp := map[string]any{
		"active":     true,
		"token_type": "Bearer",
	}

	if sub, ok := claims["sub"].(string); ok {
		resp["sub"] = sub
	}
	if aud, ok := claims["aud"].(string); ok {
		resp["client_id"] = aud
	}
	if scope, ok := claims["scope"].(string); ok {
		resp["scope"] = scope
	}
	if iss, ok := claims["iss"].(string); ok {
		resp["iss"] = iss
	}
	if exp, ok := claims["exp"].(float64); ok {
		resp["exp"] = int64(exp)
	}
	if iat, ok := claims["iat"].(float64); ok {
		resp["iat"] = int64(iat)
	}

	return resp
}

// RevocationHandler handles /oauth2/revoke endpoint (RFC 7009)
type RevocationHandler struct {
	deps *Dependencies
}

func NewRevocationHandler(deps *Dependencies) *RevocationHandler {
	return &RevocationHandler{deps: deps}
}

func (h *RevocationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}

	var clientID string
	if h.deps.Config.Revocation.RequireClientAuth {
		cl, ok := authenticateClient(h.deps.Store, r)
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=revoke")
			WriteOAuthErrorJSON(w, http.StatusUnauthorized, "invalid_client", "client authentication required")
			return
		}
		clientID = cl.ID
	}

	tokenStr := r.Form.Get("token")
	if tokenStr == "" {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, "invalid_request", "token required")
		return
	}

	tokenTypeHint := r.Form.Get("token_type_hint")

	h.revokeToken(tokenStr, tokenTypeHint, clientID)

	w.WriteHeader(http.StatusOK)
}

func (h *RevocationHandler) revokeToken(tokenStr, tokenTypeHint, clientID string) {
	if tokenTypeHint == "refresh_token" || tokenTypeHint == "" {
		rt, ok := h.deps.Store.GetRefreshToken(tokenStr)
		if ok {
			if clientID == "" || rt.ClientID == clientID {
				h.deps.Store.RevokeRefreshToken(tokenStr)
			}
			return
		}
	}

	if tokenTypeHint == "access_token" || tokenTypeHint == "" {
		claims, err := core.ParseAndValidateToken(tokenStr, &h.deps.PrivKey.PublicKey)
		if err == nil {
			if clientID != "" {
				if aud, ok := claims["aud"].(string); ok && aud != clientID {
					return
				}
			}
			if jti, ok := claims["jti"].(string); ok {
				h.deps.Store.RevokeAccessToken(jti)
			} else {
				h.deps.Store.RevokeAccessToken(tokenStr)
			}
		}
	}
}
