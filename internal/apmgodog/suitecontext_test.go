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

package apmgodog_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/apm/module/apmhttp"
)

type featureContext struct {
	apiKey         string
	secretToken    string
	serviceName    string
	serviceVersion string
	env            []string // for subprocesses

	httpServer         *httptest.Server
	httpHandler        *httpHandler
	httpRequestHeaders http.Header

	grpcServer  *grpc.Server
	grpcClient  *grpc.ClientConn
	grpcService *helloworldGRPCService

	tracer      *apmtest.RecordingTracer
	span        *apm.Span
	transaction *apm.Transaction

	cloud *model.Cloud
}

func newFeatureContext() *featureContext {
	return &featureContext{
		tracer:      apmtest.NewRecordingTracer(),
		httpHandler: &httpHandler{},
		grpcService: &helloworldGRPCService{},
	}
}

// initTestSuite initialises s with before/after suite hooks.
func (c *featureContext) initTestSuite(s *godog.TestSuiteContext) {
	s.BeforeSuite(func() {
		c.httpServer = httptest.NewServer(
			apmhttp.Wrap(c.httpHandler, apmhttp.WithTracer(c.tracer.Tracer)),
		)

		c.grpcServer = grpc.NewServer(grpc.UnaryInterceptor(
			apmgrpc.NewUnaryServerInterceptor(
				apmgrpc.WithTracer(c.tracer.Tracer),
				apmgrpc.WithRecovery(),
			),
		))
		pb.RegisterGreeterServer(c.grpcServer, c.grpcService)
		grpcListener := bufconn.Listen(1)
		grpcClient, err := grpc.Dial("bufconn",
			grpc.WithInsecure(),
			grpc.WithDialer(func(string, time.Duration) (net.Conn, error) { return grpcListener.Dial() }),
			grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
		)
		if err != nil {
			panic(err)
		}
		c.grpcClient = grpcClient
		go c.grpcServer.Serve(grpcListener)
	})
	s.AfterSuite(func() {
		c.tracer.Close()
		c.grpcServer.Stop()
		c.grpcClient.Close()
		c.httpServer.Close()
	})
}

// initScenario initialises s with step definitions, and before/after scenario hooks.
func (c *featureContext) initScenario(s *godog.ScenarioContext) {
	s.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		c.env = nil
		c.cloud = nil
		c.apiKey = ""
		c.secretToken = ""
		c.span = nil
		c.transaction = nil
		c.httpHandler.panic = false
		c.httpHandler.statusCode = http.StatusOK
		c.httpRequestHeaders = nil
		c.grpcService.panic = false
		c.grpcService.err = nil
		c.tracer.ResetPayloads()
		for _, k := range os.Environ() {
			if strings.HasPrefix(k, "ELASTIC_APM") {
				os.Unsetenv(k)
			}
		}
		return ctx, nil
	})

	s.Step("^an agent$", c.anAgent)
	s.Step("^an agent configured with$", c.anAgentConfiguredWith)
	s.Step("^the agent sends a request to APM server$", c.sendRequest)
	s.Step("^the Authorization header of the request is '(.*)'$", c.checkAuthorizationHeader)
	s.Step("^the User-Agent header of the request matches regex '(.*)'$", c.checkUserAgentHeader)

	s.Step("^an active span$", c.anActiveSpan)
	s.Step("^an active transaction$", c.anActiveTransaction)

	// Outcome
	s.Step("^a user sets the span outcome to '(.*)'$", c.userSetsSpanOutcome)
	s.Step("^a user sets the transaction outcome to '(.*)'$", c.userSetsTransactionOutcome)
	s.Step("^the agent sets the span outcome to '(.*)'$", c.agentSetsSpanOutcome)
	s.Step("^the agent sets the transaction outcome to '(.*)'$", c.agentSetsTransactionOutcome)
	s.Step("^an error is reported to the span$", c.anErrorIsReportedToTheSpan)
	s.Step("^an error is reported to the transaction$", c.anErrorIsReportedToTheTransaction)
	s.Step("^the span ends$", c.spanEnds)
	s.Step("^the transaction ends$", c.transactionEnds)
	s.Step("^the span outcome is '(.*)'$", c.spanOutcomeIs)
	s.Step("^the transaction outcome is '(.*)'$", c.transactionOutcomeIs)

	// HTTP
	s.Step("^a HTTP call is received that returns (.*)$", c.anHTTPTransactionWithStatusCode)
	s.Step("^a HTTP call is made that returns (.*)$", c.anHTTPSpanWithStatusCode)

	// gRPC
	s.Step("^a gRPC call is received that returns '(.*)'$", c.aGRPCTransactionWithStatusCode)
	s.Step("^a gRPC call is made that returns '(.*)'$", c.aGRPCSpanWithStatusCode)

	// Cloud metadata
	s.Step("an instrumented application is configured to collect cloud provider metadata for azure", func() error {
		return nil
	})
	s.Step("the following environment variables are present", func(kv *godog.Table) error {
		for _, row := range kv.Rows[1:] {
			c.env = append(c.env, row.Cells[0].Value+"="+row.Cells[1].Value)
		}
		return nil
	})
	s.Step("cloud metadata is collected", func() error {
		_, _, _, cloud, _, err := getSubprocessMetadata(append([]string{
			"ELASTIC_APM_CLOUD_PROVIDER=auto", // Explicitly enable cloud metadata detection
			"http_proxy=http://proxy.invalid", // fail all HTTP requests
		}, c.env...)...)
		if err != nil {
			return err
		}
		if *cloud != (model.Cloud{}) {
			c.cloud = cloud
		}
		return nil
	})
	s.Step("cloud metadata is not null", func() error {
		if c.cloud == nil {
			return errors.New("cloud metadata is empty")
		}
		return nil
	})
	s.Step("cloud metadata is null", func() error {
		if c.cloud != nil {
			return fmt.Errorf("cloud metadata is non-empty: %+v", *c.cloud)
		}
		return nil
	})
	s.Step("cloud metadata '(.*)' is '(.*)'", func(field, expected string) error {
		var actual string
		switch field {
		case "account.id":
			if c.cloud.Account == nil {
				return errors.New("cloud.account is nil")
			}
			actual = c.cloud.Account.ID
		case "provider":
			actual = c.cloud.Provider
		case "instance.id":
			if c.cloud.Instance == nil {
				return errors.New("cloud.instance is nil")
			}
			actual = c.cloud.Instance.ID
		case "instance.name":
			if c.cloud.Instance == nil {
				return errors.New("cloud.instance is nil")
			}
			actual = c.cloud.Instance.Name
		case "project.name":
			if c.cloud.Project == nil {
				return errors.New("cloud.project is nil")
			}
			actual = c.cloud.Project.Name
		case "region":
			actual = c.cloud.Region
		default:
			return fmt.Errorf("unexpected field %q", field)
		}
		if actual != expected {
			return fmt.Errorf("expected %q to be %q, got %q", field, expected, actual)
		}
		return nil
	})
}

