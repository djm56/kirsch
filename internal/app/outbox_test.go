package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// orderRecorder is a sendFn that records messages in delivery order. The first
// message it sees is held inside sendFn until release is closed, which is how
// the tests keep a delivery in flight while later messages are produced.
type orderRecorder struct {
	mu      sync.Mutex
	got     []string
	entered chan struct{} // closed when the held message has entered sendFn
	release chan struct{} // closes to let the held message finish
	seen    chan string   // receives every message after the held one, as it is delivered
	once    sync.Once
}

func newOrderRecorder() *orderRecorder {
	return &orderRecorder{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		seen:    make(chan string, 16),
	}
}

// send is the sendFn. Messages are strings. The first one blocks until release
// is closed and is recorded when it finishes; the rest are recorded at once.
func (r *orderRecorder) send(msg any) {
	s := msg.(string)
	held := false
	r.once.Do(func() {
		held = true
		close(r.entered)
	})
	if held {
		<-r.release
	}
	r.mu.Lock()
	r.got = append(r.got, s)
	r.mu.Unlock()
	if !held {
		r.seen <- s
	}
}

func (r *orderRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.got...)
}

// holdThenRelease waits for the held message to enter sendFn, then calls
// produce (which enqueues later messages) and releases the held message. It
// first gives a later message up to wait to be delivered while the first is
// still held: under a sender that delivers each message from its own goroutine
// that happens at once, under the outbox it cannot happen, so the wait only
// ever runs out on the passing path and decides nothing there.
func (r *orderRecorder) holdThenRelease(t *testing.T, produce func()) {
	t.Helper()
	select {
	case <-r.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the first message never reached sendFn")
	}
	produce()
	select {
	case <-time.After(100 * time.Millisecond):
	case s := <-r.seen:
		// Delivered while the first message is still held: put it back for the
		// final comparison, which will show the inversion.
		r.seen <- s
	}
	close(r.release)
}

