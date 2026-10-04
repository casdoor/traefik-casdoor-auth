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
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestRoundTrip(t *testing.T) {
	codec := NewCodec(testSecret, "session")
	user := User{Id: "1", Organization: "built-in", Name: "alice", Groups: []string{"built-in/dev"}}

	value, err := codec.Encode(user, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	var got User
	if err = codec.Decode(value, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, user) {
		t.Fatalf("got %+v, want %+v", got, user)
	}
}

func TestRejects(t *testing.T) {
	codec := NewCodec(testSecret, "session")
	value, _ := codec.Encode(User{Name: "alice"}, time.Now().Add(time.Hour))
	payload, signature, _ := strings.Cut(value, ".")

	var user User
	if err := NewCodec("another secret, at least 32 chars", "session").Decode(value, &user); !errors.Is(err, ErrInvalid) {
		t.Errorf("other secret: got %v, want ErrInvalid", err)
	}
	if err := NewCodec(testSecret, "login-state").Decode(value, &user); !errors.Is(err, ErrInvalid) {
		t.Errorf("other purpose: got %v, want ErrInvalid", err)
	}
	if err := codec.Decode(payload+"x."+signature, &user); !errors.Is(err, ErrInvalid) {
		t.Errorf("tampered payload: got %v, want ErrInvalid", err)
	}
	if err := codec.Decode(payload, &user); !errors.Is(err, ErrInvalid) {
		t.Errorf("no signature: got %v, want ErrInvalid", err)
	}

	expired, _ := codec.Encode(User{Name: "alice"}, time.Now().Add(-time.Second))
	if err := codec.Decode(expired, &user); !errors.Is(err, ErrExpired) {
		t.Errorf("expired: got %v, want ErrExpired", err)
	}
}
