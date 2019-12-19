// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

// This program is a test service for the W3C Distributed Tracing test harness:
//     https://github.com/w3c/trace-context/tree/master/test

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"go.elastic.co/apm/module/apmhttp"
)

var (
	listenAddr = flag.String("listen", ":5000", "Address to listen on for test requests")
)

func main() {
	flag.Parse()

	type TestCase struct {
		URL  string            `json:"url"`
		Args []json.RawMessage `json:"arguments,omitempty"`
	}

	client := http.DefaultClient
	client = apmhttp.WrapClient(client)

	var handler http.HandlerFunc = func(w http.ResponseWriter, req *http.Request) {
		var testCases []TestCase
		if err := json.NewDecoder(req.Body).Decode(&testCases); err != nil {
			log.Printf("decoding error: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, tc := range testCases {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(tc.Args); err != nil {
				log.Printf("encoding error: %s", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			outReq, err := http.NewRequest("POST", tc.URL, &buf)
			if err != nil {
				log.Printf("error creating request: %s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				continue
			}
			outReq = outReq.WithContext(req.Context())
			outReq.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(outReq)
			if err != nil {
				log.Printf("error sending request: %s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Printf("status: %s", resp.Status)
			}
		}
	}
	log.Printf("Starting Trace-Context test service, listening on %s", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, apmhttp.Wrap(handler)))
}
