package handler

import "traefikcasdoor/internal/httpstate"
import "log"

var stateStorage httpstate.StateStorage
func init(){
	storage,err:=httpstate.NewStateMemoryStorage()
	if err!=nil{
		log.Printf("error happened when creating StateMemoryStorage\n")
		return
	}
	stateStorage=storage
}