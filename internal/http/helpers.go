package http

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"jwtea/internal/core"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

func writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("JSON encode error: %v", err)
	}
}

func DeriveIssuer(explicit, host string, port int) string {
	if explicit != "" {
		return strings.TrimRight(explicit, "/")
	}
	lower := strings.ToLower(host)
	if lower == "127.0.0.1" || lower == "localhost" || strings.HasPrefix(lower, "127.") {
		return fmt.Sprintf("http://%s:%d", host, port)
	}
	if port == 443 {
		return fmt.Sprintf("https://%s", host)
	}
	return fmt.Sprintf("https://%s:%d", host, port)
}

func RedirectAllowed(c core.Client, redirectURI string) bool {
	return slices.Contains(c.RedirectURIs, redirectURI)
}

func RandCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func OAuthErrorRedirect(w http.ResponseWriter, r *http.Request, redirectURI, state, code, desc string) {
	if redirectURI == "" {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, code, desc)
		return
	}
	u, err := url.Parse(redirectURI)
	if err != nil {
		WriteOAuthErrorJSON(w, http.StatusBadRequest, code, desc)
		return
	}
	q := u.Query()
	q.Set("error", code)
	if desc != "" {
		q.Set("error_description", desc)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func WriteOAuthErrorJSON(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	payload := map[string]string{"error": code}
	if desc != "" {
		payload["error_description"] = desc
	}
	writeJSON(w, payload)
}

func ValidatePKCE(verifier, challenge, method string) bool {
	if method == "plain" {
		return verifier == challenge
	}
	if method == "S256" {
		h := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		return computed == challenge
	}
	return false
}

func GenerateRefreshToken() (string, error) {
	return RandCode(32)
}

func IsScopeSubset(requested, original string) bool {
	if requested == "" {
		return true
	}
	origScopes := make(map[string]bool)
	for _, s := range strings.Fields(original) {
		origScopes[s] = true
	}
	for _, s := range strings.Fields(requested) {
		if !origScopes[s] {
			return false
		}
	}
	return true
}

func HasScope(scopeStr, target string) bool {
	return slices.Contains(strings.Fields(scopeStr), target)
}
