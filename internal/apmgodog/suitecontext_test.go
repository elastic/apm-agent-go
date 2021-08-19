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

//go:build go1.9
// +build go1.9

package apmgodog_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"
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
	"go.elastic.co/apm/transport"
)

type featureContext struct {
	apiKey      string
	secretToken string
	env         []string // for subprocesses

	httpServer  *httptest.Server
	httpHandler *httpHandler

	grpcServer  *grpc.Server
	grpcClient  *grpc.ClientConn
	grpcService *helloworldGRPCService

	tracer      *apmtest.RecordingTracer
	span        *apm.Span
	transaction *apm.Transaction

	cloud *model.Cloud
}

// InitContext initialises a godoc.Suite with step definitions.
func InitContext(s *godog.Suite) {
	c := &featureContext{
		tracer: apmtest.NewRecordingTracer(),
	}

	c.httpHandler = &httpHandler{}
	c.httpServer = httptest.NewServer(
		apmhttp.Wrap(c.httpHandler, apmhttp.WithTracer(c.tracer.Tracer)),
	)

	c.grpcServer = grpc.NewServer(grpc.UnaryInterceptor(
		apmgrpc.NewUnaryServerInterceptor(
			apmgrpc.WithTracer(c.tracer.Tracer),
			apmgrpc.WithRecovery(),
		),
	))
	c.grpcService = &helloworldGRPCService{}
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

	s.BeforeScenario(func(interface{}) { c.reset() })
	s.AfterSuite(func() {
		c.tracer.Close()
		c.grpcServer.Stop()
		c.grpcClient.Close()
		c.httpServer.Close()
	})
	s.AfterScenario(func(interface{}, error) {
		c.env = nil
		c.cloud = nil
	})

	s.Step("^an agent$", c.anAgent)
	s.Step("^an api key is not set in the config$", func() error { return nil })
	s.Step("^an api key is set to '(.*)' in the config$", c.setAPIKey)
	s.Step("^a secret_token is set to '(.*)' in the config$", c.setSecretToken)
	s.Step("^the Authorization header is '(.*)'$", c.checkAuthorizationHeader)

	s.Step("^an active span$", c.anActiveSpan)
	s.Step("^an active transaction$", c.anActiveTransaction)

	// Outcome
	s.Step("^user sets span outcome to '(.*)'$", c.userSetsSpanOutcome)
	s.Step("^user sets transaction outcome to '(.*)'$", c.userSetsTransactionOutcome)
	s.Step("^span terminates with outcome '(.*)'$", c.spanTerminatesWithOutcome)
	s.Step("^transaction terminates with outcome '(.*)'$", c.transactionTerminatesWithOutcome)
	s.Step("^span terminates with an error$", func() error { return c.spanTerminatesWithOutcome("failure") })
	s.Step("^span terminates without error$", func() error { return c.spanTerminatesWithOutcome("success") })
	s.Step("^transaction terminates with an error$", func() error { return c.transactionTerminatesWithOutcome("failure") })
	s.Step("^transaction terminates without error$", func() error { return c.transactionTerminatesWithOutcome("success") })
	s.Step("^span outcome is '(.*)'$", c.spanOutcomeIs)
	s.Step("^span outcome is \"(.*)\"$", c.spanOutcomeIs)
	s.Step("^transaction outcome is '(.*)'$", c.transactionOutcomeIs)
	s.Step("^transaction outcome is \"(.*)\"$", c.transactionOutcomeIs)

	// HTTP
	s.Step("^an HTTP transaction with (.*) response code$", c.anHTTPTransactionWithStatusCode)
	s.Step("^an HTTP span with (.*) response code$", c.anHTTPSpanWithStatusCode)

	// gRPC
	s.Step("^a gRPC transaction with '(.*)' status$", c.aGRPCTransactionWithStatusCode)
	s.Step("^a gRPC span with '(.*)' status$", c.aGRPCSpanWithStatusCode)

	// Cloud metadata
	s.Step("an instrumented application is configured to collect cloud provider metadata for azure", func() error {
		return nil
	})
	s.Step("the following environment variables are present", func(kv *gherkin.DataTable) error {
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

func (c *featureContext) reset() {
	c.apiKey = ""
	c.secretToken = ""
	c.span = nil
	c.transaction = nil
	c.httpHandler.panic = false
	c.httpHandler.statusCode = http.StatusOK
	c.grpcService.panic = false
	c.grpcService.err = nil
	c.tracer.ResetPayloads()

	for _, k := range os.Environ() {
		if strings.HasPrefix(k, "ELASTIC_APM") {
			os.Unsetenv(k)
		}
	}
}

func (c *featureContext) anAgent() error {
	// No-op; we create the tracer in the suite setup.
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

func (c *featureContext) spanTerminatesWithOutcome(outcome string) error {
	switch outcome {
	case "unknown":
	case "success":
		c.span.Context.SetHTTPStatusCode(200)
	case "failure":
		c.span.Context.SetHTTPStatusCode(400)
	}
	c.span.End()
	return nil
}

func (c *featureContext) transactionTerminatesWithOutcome(outcome string) error {
	switch outcome {
	case "unknown":
	case "success":
		c.transaction.Context.SetHTTPStatusCode(200)
	case "failure":
		c.transaction.Context.SetHTTPStatusCode(500)
	}
	c.transaction.End()
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
