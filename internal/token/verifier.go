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

package token

import (
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// jwksRefreshInterval limits how often an unknown key ID makes us fetch the JWKS again.
const jwksRefreshInterval = 30 * time.Second

// Verifier checks the access tokens issued by Casdoor. The signing certificate is either
// configured explicitly or looked up by the token's key ID in Casdoor's JWKS, which
// keeps working after the certificate of the application is changed or rotated.
type Verifier struct {
	endpoint    string
	clientId    string
	certificate string
	httpClient  *http.Client

	mutex     sync.Mutex
	keys      map[string]string
	fetchedAt time.Time
}

func NewVerifier(endpoint string, clientId string, certificate string, httpClient *http.Client) *Verifier {
	return &Verifier{
		endpoint:    endpoint,
		clientId:    clientId,
		certificate: certificate,
		httpClient:  httpClient,
	}
}

func (v *Verifier) Verify(accessToken string) (*casdoorsdk.Claims, error) {
	certificate := v.certificate
	if certificate == "" {
		kid, err := getKeyId(accessToken)
		if err != nil {
			return nil, err
		}

		certificate, err = v.getCertificate(kid)
		if err != nil {
			return nil, err
		}
	}

	client := casdoorsdk.NewClient(v.endpoint, v.clientId, "", certificate, "", "")
	claims, err := client.ParseJwtToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}
	if claims.IsRefreshToken() {
		return nil, fmt.Errorf("invalid access token: got a refresh token")
	}
	if !containsString(claims.Audience, v.clientId) {
		return nil, fmt.Errorf("invalid access token: audience %v doesn't contain client ID %s", claims.Audience, v.clientId)
	}
	if claims.Name == "" {
		return nil, fmt.Errorf("invalid access token: no user name")
	}

	return claims, nil
}

func (v *Verifier) getCertificate(kid string) (string, error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if certificate, ok := v.keys[kid]; ok {
		return certificate, nil
	}

	if time.Since(v.fetchedAt) < jwksRefreshInterval {
		return "", fmt.Errorf("signing key %q not found in the JWKS of Casdoor", kid)
	}

	keys, err := v.fetchJwks()
	if err != nil {
		return "", err
	}
	v.keys = keys
	v.fetchedAt = time.Now()

	if certificate, ok := v.keys[kid]; ok {
		return certificate, nil
	}
	return "", fmt.Errorf("signing key %q not found in the JWKS of Casdoor", kid)
}

func (v *Verifier) fetchJwks() (map[string]string, error) {
	resp, err := v.httpClient.Get(v.endpoint + "/.well-known/jwks")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the JWKS of Casdoor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch the JWKS of Casdoor: HTTP %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string   `json:"kid"`
			X5c []string `json:"x5c"`
		} `json:"keys"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to parse the JWKS of Casdoor: %w", err)
	}

	keys := map[string]string{}
	for _, key := range jwks.Keys {
		if len(key.X5c) == 0 {
			continue
		}

		der, err := base64.StdEncoding.DecodeString(key.X5c[0])
		if err != nil {
			continue
		}
		keys[key.Kid] = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	}

	return keys, nil
}

func getKeyId(accessToken string) (string, error) {
	encodedHeader, _, ok := strings.Cut(accessToken, ".")
	if !ok {
		return "", fmt.Errorf("invalid access token: not a JWT")
	}

	header, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return "", fmt.Errorf("invalid access token: %w", err)
	}

	var jwtHeader struct {
		Kid string `json:"kid"`
	}
	if err = json.Unmarshal(header, &jwtHeader); err != nil {
		return "", fmt.Errorf("invalid access token: %w", err)
	}
	return jwtHeader.Kid, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
