package plugindemo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"time"
)

// Config the plugin configuration.
type Config struct {
	MultationWebhook string `json:"multationWebhook,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		MultationWebhook: "",
	}
}

// Demo a Demo plugin.
type Plugin struct {
	next http.Handler
	webhook string
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	
	return &Plugin{
		next: next,
		webhook: config.MultationWebhook,
	}, nil
}

func (a *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if a.webhook==""{
		a.next.ServeHTTP(rw, req)//webhook is disabled, let it pass
		return
	}
	//forward this to the specified webhook
	//read the body
	requestBody,err:=ioutil.ReadAll(req.Body)
	if err!=nil{
		fmt.Fprintf(rw,"error when reading body: %s",err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	//construct request
	client := &http.Client{Timeout: 5 * time.Second}
	reqForWebhook, err := http.NewRequest(req.Method,a.webhook,bytes.NewReader(requestBody))
	if err!=nil{
		fmt.Fprintf(rw,"error when creating request: %s",err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	//copy the header
	reqForWebhook.Header=req.Header.Clone()
	// for k,vlist:=range req.Header {
	// 	for _,v:=range vlist{
	// 		reqForWebhook.Header.Add(k,v)
	// 	}
	// }
	//encode necessary information into header
	//copy the cookie
	cookie,err:=req.Cookie("Casbin-Plugin-ClientCode")
	if err!=http.ErrNoCookie{
		reqForWebhook.AddCookie(cookie)
	}
	_,_=client.Do(reqForWebhook)
	req.Body= ioutil.NopCloser(bytes.NewReader(requestBody))
	a.next.ServeHTTP(rw, req)
}