func (c *featureContext) anAgent() error {
	// No-op; we create the tracer in the suite setup.
	return nil
}

func (c *featureContext) anAgentConfiguredWith(settings *godog.Table) error {
	for _, row := range settings.Rows[1:] {
		setting := row.Cells[0].Value
		value := row.Cells[1].Value
		switch setting {
		case "api_key":
			c.apiKey = value
		case "cloud_provider":
			c.env = append(c.env, "ELASTIC_APM_CLOUD_PROVIDER="+value)
		case "secret_token":
			c.secretToken = value
		case "service_name":
			c.serviceName = value
		case "service_version":
			c.serviceVersion = value
		default:
			return fmt.Errorf("unhandled setting %q", setting)
		}
	}
	return nil
}

func (c *featureContext) sendRequest() error {
	var h http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		c.httpRequestHeaders = r.Header
	}
	server := httptest.NewServer(h)
	defer server.Close()

	os.Setenv("ELASTIC_APM_SECRET_TOKEN", c.secretToken)
	os.Setenv("ELASTIC_APM_API_KEY", c.apiKey)
	os.Setenv("ELASTIC_APM_SERVER_URL", server.URL)

	tracer, err := apm.NewTracer(c.serviceName, c.serviceVersion)
	if err != nil {
		// The User-Agent tests set a service name with invalid characters, and then
		// checks that the resulting User-Agent is valid. We prevent invalid service
		// names from being set, so mimic other agents' behaviour by not using
		// serviceName and serviceVersion if NewTracer returns an error.
		tracer, err = apm.NewTracer("godog", "")
	}
	if err != nil {
		return err
	}

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)
	return nil
}

func (c *featureContext) checkAuthorizationHeader(expected string) error {
	authHeader := c.httpRequestHeaders["Authorization"]
	if n := len(authHeader); n != 1 {
		return fmt.Errorf("got %d Authorization headers, expected 1", n)
	}
	return nil
}

func (c *featureContext) checkUserAgentHeader(expectedRegex string) error {
	re, err := regexp.Compile(expectedRegex)
	if err != nil {
		return err
	}
	userAgentHeader := c.httpRequestHeaders["User-Agent"]
	if n := len(userAgentHeader); n != 1 {
		return fmt.Errorf("got %d User-Agent headers, expected 1", n)
	}
	if !re.MatchString(userAgentHeader[0]) {
		return fmt.Errorf(
			"User-Agent header %q does not match regex %q",
			userAgentHeader[0], expectedRegex,
		)
	}
	return nil
}

func (c *featureContext) anActiveSpan() error {
	if err := c.anActiveTransaction(); err != nil {
		return err
	}
	c.span = c.transaction.StartSpan("name", "type", nil)
	return nil
}

