package apmhttp

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/elastic/apm-agent-go/model"
)

// RequestName returns the name to use in model.Transaction.Name
// for HTTP requests.
func RequestName(req *http.Request) string {
	return fmt.Sprintf("%s %s", req.Method, req.URL.Path)
}

// RequestContext returns the context to use in model.Transaction.Context
// for HTTP requests.
//
// Context.Request.Body will be nil. The caller is responsible for setting
// this, taking care to copy and replace the request body as necessary.
//
// Context.Response will be nil. When the request has been handled, this can
// be set using ResponseContext.
//
// Context.User will be initialised with the HTTP Basic Authentication username
// if specified, or else the username in the URL if specified. Otherwise, the
// Context.User field will be nil. In either case, application's may wish to
// override the user values.
func RequestContext(req *http.Request) *model.Context {
	username, _, ok := req.BasicAuth()
	if !ok && req.URL.User != nil {
		username = req.URL.User.Username()
	}

	ctx := model.Context{
		Request: &model.Request{
			URL:         RequestURL(req),
			Method:      req.Method,
			Headers:     RequestHeaders(req),
			HTTPVersion: fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor),
			Cookies:     req.Cookies(),
			Socket: &model.RequestSocket{
				Encrypted:     req.TLS != nil,
				RemoteAddress: RequestRemoteAddress(req),
			},
		},
	}
	if username != "" {
		ctx.User = &model.User{
			Username: username,
		}
	}
	return &ctx
}

// RequestURL returns a model.URL for the given HTTP request.
//
// If the URL contains user info, it will be removed and
// excluded from the URL's "full" field.
func RequestURL(req *http.Request) model.URL {
	fullHost := req.Host
	if fullHost == "" {
		fullHost = req.URL.Host
	}
	host, port, err := net.SplitHostPort(fullHost)
	if err != nil {
		host = fullHost
		port = ""
	}
	out := model.URL{
		Hostname: host,
		Port:     port,
		Path:     req.URL.Path,
		Search:   req.URL.RawQuery,
		Hash:     req.URL.Fragment,
	}
	if req.URL.Scheme != "" {
		// If the URL contains user info, remove it before formatting
		// so it doesn't make its way into the "full" URL, to avoid
		// leaking PII or secrets.
		user := req.URL.User
		req.URL.User = nil
		out.Full = req.URL.String()
		out.Protocol = req.URL.Scheme
		req.URL.User = user
	} else {
		// Server-side, req.URL contains the
		// URI only. We synthesize the URL
		// by adding in req.Host, and the
		// scheme determined by the presence
		// of TLS configuration.
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		u := *req.URL
		u.Scheme = scheme
		u.User = nil
		u.Host = fullHost
		out.Full = u.String()
		out.Protocol = scheme
	}
	return out
}

// RequestHeaders returns the headers for the HTTP request relevant to tracing.
func RequestHeaders(req *http.Request) *model.RequestHeaders {
	return &model.RequestHeaders{
		ContentType: req.Header.Get("Content-Type"),
		Cookie:      strings.Join(req.Header["Cookie"], ";"),
		UserAgent:   req.UserAgent(),
	}
}

// RequestRemoteAddress returns the remote address for the HTTP request.
//
// In order:
//  - if the X-Real-IP header is set, then its value is returned.
//  - if the X-Forwarded-For header is set, then the first value
//    in the comma-separated list is returned.
//  - otherwise, the host portion of req.RemoteAddr is returned.
func RequestRemoteAddress(req *http.Request) string {
	if realIP := req.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		if sep := strings.IndexRune(xff, ','); sep > 0 {
			xff = xff[:sep]
		}
		return strings.TrimSpace(xff)
	}
	remoteAddr, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		remoteAddr = req.RemoteAddr
	}
	return remoteAddr
}

// ResponseHeaders returns the headers for the HTTP response relevant to tracing.
func ResponseHeaders(w http.ResponseWriter) *model.ResponseHeaders {
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		return nil
	}
	return &model.ResponseHeaders{
		ContentType: contentType,
	}
}