// waitIdle waits for the outbox goroutine to exit, failing after 5 seconds.
func waitIdle(t *testing.T, a *App) {
	t.Helper()
	done := make(chan struct{})
	go func() { a.out.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the outbox goroutine did not exit")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestOutboxDeliversAsyncAndBlockingSendsInProductionOrder holds the first
// delivery (an async fenced message) inside sendFn, then produces a second async
// fenced message and a blocking send, in that order. Entry points are
// sendAsyncSequenceFenced and send, the two senders RunTurn's emit and the tool
// executor use. Delivery order must equal production order: with a goroutine per
// message the second message is delivered while the first is held. The blocking
// send must also return only after its own message is recorded.
func TestOutboxDeliversAsyncAndBlockingSendsInProductionOrder(t *testing.T) {
	r := newOrderRecorder()
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()
	a.sendFn = r.send
	gen := a.currentGen()

	a.sendAsyncSequenceFenced(gen, []tea.Msg{"text-delta"})
	var recordedAtReturn []string
	sendReturned := make(chan struct{})
	r.holdThenRelease(t, func() {
		a.sendAsyncSequenceFenced(gen, []tea.Msg{"text-delta-2"})
		go func() {
			a.send(gen, "tool-started")
			recordedAtReturn = r.snapshot()
			close(sendReturned)
		}()
	})
	select {
	case <-sendReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("the blocking send never returned")
	}

	want := []string{"text-delta", "text-delta-2", "tool-started"}
	if got := r.snapshot(); !equalStrings(got, want) {
		t.Errorf("delivery order = %v, want %v", got, want)
	}
	if !equalStrings(recordedAtReturn, want) {
		t.Errorf("when send returned the delivered messages were %v, want all of %v", recordedAtReturn, want)
	}
}

// TestOutboxDropsStaleGenerationAndStillDeliversCurrent queues, behind a held
// delivery, a fenced message from an old generation, a fenced message from the
// current one (the positive control: same path, delivered) and an old-generation
// blocking send, which must return without delivering. Entry points are
// sendAsyncSequenceFenced and send.
func TestOutboxDropsStaleGenerationAndStillDeliversCurrent(t *testing.T) {
	r := newOrderRecorder()
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()
	a.sendFn = r.send
	a.sessionGen.Store(5)

	a.sendAsyncSequenceFenced(5, []tea.Msg{"first"})
	sendReturned := make(chan struct{})
	r.holdThenRelease(t, func() {
		a.sendAsyncSequenceFenced(4, []tea.Msg{"stale-async"})
		a.sendAsyncSequenceFenced(5, []tea.Msg{"current"})
		go func() {
			a.send(4, "stale-blocking")
			close(sendReturned)
		}()
	})
	select {
	case <-sendReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("the stale blocking send never returned")
	}
	waitIdle(t, a)

	want := []string{"first", "current"}
	if got := r.snapshot(); !equalStrings(got, want) {
		t.Errorf("delivered = %v, want %v (stale-generation messages dropped, current one kept)", got, want)
	}
}

// TestOutboxGoroutineExitsAndRestarts checks the sender goroutine's lifetime:
// it exits once the queue is empty (a.out.wg returns, running is false), and a
// later enqueue starts a new one that delivers. Then a root-context
// cancellation while a delivery is held: the messages still queued are not
// delivered, a blocked send returns, and the goroutine exits.
func TestOutboxGoroutineExitsAndRestarts(t *testing.T) {
	var mu sync.Mutex
	var got []string
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()
	a.sendFn = func(m any) { mu.Lock(); got = append(got, m.(string)); mu.Unlock() }
	gen := a.currentGen()

	a.send(gen, "one")
	waitIdle(t, a)
	a.out.mu.Lock()
	running, queued := a.out.running, len(a.out.queue)
	a.out.mu.Unlock()
	if running || queued != 0 {
		t.Errorf("after the last delivery running=%v queued=%d, want false and 0", running, queued)
	}

	a.send(gen, "two")
	waitIdle(t, a)
	mu.Lock()
	if !equalStrings(got, []string{"one", "two"}) {
		t.Errorf("delivered = %v, want [one two] (a second goroutine must start after the first exits)", got)
	}
	mu.Unlock()

	// Cancellation while a delivery is held.
	r := newOrderRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	a.rootCtx = ctx
	a.sendFn = r.send
	a.sendAsyncSequenceFenced(gen, []tea.Msg{"held"})
	sendReturned := make(chan struct{})
	select {
	case <-r.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the held message never reached sendFn")
	}
	a.sendAsyncSequenceFenced(gen, []tea.Msg{"queued"})
	go func() {
		a.send(gen, "blocked")
		close(sendReturned)
	}()
	cancel()
	// A blocked send returns on cancellation even though its delivery is not
	// finished (the held message is still inside sendFn).
	select {
	case <-sendReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("a blocked send did not return after the root context was cancelled")
	}
	close(r.release)
	waitIdle(t, a)
	if got := r.snapshot(); !equalStrings(got, []string{"held"}) {
		t.Errorf("delivered after cancel = %v, want only the held message", got)
	}
}

// outboxItemID names one message in the stress test: producer p's seq-th item.
type outboxItemID struct{ p, seq int }

// TestOutboxConcurrentProducersDeliverEachItemOnceInProducerOrder runs eight
// producers into one outbox at once, in lock-step rounds. Entry points are
// sendAsyncSequenceFenced and send, not RunTurn. In each round every producer
// queues a two-item sendAsyncSequenceFenced call and then makes a blocking send;
// the test waits for all eight blocking sends to return before it starts the
// next round. At the end it requires each producer's items to have arrived
// exactly once, in the order that producer made them (the order between
// producers is free).
//
// A producer's blocking send is its last item of the round and is queued behind
// its own async items, and the outbox is FIFO, so once every blocking send of a
// round has returned every item of the round has been delivered. The count check
// after each round therefore needs no wait. A count above the expected one is an
// item delivered twice; one below it is an item lost, or a blocking send that
// returned before the items queued ahead of it had been delivered.
//
// The lock-step is there for the stranding defect that
// TestOutboxEnqueueBetweenDrainEmptyCheckAndExitIsDelivered targets: an item left
// queued with no drain goroutine running. The next enqueue from any producer
// starts a drain goroutine and delivers it, so a strand persists only when it
// falls on the last enqueue of a round. That enqueue is always a blocking send,
// and no producer enqueues again until the round is over, so the stranded send
// never returns and the test fails after roundTimeout. Whether a run lands in that window is down to
// scheduling, so this test is not the one that pins the strand case; the
// targeted test is.
func TestOutboxConcurrentProducersDeliverEachItemOnceInProducerOrder(t *testing.T) {
	const producers, rounds = 8, 500
	// Each round a producer makes one two-item async call and one blocking send.
	const perRound = 3
	const roundTimeout = 5 * time.Second

	var mu sync.Mutex
	var delivered []outboxItemID
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()
	a.sendFn = func(m any) {
		mu.Lock()
		delivered = append(delivered, m.(outboxItemID))
		mu.Unlock()
	}
	gen := a.currentGen()

	for r := 0; r < rounds; r++ {
		seq := r * perRound
		var wg sync.WaitGroup
		start := make(chan struct{})
		for p := 0; p < producers; p++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				a.sendAsyncSequenceFenced(gen, []tea.Msg{outboxItemID{p, seq}, outboxItemID{p, seq + 1}})
				a.send(gen, outboxItemID{p, seq + 2})
			}()
		}
		close(start)

		finished := make(chan struct{})
		go func() { wg.Wait(); close(finished) }()
		select {
		case <-finished:
		case <-time.After(roundTimeout):
			t.Fatalf("round %d: a blocking send did not return within %v (its item was left in the outbox with no goroutine to deliver it)", r, roundTimeout)
		}
		mu.Lock()
		n := len(delivered)
		mu.Unlock()
		if want := (r + 1) * producers * perRound; n != want {
			t.Fatalf("after round %d: %d items delivered, want %d", r, n, want)
		}
	}
	waitIdle(t, a)

	mu.Lock()
	defer mu.Unlock()
	next := make([]int, producers)
	for i, id := range delivered {
		if id.seq != next[id.p] {
			t.Fatalf("delivery %d: producer %d item %d, want item %d (lost, repeated or reordered)", i, id.p, id.seq, next[id.p])
		}
		next[id.p]++
	}
	for p, n := range next {
		if n != rounds*perRound {
			t.Errorf("producer %d: %d items delivered, want %d", p, n, rounds*perRound)
		}
	}
}

