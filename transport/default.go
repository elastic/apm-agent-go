package transport

import (
	"os"
)

var (
	// Default is the default Transport, using the
	// ELASTIC_APM_* environment variables.
	//
	// If ELASTIC_APM_SERVER_URL is not defined, then
	// Defaultwill be set to Discard. If it is defined,
	// but invalid, then Default will be set to a transport
	// returning an error for every operation.
	Default Transport

	// Discard is a Transport on which all operations
	// succeed without doing anything.
	Discard = discardTransport{}
)

func init() {
	_, _ = InitDefault()
}

// InitDefault (re-)initializes Default, the default transport, returning
// its new value along with the error that will be returned by the transport
// if the environment variable configuration is invalid. The Transport returned
// is always non-nil.
func InitDefault() (Transport, error) {
	url := os.Getenv(envServerURL)
	if url == "" {
		Default = Discard
		return Default, nil
	}
	t, err := NewHTTPTransport(url, "")
	if err != nil {
		Default = discardTransport{err}
		return Default, err
	}
	Default = t
	return Default, nil
}
