package apmrestful_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmrestful"
	"go.elastic.co/apm/transport/transporttest"
)

func TestHandlerHTTPSuite(t *testing.T) {
	tracer, recorder := transporttest.NewRecorderTracer()
	var ws restful.WebService
	ws.Path("/").Consumes(restful.MIME_JSON, restful.MIME_XML).Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/implicit_write").To(func(req *restful.Request, resp *restful.Response) {}))
	ws.Route(ws.GET("/panic_before_write").To(func(req *restful.Request, resp *restful.Response) {
		panic("boom")
	}))
	ws.Route(ws.GET("/panic_after_write").To(func(req *restful.Request, resp *restful.Response) {
		resp.Write([]byte("hello, world"))
		panic("boom")
	}))
	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestful.Filter(apmrestful.WithTracer(tracer)))

	suite.Run(t, &apmtest.HTTPTestSuite{
		Handler:  container,
		Tracer:   tracer,
		Recorder: recorder,
	})
}

func TestContainerFilter(t *testing.T) {
	type Thing struct {
		ID string
	}

	var ws restful.WebService
	ws.Path("/things").Consumes(restful.MIME_JSON, restful.MIME_XML).Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/{id:[0-1]+}").To(func(req *restful.Request, resp *restful.Response) {
		if apm.TransactionFromContext(req.Request.Context()) == nil {
			panic("no transaction in context")
		}
		resp.WriteHeaderAndEntity(http.StatusTeapot, Thing{
			ID: req.PathParameter("id"),
		})
	}))

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestful.Filter(apmrestful.WithTracer(tracer)))

	server := httptest.NewServer(container)
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	serverHost, serverPort, err := net.SplitHostPort(serverURL.Host)
	require.NoError(t, err)

	resp, err := http.Get(server.URL + "/things/123")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	assert.Len(t, payloads.Transactions, 1)
	transaction := payloads.Transactions[0]

	assert.Equal(t, "GET /things/{id}", transaction.Name)
	assert.Equal(t, "request", transaction.Type)
	assert.Equal(t, "HTTP 4xx", transaction.Result)

	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "go-restful",
				Version: "unspecified",
			},
		},
		Request: &model.Request{
			Socket: &model.RequestSocket{
				RemoteAddress: "127.0.0.1",
			},
			URL: model.URL{
				Full:     server.URL + "/things/123",
				Protocol: "http",
				Hostname: serverHost,
				Port:     serverPort,
				Path:     "/things/123",
			},
			Method:      "GET",
			HTTPVersion: "1.1",
			Headers: model.Headers{{
				Key:    "Accept-Encoding",
				Values: []string{"gzip"},
			}, {
				Key:    "User-Agent",
				Values: []string{"Go-http-client/1.1"},
			}},
		},
		Response: &model.Response{
			StatusCode: 418,
			Headers: model.Headers{{
				Key:    "Content-Type",
				Values: []string{"application/json"},
			}},
		},
	}, transaction.Context)
}

func TestContainerFilterPanic(t *testing.T) {
	var ws restful.WebService
	ws.Path("/things").Consumes(restful.MIME_JSON, restful.MIME_XML).Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.Route(ws.GET("/{id}/foo").To(handlePanic))

	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()

	container := restful.NewContainer()
	container.Add(&ws)
	container.Filter(apmrestful.Filter(apmrestful.WithTracer(tracer)))

	server := httptest.NewServer(container)
	defer server.Close()
	resp, err := http.Get(server.URL + "/things/123/foo")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	require.Len(t, payloads.Transactions, 1)
	require.Len(t, payloads.Errors, 1)
	panicError := payloads.Errors[0]
	assert.Equal(t, payloads.Transactions[0].Context.Service, panicError.Context.Service)
	assert.Equal(t, payloads.Transactions[0].ID, panicError.ParentID)
	assert.Equal(t, "kablamo", panicError.Exception.Message)
	assert.Equal(t, "handlePanic", panicError.Culprit)
}

func handlePanic(req *restful.Request, resp *restful.Response) {
	panic("kablamo")
}
