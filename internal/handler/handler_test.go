// Copyright 2026 The Casdoor Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/casdoor/casdoor-forward-auth/internal/config"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v4"
)

const (
	testClientId     = "test-client-id"
	testClientSecret = "test-client-secret"
	testKid          = "cert-test"
)

// fakeCasdoor issues RS256 access tokens for the codes it hands out and serves its
// certificate in the JWKS, like a real Casdoor server.
type fakeCasdoor struct {
	server   *httptest.Server
	key      *rsa.PrivateKey
	certDer  []byte
	codes    map[string]bool
	audience string
}

func newFakeCasdoor(t *testing.T) *fakeCasdoor {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Casdoor Cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDer, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	fc := &fakeCasdoor{key: key, certDer: certDer, codes: map[string]bool{}, audience: testClientId}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]interface{}{
				{"kid": testKid, "kty": "RSA", "x5c": []string{base64.StdEncoding.EncodeToString(fc.certDer)}},
			},
		})
	})
	mux.HandleFunc("/api/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		code := r.Form.Get("code")
		if r.Form.Get("client_secret") != testClientSecret || !fc.codes[code] {
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": "authorization code is invalid"})
			return
		}
		// authorization codes can be used only once
		delete(fc.codes, code)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": fc.issueToken(t),
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})
	fc.server = httptest.NewServer(mux)
	t.Cleanup(fc.server.Close)
	return fc
}

