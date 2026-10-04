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

package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddr             string   `json:"listenAddr"`
	CasdoorEndpoint        string   `json:"casdoorEndpoint"`
	ClientId               string   `json:"clientId"`
	ClientSecret           string   `json:"clientSecret"`
	Certificate            string   `json:"certificate"`
	ExternalUrl            string   `json:"externalUrl"`
	CookieSecret           string   `json:"cookieSecret"`
	CookieName             string   `json:"cookieName"`
	CookieDomain           string   `json:"cookieDomain"`
	SessionTtl             string   `json:"sessionTtl"`
	AllowedRedirectDomains []string `json:"allowedRedirectDomains"`

	SessionDuration time.Duration `json:"-"`
}

// Load reads the config from the JSON file at path (if path is not empty), lets the
// environment variables override it, fills in the defaults and validates the result.
func Load(path string) (*Config, error) {
	conf := &Config{}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err = json.Unmarshal(data, conf); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
		}
	}

	conf.loadEnv()
	if err := conf.normalize(); err != nil {
		return nil, err
	}

	return conf, nil
}

func (conf *Config) loadEnv() {
	vars := map[string]*string{
		"LISTEN_ADDR":      &conf.ListenAddr,
		"CASDOOR_ENDPOINT": &conf.CasdoorEndpoint,
		"CLIENT_ID":        &conf.ClientId,
		"CLIENT_SECRET":    &conf.ClientSecret,
		"CERTIFICATE":      &conf.Certificate,
		"EXTERNAL_URL":     &conf.ExternalUrl,
		"COOKIE_SECRET":    &conf.CookieSecret,
		"COOKIE_NAME":      &conf.CookieName,
		"COOKIE_DOMAIN":    &conf.CookieDomain,
		"SESSION_TTL":      &conf.SessionTtl,
	}
	for name, field := range vars {
		if value, ok := os.LookupEnv(name); ok {
			*field = value
		}
	}

	if value, ok := os.LookupEnv("ALLOWED_REDIRECT_DOMAINS"); ok {
		conf.AllowedRedirectDomains = nil
		for _, domain := range strings.Split(value, ",") {
			if domain = strings.TrimSpace(domain); domain != "" {
				conf.AllowedRedirectDomains = append(conf.AllowedRedirectDomains, domain)
			}
		}
	}
}

func (conf *Config) normalize() error {
	if conf.ListenAddr == "" {
		conf.ListenAddr = ":9999"
	}
	if conf.CookieName == "" {
		conf.CookieName = "casdoor_forward_auth"
	}
	if conf.SessionTtl == "" {
		conf.SessionTtl = "24h"
	}

	required := map[string]string{
		"casdoorEndpoint": conf.CasdoorEndpoint,
		"clientId":        conf.ClientId,
		"clientSecret":    conf.ClientSecret,
		"externalUrl":     conf.ExternalUrl,
		"cookieSecret":    conf.CookieSecret,
	}
	for _, name := range []string{"casdoorEndpoint", "clientId", "clientSecret", "externalUrl", "cookieSecret"} {
		if required[name] == "" {
			return fmt.Errorf("config %s is required", name)
		}
	}

	conf.CasdoorEndpoint = strings.TrimRight(conf.CasdoorEndpoint, "/")
	if _, err := parseHttpUrl(conf.CasdoorEndpoint); err != nil {
		return fmt.Errorf("invalid casdoorEndpoint: %w", err)
	}

	conf.ExternalUrl = strings.TrimRight(conf.ExternalUrl, "/")
	externalUrl, err := parseHttpUrl(conf.ExternalUrl)
	if err != nil {
		return fmt.Errorf("invalid externalUrl: %w", err)
	}

	if len(conf.CookieSecret) < 32 {
		return fmt.Errorf("cookieSecret must be at least 32 characters long")
	}

	conf.SessionDuration, err = time.ParseDuration(conf.SessionTtl)
	if err != nil || conf.SessionDuration <= 0 {
		return fmt.Errorf("invalid sessionTtl: %s", conf.SessionTtl)
	}

	// a cookie domain always covers its subdomains, with or without the leading dot
	conf.CookieDomain = strings.TrimPrefix(strings.ToLower(conf.CookieDomain), ".")
	if conf.CookieDomain != "" && !MatchDomain("."+conf.CookieDomain, externalUrl.Hostname()) {
		return fmt.Errorf("the host of externalUrl (%s) must be inside cookieDomain (%s), otherwise the browser rejects the session cookie", externalUrl.Hostname(), conf.CookieDomain)
	}

	if len(conf.AllowedRedirectDomains) == 0 {
		conf.AllowedRedirectDomains = []string{externalUrl.Hostname()}
		if conf.CookieDomain != "" {
			conf.AllowedRedirectDomains = append(conf.AllowedRedirectDomains, "."+conf.CookieDomain)
		}
	}
	for i, domain := range conf.AllowedRedirectDomains {
		conf.AllowedRedirectDomains[i] = strings.ToLower(domain)
	}

	return nil
}

func (conf *Config) IsSecure() bool {
	return strings.HasPrefix(conf.ExternalUrl, "https://")
}

// BasePath is the path of externalUrl, so the service can be mounted under a path
// prefix of an existing host, e.g., "https://app.example.com/_auth".
func (conf *Config) BasePath() string {
	u, _ := url.Parse(conf.ExternalUrl)
	return strings.TrimRight(u.Path, "/")
}

// MatchDomain reports whether host is covered by domain: "example.com" only matches
// itself, ".example.com" matches example.com and all of its subdomains.
func MatchDomain(domain string, host string) bool {
	domain = strings.ToLower(domain)
	host = strings.ToLower(host)
	if strings.HasPrefix(domain, ".") {
		return host == domain[1:] || strings.HasSuffix(host, domain)
	}
	return host == domain
}

func parseHttpUrl(rawUrl string) (*url.URL, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%s is not an absolute http(s) URL", rawUrl)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain a query or fragment", rawUrl)
	}
	return u, nil
}
