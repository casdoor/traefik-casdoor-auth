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
	//TODO: check cookie
	// fmt.Println(c.Request.Host)
	// fmt.Println(body)
	// fmt.Println(c.Request.Header)
	//generate state: we need to storage the body and header. These information will be used when callback is called
	body, _ := ioutil.ReadAll(c.Request.Body)
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

	c.Redirect(http.StatusTemporaryRedirect,redirectURL)
}

func CasdoorCallbackHandler(c *gin.Context){
	state:=c.Query("state")
	code:=c.Query("code")
	//write into cookie
	c.SetCookie("client-code",code,3600,"/",config.PluginDomain,false,true)
	c.SetCookie("client-state",code,3600,"/",config.PluginDomain,false,true)
	_=state
	//construct the redirect 
}



func checkCode(code ,state string)error{
	_, err := auth.GetOAuthToken(code, state)
	return err
}