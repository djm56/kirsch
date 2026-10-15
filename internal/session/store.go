package session

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/djm56/kirsch/internal/agent"
)

// ErrClosed is returned by Flush when the store has already been closed.
var ErrClosed = errors.New("session store closed")

// Option configures a Store.
type Option func(*Store)

// WithCadence sets the background fsync interval. The default is 500ms.
func WithCadence(d time.Duration) Option {
	return func(s *Store) {
		s.cadence = d
	}
}

// WithOnError sets the callback used for non-fatal write/fsync errors.
//
// Mid-turn write and fsync failures are reported here; they do not surface to
// the caller of Record because the Recorder seam is intentionally no-error.
// The consumer is expected to wire this to the telemetry debug log. A nil
// callback is treated as a no-op.
func WithOnError(fn func(error)) Option {
	return func(s *Store) {
		if fn != nil {
			s.onError = fn
		}
	}
}

// Store implements agent.Recorder by appending JSONL events to a file.
//
// A single writer goroutine owns all file writes and fsyncs. Record is safe
// for concurrent callers; it serialises the message immediately and posts the
// resulting lines to the writer's queue. Buffered output is flushed and
// fsynced on a ticker and on explicit Flush calls.
type Store struct {
	// path is the caller-supplied file path. Path containment is not this
	// layer's responsibility; the caller (m4-d2 layout) decides where the
	// file lives.
	path string

	// file is the open session file.
	file *os.File

	// buf batches writes to file.
	buf *bufio.Writer

	// events carries batches of JSONL lines waiting to be written.
	// Each batch belongs to one Record call and is written contiguously.
	events chan [][]byte

	// flushReq carries response channels for explicit Flush calls.
	flushReq chan chan error

	// closed is closed when Close is called.
	closed chan struct{}

	// closeOnce ensures closed is closed exactly once.
	closeOnce sync.Once

	// cadence is the background fsync interval.
	cadence time.Duration

	// onError reports non-fatal write/fsync errors.
	onError func(error)

	// recovered is true if the session file was repaired when it was opened.
	recovered bool

	// wg waits for the writer goroutine.
	wg sync.WaitGroup
}

// New opens or creates a session store at path.
//
// The caller owns path selection; this function does not validate containment.
// The returned Store starts its writer goroutine immediately and must be
// closed with Close to release the file and goroutine.
func New(path string, opts ...Option) (*Store, error) {
	s := &Store{
		path:     path,
		events:   make(chan [][]byte, 256),
		flushReq: make(chan chan error),
		closed:   make(chan struct{}),
		cadence:  500 * time.Millisecond,
		onError:  func(error) {},
	}
	for _, opt := range opts {
		opt(s)
	}

	f, recovered, err := openAndRepair(path, s.report)
	if err != nil {
		return nil, err
	}
	s.file = f
	s.recovered = recovered
	s.buf = bufio.NewWriter(f)

	s.wg.Add(1)
	go s.writer()

	return s, nil
}

// Recovered reports whether the session file was repaired when it was opened.
//
// A recovered session had a corrupt or torn final line that was truncated to
// the last good line.
func (s *Store) Recovered() bool {
	return s.recovered
}

// openAndRepair opens path for append and repairs any torn trailing line.
//
// The file is opened read-write so repairSession can read and truncate it.
// Repair errors are reported through the supplied hook but do not prevent the
// store from opening, satisfying the requirement that loading never fails
// startup on one bad file.
func openAndRepair(path string, report func(error)) (*os.File, bool, error) {
	// #nosec G304 -- path is caller-supplied; containment is the caller's
	// responsibility, matching internal/workspace/detect.go,
	// internal/tool/readfile.go and internal/agent/prompt/loader.go.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, false, err
	}

	recovered, repairErr := repairSession(f)
	if repairErr != nil {
		report(fmt.Errorf("repair session %s: %w", path, repairErr))
		// Continue anyway; a bad session file must not block startup.
	}

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		return nil, false, err
	}
	return f, recovered, nil
}

// Record implements agent.Recorder.
//
// It serialises the message to one or more JSONL lines and posts them to the
// writer goroutine. Record blocks until the batch is accepted by the queue;
// it does not return errors because the Recorder seam keeps durability
// concerns in the store layer. Encoding errors and mid-turn write failures
// are reported through the OnError hook instead.
func (s *Store) Record(kind agent.RecordKind, m agent.Message) {
	lines, err := encode(kind, m)
	if err != nil {
		s.report(err)
		return
	}
	if len(lines) == 0 {
		return
	}

	select {
	case <-s.closed:
		s.report(ErrClosed)
		return
	default:
	}

	select {
	case s.events <- lines:
	case <-s.closed:
		s.report(ErrClosed)
	}
}

