package stacktrace

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/elastic/apm-agent-go/model"
)

//go:generate /bin/bash generate_library.bash std ..

// TODO(axw) add a function for marking frames as library
// frames, based on configuration. Possibly use
// in-the-same-repo as a heuristic?

// Stacktrace returns a slice of StacktraceFrame
// of at most n frames, skipping skip frames starting
// with Stacktrace. If n is negative, then all stack
// frames will be returned.
//
// See RuntimeStacktraceFrame for information on what
// details are included.
func Stacktrace(skip, n int) []model.StacktraceFrame {
	if n == 0 {
		return nil
	}
	var pc []uintptr
	if n > 0 {
		pc = make([]uintptr, n)
		pc = pc[:runtime.Callers(skip+1, pc)]
	} else {
		// n is negative, get all frames.
		n = 0
		pc = make([]uintptr, 10)
		for {
			n += runtime.Callers(skip+n+1, pc[n:])
			if n < len(pc) {
				pc = pc[:n]
				break
			}
			pc = append(pc, 0)
		}
	}
	return Callers(pc)
}

// Callers returns a slice of StacktraceFrame
// for the given callers (program counter values).
//
// See RuntimeStacktraceFrame for information on what
// details are included.
func Callers(callers []uintptr) []model.StacktraceFrame {
	if len(callers) == 0 {
		return nil
	}
	frames := runtime.CallersFrames(callers)
	out := make([]model.StacktraceFrame, 0, len(callers))
	for {
		frame, more := frames.Next()
		out = append(out, RuntimeStacktraceFrame(&frame))
		if !more {
			break
		}
	}
	return out
}

// RuntimeStacktraceFrame returns a StacktraceFrame
// based on the given runtime.Frame.
//
// The resulting StacktraceFrame will have the file,
// function, module (package path), and function
// information set. Context (source code) and vars
// can be filled in separately if desired.
//
// Only stack frames pertaining to code in the standard
// library will be marked as "library" frames. Logic
// for classifying non-standard library packages may
// be applied on the results.
func RuntimeStacktraceFrame(in *runtime.Frame) model.StacktraceFrame {
	var abspath string
	file := in.File
	if filepath.IsAbs(file) {
		abspath = file
		file = filepath.Base(file)
	}

	packagePath, function := SplitFunctionName(in.Function)
	return model.StacktraceFrame{
		AbsolutePath: abspath,
		File:         file,
		Line:         in.Line,
		Function:     function,
		Module:       packagePath,
		LibraryFrame: IsLibraryPackage(packagePath),
	}
}

// SplitFunctionName splits the function name as formatted in
// runtime.Frame.Function, and returns the package path and
// function name components.
func SplitFunctionName(in string) (packagePath, function string) {
	function = in
	if function == "" {
		return "", ""
	}
	// The last part of a package path will always have "."
	// encoded as "%2e", so we can pick off the package path
	// by finding the last part of the package path, and then
	// the proceeding ".".
	//
	// Unexported method names may contain the package path.
	// In these cases, the method receiver will be enclosed
	// in parentheses, so we can treat that as the start of
	// the function name.
	sep := strings.Index(function, ".(")
	if sep >= 0 {
		packagePath = unescape(function[:sep])
		function = function[sep+1:]
	} else {
		offset := 0
		if sep := strings.LastIndex(function, "/"); sep >= 0 {
			offset = sep
		}
		if sep := strings.IndexRune(function[offset+1:], '.'); sep >= 0 {
			packagePath = unescape(function[:offset+1+sep])
			function = function[offset+1+sep+1:]
		}
	}
	return packagePath, function
}

func unescape(s string) string {
	var n int
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			n++
		}
	}
	if n == 0 {
		return s
	}
	bytes := make([]byte, 0, len(s)-2*n)
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '%' {
			b = fromhex(s[i+1])<<4 | fromhex(s[i+2])
			i += 2
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}

func fromhex(b byte) byte {
	if b >= 'a' {
		return 10 + b - 'a'
	}
	return b - '0'
}
