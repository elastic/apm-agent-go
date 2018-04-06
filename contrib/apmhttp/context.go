package apmhttp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/elastic/apm-agent-go/internal/apmhttputil"
	"github.com/elastic/apm-agent-go/model"
)

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
//
// TODO(axw) move this out of apmhttp, into the elasticapm package, or a
// sub-package specifically for context creation.
func RequestContext(req *http.Request) *model.Context {
	username, _, ok := req.BasicAuth()
	if !ok && req.URL.User != nil {
		username = req.URL.User.Username()
	}

	// Special cases to avoid calling into fmt.Sprintf in most cases.
	var httpVersion string
	switch {
	case req.ProtoMajor == 1 && req.ProtoMinor == 1:
		httpVersion = "1.1"
	case req.ProtoMajor == 2 && req.ProtoMinor == 0:
		httpVersion = "2.0"
	default:
		httpVersion = fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor)
	}

	var forwarded *apmhttputil.ForwardedHeader
	if fwd := req.Header.Get("Forwarded"); fwd != "" {
		parsed := apmhttputil.ParseForwarded(fwd)
		forwarded = &parsed
	}

	ctx := model.Context{
		Request: &model.Request{
			URL:         apmhttputil.RequestURL(req, forwarded),
			Method:      req.Method,
			Headers:     RequestHeaders(req),
			HTTPVersion: httpVersion,
			Cookies:     req.Cookies(),
			Socket: &model.RequestSocket{
				Encrypted:     req.TLS != nil,
				RemoteAddress: apmhttputil.RemoteAddr(req, forwarded),
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

// RequestHeaders returns the headers for the HTTP request relevant to tracing.
func RequestHeaders(req *http.Request) *model.RequestHeaders {
	return &model.RequestHeaders{
		ContentType: req.Header.Get("Content-Type"),
		Cookie:      strings.Join(req.Header["Cookie"], ";"),
		UserAgent:   req.UserAgent(),
	}
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

// StatusCodeString returns the stringified status code. Prefer this to
// strconv.Itoa to avoid allocating memory for well known status codes.
func StatusCodeString(statusCode int) string {
	switch statusCode {
	case http.StatusContinue:
		return "100"
	case http.StatusSwitchingProtocols:
		return "101"
	case http.StatusProcessing:
		return "102"

	case http.StatusOK:
		return "200"
	case http.StatusCreated:
		return "201"
	case http.StatusAccepted:
		return "202"
	case http.StatusNonAuthoritativeInfo:
		return "203"
	case http.StatusNoContent:
		return "204"
	case http.StatusResetContent:
		return "205"
	case http.StatusPartialContent:
		return "206"
	case http.StatusMultiStatus:
		return "207"
	case http.StatusAlreadyReported:
		return "208"
	case http.StatusIMUsed:
		return "226"

	case http.StatusMultipleChoices:
		return "300"
	case http.StatusMovedPermanently:
		return "301"
	case http.StatusFound:
		return "302"
	case http.StatusSeeOther:
		return "303"
	case http.StatusNotModified:
		return "304"
	case http.StatusUseProxy:
		return "305"

	case http.StatusTemporaryRedirect:
		return "307"
	case http.StatusPermanentRedirect:
		return "308"

	case http.StatusBadRequest:
		return "400"
	case http.StatusUnauthorized:
		return "401"
	case http.StatusPaymentRequired:
		return "402"
	case http.StatusForbidden:
		return "403"
	case http.StatusNotFound:
		return "404"
	case http.StatusMethodNotAllowed:
		return "405"
	case http.StatusNotAcceptable:
		return "406"
	case http.StatusProxyAuthRequired:
		return "407"
	case http.StatusRequestTimeout:
		return "408"
	case http.StatusConflict:
		return "409"
	case http.StatusGone:
		return "410"
	case http.StatusLengthRequired:
		return "411"
	case http.StatusPreconditionFailed:
		return "412"
	case http.StatusRequestEntityTooLarge:
		return "413"
	case http.StatusRequestURITooLong:
		return "414"
	case http.StatusUnsupportedMediaType:
		return "415"
	case http.StatusRequestedRangeNotSatisfiable:
		return "416"
	case http.StatusExpectationFailed:
		return "417"
	case http.StatusTeapot:
		return "418"
	case http.StatusUnprocessableEntity:
		return "422"
	case http.StatusLocked:
		return "423"
	case http.StatusFailedDependency:
		return "424"
	case http.StatusUpgradeRequired:
		return "426"
	case http.StatusPreconditionRequired:
		return "428"
	case http.StatusTooManyRequests:
		return "429"
	case http.StatusRequestHeaderFieldsTooLarge:
		return "431"
	case http.StatusUnavailableForLegalReasons:
		return "451"

	case http.StatusInternalServerError:
		return "500"
	case http.StatusNotImplemented:
		return "501"
	case http.StatusBadGateway:
		return "502"
	case http.StatusServiceUnavailable:
		return "503"
	case http.StatusGatewayTimeout:
		return "504"
	case http.StatusHTTPVersionNotSupported:
		return "505"
	case http.StatusVariantAlsoNegotiates:
		return "506"
	case http.StatusInsufficientStorage:
		return "507"
	case http.StatusLoopDetected:
		return "508"
	case http.StatusNotExtended:
		return "510"
	case http.StatusNetworkAuthenticationRequired:
		return "511"
	}
	return strconv.Itoa(statusCode)
}