// RecordTurnStarted writes a turn.started event for the given turn number.
//
// This is a non-conversation event; it does not flow through agent.Recorder.
func (s *Store) RecordTurnStarted(turn int) {
	line, err := marshalTurnStarted(turn)
	if err != nil {
		s.report(err)
		return
	}
	s.postLine(line)
}

// RecordTurnCompleted writes a turn.completed event with the supplied status.
//
// Status values follow plan §7: "success", "cancelled", or "error".
func (s *Store) RecordTurnCompleted(status string) {
	line, err := marshalTurnCompleted(status)
	if err != nil {
		s.report(err)
		return
	}
	s.postLine(line)
}

// RecordUsage writes a usage event carrying per-turn token counts.
//
// The JSONL event records input, output, cache read and cache write tokens.
// Aggregate usage is updated separately via AddUsage in the app wiring layer.
func (s *Store) RecordUsage(u agent.Usage) {
	line, err := marshalUsage(u)
	if err != nil {
		s.report(err)
		return
	}
	s.postLine(line)
}

// RecordScopeGranted writes an approval.scope_granted event.
//
// Action is the operation kind (e.g. "run_command") and pattern is the granted
// argument pattern. This is emitted only after a successful session-scope grant.
func (s *Store) RecordScopeGranted(action, pattern string) {
	line, err := marshalApprovalScopeGranted(action, pattern)
	if err != nil {
		s.report(err)
		return
	}
	s.postLine(line)
}

// postLine enqueues a single JSONL line for the writer goroutine.
func (s *Store) postLine(line []byte) {
	if len(line) == 0 {
		return
	}

	select {
	case <-s.closed:
		s.report(ErrClosed)
		return
	default:
	}

	select {
	case s.events <- [][]byte{line}:
	case <-s.closed:
		s.report(ErrClosed)
	}
}

// Flush flushes the buffered writer and fsyncs the file.
//
// It returns any write or fsync error so that turn completion can surface
// durability failures to the user. Flush returns ErrClosed if the store has
// already been closed.
func (s *Store) Flush() error {
	select {
	case <-s.closed:
		return ErrClosed
	default:
	}

	resp := make(chan error, 1)
	select {
	case s.flushReq <- resp:
	case <-s.closed:
		return ErrClosed
	}

	select {
	case err := <-resp:
		return err
	case <-s.closed:
		return ErrClosed
	}
}

// Close shuts down the store.
//
// It stops accepting new events, drains any queued events, flushes and fsyncs
// the file, stops the writer goroutine, and closes the file. Close is safe to
// call more than once; subsequent calls return nil.
func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		s.wg.Wait()
	})
	return nil
}

// writer is the single goroutine that owns all file writes.
func (s *Store) writer() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cadence)
	defer ticker.Stop()

	for {
		select {
		case lines := <-s.events:
			s.writeLines(lines)

		case resp := <-s.flushReq:
			// Include every event that was queued before this flush request.
			s.drainEvents()
			resp <- s.flush()

		case <-ticker.C:
			if err := s.flush(); err != nil {
				s.report(err)
			}

		case <-s.closed:
			s.drain()
			if err := s.flush(); err != nil {
				s.report(err)
			}
			_ = s.file.Close()
			return
		}
	}
}

// writeLines writes every line in the batch to the buffered writer.
// If any write fails the remaining lines are dropped and the error is
// reported; the caller (the writer loop) will continue running.
func (s *Store) writeLines(lines [][]byte) {
	for _, line := range lines {
		if _, err := s.buf.Write(line); err != nil {
			s.report(err)
			return
		}
	}
}

// flush flushes the buffered writer and fsyncs the underlying file.
func (s *Store) flush() error {
	if err := s.buf.Flush(); err != nil {
		return err
	}
	return s.file.Sync()
}

// drain empties the event and flush queues without blocking.
func (s *Store) drain() {
	for {
		select {
		case lines := <-s.events:
			s.writeLines(lines)
		case resp := <-s.flushReq:
			resp <- s.flush()
		default:
			return
		}
	}
}

// drainEvents empties queued events without blocking.
func (s *Store) drainEvents() {
	for {
		select {
		case lines := <-s.events:
			s.writeLines(lines)
		default:
			return
		}
	}
}

// report delivers a non-fatal error to the configured hook.
func (s *Store) report(err error) {
	if err == nil {
		return
	}
	s.onError(err)
}
