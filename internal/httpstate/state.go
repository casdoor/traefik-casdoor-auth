package httpstate

import "net/http"

type State struct{
	Header http.Header
	Method string
	Body []byte
}

func NewState(method string, header http.Header,body []byte)*State{
	var tmp State
	tmp.Method=method
	tmp.Header=header.Clone()
	tmp.Body=make([]byte,len(body))
	copy(tmp.Body,body)
	return &tmp
}
