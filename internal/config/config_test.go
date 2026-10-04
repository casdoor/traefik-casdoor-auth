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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, content string) string {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validConfig = `{
  "casdoorEndpoint": "https://door.casdoor.com/",
  "clientId": "id",
  "clientSecret": "secret",
  "externalUrl": "https://auth.example.com/",
  "cookieSecret": "0123456789abcdef0123456789abcdef",
  "cookieDomain": ".Example.com"
}`

func TestLoadDefaults(t *testing.T) {
	conf, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}

	if conf.CasdoorEndpoint != "https://door.casdoor.com" || conf.ExternalUrl != "https://auth.example.com" {
		t.Errorf("trailing slashes not trimmed: %s, %s", conf.CasdoorEndpoint, conf.ExternalUrl)
	}
	if conf.ListenAddr != ":9999" || conf.CookieName != "casdoor_forward_auth" || conf.SessionDuration != 24*time.Hour {
		t.Errorf("unexpected defaults: %+v", conf)
	}
	if conf.CookieDomain != "example.com" {
		t.Errorf("cookie domain: got %s", conf.CookieDomain)
	}
	if want := []string{"auth.example.com", ".example.com"}; !reflect.DeepEqual(conf.AllowedRedirectDomains, want) {
		t.Errorf("allowed redirect domains: got %v, want %v", conf.AllowedRedirectDomains, want)
	}
	if !conf.IsSecure() || conf.BasePath() != "" {
		t.Errorf("IsSecure() = %v, BasePath() = %q", conf.IsSecure(), conf.BasePath())
	}
}

func TestLoadEnv(t *testing.T) {
	t.Setenv("CLIENT_SECRET", "secret-from-env")
	t.Setenv("EXTERNAL_URL", "http://app.example.com/_auth")
	t.Setenv("SESSION_TTL", "1h30m")
	t.Setenv("ALLOWED_REDIRECT_DOMAINS", "app.example.com, .example.org")

	conf, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatal(err)
	}

	if conf.ClientSecret != "secret-from-env" || conf.SessionDuration != 90*time.Minute {
		t.Errorf("env not applied: %+v", conf)
	}
	if conf.IsSecure() || conf.BasePath() != "/_auth" {
		t.Errorf("IsSecure() = %v, BasePath() = %q", conf.IsSecure(), conf.BasePath())
	}
	if want := []string{"app.example.com", ".example.org"}; !reflect.DeepEqual(conf.AllowedRedirectDomains, want) {
		t.Errorf("allowed redirect domains: got %v, want %v", conf.AllowedRedirectDomains, want)
	}
}

func TestLoadInvalid(t *testing.T) {
	cases := map[string]string{
		`"clientSecret": "secret",`:                           ``,
		`"cookieSecret": "0123456789abcdef0123456789abcdef",`: `"cookieSecret": "short",`,
		`"externalUrl": "https://auth.example.com/",`:         `"externalUrl": "auth.example.com",`,
		`"cookieDomain": ".Example.com"`:                      `"cookieDomain": "example.org"`,
		`"casdoorEndpoint": "https://door.casdoor.com/",`:     `"casdoorEndpoint": "ftp://door.casdoor.com",`,
		`"clientId": "id",`:                                   `"clientId": "id", "sessionTtl": "forever",`,
	}
	for old, replacement := range cases {
		content := strings.Replace(validConfig, old, replacement, 1)
		if _, err := Load(writeConfig(t, content)); err == nil {
			t.Errorf("expected an error after replacing %s with %q", old, replacement)
		}
	}
}

func TestMatchDomain(t *testing.T) {
	cases := []struct {
		domain string
		host   string
		want   bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "app.example.com", false},
		{".example.com", "example.com", true},
		{".example.com", "app.example.com", true},
		{".example.com", "a.b.example.com", true},
		{".example.com", "evilexample.com", false},
		{".example.com", "example.com.evil.com", false},
	}
	for _, c := range cases {
		if got := MatchDomain(c.domain, c.host); got != c.want {
			t.Errorf("MatchDomain(%q, %q) = %v, want %v", c.domain, c.host, got, c.want)
		}
	}
}