func (fc *fakeCasdoor) issueToken(t *testing.T) string {
	claims := casdoorsdk.Claims{
		User: casdoorsdk.User{
			Owner:  "built-in",
			Name:   "alice",
			Id:     "8a1b2c3d",
			Email:  "alice@example.com",
			Groups: []string{"built-in/dev"},
			Roles:  []*casdoorsdk.Role{{Owner: "built-in", Name: "admin"}},
		},
		TokenType: "access-token",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    fc.server.URL,
			Subject:   "8a1b2c3d",
			Audience:  []string{fc.audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKid
	signed, err := token.SignedString(fc.key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func newTestHandler(t *testing.T, fc *fakeCasdoor, externalUrl string) *Handler {
	t.Setenv("CASDOOR_ENDPOINT", fc.server.URL)
	t.Setenv("CLIENT_ID", testClientId)
	t.Setenv("CLIENT_SECRET", testClientSecret)
	t.Setenv("EXTERNAL_URL", externalUrl)
	t.Setenv("COOKIE_SECRET", strings.Repeat("s", 32))
	t.Setenv("COOKIE_DOMAIN", "example.com")

	conf, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	return New(conf)
}

func serve(h http.Handler, method string, target string, header http.Header, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for name, values := range header {
		req.Header[name] = values
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func findCookie(rec *httptest.ResponseRecorder, prefix string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if strings.HasPrefix(cookie.Name, prefix) && cookie.MaxAge >= 0 {
			return cookie
		}
	}
	return nil
}

// login runs /login and /callback, and returns the session cookie.
func login(t *testing.T, h *Handler, fc *fakeCasdoor, rd string) (*http.Cookie, *httptest.ResponseRecorder) {
	rec := serve(h, http.MethodGet, "https://auth.example.com/login?rd="+url.QueryEscape(rd), nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("login: got status %d", rec.Code)
	}

	authorizeUrl, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(authorizeUrl.String(), fc.server.URL+"/login/oauth/authorize?") {
		t.Fatalf("login: unexpected redirect %s", authorizeUrl)
	}
	if got := authorizeUrl.Query().Get("redirect_uri"); got != "https://auth.example.com/callback" {
		t.Fatalf("login: unexpected redirect_uri %s", got)
	}

	stateCookie := findCookie(rec, "casdoor_forward_auth_state_")
	if stateCookie == nil {
		t.Fatal("login: no state cookie")
	}

	fc.codes["code-1"] = true
	state := authorizeUrl.Query().Get("state")
	rec = serve(h, http.MethodGet, "https://auth.example.com/callback?code=code-1&state="+state, nil, stateCookie)
	return findCookie(rec, "casdoor_forward_auth"), rec
}

func TestFullLogin(t *testing.T) {
	fc := newFakeCasdoor(t)
	h := newTestHandler(t, fc, "https://auth.example.com")

	forwarded := http.Header{
		"X-Forwarded-Method": {"GET"},
		"X-Forwarded-Proto":  {"https"},
		"X-Forwarded-Host":   {"app.example.com"},
		"X-Forwarded-Uri":    {"/dashboard?tab=1"},
		"X-Forwarded-User":   {"mallory"},
	}

	rec := serve(h, http.MethodGet, "/auth", forwarded)
	if rec.Code != http.StatusFound {
		t.Fatalf("auth without session: got status %d", rec.Code)
	}
	wantLogin := "https://auth.example.com/login?rd=" + url.QueryEscape("https://app.example.com/dashboard?tab=1")
	if got := rec.Header().Get("Location"); got != wantLogin {
		t.Fatalf("auth without session: got redirect %s, want %s", got, wantLogin)
	}

	sessionCookie, rec := login(t, h, fc, "https://app.example.com/dashboard?tab=1")
	if rec.Code != http.StatusFound {
		t.Fatalf("callback: got status %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "https://app.example.com/dashboard?tab=1" {
		t.Fatalf("callback: got redirect %s", got)
	}
	if sessionCookie == nil || sessionCookie.Domain != "example.com" || !sessionCookie.HttpOnly || !sessionCookie.Secure {
		t.Fatalf("callback: bad session cookie %+v", sessionCookie)
	}

	// the session keeps working for any number of requests, unlike an authorization code
	for i := 0; i < 3; i++ {
		rec = serve(h, http.MethodGet, "/auth", forwarded, sessionCookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("auth with session: got status %d", rec.Code)
		}
	}

	want := map[string]string{
		HeaderUser:         "alice",
		HeaderUserId:       "8a1b2c3d",
		HeaderOrganization: "built-in",
		HeaderEmail:        "alice@example.com",
		HeaderGroups:       "built-in/dev",
		HeaderRoles:        "admin",
	}
	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Errorf("header %s: got %q, want %q", name, got, value)
		}
	}

	rec = serve(h, http.MethodPost, "/verify", nil, sessionCookie)
	if rec.Code != http.StatusOK || rec.Header().Get(HeaderUser) != "alice" {
		t.Fatalf("verify with session: got status %d", rec.Code)
	}

	rec = serve(h, http.MethodGet, "/logout", nil, sessionCookie)
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "casdoor_forward_auth" && cookie.MaxAge >= 0 {
			t.Fatalf("logout: session cookie not cleared: %+v", cookie)
		}
	}
}

func TestAllowedRolesAndGroups(t *testing.T) {
	fc := newFakeCasdoor(t)
	h := newTestHandler(t, fc, "https://auth.example.com")
	sessionCookie, _ := login(t, h, fc, "")

	// alice has the role "admin" and the group "built-in/dev"
	cases := map[string]int{
		"/auth":                                   http.StatusOK,
		"/auth?roles=admin":                       http.StatusOK,
		"/auth?roles=ops,%20admin":                http.StatusOK,
		"/auth?roles=ops":                         http.StatusForbidden,
		"/auth?groups=built-in/dev":               http.StatusOK,
		"/auth?groups=built-in/ops":               http.StatusForbidden,
		"/auth?roles=admin&groups=built-in/ops":   http.StatusForbidden,
		"/verify?roles=admin&groups=built-in/dev": http.StatusOK,
		"/verify?roles=ops":                       http.StatusForbidden,
	}
	for target, want := range cases {
		rec := serve(h, http.MethodGet, target, nil, sessionCookie)
		if rec.Code != want {
			t.Errorf("%s: got status %d, want %d", target, rec.Code, want)
		}
		if want == http.StatusForbidden && rec.Header().Get(HeaderUser) != "" {
			t.Errorf("%s: identity headers on a forbidden response", target)
		}
	}

	h.conf.AllowedRoles = []string{"ops"}
	if rec := serve(h, http.MethodGet, "/auth", nil, sessionCookie); rec.Code != http.StatusForbidden {
		t.Errorf("allowedRoles: got status %d", rec.Code)
	}
	if rec := serve(h, http.MethodGet, "/auth?roles=admin", nil, sessionCookie); rec.Code != http.StatusOK {
		t.Errorf("allowedRoles plus query: got status %d", rec.Code)
	}
}

func TestUnauthenticated(t *testing.T) {
	fc := newFakeCasdoor(t)
	h := newTestHandler(t, fc, "https://auth.example.com")

	rec := serve(h, http.MethodGet, "/auth", http.Header{"X-Forwarded-Method": {"POST"}, "X-Forwarded-Host": {"app.example.com"}})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("auth for POST: got status %d, want 401", rec.Code)
	}

	rec = serve(h, http.MethodGet, "/verify", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("verify: got status %d, want 401", rec.Code)
	}

	sessionCookie, _ := login(t, h, fc, "")
	sessionCookie.Value = strings.Replace(sessionCookie.Value, sessionCookie.Value[:4], "AAAA", 1)
	rec = serve(h, http.MethodGet, "/verify", nil, sessionCookie)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("verify with tampered cookie: got status %d, want 401", rec.Code)
	}
}

func TestCallbackRejectsBadState(t *testing.T) {
	fc := newFakeCasdoor(t)
	h := newTestHandler(t, fc, "https://auth.example.com")

	rec := serve(h, http.MethodGet, "https://auth.example.com/login", nil)
	stateCookie := findCookie(rec, "casdoor_forward_auth_state_")
	authorizeUrl, _ := url.Parse(rec.Header().Get("Location"))
	state := authorizeUrl.Query().Get("state")
	fc.codes["code-1"] = true

	// no state cookie, e.g., a login CSRF where the attacker's code is sent to the victim
	rec = serve(h, http.MethodGet, "/callback?code=code-1&state="+state, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("callback without state cookie: got status %d, want 400", rec.Code)
	}

	forged := state[:16] + strings.Repeat("0", len(state)-16)
	rec = serve(h, http.MethodGet, "/callback?code=code-1&state="+forged, nil, stateCookie)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("callback with forged state: got status %d, want 400", rec.Code)
	}

	rec = serve(h, http.MethodGet, "/callback?error=access_denied", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("callback with error: got status %d, want 400", rec.Code)
	}
}

