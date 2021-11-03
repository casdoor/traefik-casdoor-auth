package httpstate

import (
	"fmt"
	"math/rand"
	"sync"
)

type StateMemoryStorage struct{
	sync.Mutex
	content map[int]*State
}

func NewStateMemoryStorage()(*StateMemoryStorage,error){
	var res StateMemoryStorage
	res.content=make(map[int]*State)
	return &res,nil
}


func(s *StateMemoryStorage)SetState(state *State)(int,error){
	s.Lock()
	defer s.Unlock()
	
	nonce:=rand.Int()
	s.content[nonce]=state
	return nonce,nil
}

func (s *StateMemoryStorage)PopState(nonce int)(*State,error){
	s.Lock()
	defer s.Unlock()
	state,ok:=s.content[nonce]
	if !ok{
		return nil,fmt.Errorf("state %d not found",nonce)
	}
	delete(s.content,nonce)
	return state,nil

}