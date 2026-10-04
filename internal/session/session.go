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

package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalid = errors.New("invalid cookie value")
	ErrExpired = errors.New("cookie value has expired")
)

// User is the identity kept in the session cookie and passed to the backend.
type User struct {
	Id           string   `json:"id"`
	Organization string   `json:"organization"`
	Name         string   `json:"name"`
	Email        string   `json:"email,omitempty"`
	Groups       []string `json:"groups,omitempty"`
	Roles        []string `json:"roles,omitempty"`
}

// Codec signs values into cookie-safe strings with HMAC-SHA256. Each purpose gets its
// own key, so a value signed for one cookie can't be replayed as another cookie.
type Codec struct {
	key []byte
}

type envelope struct {
	ExpiresAt int64           `json:"exp"`
	Data      json.RawMessage `json:"data"`
}

func NewCodec(secret string, purpose string) *Codec {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(purpose))
	return &Codec{key: mac.Sum(nil)}
}

func (c *Codec) Encode(value interface{}, expiresAt time.Time) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(envelope{ExpiresAt: expiresAt.Unix(), Data: data})
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(c.sign(encodedPayload)), nil
}

func (c *Codec) Decode(s string, value interface{}) error {
	encodedPayload, encodedSignature, ok := strings.Cut(s, ".")
	if !ok {
		return ErrInvalid
	}

	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil || !hmac.Equal(signature, c.sign(encodedPayload)) {
		return ErrInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return ErrInvalid
	}

	var env envelope
	if err = json.Unmarshal(payload, &env); err != nil {
		return ErrInvalid
	}
	if time.Now().Unix() >= env.ExpiresAt {
		return ErrExpired
	}

	if err = json.Unmarshal(env.Data, value); err != nil {
		return ErrInvalid
	}
	return nil
}

func (c *Codec) sign(s string) []byte {
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(s))
	return mac.Sum(nil)
}
