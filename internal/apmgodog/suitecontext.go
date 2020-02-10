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

package apmgodog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/cucumber/godog"

	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"
)

const (
	aSecretToken = "a_secret_token"
	anAPIKey     = "an_api_key"
)

type featureContext struct {
	apiKey      string
	secretToken string
}

// InitContext initialises a godoc.Suite with step definitions.
func InitContext(s *godog.Suite) {
	c := &featureContext{}
	s.BeforeScenario(func(interface{}) { c.reset() })

	s.Step("^an agent$", c.anAgent)
	s.Step("^an api key is not set in the config$", func() error { return nil })
	s.Step("^an api key is set in the config$", func() error { return c.setAPIKey(anAPIKey) })
	s.Step("^an api key is set to '(.*)' in the config$", c.setAPIKey)
	s.Step("^a secret_token is set in the config$", func() error { return c.setSecretToken(aSecretToken) })
	s.Step("^the Authorization header is '(.*)'$", c.checkAuthorizationHeader)
	s.Step("^the secret token is sent in the Authorization header$", c.secretTokenSentInAuthorizationHeader)
	s.Step("^the api key is sent in the Authorization header$", c.apiKeySentInAuthorizationHeader)
}

func (c *featureContext) reset() {
	c.apiKey = ""
	c.secretToken = ""
	for _, k := range os.Environ() {
		if strings.HasPrefix(k, "ELASTIC_APM") {
			os.Unsetenv(k)
		}
	}
}

func (c *featureContext) anAgent() error {
	// No-op; we create the tracer as needed to test steps.
	return nil
}

func (c *featureContext) setAPIKey(v string) error {
	c.apiKey = v
	return nil
}

func (c *featureContext) setSecretToken(v string) error {
	c.secretToken = v
	return nil
}

func (c *featureContext) secretTokenSentInAuthorizationHeader() error {
	return c.checkAuthorizationHeader("Bearer " + c.secretToken)
}

func (c *featureContext) apiKeySentInAuthorizationHeader() error {
	return c.checkAuthorizationHeader("ApiKey " + c.apiKey)
}

func (c *featureContext) checkAuthorizationHeader(expected string) error {
	var authHeader []string
	var h http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header["Authorization"]
	}
	server := httptest.NewServer(h)
	defer server.Close()

	os.Setenv("ELASTIC_APM_SECRET_TOKEN", c.secretToken)
	os.Setenv("ELASTIC_APM_API_KEY", c.apiKey)
	os.Setenv("ELASTIC_APM_SERVER_URL", server.URL)
	if _, err := transport.InitDefault(); err != nil {
		return err
	}

	tracer, err := apm.NewTracer("godog", "")
	if err != nil {
		return err
	}
	defer tracer.Close()

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)

	if n := len(authHeader); n != 1 {
		return fmt.Errorf("got %d Authorization headers, expected 1", n)
	}
	if authHeader[0] != expected {
		return fmt.Errorf("got Authorization header value %q, expected %q", authHeader, expected)
	}
	return nil
}
