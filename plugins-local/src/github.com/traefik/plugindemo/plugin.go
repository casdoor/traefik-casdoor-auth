package plugindemo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"time"
)

type Replacement struct {
	ShouldReplaceBody bool   `json:"shouldReplaceBody"`
	Body              string `json:"body"`
	// ShouldReplaceUri bool `json:"shouldReplaceUri"`
	// Uri string `json:"uri"`
	ShouldReplaceHeader bool                `json:"shouldReplaceHeader"`
	Header              map[string][]string `json:"Header"`
}

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
	next    http.Handler
	webhook string
}

/* New created a new Demo plugin.
This plugin do the following acitions successively

1. forward the request to the casbin auth webhook to ask for the opinion.
2. if the status code is 2xx，this plugin will modify the original request according to the response(which is actually a json-marshaled Replacement struct). Otherwise, the body as long as the status code will be directly send back(to client), without sending original request to following handlers.
*/
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {

	return &Plugin{
		next:    next,
		webhook: config.MultationWebhook,
	}, nil
}

func (p *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if p.webhook == "" {
		p.next.ServeHTTP(rw, req) //webhook is disabled, let it pass
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	client.CheckRedirect=func(req *http.Request, via []*http.Request) error {return http.ErrUseLastResponse}
	//forward this to the specified webhook
	reqForWebhook, err := p.copyRequestForWebhook(req)
	if err != nil {
		fmt.Fprintf(rw, "%s", err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	//send out the request
	resp, err := client.Do(reqForWebhook)
	if err != nil {
		fmt.Fprintf(rw, "error when forwarding request: %s", err.Error())
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	//if the status is 2xx,then the request will be allowed to be proceeded.
	//but, the body, the header, the uri(not url) may be replaced
	//if the response want we to do so.
	// if the status code is not what we want, the response will be directly returned to user
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		//pass, replace the header if necessary
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(rw, "error when reading response: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		var replacement Replacement
		err = json.Unmarshal(responseBody, &replacement)
		if err != nil {
			fmt.Fprintf(rw, "error when unmarshal response body: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		modifiedRequest, err := p.modifyRequestForTraefik(req, replacement)
		if err != nil {
			fmt.Fprintf(rw, " %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		p.next.ServeHTTP(rw, modifiedRequest)
	} else {
		//cannot let it pass
		responseHeader := rw.Header()
		for k, v := range resp.Header {
			responseHeader[k] = v
		}
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(rw, "error when reading response: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(resp.StatusCode)
		_, err = rw.Write(responseBody)
		if err != nil {
			fmt.Fprintf(rw, "error when writing response: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

}

/*
construct a request used to forward to the webhook plugin.
The request method,and the body and the header sent into the plugin will be copied.

(because when a request enters plugin of traefik, the original information like uri and url will be put into header, so replicating the header will be enough for webhook to get all information.)

Considering that the Body field in http.Request is a read closer, thus unavailable for subsequent procedures to read. Therefore, the Body will be restored by ioutil.NopCloser

*/
func (p *Plugin) copyRequestForWebhook(req *http.Request) (*http.Request, error) {
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error when reading body: %s", err.Error())
	}
	req.Body.Close()
	//restore this body into the req so that it can still be read
	req.Body = ioutil.NopCloser(bytes.NewReader(requestBody))

	//construct request with extracted body
	reqForWebhook, err := http.NewRequest(req.Method, p.webhook, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error when creating request: %s", err.Error())
	}
	//copy the header
	reqForWebhook.Header = req.Header.Clone()
	//add uri if there is
	reqForWebhook.Header.Set("X-Forwarded-URI",req.RequestURI)

	//and the cookie for casbin-plugin
	// cookie,err:=req.Cookie("Casbin-Plugin-ClientCode")
	// if err==nil{
	// 	reqForWebhook.AddCookie(cookie)
	// }else if err!=http.ErrNoCookie{
	// 	return nil,fmt.Errorf("error when copting cookie: %s",err.Error())
	// }
	return reqForWebhook, nil

}

/*
construct a request based on the original and the directives of webhook to forward to subsequent handler on the middleware chain.
*/

func (p *Plugin) modifyRequestForTraefik(req *http.Request, replacement Replacement) (*http.Request, error) {
	var err error
	oldBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error when read request body: %s", err.Error())
	}
	req.Body.Close()
	var newRequest *http.Request
	if replacement.ShouldReplaceBody {
		newRequest, err = http.NewRequest(req.Method, p.webhook, bytes.NewReader([]byte(replacement.Body)))
	} else {
		newRequest, err = http.NewRequest(req.Method, p.webhook, bytes.NewReader(oldBody))
	}

	if err != nil {
		return nil, fmt.Errorf("error when construct new request: %s", err.Error())
	}

	if replacement.ShouldReplaceHeader {
		newRequest.Header = replacement.Header
	} else {
		newRequest.Header = req.Header.Clone()
	}
	newRequest.Host = req.Host
	cookies := req.Cookies()
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	return newRequest, nil

}
