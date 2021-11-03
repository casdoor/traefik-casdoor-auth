package handler

import (
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
)

func EchoHandler(c *gin.Context) {
	body, _ := ioutil.ReadAll(c.Request.Body)
	fmt.Println(c.Request.Header,body)
	c.JSON(200,gin.H{
		"header":c.Request.Header,
		"body":body,
	})
}
