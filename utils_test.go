package elasticapm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGracePeriod(t *testing.T) {
	var p time.Duration = -1
	var seq []time.Duration
	for i := 0; i < 1000; i++ {
		next := nextGracePeriod(p)
		if next == p {
			assert.Equal(t, []time.Duration{
				0,
				time.Second,
				4 * time.Second,
				9 * time.Second,
				16 * time.Second,
				25 * time.Second,
				36 * time.Second,
			}, seq)
			return
		}
		p = next
		seq = append(seq, p)
	}
	t.Fatal("failed to find fixpoint")
}
