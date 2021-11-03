package handler

import (
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
)

func EchoHandler(c *gin.Context) {
	body, _ := ioutil.ReadAll(c.Request.Body)
	fmt.Println(c.Request.Header,string(body))
	c.JSON(200,gin.H{
		"header":c.Request.Header,
		"body":string(body),
	})
}

func TestHandler(c *gin.Context){
	body, _ := ioutil.ReadAll(c.Request.Body)
	fmt.Println(c.Request.Header,string(body))
	var replacement Replacement 
	replacement.ShouldReplaceBody=true
	replacement.ShouldReplaceHeader=true
	replacement.Body=string(body)+"plus"
	replacement.Header=c.Request.Header.Clone()
	replacement.Header["Test"]=[]string{"modified"}
	c.JSON(200,replacement)
}
