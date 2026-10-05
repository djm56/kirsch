package anthropic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/djm56/kirsch/internal/provider"
)

// TestClientCancelledDuringErrorBody: a cancel while an error body is being
// read returns ctx.Err() with no event, no retry and no wait, whatever the status.
func TestClientCancelledDuringErrorBody(t *testing.T) {
	for _, code := range []int{401, 400, 503, 429} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			release := make(chan struct{})
			rec, sl := &recorder{}, &sleeps{}
			srv := serve(t, rec, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "3600")
				w.WriteHeader(code)
				_, _ = io.WriteString(w, `{"error":`)
				w.(http.Flusher).Flush()
				cancel()
				select {
				case <-r.Context().Done():
				case <-release:
				}
			})
			defer close(release)
			c := newClient(t, srv, "opencode", sl)
			var evs []provider.StreamEvent
			err := c.Stream(ctx, simpleReq(), func(e provider.StreamEvent) { evs = append(evs, e) })
			var pe *Error
			if !errors.Is(err, context.Canceled) || errors.As(err, &pe) || len(evs) != 0 ||
				rec.count() != 1 || len(sl.get()) != 0 {
				t.Fatalf("err=%v events=%d requests=%d sleeps=%v", err, len(evs), rec.count(), sl.get())
			}
		})
	}
}

// TestClientLockoutCalledOnce: the Lockout verdict is taken once per 429 and
// used for both the retry decision and the error kind.
func TestClientLockoutCalledOnce(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, status(429, "{}"))
	calls := 0
	c := newClient(t, srv, "opencode", &sleeps{}, func(o *Options) {
		o.Lockout = func(int, http.Header, []byte) bool { calls++; return calls == 1 }
	})
	evs, err := run(t, c)
	failure(t, evs, err, KindLockout)
	if calls != 1 || rec.count() != 1 {
		t.Fatalf("lockout calls=%d requests=%d", calls, rec.count())
	}
}

// TestClientUnavailableKeepsCause: after the last dropped connection the
// error still carries the network cause.
func TestClientUnavailableKeepsCause(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, hangup, hangup, hangup, hangup)
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	if pe := failure(t, evs, err, KindUnavailable); pe.Err == nil {
		t.Fatal("KindUnavailable lost the network cause")
	}
}
