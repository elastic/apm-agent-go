package apmschema

import (
	"go/build"
	"log"
	"path"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema"
)

var (
	// Error is the compiled JSON Schema for an error.
	Error *jsonschema.Schema

	// Metadata is the compiled JSON Schema for metadata.
	Metadata *jsonschema.Schema

	// Metrics is the compiled JSON Schema for a set of metrics.
	Metrics *jsonschema.Schema

	// Span is the compiled JSON Schema for a span.
	Span *jsonschema.Schema

	// Transaction is the compiled JSON Schema for a transaction.
	Transaction *jsonschema.Schema
)

func init() {
	pkg, err := build.Default.Import("github.com/elastic/apm-agent-go/internal/apmschema", "", build.FindOnly)
	if err != nil {
		log.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft4
	schemaDir := path.Join(filepath.ToSlash(pkg.Dir), "jsonschema")
	compile := func(filepath string, out **jsonschema.Schema) {
		schema, err := compiler.Compile("file://" + path.Join(schemaDir, filepath))
		if err != nil {
			log.Fatal(err)
		}
		*out = schema
	}
	compile("errors/v2_error.json", &Error)
	compile("metadata.json", &Metadata)
	compile("metrics/metric.json", &Metrics)
	compile("spans/v2_span.json", &Span)
	compile("transactions/v2_transaction.json", &Transaction)
}
