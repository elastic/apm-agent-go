package trace

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/apm-agent-go/model"
	"github.com/elastic/apm-agent-go/stacktrace"
	"github.com/elastic/apm-agent-go/transport"
)

const (
	defaultPreContext      = 3
	defaultPostContext     = 3
	transactionsChannelCap = 1000
	errorsChannelCap       = 1000

	// defaultMaxErrorQueueSize is the default maximum number
	// of errors to enqueue in the tracer. When this fills up,
	// errors will start being dropped (when the channel is
	// also full).
	defaultMaxErrorQueueSize = 1000
)

// Tracer manages the sampling and sending of transactions to
// Elastic APM.
//
// Transactions are buffered until they are flushed (forcibly
// with a Flush call, or when the flush timer expires), or when
// the maximum transaction queue size is reached. Failure to
// send will be periodically retried. Once the queue limit has
// been reached, new transactions will replace older ones in
// the queue.
//
// Errors are sent as soon as possible, but will buffered and
// later sent in bulk if the tracer is busy, or otherwise cannot
// send to the server, e.g. due to network failure. There is
// a limit to the number of errors that will be buffered, and
// once that limit has been reached, new errors will be dropped
// until the queue is drained.
//
// The exported fields be altered or replaced any time up until
// any Tracer methods have been invoked.
type Tracer struct {
	Transport transport.Transport
	Service   *model.Service

	process *model.Process
	system  *model.System

	closing                    chan struct{}
	closed                     chan struct{}
	forceFlush                 chan chan<- struct{}
	setFlushInterval           chan time.Duration
	setMaxTransactionQueueSize chan int
	setMaxErrorQueueSize       chan int
	setPreContext              chan int
	setPostContext             chan int
	setContextSetter           chan stacktrace.ContextSetter
	setLogger                  chan Logger
	setProcessor               chan Processor
	transactions               chan *Transaction
	errors                     chan *Error

	statsMu sync.Mutex
	stats   TracerStats

	maxSpansMu sync.RWMutex
	maxSpans   int

	errorPool       sync.Pool
	spanPool        sync.Pool
	transactionPool sync.Pool
}

// NewTracer returns a new Tracer, using the default transport,
// initializing a Service with the specified name and version,
// or taking the service name and version from the environment
// if unspecified.
//
// If service is nil, then the service will be defined using the
// ELASTIC_APM_* environment variables.
func NewTracer(serviceName, serviceVersion string) (*Tracer, error) {
	service := envService
	if serviceName != "" {
		service = newService(serviceName, serviceVersion)
	} else if service == nil {
		return nil, errors.Errorf(
			"no service name specified, and %s not specified",
			envServiceName,
		)
	}

	flushInterval, err := initialFlushInterval()
	if err != nil {
		return nil, err
	}
	maxTransactionQueueSize, err := initialMaxTransactionQueueSize()
	if err != nil {
		return nil, err
	}
	maxSpans, err := initialMaxSpans()
	if err != nil {
		return nil, err
	}
	t := &Tracer{
		Transport:                  transport.Default,
		Service:                    service,
		process:                    &currentProcess,
		system:                     &localSystem,
		closing:                    make(chan struct{}),
		closed:                     make(chan struct{}),
		forceFlush:                 make(chan chan<- struct{}),
		setFlushInterval:           make(chan time.Duration),
		setMaxTransactionQueueSize: make(chan int),
		setMaxErrorQueueSize:       make(chan int),
		setPreContext:              make(chan int),
		setPostContext:             make(chan int),
		setContextSetter:           make(chan stacktrace.ContextSetter),
		setLogger:                  make(chan Logger),
		setProcessor:               make(chan Processor),
		transactions:               make(chan *Transaction, transactionsChannelCap),
		errors:                     make(chan *Error, errorsChannelCap),
		maxSpans:                   maxSpans,
	}
	go t.loop()
	t.setFlushInterval <- flushInterval
	t.setMaxTransactionQueueSize <- maxTransactionQueueSize
	t.setMaxErrorQueueSize <- defaultMaxErrorQueueSize
	t.setPreContext <- defaultPreContext
	t.setPostContext <- defaultPostContext
	return t, nil
}

// Close closes the Tracer, preventing transactions from being
// sent to the APM server.
func (t *Tracer) Close() {
	select {
	case <-t.closing:
	default:
		close(t.closing)
	}
	<-t.closed
}

