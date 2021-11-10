// Copyright 2021 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package handler

import (
	"fmt"
	"io/ioutil"

	"github.com/gin-gonic/gin"
)

func EchoHandler(c *gin.Context) {
	body, _ := ioutil.ReadAll(c.Request.Body)
	fmt.Println(c.Request.Header, string(body))
	c.JSON(200, gin.H{
		"header": c.Request.Header,
		"body":   string(body),
	})
}

func TestHandler(c *gin.Context) {
	body, _ := ioutil.ReadAll(c.Request.Body)
	fmt.Println(c.Request.Header, string(body))
	var replacement Replacement
	replacement.ShouldReplaceBody = true
	replacement.ShouldReplaceHeader = true
	replacement.Body = string(body) + "plus"
	replacement.Header = c.Request.Header.Clone()
	replacement.Header["Test"] = []string{"modified"}
	c.JSON(200, replacement)
}
