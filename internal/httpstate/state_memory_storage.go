// Copyright 2021 The casbin Authors. All Rights Reserved.
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
package httpstate

import (
	"fmt"
	"math/rand"
	"sync"
)

type StateMemoryStorage struct {
	sync.Mutex
	content map[int]*State
}

func NewStateMemoryStorage() (*StateMemoryStorage, error) {
	var res StateMemoryStorage
	res.content = make(map[int]*State)
	return &res, nil
}

func (s *StateMemoryStorage) SetState(state *State) (int, error) {
	s.Lock()
	defer s.Unlock()

	nonce := rand.Int()
	s.content[nonce] = state
	return nonce, nil
}

func (s *StateMemoryStorage) PopState(nonce int) (*State, error) {
	s.Lock()
	defer s.Unlock()
	state, ok := s.content[nonce]
	if !ok {
		return nil, fmt.Errorf("state %d not found", nonce)
	}
	delete(s.content, nonce)
	return state, nil

}
func (s *StateMemoryStorage) GetState(nonce int) (*State, error) {
	s.Lock()
	defer s.Unlock()
	state, ok := s.content[nonce]
	if !ok {
		return nil, fmt.Errorf("state %d not found", nonce)
	}
	return state, nil

}
