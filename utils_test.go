package apm

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentProcess(t *testing.T) {
	process := getCurrentProcess()
	expected := filepath.Base(os.Args[0])

	// On Linux, the process title can be at most
	// 16 bytes, including the null terminator.
	if runtime.GOOS == "linux" && len(expected) >= 16 {
		expected = expected[:15]
	}

	assert.Equal(t, expected, process.Title)
}

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

func TestJitterDuration(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	assert.Equal(t, time.Duration(0), jitterDuration(0, rng, 0.1))
	assert.Equal(t, time.Second, jitterDuration(time.Second, rng, 0))
	for i := 0; i < 100; i++ {
		assert.InDelta(t, time.Second, jitterDuration(time.Second, rng, 0.1), float64(100*time.Millisecond))
	}
}
