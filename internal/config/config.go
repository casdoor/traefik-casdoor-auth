package config
import "github.com/casdoor/casdoor-go-sdk/auth"

var CasdoorEndpoint = "http://webhook.domain.local:8000"
var CasfoorClientId = "88b2457a123984b48392"
var CasdoorClientSecret = "1a3f5eb7990b92f135a78fab5d0327890f2ae8df"
var CasdoorJwtSecret = "CasdoorSecret"
var CasdoorOrganization = "Traefik ForwardAuth"
var CasdoorApplication="TraefikForwardAuthPlugin"
var PluginDomain="webhook.domain.local:9999"
var PluginCallback="http://webhook.domain.local:9999/callback"
func init() {
    auth.InitConfig(CasdoorEndpoint, CasfoorClientId, CasdoorClientSecret, CasdoorJwtSecret, CasdoorOrganization,CasdoorApplication)
}