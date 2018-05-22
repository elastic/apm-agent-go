package elasticapm

import (
	"context"
	"regexp"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
)

type sender struct {
	tracer                  *Tracer
	logger                  Logger
	processor               Processor
	contextSetter           stacktrace.ContextSetter
	preContext, postContext int
	stats                   *TracerStats
	sanitizedFieldNames     *regexp.Regexp
}

// sendTransactions attempts to send enqueued transactions to the APM server,
// returning true if the transactions were successfully sent.
func (s *sender) sendTransactions(ctx context.Context, transactions []*Transaction) bool {
	if len(transactions) == 0 {
		return false
	}
	if s.contextSetter != nil {
		var err error
		for _, tx := range transactions {
			if err = tx.setContext(s.contextSetter, s.preContext, s.postContext); err != nil {
				break
			}
		}
		if err != nil {
			if s.logger != nil {
				s.logger.Debugf("setting context failed: %s", err)
			}
			s.stats.Errors.SetContext++
		}
	}
	service := makeService(s.tracer.Service.Name, s.tracer.Service.Version, s.tracer.Service.Environment)
	payload := model.TransactionsPayload{
		Service:      &service,
		Process:      s.tracer.process,
		System:       s.tracer.system,
		Transactions: make([]*model.Transaction, len(transactions)),
	}
	for i, tx := range transactions {
		tx.setID()
		if tx.Sampled() {
			tx.model.Context = tx.Context.build()
			if s.sanitizedFieldNames != nil && tx.model.Context != nil && tx.model.Context.Request != nil {
				sanitizeRequest(tx.model.Context.Request, s.sanitizedFieldNames)
			}
		}
		if s.processor != nil {
			s.processor.ProcessTransaction(&tx.model)
		}
		payload.Transactions[i] = &tx.model
	}
	if err := s.tracer.Transport.SendTransactions(ctx, &payload); err != nil {
		if s.logger != nil {
			s.logger.Debugf("sending transactions failed: %s", err)
		}
		s.stats.Errors.SendTransactions++
		return false
	}
	s.stats.TransactionsSent += uint64(len(transactions))
	return true
}

// sendErrors attempts to send enqueued errors to the APM server,
// returning true if the errors were successfully sent.
func (s *sender) sendErrors(ctx context.Context, errors []*Error) bool {
	if len(errors) == 0 {
		return false
	}
	if s.contextSetter != nil {
		var err error
		for _, e := range errors {
			if err = e.setContext(s.contextSetter, s.preContext, s.postContext); err != nil {
				break
			}
		}
		if err != nil {
			if s.logger != nil {
				s.logger.Debugf("setting context failed: %s", err)
			}
			s.stats.Errors.SetContext++
		}
	}
	service := makeService(s.tracer.Service.Name, s.tracer.Service.Version, s.tracer.Service.Environment)
	payload := model.ErrorsPayload{
		Service: &service,
		Process: s.tracer.process,
		System:  s.tracer.system,
		Errors:  make([]*model.Error, len(errors)),
	}
	for i, e := range errors {
		if e.Transaction != nil {
			e.Transaction.setID()
			e.model.Transaction.ID = e.Transaction.model.ID
		}
		e.setStacktrace()
		e.setCulprit()
		e.model.ID = e.ID
		e.model.Timestamp = model.Time(e.Timestamp.UTC())
		e.model.Context = e.Context.build()
		e.model.Exception.Handled = e.Handled
		if s.processor != nil {
			s.processor.ProcessError(&e.model)
		}
		payload.Errors[i] = &e.model
	}
	if err := s.tracer.Transport.SendErrors(ctx, &payload); err != nil {
		if s.logger != nil {
			s.logger.Debugf("sending errors failed: %s", err)
		}
		s.stats.Errors.SendErrors++
		return false
	}
	s.stats.ErrorsSent += uint64(len(errors))
	return true
}
