#curl -H Host:webhook.domain.local 127.0.0.1:8888
go run cmd/webhook/main.go  -configFile="conf/plugin.json"