package anthropic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/provider"
)

// cancelOnRead wraps a response body so the first Read cancels ctx and fails,
// as a real body does when its request is cancelled mid-read.
type cancelOnRead struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b cancelOnRead) Read([]byte) (int, error) {
	b.cancel()
	return 0, context.Canceled
}

// cancelBodyTransport returns real responses whose bodies cancel on first read.
type cancelBodyTransport struct{ cancel context.CancelFunc }

func (tr cancelBodyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err == nil {
		resp.Body = cancelOnRead{ReadCloser: resp.Body, cancel: tr.cancel}
	}
	return resp, err
}

// TestClientCancelledReadingErrorBody: the cancel lands during the error-body
// read itself, deterministically, for every status class.
func TestClientCancelledReadingErrorBody(t *testing.T) {
	for _, code := range []int{400, 401, 302, 404, 429, 503, 204} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rec, sl := &recorder{}, &sleeps{}
			srv := serve(t, rec, status(code, `{"error":{"type":"x"}}`))
			c := newClient(t, srv, "opencode", sl, func(o *Options) {
				o.HTTPClient = &http.Client{Transport: cancelBodyTransport{cancel: cancel}}
			})
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

// TestClientSleepFailureIsReported: a non-cancellation error from Sleep is a
// reported failure, not a silent return.
func TestClientSleepFailureIsReported(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, status(503, "busy"))
	boom := errors.New("clock broke")
	c := newClient(t, srv, "opencode", &sleeps{}, func(o *Options) {
		o.Sleep = func(context.Context, time.Duration) error { return boom }
	})
	evs, err := run(t, c)
	if pe := failure(t, evs, err, KindUnavailable); !errors.Is(pe, boom) || rec.count() != 1 {
		t.Fatalf("err=%v requests=%d", err, rec.count())
	}
}

// TestClientUnavailableCauseIsLatest: a network cause from an early attempt is
// not reported against a later 5xx.
func TestClientUnavailableCauseIsLatest(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, hangup, status(503, "a"), status(503, "b"), status(503, "c"))
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	if pe := failure(t, evs, err, KindUnavailable); pe.Status != 503 || pe.Err != nil {
		t.Fatalf("status=%d err=%v", pe.Status, pe.Err)
	}
}
