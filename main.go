// Copyright 2026 The Casdoor Authors. All Rights Reserved.
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

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/casdoor/casdoor-forward-auth/internal/config"
	"github.com/casdoor/casdoor-forward-auth/internal/handler"
)

func main() {
	configFile := flag.String("config", os.Getenv("CONFIG_FILE"), "path to the JSON config file, can be omitted when everything is set by environment variables")
	flag.Parse()

	conf, err := config.Load(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              conf.ListenAddr,
		Handler:           handler.New(conf),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("casdoor-forward-auth is listening on %s, external URL: %s", conf.ListenAddr, conf.ExternalUrl)
	log.Fatal(server.ListenAndServe())
}
