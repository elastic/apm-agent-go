package apmtest

// TestLogger is an implementation of elasticapm.Logger,
// logging to a testing.T.
type TestLogger struct {
	l LogfLogger
}

// NewTestLogger returns a new TestLogger that logs messages to l.
func NewTestLogger(l LogfLogger) TestLogger {
	return TestLogger{l: l}
}

// Debugf logs debug messages.
func (t TestLogger) Debugf(format string, args ...interface{}) {
	t.l.Logf("[DEBUG] "+format, args...)
}

// Errorf logs error messages.
func (t TestLogger) Errorf(format string, args ...interface{}) {
	t.l.Logf("[ERROR] "+format, args...)
}

// LogfLogger is an interface with the a Logf method,
// implemented by *testing.T and *testing.B.
type LogfLogger interface {
	Logf(string, ...interface{})
}
