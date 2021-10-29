package httpstate


type StateStorage interface{
	SetState(state *State)(int,error)
	PopState(nonce int)(*State,error)
}