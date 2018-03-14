package apmgin

import (
	"strings"

	"github.com/elastic/apm-agent-go/model"
)

const (
	ginPackagePrefix        = "github.com/gin-gonic/"
	ginContribPackagePrefix = "github.com/gin-contrib/"
)

// Processor is an implementation of elasticapm.Processor for making
// Gin-specific adjustments to model values before they are sent to
// the APM server.
type Processor struct{}

// ProcessError processes the error by marking its exception and log
// stacktrace frames originating within gin packages as being library
// frames.
func (Processor) ProcessError(e *model.Error) {
	if e.Exception != nil {
		setGinLibraryFrames(e.Exception.Stacktrace)
	}
	if e.Log != nil {
		setGinLibraryFrames(e.Log.Stacktrace)
	}
}

// ProcessTransaction processes the transaction by marking its spans'
// stacktrace frames originating within gin packages as being library
// frames.
func (Processor) ProcessTransaction(t *model.Transaction) {
	for _, s := range t.Spans {
		setGinLibraryFrames(s.Stacktrace)
	}
}

func setGinLibraryFrames(frames []model.StacktraceFrame) {
	for i, f := range frames {
		if f.LibraryFrame {
			continue
		}
		if !strings.HasPrefix(f.Module, ginPackagePrefix) && !strings.HasPrefix(f.Module, ginContribPackagePrefix) {
			continue
		}
		frames[i].LibraryFrame = true
	}
}
