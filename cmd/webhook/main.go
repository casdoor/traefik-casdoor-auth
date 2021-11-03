package main

import (
	"github.com/gin-gonic/gin"
	"traefikcasdoor/internal/handler"
)

func main(){
	r := gin.Default()
	r.Any("/echo", handler.TestHandler)
	r.Any("/auth",handler.ForwardAuthHandler)
	r.Any("/callback",handler.CasdoorCallbackHandler)
	r.Run(":9999")
}