// Flush waits for the Tracer to flush any transactions and errors it currently
// has queued to the APM server, the tracer is stopped, or the abort channel
// is signaled.
func (t *Tracer) Flush(abort <-chan struct{}) {
	flushed := make(chan struct{}, 1)
	select {
	case t.forceFlush <- flushed:
		select {
		case <-abort:
		case <-flushed:
		case <-t.closed:
		}
	case <-t.closed:
	}
}

// SetFlushInterval sets the flush interval -- the amount of time
// to wait before flushing enqueued transactions to the APM server.
func (t *Tracer) SetFlushInterval(d time.Duration) {
	select {
	case t.setFlushInterval <- d:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetMaxTransactionQueueSize sets the maximum transaction queue size -- the
// maximum number of transactions to buffer before flushing to the APM server.
// If set to a non-positive value, the queue size is unlimited.
func (t *Tracer) SetMaxTransactionQueueSize(n int) {
	select {
	case t.setMaxTransactionQueueSize <- n:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetMaxErrorQueueSize sets the maximum error queue size -- the
// maximum number of errors to buffer before they will start getting
// dropped. If set to a non-positive value, the queue size is unlimited.
func (t *Tracer) SetMaxErrorQueueSize(n int) {
	select {
	case t.setMaxErrorQueueSize <- n:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetContextSetter sets the stacktrace.ContextSetter to be used for
// setting stacktrace source context. If nil (which is the initial
// value), no context will be set.
func (t *Tracer) SetContextSetter(setter stacktrace.ContextSetter) {
	select {
	case t.setContextSetter <- setter:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetLogger sets the Logger to be used for logging the operation of
// the tracer.
func (t *Tracer) SetLogger(logger Logger) {
	select {
	case t.setLogger <- logger:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetProcessor sets the processors for the tracer.
func (t *Tracer) SetProcessor(p ...Processor) {
	var processor Processor
	if len(p) > 0 {
		processor = Processors(p)
	}
	select {
	case t.setProcessor <- processor:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetMaxSpans sets the maximum number of spans that will be added
// to a transaction before dropping. If set to a non-positive value,
// the number of spans is unlimited.
func (t *Tracer) SetMaxSpans(n int) {
	t.maxSpansMu.Lock()
	t.maxSpans = n
	t.maxSpansMu.Unlock()
}

// Stats returns the current TracerStats. This will return the most
// recent values even after the tracer has been closed.
func (t *Tracer) Stats() TracerStats {
	t.statsMu.Lock()
	stats := t.stats
	t.statsMu.Unlock()
	return stats
}

func (t *Tracer) loop() {
	defer close(t.closed)

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()
	go func() {
		select {
		case <-t.closing:
			cancelContext()
		}
	}()

	var flushInterval time.Duration
	var maxTransactionQueueSize int
	var maxErrorQueueSize int
	var flushC <-chan time.Time
	var transactions []*Transaction
	var errors []*Error
	var statsUpdates TracerStats
	sender := sender{
		tracer: t,
		stats:  &statsUpdates,
	}

	errorsC := t.errors
	forceFlush := t.forceFlush
	flushTimer := time.NewTimer(0)
	if !flushTimer.Stop() {
		<-flushTimer.C
	}
	startTimer := func() {
		if flushC != nil {
			// Timer already started.
			return
		}
		if !flushTimer.Stop() {
			select {
			case <-flushTimer.C:
			default:
			}
		}
		flushTimer.Reset(flushInterval)
		flushC = flushTimer.C
	}
	receivedTransaction := func(tx *Transaction, stats *TracerStats) {
		if maxTransactionQueueSize > 0 && len(transactions) >= maxTransactionQueueSize {
			// The queue is full, so pop the oldest item.
			// TODO(axw) use container/ring? implement
			// ring buffer on top of slice? profile
			n := uint64(len(transactions) - maxTransactionQueueSize + 1)
			for _, tx := range transactions[:n] {
				tx.reset()
				t.transactionPool.Put(tx)
			}
			transactions = transactions[n:]
			stats.TransactionsDropped += n
		}
		transactions = append(transactions, tx)
	}

	for {
		var sendTransactions bool
		var flushed chan<- struct{}
		statsUpdates = TracerStats{}

		select {
		case <-t.closing:
			return
		case flushInterval = <-t.setFlushInterval:
			continue
		case maxTransactionQueueSize = <-t.setMaxTransactionQueueSize:
			if maxTransactionQueueSize <= 0 || len(transactions) < maxTransactionQueueSize {
				continue
			}
		case maxErrorQueueSize = <-t.setMaxErrorQueueSize:
			if maxErrorQueueSize <= 0 || len(errors) < maxErrorQueueSize {
				errorsC = t.errors
			}
			continue
		case sender.preContext = <-t.setPreContext:
			continue
		case sender.postContext = <-t.setPostContext:
			continue
		case sender.contextSetter = <-t.setContextSetter:
			continue
		case sender.logger = <-t.setLogger:
			continue
		case sender.processor = <-t.setProcessor:
			continue
		case e := <-errorsC:
			errors = append(errors, e)
		case tx := <-t.transactions:
			beforeLen := len(transactions)
			receivedTransaction(tx, &statsUpdates)
			if len(transactions) == beforeLen && flushC != nil {
				// The queue was already full, and a retry
				// timer is running; wait for it to fire.
				t.statsMu.Lock()
				t.stats.accumulate(statsUpdates)
				t.statsMu.Unlock()
				continue
			}
			if maxTransactionQueueSize <= 0 || len(transactions) < maxTransactionQueueSize {
				startTimer()
				continue
			}
			sendTransactions = true
		case <-flushC:
			flushC = nil
			sendTransactions = true
		case flushed = <-forceFlush:
			// The caller has explicitly requested a flush, so
			// drain any transactions buffered in the channel.
			for n := len(t.transactions); n > 0; n-- {
				tx := <-t.transactions
				receivedTransaction(tx, &statsUpdates)
			}
			// flushed will be signaled, and forceFlush set back to
			// t.forceFlush, when the queued transactions and/or
			// errors are successfully sent.
			forceFlush = nil
			flushC = nil
			sendTransactions = true
		}

		if remainder := maxErrorQueueSize - len(errors); remainder > 0 {
			// Drain any errors in the channel, up to the maximum queue size.
			for n := len(t.errors); n > 0 && remainder > 0; n-- {
				errors = append(errors, <-t.errors)
				remainder--
			}
		}
		if sender.sendErrors(ctx, errors) {
			for _, e := range errors {
				e.reset()
				t.errorPool.Put(e)
			}
			errors = errors[:0]
			errorsC = t.errors
		} else if len(errors) == maxErrorQueueSize {
			errorsC = nil
		}
		if sendTransactions {
			if sender.sendTransactions(ctx, transactions) {
				for _, tx := range transactions {
					tx.reset()
					t.transactionPool.Put(tx)
				}
				transactions = transactions[:0]
			}
		}

		if !statsUpdates.isZero() {
			t.statsMu.Lock()
			t.stats.accumulate(statsUpdates)
			t.statsMu.Unlock()

			if statsUpdates.Errors.SendTransactions != 0 || statsUpdates.Errors.SendErrors != 0 {
				// Sending transactions or errors failed, start a new timer to resend.
				startTimer()
				continue
			}
		}
		if sendTransactions && flushed != nil {
			forceFlush = t.forceFlush
			flushed <- struct{}{}
			flushed = nil
		}
	}
}

type sender struct {
	tracer                  *Tracer
	logger                  Logger
	processor               Processor
	contextSetter           stacktrace.ContextSetter
	preContext, postContext int
	stats                   *TracerStats
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
	payload := model.TransactionsPayload{
		Service:      s.tracer.Service,
		Process:      s.tracer.process,
		System:       s.tracer.system,
		Transactions: make([]*model.Transaction, len(transactions)),
	}
	for i, tx := range transactions {
		tx.setID()
		if s.processor != nil {
			s.processor.ProcessTransaction(&tx.Transaction)
		}
		payload.Transactions[i] = &tx.Transaction
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
	payload := model.ErrorsPayload{
		Service: s.tracer.Service,
		Process: s.tracer.process,
		System:  s.tracer.system,
		Errors:  make([]*model.Error, len(errors)),
	}
	for i, e := range errors {
		if e.Transaction != nil {
			e.Transaction.setID()
			e.TransactionID = e.Transaction.ID
		}
		if s.processor != nil {
			s.processor.ProcessError(&e.Error)
		}
		e.setCulprit()
		payload.Errors[i] = &e.Error
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
