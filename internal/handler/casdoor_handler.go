package handler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/casdoor/casdoor-go-sdk/auth"
	"traefikcasdoor/internal/config"
	"traefikcasdoor/internal/httpstate"

	"github.com/gin-gonic/gin"
)

func ForwardAuthHandler(c *gin.Context){
	// fmt.Println(c.Request.Host)
	// fmt.Println(body)
	fmt.Println(c.Request.Header)
	
	clientcode,err:=c.Cookie("client-code")
	if err!=nil{
		fmt.Println("no client code found in cookie")
		ForwardAuthHandlerWithoutState(c)
		return
	}
	clientstate,err:=c.Cookie("client-state")
	if err!=nil{
		fmt.Println("no state found in cookie")
		ForwardAuthHandlerWithoutState(c)
		return
	}
	if err:=checkCode(clientcode,clientstate);err!=nil{
		fmt.Printf("invalid code and state %s\n",err.Error())
		ForwardAuthHandlerWithoutState(c)
		return
	}
	ForwardAuthHandlerWithState(c)
}

func ForwardAuthHandlerWithoutState(c *gin.Context){
	body, _ := ioutil.ReadAll(c.Request.Body)
	//generate state: we need to storage the body and header. These information will be used when callback is called
	state:=httpstate.NewState(c.Request.Method,c.Request.Header,body)
	stateNonce,err:=stateStorage.SetState(state)
	if err!=nil{
		log.Printf("error happened when setting state: %s\n",err.Error())
		c.JSON(http.StatusInternalServerError,err.Error())
		return
	}
	//generate redirect url
	redirectURL:=fmt.Sprintf("%s/login/oauth/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=read&state=%s",config.CasdoorEndpoint,
	config.CasfoorClientId,
	config.PluginCallback,
	strconv.Itoa(stateNonce))

	c.Redirect(307,redirectURL)
}
func ForwardAuthHandlerWithState(c *gin.Context){
	fmt.Println("client code checked")
	stateString,_:=c.Cookie("client-state")
	stateNonce,_:=strconv.Atoi(stateString)
	state,err:=stateStorage.PopState(stateNonce)
	var replacement Replacement 
	replacement.ShouldReplaceBody=true
	replacement.ShouldReplaceHeader=true
	if err!=nil{
		fmt.Printf("no related state found, state nonce %s\n",stateString)
		replacement.ShouldReplaceBody=false
		replacement.ShouldReplaceHeader=false
		c.JSON(200,replacement)
		return 
	}
	
	replacement.Body=string(state.Body)
	replacement.Header=state.Header
	c.JSON(200,replacement)
	return 

}


func CasdoorCallbackHandler(c *gin.Context){
	stateString:=c.Query("state")
	code:=c.Query("code")
	//write into cookie
	c.SetCookie("client-code",code,3600,"/",config.PluginDomain,false,true)
	c.SetCookie("client-state",stateString,3600,"/",config.PluginDomain,false,true)
	stateNonce,_:=strconv.Atoi(stateString)
	state,err:=stateStorage.GetState(stateNonce)
	if err!=nil{
		fmt.Printf("no related state found, state nonce %s\n",stateNonce)
		c.JSON(500,gin.H{
			"error":"no related state found, state nonce "+stateString,
		})
		return
	}
	//construct the redirect 
	scheme:=state.Header.Get("X-Forwarded-Proto")
	host:=state.Header.Get("X-Forwarded-Host")
	uri:=state.Header.Get("X-Forwarded-URI")
	url:=fmt.Sprintf("%s://%s%s",scheme,host,uri)
	c.Redirect(307,url)


}



func checkCode(code ,state string)error{
	_, err := auth.GetOAuthToken(code, state)
	return err
}