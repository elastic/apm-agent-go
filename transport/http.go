package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/apm-agent-go/internal/apmconfig"
)

const (
	intakePath = "/v2/intake"

	envSecretToken      = "ELASTIC_APM_SECRET_TOKEN"
	envServerURL        = "ELASTIC_APM_SERVER_URL"
	envServerTimeout    = "ELASTIC_APM_SERVER_TIMEOUT"
	envVerifyServerCert = "ELASTIC_APM_VERIFY_SERVER_CERT"
)

var (
	// Take a copy of the http.DefaultTransport pointer,
	// in case another package replaces the value later.
	defaultHTTPTransport = http.DefaultTransport.(*http.Transport)

	defaultServerURL     = "http://localhost:8200"
	defaultServerTimeout = 30 * time.Second
)

// HTTPTransport is an implementation of Transport, sending payloads via
// a net/http client.
type HTTPTransport struct {
	Client    *http.Client
	baseURL   *url.URL
	intakeURL *url.URL
	headers   http.Header
}

// NewHTTPTransport returns a new HTTPTransport, which can be used for
// streaming data to the APM server at the specified URL, with the given
// secret token.
//
// If the URL specified is the empty string, then NewHTTPTransport will use the
// value of the ELASTIC_APM_SERVER_URL environment variable, if defined; if
// the environment variable is also undefined, then the transport will use the
// default URL "http://localhost:8200". The URL must be the base server URL,
// excluding any transactions or errors path. e.g. "http://server.example:8200".
//
// If the secret token specified is the empty string, then NewHTTPTransport
// will use the value of the ELASTIC_APM_SECRET_TOKEN environment variable, if
// defined; if the environment variable is also undefined, then requests will
// not be authenticated.
//
// If ELASTIC_APM_VERIFY_SERVER_CERT is set to "false", then the transport
// will not verify the APM server's TLS certificate.
//
// The Client field will be initialized with a new http.Client configured from
// ELASTIC_APM_* environment variables. The Client field may be modified or
// replaced, e.g. in order to specify TLS root CAs.
func NewHTTPTransport(serverURL, secretToken string) (*HTTPTransport, error) {
	if serverURL == "" {
		serverURL = os.Getenv(envServerURL)
		if serverURL == "" {
			serverURL = defaultServerURL
		}
	}
	req, err := http.NewRequest("POST", serverURL, nil)
	if err != nil {
		return nil, err
	}

	httpTransport := &http.Transport{
		Proxy:                 defaultHTTPTransport.Proxy,
		DialContext:           defaultHTTPTransport.DialContext,
		MaxIdleConns:          defaultHTTPTransport.MaxIdleConns,
		IdleConnTimeout:       defaultHTTPTransport.IdleConnTimeout,
		TLSHandshakeTimeout:   defaultHTTPTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultHTTPTransport.ExpectContinueTimeout,
	}
	if req.URL.Scheme == "https" && os.Getenv(envVerifyServerCert) == "false" {
		httpTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	client := &http.Client{Transport: httpTransport}

	timeout, err := apmconfig.ParseDurationEnv(envServerTimeout, defaultServerTimeout)
	if err != nil {
		return nil, err
	}
	if timeout > 0 {
		client.Timeout = timeout
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/x-ndjson")
	headers.Set("Content-Encoding", "deflate")
	headers.Set("Transfer-Encoding", "chunked")
	if secretToken == "" {
		secretToken = os.Getenv(envSecretToken)
	}
	if secretToken != "" {
		headers.Set("Authorization", "Bearer "+secretToken)
	}

	t := &HTTPTransport{
		Client:    client,
		baseURL:   req.URL,
		intakeURL: urlWithPath(req.URL, intakePath),
		headers:   headers,
	}
	return t, nil
}

// SetUserAgent sets the User-Agent header that will be
// sent with each request.
func (t *HTTPTransport) SetUserAgent(ua string) {
	t.headers.Set("User-Agent", ua)
}

// SendStream sends the stream over HTTP.
func (t *HTTPTransport) SendStream(ctx context.Context, r io.Reader) error {
	req := t.newRequest(t.intakeURL)
	req.Body = ioutil.NopCloser(r)

	resp, err := t.Client.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending stream failed")
	}
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		resp.Body.Close()
		return nil
	}
	defer resp.Body.Close()

	// apm-server will return 503 Service Unavailable
	// if the data cannot be published to Elasticsearch,
	// but there is no Retry-After header included, so
	// we treat it as any other internal server error.
	bodyContents, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		resp.Body = ioutil.NopCloser(bytes.NewReader(bodyContents))
	}
	return &HTTPError{
		Response: resp,
		Message:  strings.TrimSpace(string(bodyContents)),
	}
}

func (t *HTTPTransport) newRequest(url *url.URL) *http.Request {
	req := &http.Request{
		Method:     "POST",
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     t.headers,
		Host:       url.Host,
	}
	return req
}

func urlWithPath(url *url.URL, p string) *url.URL {
	urlCopy := *url
	urlCopy.Path += p
	if urlCopy.RawPath != "" {
		urlCopy.RawPath += p
	}
	return &urlCopy
}

// HTTPError is an error returned by HTTPTransport methods when requests fail.
type HTTPError struct {
	Response *http.Response
	Message  string
}

func (e *HTTPError) Error() string {
	msg := fmt.Sprintf("request failed with %s", e.Response.Status)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	return msg
}

func requestWithContext(ctx context.Context, req *http.Request) *http.Request {
	url := req.URL
	req.URL = nil
	reqCopy := req.WithContext(ctx)
	reqCopy.URL = url
	req.URL = url
	return reqCopy
}