func (c *featureContext) anActiveTransaction() error {
	c.transaction = c.tracer.StartTransaction("name", "type")
	return nil
}

func (c *featureContext) userSetsSpanOutcome(outcome string) error {
	c.span.Outcome = outcome
	return nil
}

func (c *featureContext) userSetsTransactionOutcome(outcome string) error {
	c.transaction.Outcome = outcome
	return nil
}

func (c *featureContext) agentSetsSpanOutcome(outcome string) error {
	switch outcome {
	case "unknown":
	case "success":
		c.span.Context.SetHTTPStatusCode(200)
	case "failure":
		c.span.Context.SetHTTPStatusCode(400)
	}
	return nil
}

func (c *featureContext) agentSetsTransactionOutcome(outcome string) error {
	switch outcome {
	case "unknown":
	case "success":
		c.transaction.Context.SetHTTPStatusCode(200)
	case "failure":
		c.transaction.Context.SetHTTPStatusCode(500)
	}
	return nil
}

func (c *featureContext) anErrorIsReportedToTheSpan() error {
	e := c.tracer.NewError(errors.New("an error"))
	e.SetSpan(c.span)
	return nil
}

func (c *featureContext) anErrorIsReportedToTheTransaction() error {
	e := c.tracer.NewError(errors.New("an error"))
	e.SetTransaction(c.transaction)
	return nil
}

func (c *featureContext) spanEnds() error {
	c.span.End()
	c.tracer.Flush(nil)
	return nil
}

func (c *featureContext) transactionEnds() error {
	c.transaction.End()
	c.tracer.Flush(nil)
	return nil
}

func (c *featureContext) spanOutcomeIs(expected string) error {
	c.tracer.Flush(nil)
	payloads := c.tracer.Payloads()
	if actual := payloads.Spans[0].Outcome; actual != expected {
		return fmt.Errorf("expected outcome %q, got %q", expected, actual)
	}
	return nil
}

func (c *featureContext) transactionOutcomeIs(expected string) error {
	c.tracer.Flush(nil)
	payloads := c.tracer.Payloads()
	if actual := payloads.Transactions[0].Outcome; actual != expected {
		return fmt.Errorf("expected outcome %q, got %q", expected, actual)
	}
	return nil
}

func (c *featureContext) anHTTPTransactionWithStatusCode(statusCode int) error {
	if statusCode < 0 {
		c.httpHandler.panic = true
	} else {
		c.httpHandler.statusCode = statusCode
	}
	return c.sendHTTPRequest(context.Background())
}

func (c *featureContext) anHTTPSpanWithStatusCode(statusCode int) error {
	if statusCode < 0 {
		c.httpHandler.panic = true
	}
	tx := c.tracer.StartTransaction("name", "type")
	defer tx.End()
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	return c.sendHTTPRequest(ctx)
}

func (c *featureContext) sendHTTPRequest(ctx context.Context) error {
	client := apmhttp.WrapClient(c.httpServer.Client())
	req, _ := http.NewRequest("GET", c.httpServer.URL, nil)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *featureContext) aGRPCTransactionWithStatusCode(statusCode string) error {
	if statusCode == "n/a" {
		c.grpcService.panic = true
	} else {
		code, err := parseGRPCStatusCode(statusCode)
		if err != nil {
			return err
		}
		if code != codes.OK {
			c.grpcService.err = status.Error(code, code.String())
		}
	}
	pb.NewGreeterClient(c.grpcClient).SayHello(context.Background(), &pb.HelloRequest{Name: "world"})
	return nil
}

func (c *featureContext) aGRPCSpanWithStatusCode(statusCode string) error {
	if statusCode == "n/a" {
		c.grpcService.panic = true
	} else {
		code, err := parseGRPCStatusCode(statusCode)
		if err != nil {
			return err
		}
		if code != codes.OK {
			c.grpcService.err = status.Error(code, code.String())
		}
	}
	tx := c.tracer.StartTransaction("name", "type")
	defer tx.End()
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	pb.NewGreeterClient(c.grpcClient).SayHello(ctx, &pb.HelloRequest{Name: "world"})
	return nil
}

func parseGRPCStatusCode(s string) (codes.Code, error) {
	var code codes.Code
	err := code.UnmarshalJSON([]byte(strconv.Quote(s)))
	return code, err
}

type helloworldGRPCService struct {
	panic bool
	err   error
}

func (h *helloworldGRPCService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if h.panic {
		panic("boom")
	}
	return &pb.HelloReply{Message: "hello, " + req.Name}, h.err
}

type httpHandler struct {
	panic      bool
	statusCode int
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.panic {
		panic("boom")
	}
	statusCode := h.statusCode
	if statusCode == 100 {
		// Cheat to avoid complexity of dealing with 100 Continue responses.
		statusCode = 101
	}
	w.WriteHeader(statusCode)
}
