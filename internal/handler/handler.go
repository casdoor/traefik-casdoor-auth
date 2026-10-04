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
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/casdoor/casdoor-forward-auth/internal/config"
	"github.com/casdoor/casdoor-forward-auth/internal/session"
	"github.com/casdoor/casdoor-forward-auth/internal/token"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

const (
	HeaderUser         = "X-Forwarded-User"
	HeaderUserId       = "X-Forwarded-User-Id"
	HeaderOrganization = "X-Forwarded-Organization"
	HeaderEmail        = "X-Forwarded-Email"
	HeaderGroups       = "X-Forwarded-Groups"
	HeaderRoles        = "X-Forwarded-Roles"

	loginStateTtl = 10 * time.Minute
)

type Handler struct {
	conf       *config.Config
	basePath   string
	casdoor    *casdoorsdk.Client
	verifier   *token.Verifier
	httpClient *http.Client
	sessions   *session.Codec
	states     *session.Codec
	mux        *http.ServeMux
}

type loginState struct {
	State    string `json:"state"`
	Redirect string `json:"redirect"`
}

func New(conf *config.Config) *Handler {
	httpClient := &http.Client{Timeout: 15 * time.Second}
	h := &Handler{
		conf:       conf,
		basePath:   conf.BasePath(),
		casdoor:    casdoorsdk.NewClient(conf.CasdoorEndpoint, conf.ClientId, conf.ClientSecret, conf.Certificate, "", ""),
		verifier:   token.NewVerifier(conf.CasdoorEndpoint, conf.ClientId, conf.Certificate, httpClient),
		httpClient: httpClient,
		sessions:   session.NewCodec(conf.CookieSecret, "session"),
		states:     session.NewCodec(conf.CookieSecret, "login-state"),
		mux:        http.NewServeMux(),
	}

	// Traefik and Caddy send GET to the auth endpoint, while Nginx's auth_request
	// keeps the method of the original request, so these two accept any method.
	h.mux.HandleFunc(h.basePath+"/auth", h.handleAuth)
	h.mux.HandleFunc(h.basePath+"/verify", h.handleVerify)
	h.mux.HandleFunc("GET "+h.basePath+"/login", h.handleLogin)
	h.mux.HandleFunc("GET "+h.basePath+"/callback", h.handleCallback)
	h.mux.HandleFunc("GET "+h.basePath+"/logout", h.handleLogout)
	h.mux.HandleFunc("GET "+h.basePath+"/healthz", h.handleHealthz)
	h.mux.HandleFunc("GET "+h.basePath+"/{$}", h.handleIndex)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleAuth is the endpoint for Traefik's forwardAuth and Caddy's forward_auth:
// 200 with the identity headers when signed in, otherwise a redirect to the login.
func (h *Handler) handleAuth(w http.ResponseWriter, r *http.Request) {
	if user := h.getSessionUser(r); user != nil {
		setIdentityHeaders(w.Header(), user)
		w.WriteHeader(http.StatusOK)
		return
	}

	// a redirect only makes sense for page loads, other requests can't follow it to the login page
	method := r.Header.Get("X-Forwarded-Method")
	if method != "" && method != http.MethodGet && method != http.MethodHead {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, h.getLoginUrl(getOriginalUrl(r)), http.StatusFound)
}

// handleVerify is the endpoint for Nginx's auth_request, which only understands
// 2xx, 401 and 403, so it never redirects. Nginx sends the user to /login on 401.
func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	user := h.getSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	setIdentityHeaders(w.Header(), user)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	redirect := h.getSafeRedirect(r.URL.Query().Get("rd"))

	state, err := randomHex(32)
	if err != nil {
		h.fail(w, http.StatusInternalServerError, "failed to generate the login state", err)
		return
	}

	value, err := h.states.Encode(loginState{State: state, Redirect: redirect}, time.Now().Add(loginStateTtl))
	if err != nil {
		h.fail(w, http.StatusInternalServerError, "failed to encode the login state", err)
		return
	}

	// one cookie per login attempt, so signing in from several tabs at once doesn't
	// overwrite the state of the other tabs
	http.SetCookie(w, &http.Cookie{
		Name:     h.getStateCookieName(state),
		Value:    value,
		Path:     h.basePath + "/callback",
		MaxAge:   int(loginStateTtl.Seconds()),
		HttpOnly: true,
		Secure:   h.conf.IsSecure(),
		SameSite: http.SameSiteLaxMode,
	})

	query := url.Values{}
	query.Set("client_id", h.conf.ClientId)
	query.Set("response_type", "code")
	query.Set("redirect_uri", h.conf.ExternalUrl+"/callback")
	query.Set("scope", "read")
	query.Set("state", state)
	http.Redirect(w, r, h.conf.CasdoorEndpoint+"/login/oauth/authorize?"+query.Encode(), http.StatusFound)
}

