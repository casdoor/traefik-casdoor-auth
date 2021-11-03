package main

import (
	"github.com/gin-gonic/gin"
	"traefikcasdoor/internal/handler"
)

func main(){
	r := gin.Default()
	r.GET("/echo", handler.EchoHandler)
	r.GET("/auth",handler.ForwardAuthHandler)
	r.GET("/callback",handler.CasdoorCallbackHandler)
	r.Run(":9999")
}