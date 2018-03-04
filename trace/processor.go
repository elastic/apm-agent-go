package trace

import "github.com/elastic/apm-agent-go/model"

// Processor providing methods for making adjustments to transaction
// and error values before sending them to the APM server.
type Processor interface {
	ErrorProcessor
	TransactionProcessor
}

// ErrorProcessor providing methods for making adjustments to error
// values before sending them to the APM server.
type ErrorProcessor interface {
	// ProcessError processes a model.Error, possibly making adjustments.
	//
	// ProcessError will be called from the Tracer's internal goroutine,
	// and can safely make changes to the supplied Error without any
	// additional synchronization.
	ProcessError(*model.Error)
}

// TransactionProcessor providing methods for making adjustments to transaction
// values before sending them to the APM server.
type TransactionProcessor interface {
	// ProcessTransaction processes a model.Transaction, possibly making
	// adjustments.
	//
	// ProcessTransaction will be called from the Tracer's internal
	// goroutine, and can safely make changes to the supplied Transaction
	// without any additional synchronization.
	ProcessTransaction(*model.Transaction)
}

// Processors is a slice of Processors; each entry's Process methods
// will be invoked in series.
type Processors []Processor

func (p Processors) ProcessError(e *model.Error) {
	for _, p := range p {
		p.ProcessError(e)
	}
}

func (p Processors) ProcessTransaction(t *model.Transaction) {
	for _, p := range p {
		p.ProcessTransaction(t)
	}
}