func (h *Handler) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if errorCode := query.Get("error"); errorCode != "" {
		h.fail(w, http.StatusBadRequest, fmt.Sprintf("Casdoor login failed: %s %s", errorCode, query.Get("error_description")), nil)
		return
	}

	code := query.Get("code")
	state := query.Get("state")
	if code == "" || len(state) < 16 {
		h.fail(w, http.StatusBadRequest, "missing code or state in the callback", nil)
		return
	}

	cookieName := h.getStateCookieName(state)
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		h.fail(w, http.StatusBadRequest, "the login state is missing or has expired, please sign in again", nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Path:     h.basePath + "/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.conf.IsSecure(),
		SameSite: http.SameSiteLaxMode,
	})

	var ls loginState
	if err = h.states.Decode(cookie.Value, &ls); err != nil || subtle.ConstantTimeCompare([]byte(ls.State), []byte(state)) != 1 {
		h.fail(w, http.StatusBadRequest, "invalid login state, please sign in again", err)
		return
	}

	oauthToken, err := h.casdoor.GetOAuthToken(code, state, casdoorsdk.WithHTTPClient(h.httpClient))
	if err != nil {
		h.fail(w, http.StatusBadGateway, "failed to get the access token from Casdoor", err)
		return
	}

	claims, err := h.verifier.Verify(oauthToken.AccessToken)
	if err != nil {
		h.fail(w, http.StatusBadGateway, "failed to verify the access token from Casdoor", err)
		return
	}

	user := &session.User{
		Id:           claims.Id,
		Organization: claims.Owner,
		Name:         claims.Name,
		Email:        claims.Email,
		Groups:       claims.Groups,
	}
	for _, role := range claims.Roles {
		if role != nil {
			user.Roles = append(user.Roles, role.Name)
		}
	}

	expiresAt := time.Now().Add(h.conf.SessionDuration)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(expiresAt) {
		expiresAt = claims.ExpiresAt.Time
	}

	value, err := h.sessions.Encode(user, expiresAt)
	if err != nil {
		h.fail(w, http.StatusInternalServerError, "failed to encode the session", err)
		return
	}

	http.SetCookie(w, h.newSessionCookie(value, expiresAt))
	log.Printf("user %s/%s signed in", user.Organization, user.Name)
	http.Redirect(w, r, ls.Redirect, http.StatusFound)
}

// handleLogout only ends the session of this service. The Casdoor session stays,
// so the next login is silent unless the user also signs out of Casdoor.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := h.newSessionCookie("", time.Unix(0, 0))
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)

	if rd := r.URL.Query().Get("rd"); rd != "" {
		http.Redirect(w, r, h.getSafeRedirect(rd), http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Signed out.")
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	user := h.getSessionUser(r)
	if user == nil {
		http.Redirect(w, r, h.getLoginUrl(""), http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Signed in as %s/%s.\n", user.Organization, user.Name)
}

func (h *Handler) getSessionUser(r *http.Request) *session.User {
	cookie, err := r.Cookie(h.conf.CookieName)
	if err != nil {
		return nil
	}

	var user session.User
	if err = h.sessions.Decode(cookie.Value, &user); err != nil {
		return nil
	}
	return &user
}

func (h *Handler) newSessionCookie(value string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     h.conf.CookieName,
		Value:    value,
		Domain:   h.conf.CookieDomain,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.conf.IsSecure(),
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handler) getStateCookieName(state string) string {
	return h.conf.CookieName + "_state_" + state[:16]
}

func (h *Handler) getLoginUrl(redirect string) string {
	if redirect == "" {
		return h.conf.ExternalUrl + "/login"
	}
	return h.conf.ExternalUrl + "/login?rd=" + url.QueryEscape(redirect)
}

// getSafeRedirect returns rd if it points to an allowed domain, otherwise the root of
// this service, so the login can't be abused as an open redirect.
func (h *Handler) getSafeRedirect(rd string) string {
	defaultRedirect := h.conf.ExternalUrl + "/"
	if rd == "" {
		return defaultRedirect
	}

	// a path on the host of this service, but not "//host" or "/\host" which browsers treat as another host
	if strings.HasPrefix(rd, "/") && !strings.HasPrefix(rd, "//") && !strings.HasPrefix(rd, "/\\") {
		externalUrl, _ := url.Parse(h.conf.ExternalUrl)
		return externalUrl.Scheme + "://" + externalUrl.Host + rd
	}

	u, err := url.Parse(rd)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return defaultRedirect
	}

	for _, domain := range h.conf.AllowedRedirectDomains {
		if config.MatchDomain(domain, u.Hostname()) {
			return rd
		}
	}
	return defaultRedirect
}

func (h *Handler) fail(w http.ResponseWriter, status int, message string, err error) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	} else {
		log.Print(message)
	}
	http.Error(w, message, status)
}

// getOriginalUrl rebuilds the URL the user asked for from the headers set by the reverse proxy.
func getOriginalUrl(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		return ""
	}

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}

	uri := r.Header.Get("X-Forwarded-Uri")
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}
	return scheme + "://" + host + uri
}

// setIdentityHeaders always sets every header, even empty ones, so the reverse proxy
// overwrites whatever the client sent in the same headers.
func setIdentityHeaders(header http.Header, user *session.User) {
	header.Set(HeaderUser, user.Name)
	header.Set(HeaderUserId, user.Id)
	header.Set(HeaderOrganization, user.Organization)
	header.Set(HeaderEmail, user.Email)
	header.Set(HeaderGroups, strings.Join(user.Groups, ","))
	header.Set(HeaderRoles, strings.Join(user.Roles, ","))
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