func TestCallbackRejectsTokenForOtherClient(t *testing.T) {
	fc := newFakeCasdoor(t)
	fc.audience = "another-client"
	h := newTestHandler(t, fc, "https://auth.example.com")

	sessionCookie, rec := login(t, h, fc, "")
	if rec.Code != http.StatusBadGateway || sessionCookie != nil {
		t.Fatalf("callback: got status %d, want 502 and no session", rec.Code)
	}
}

func TestSafeRedirect(t *testing.T) {
	fc := newFakeCasdoor(t)
	h := newTestHandler(t, fc, "https://auth.example.com")

	cases := map[string]string{
		"":                                    "https://auth.example.com/",
		"https://app.example.com/x?y=1":       "https://app.example.com/x?y=1",
		"http://example.com:8080/":            "http://example.com:8080/",
		"/profile":                            "https://auth.example.com/profile",
		"https://evil.com/":                   "https://auth.example.com/",
		"https://example.com.evil.com/":       "https://auth.example.com/",
		"https://evilexample.com/":            "https://auth.example.com/",
		"//evil.com/":                         "https://auth.example.com/",
		"/\\evil.com/":                        "https://auth.example.com/",
		"javascript:alert(1)":                 "https://auth.example.com/",
		"https://app.example.com@evil.com/":   "https://auth.example.com/",
		"https://evil.com\\@app.example.com/": "https://auth.example.com/",
	}
	for rd, want := range cases {
		if got := h.getSafeRedirect(rd); got != want {
			t.Errorf("getSafeRedirect(%q) = %q, want %q", rd, got, want)
		}
	}
}

func TestBasePath(t *testing.T) {
	fc := newFakeCasdoor(t)
	h := newTestHandler(t, fc, "https://app.example.com/_auth")

	rec := serve(h, http.MethodGet, "/_auth/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz: got status %d", rec.Code)
	}

	rec = serve(h, http.MethodGet, "/_auth/login?rd=/dashboard", nil)
	authorizeUrl, _ := url.Parse(rec.Header().Get("Location"))
	if got := authorizeUrl.Query().Get("redirect_uri"); got != "https://app.example.com/_auth/callback" {
		t.Fatalf("login: unexpected redirect_uri %s", got)
	}
	if cookie := findCookie(rec, "casdoor_forward_auth_state_"); cookie == nil || cookie.Path != "/_auth/callback" {
		t.Fatalf("login: bad state cookie %+v", cookie)
	}
}
