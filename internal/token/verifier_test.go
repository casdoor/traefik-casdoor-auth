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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"github.com/golang-jwt/jwt/v4"
)

type keyPair struct {
	key     *rsa.PrivateKey
	certDer []byte
}

func newKeyPair(t *testing.T) *keyPair {
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
	return &keyPair{key: key, certDer: certDer}
}

func (kp *keyPair) sign(t *testing.T, kid string, tokenType string) string {
	claims := casdoorsdk.Claims{
		User:      casdoorsdk.User{Owner: "built-in", Name: "alice"},
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  []string{"client-id"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(kp.key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func newJwksServer(t *testing.T, kid string, kp *keyPair) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]interface{}{
				{"kid": kid, "x5c": []string{base64.StdEncoding.EncodeToString(kp.certDer)}},
			},
		})
	}))
	t.Cleanup(server.Close)
	return server
}

func TestVerifyWithJwks(t *testing.T) {
	kp := newKeyPair(t)
	server := newJwksServer(t, "cert-built-in", kp)
	verifier := NewVerifier(server.URL, "client-id", "", server.Client())

	claims, err := verifier.Verify(kp.sign(t, "cert-built-in", "access-token"))
	if err != nil || claims.Name != "alice" {
		t.Fatalf("valid token: got %v, %v", claims, err)
	}

	if _, err = verifier.Verify(newKeyPair(t).sign(t, "cert-built-in", "access-token")); err == nil {
		t.Error("token signed by another key: expected an error")
	}
	if _, err = verifier.Verify(kp.sign(t, "cert-unknown", "access-token")); err == nil {
		t.Error("token with an unknown key ID: expected an error")
	}
	if _, err = verifier.Verify(kp.sign(t, "cert-built-in", "refresh-token")); err == nil {
		t.Error("refresh token: expected an error")
	}
}

func TestVerifyWithCertificate(t *testing.T) {
	kp := newKeyPair(t)
	certificate := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: kp.certDer}))
	verifier := NewVerifier("http://127.0.0.1:1", "client-id", certificate, http.DefaultClient)

	if _, err := verifier.Verify(kp.sign(t, "", "access-token")); err != nil {
		t.Fatalf("valid token: %v", err)
	}

	verifier = NewVerifier("http://127.0.0.1:1", "another-client-id", certificate, http.DefaultClient)
	if _, err := verifier.Verify(kp.sign(t, "", "access-token")); err == nil {
		t.Error("token for another client: expected an error")
	}
}
