package apmdebug

import (
	"log"
	"os"
	"strings"
)

var (
	// DebugLog reports whether debug logging is enabled.
	DebugLog bool
)

func init() {
	v := os.Getenv("ELASTIC_APM_DEBUG")
	if v == "" {
		return
	}
	for _, field := range strings.Split(v, ",") {
		pos := strings.IndexRune(field, '=')
		if pos == -1 {
			invalidField(field)
			continue
		}
		k, _ := field[:pos], field[pos+1:]
		switch k {
		case "log":
			DebugLog = true
		default:
			unknownKey(k)
			continue
		}
	}
}

func unknownKey(key string) {
	log.Println("unknown ELASTIC_APM_DEBUG field:", key)
}

func invalidField(field string) {
	log.Println("invalid ELASTIC_APM_DEBUG field:", field)
}

// Logger provides methods for logging.
type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// LogLogger is a Logger that uses the standard "log" package.
type LogLogger struct{}

// Debugf logs a message with log.Printf, with a DEBUG prefix.
func (l LogLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

// Errorf logs a message with log.Printf, with an ERROR prefix.
func (l LogLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// ChainedLogger is a chain of Loggers, which dispatches method calls
// to each element in the chain in sequence.
type ChainedLogger []Logger

// Debugf calls Debugf(format, args..) for each logger in the chain.
func (l ChainedLogger) Debugf(format string, args ...interface{}) {
	for _, l := range l {
		l.Debugf(format, args...)
	}
}

// Errorf calls Errorf(format, args..) for each logger in the chain.
func (l ChainedLogger) Errorf(format string, args ...interface{}) {
	for _, l := range l {
		l.Errorf(format, args...)
	}
}