// TestOutboxEnqueueBetweenDrainEmptyCheckAndExitIsDelivered checks that an item
// enqueued after drainOutbox has found the queue empty, but before the drain
// goroutine has exited, is still delivered. Entry point is
// sendAsyncSequenceFenced. The defect it targets is a drainOutbox that clears
// running under a separate lock acquisition from its empty check: an enqueue
// that takes the mutex between the two sees running still true, starts no drain
// goroutine, and its item is left queued with nothing to deliver it.
//
// That gap is one mutex release followed at once by a re-acquire, too narrow to
// hit by timing alone, so the test takes the outbox's own mutex (a.out.mu) and
// uses sync.Mutex's starvation mode to order the two waiters. Starvation mode is
// described in the sync package source, not its API: once a waiter has been
// blocked for more than 1ms, Unlock hands the mutex to the waiter at the front
// of the queue, and a goroutine that calls Lock then queues behind rather than
// taking it. Each probe:
//
//  1. "first" is queued; the drain goroutine delivers it and is held inside
//     sendFn.
//  2. The test locks a.out.mu and lets sendFn return, so the drain blocks in
//     the Lock that precedes its empty check.
//  3. After 1ms a producer goroutine calls sendAsyncSequenceFenced("second"),
//     which blocks in enqueue's Lock, queued behind the drain.
//  4. After 2ms more the test unlocks and at once locks again. The drain is
//     woken, finds the mutex held, and having waited more than 1ms switches it
//     to starvation mode.
//  5. After 1ms more the test unlocks. The mutex goes to the drain, which finds
//     the queue empty, and from the drain's Unlock straight to the producer.
//
// The correct drainOutbox clears running in the critical section where it found
// the queue empty, so the producer sees running false, starts a drain goroutine,
// and "second" is delivered. The split-lock version has not yet cleared running
// when the producer appends, so "second" stays queued and the probe fails after
// 2 seconds.
//
// The sleeps make that order likely, not certain: if the drain has not reached
// its Lock by step 3, or takes the mutex back at step 4, the producer appends
// before the empty check and the probe passes whatever drainOutbox does, which
// is why it runs probes times. No ordering of the steps can fail the correct
// drainOutbox. If a later Go release changed sync.Mutex's hand-off, the test
// would still pass on correct code and could stop catching the defect.
func TestOutboxEnqueueBetweenDrainEmptyCheckAndExitIsDelivered(t *testing.T) {
	const probes = 5
	a := testApp(t, "repo-small", newCollector(-1))
	defer a.Close()
	var hold atomic.Bool
	entered := make(chan struct{})
	proceed := make(chan struct{})
	delivered := make(chan string, 4)
	a.sendFn = func(m any) {
		if hold.CompareAndSwap(true, false) {
			entered <- struct{}{}
			<-proceed
		}
		delivered <- m.(string)
	}
	gen := a.currentGen()

	for i := 0; i < probes; i++ {
		// Step 1.
		hold.Store(true)
		a.sendAsyncSequenceFenced(gen, []tea.Msg{"first"})
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatalf("probe %d: the first message never reached sendFn", i)
		}
		// Step 2.
		a.out.mu.Lock()
		proceed <- struct{}{}
		// Step 3.
		time.Sleep(time.Millisecond)
		go a.sendAsyncSequenceFenced(gen, []tea.Msg{"second"})
		// Step 4.
		time.Sleep(2 * time.Millisecond)
		a.out.mu.Unlock()
		a.out.mu.Lock()
		// Step 5.
		time.Sleep(time.Millisecond)
		a.out.mu.Unlock()

		for _, want := range []string{"first", "second"} {
			select {
			case got := <-delivered:
				if got != want {
					t.Fatalf("probe %d: delivered %q, want %q", i, got, want)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("probe %d: %q was never delivered: it was left in the outbox with no goroutine to deliver it", i, want)
			}
		}
		waitIdle(t, a)
	}
}
