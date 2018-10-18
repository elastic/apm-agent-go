package apmgrpc

import "go.elastic.co/apm/stacktrace"

func init() {
	stacktrace.RegisterLibraryPackage(
		"google.golang.org/grpc",
		"github.com/grpc-ecosystem",
	)
}
