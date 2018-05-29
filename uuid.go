package elasticapm

import (
	"github.com/elastic/apm-agent-go/internal/uuid"
)

// newUUID returns a new hex-encoded UUID, suitable
// for use as a transaction or error ID, and a bool
// indicating whether or not the returned UUID is
// valid.
func newUUID() (string, bool) {
	uuid, ok := <-uuids
	return uuid, ok
}

var uuids = make(chan string, 1024)

func init() {
	go func() {
		defer close(uuids)
		for {
			uuid, err := uuid.NewV4()
			if err != nil {
				return
			}
			uuids <- uuid.String()
		}
	}()
}
