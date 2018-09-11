package transporttest

import (
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/apm-agent-go"
	"github.com/elastic/apm-agent-go/model"
)

// NewRecorderTracer returns a new elasticapm.Tracer and
// RecorderTransport, which is set as the tracer's transport.
func NewRecorderTracer() (*elasticapm.Tracer, *RecorderTransport) {
	var transport RecorderTransport
	tracer, err := elasticapm.NewTracer("transporttest", "")
	if err != nil {
		panic(err)
	}
	tracer.Transport = &transport
	return tracer, &transport
}

// RecorderTransport implements transport.Transport, recording the
// streams sent. The streams can be retrieved using the Payloads
// method.
type RecorderTransport struct {
	mu       sync.Mutex
	metadata *metadata
	payloads Payloads
}

// SendStream records the stream such that it can later be obtained via Payloads.
func (r *RecorderTransport) SendStream(ctx context.Context, stream io.Reader) error {
	return r.record(stream)
}

// Metadata returns the metadata recorded by the transport. If metadata is yet to
// be received, this method will panic.
func (r *RecorderTransport) Metadata() (model.System, model.Process, model.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.metadata.System, r.metadata.Process, r.metadata.Service
}

// Payloads returns the payloads recorded by SendStream.
func (r *RecorderTransport) Payloads() Payloads {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.payloads
}

func (r *RecorderTransport) record(stream io.Reader) error {
	reader, err := zlib.NewReader(stream)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(reader)

	// The first object of any request must be a metadata struct.
	var metadataPayload struct {
		Metadata metadata `json:"metadata"`
	}
	if err := decoder.Decode(&metadataPayload); err != nil {
		panic(err)
	}
	r.recordMetadata(&metadataPayload.Metadata)

	for {
		var payload struct {
			Error       *model.Error       `json:"error"`
			Metrics     *model.Metrics     `json:"metricset"`
			Span        *model.Span        `json:"span"`
			Transaction *model.Transaction `json:"transaction"`
		}
		err := decoder.Decode(&payload)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		r.mu.Lock()
		switch {
		case payload.Error != nil:
			r.payloads.Errors = append(r.payloads.Errors, *payload.Error)
		case payload.Metrics != nil:
			r.payloads.Metrics = append(r.payloads.Metrics, *payload.Metrics)
		case payload.Span != nil:
			r.payloads.Spans = append(r.payloads.Spans, *payload.Span)
		case payload.Transaction != nil:
			r.payloads.Transactions = append(r.payloads.Transactions, *payload.Transaction)
		}
		r.mu.Unlock()
	}
	return nil
}

func (r *RecorderTransport) recordMetadata(m *metadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.metadata == nil {
		r.metadata = m
	} else {
		// Make sure the metadata doesn't change between requests.
		if diff := cmp.Diff(r.metadata, m); diff != "" {
			panic(fmt.Errorf("metadata changed\n%s", diff))
		}
	}
}

// Payloads holds the recorded payloads.
type Payloads struct {
	Errors       []model.Error
	Metrics      []model.Metrics
	Spans        []model.Span
	Transactions []model.Transaction
}

// Len returns the number of recorded payloads.
func (p *Payloads) Len() int {
	return len(p.Transactions) + len(p.Errors) + len(p.Metrics)
}

type metadata struct {
	System  model.System  `json:"system"`
	Process model.Process `json:"process"`
	Service model.Service `json:"service"`
}
