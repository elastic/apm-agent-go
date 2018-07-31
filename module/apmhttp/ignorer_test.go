package apmhttp_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/module/apmhttp"
)

func TestDefaultServerRequestIgnorer(t *testing.T) {
	r1 := &http.Request{URL: &url.URL{Path: "/foo"}}
	r2 := &http.Request{URL: &url.URL{Path: "/foo", RawQuery: "bar=baz"}}
	r3 := &http.Request{URL: &url.URL{Scheme: "http", Host: "testing.invalid", Path: "/foo", RawQuery: "bar=baz"}}

	testDefaultServerRequestIgnorer(t, "", r1, false)
	testDefaultServerRequestIgnorer(t, "", r2, false)
	testDefaultServerRequestIgnorer(t, "", r3, false)
	testDefaultServerRequestIgnorer(t, "[", r1, false) // invalid regexp matches nothing

	testDefaultServerRequestIgnorer(t, "/foo", r1, true)
	testDefaultServerRequestIgnorer(t, "/foo", r2, true)
	testDefaultServerRequestIgnorer(t, "/foo", r3, true)
	testDefaultServerRequestIgnorer(t, "/FOO", r3, true) // case insensitive by default

	testDefaultServerRequestIgnorer(t, "/foo\\?bar=baz", r1, false)
	testDefaultServerRequestIgnorer(t, "/foo\\?bar=baz", r2, true)
	testDefaultServerRequestIgnorer(t, "/foo\\?bar=baz", r3, true)

	testDefaultServerRequestIgnorer(t, "http://.*", r1, false)
	testDefaultServerRequestIgnorer(t, "http://.*", r2, false)
	testDefaultServerRequestIgnorer(t, "http://.*", r3, true)
}

func testDefaultServerRequestIgnorer(t *testing.T, ignoreURLs string, r *http.Request, expect bool) {
	testName := fmt.Sprintf("%s_%s", ignoreURLs, r.URL.String())
	t.Run(testName, func(t *testing.T) {
		if os.Getenv("_INSIDE_TEST") != "1" {
			cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$")
			cmd.Env = append(os.Environ(), "_INSIDE_TEST=1")
			cmd.Env = append(cmd.Env, "ELASTIC_APM_IGNORE_URLS="+ignoreURLs)
			assert.NoError(t, cmd.Run())
			return
		}
		ignorer := apmhttp.DefaultServerRequestIgnorer()
		assert.Equal(t, expect, ignorer(r))
	})
}
