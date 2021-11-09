package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main(){
	client := &http.Client{Timeout: 5 * time.Second}
	client.CheckRedirect=func(req *http.Request, via []*http.Request) error {return http.ErrUseLastResponse}
	reqForWebhook, _ := http.NewRequest("GET", "http://127.0.0.1:9999/auth", bytes.NewReader([]byte{'a','b'}))
	resp,_:=client.Do(reqForWebhook)
	fmt.Print(resp.StatusCode)
	responseBody, err := ioutil.ReadAll(resp.Body)
	fmt.Print(string(responseBody),err)

}