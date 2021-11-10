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

import "github.com/casdoor/casdoor-go-sdk/auth"

var CasdoorEndpoint = "http://webhook.domain.local:8000"
var CasfoorClientId = "88b2457a123984b48392"
var CasdoorClientSecret = "1a3f5eb7990b92f135a78fab5d0327890f2ae8df"
var CasdoorJwtSecret = "CasdoorSecret"
var CasdoorOrganization = "Traefik ForwardAuth"
var CasdoorApplication = "TraefikForwardAuthPlugin"
var PluginDomain = "webhook.domain.local:9999"
var PluginCallback = "http://webhook.domain.local:9999/callback"

func init() {
	auth.InitConfig(CasdoorEndpoint, CasfoorClientId, CasdoorClientSecret, CasdoorJwtSecret, CasdoorOrganization, CasdoorApplication)
}
