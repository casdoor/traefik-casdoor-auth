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
package config

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/casdoor/casdoor-go-sdk/auth"
)
var CasdoorJwtSecret = "CasdoorSecret"
type Config struct{
	CasdoorEndpoint string `json:"casdoorEndpoint"`
	CasdoorClientId string `json:"casdoorClientId"`
	CasdoorClientSecret string `json:"casdoorClientSecret"`
	CasdoorOrganization string `json:"casdoorOrganization"`
	CasdoorApplication string `json:"casdoorApplication"`
	PluginEndpoint string `json:"pluginEndpoint"`
}

var CurrentConfig Config
func LoadConfigFile(path string){
	data, err := ioutil.ReadFile(path)
	if err!=nil{
		log.Fatalf("failed to read config file %s",path)
	}

	err=json.Unmarshal(data,&CurrentConfig)
	if err!=nil{
		log.Fatalf("failed to unmarshal config file %s: %s",path,err.Error())
	}
	auth.InitConfig(CurrentConfig.CasdoorEndpoint, 
		CurrentConfig.CasdoorClientId,
		CurrentConfig.CasdoorClientSecret,
		CasdoorJwtSecret, 
		CurrentConfig.CasdoorOrganization, 
		CurrentConfig.CasdoorApplication)
}
