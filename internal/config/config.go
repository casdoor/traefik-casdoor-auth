// Copyright 2021 The Casdoor Authors. All Rights Reserved.
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
	_ "embed"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/casdoor/casdoor-go-sdk/auth"
)

//go:embed token.pem
var EmbeddedJwtSecret string

type Config struct {
	CasdoorEndpoint     string `json:"casdoorEndpoint"`
	CasdoorClientId     string `json:"casdoorClientId"`
	CasdoorClientSecret string `json:"casdoorClientSecret"`
	CasdoorOrganization string `json:"casdoorOrganization"`
	CasdoorApplication  string `json:"casdoorApplication"`
	PluginEndpoint      string `json:"pluginEndpoint"`
}

var CurrentConfig Config

// JWKSResponse represents the JWKS endpoint response
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string   `json:"kid"`
	X5c []string `json:"x5c"`
}

// fetchCertificateFromJWKS fetches the certificate from Casdoor's JWKS endpoint.
// This ensures the auth service always uses the correct certificate that matches
// the Casdoor instance, regardless of when the certificate was generated.
func fetchCertificateFromJWKS(endpoint string) (string, error) {
	jwksURL := fmt.Sprintf("%s/.well-known/jwks", endpoint)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(jwksURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks JWKSResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return "", fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Find the cert-built-in key or use the first available key
	var x5c string
	for _, key := range jwks.Keys {
		if len(key.X5c) > 0 {
			if key.Kid == "cert-built-in" {
				x5c = key.X5c[0]
				break
			}
			if x5c == "" {
				x5c = key.X5c[0] // fallback to first key with x5c
			}
		}
	}

	if x5c == "" {
		return "", fmt.Errorf("no x5c certificate found in JWKS")
	}

	// Convert base64 DER to PEM format
	derBytes, err := base64.StdEncoding.DecodeString(x5c)
	if err != nil {
		return "", fmt.Errorf("failed to decode x5c: %w", err)
	}

	// Parse to verify it's a valid certificate
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Encode as PEM
	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}

func LoadConfigFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read config file %s", path)
	}

	err = json.Unmarshal(data, &CurrentConfig)
	if err != nil {
		log.Fatalf("failed to unmarshal config file %s: %s", path, err.Error())
	}

	// Try to fetch certificate from JWKS endpoint first.
	// This ensures we always use the correct certificate from the actual Casdoor instance,
	// which fixes the issue where embedded token.pem doesn't match the deployed Casdoor's certificate.
	jwtSecret := EmbeddedJwtSecret
	if fetchedCert, err := fetchCertificateFromJWKS(CurrentConfig.CasdoorEndpoint); err == nil {
		log.Printf("Successfully fetched certificate from JWKS endpoint")
		jwtSecret = fetchedCert
	} else {
		log.Printf("Warning: Failed to fetch certificate from JWKS (%v), falling back to embedded certificate", err)
	}

	auth.InitConfig(
		CurrentConfig.CasdoorEndpoint,
		CurrentConfig.CasdoorClientId,
		CurrentConfig.CasdoorClientSecret,
		jwtSecret,
		CurrentConfig.CasdoorOrganization,
		CurrentConfig.CasdoorApplication,
	)
}
