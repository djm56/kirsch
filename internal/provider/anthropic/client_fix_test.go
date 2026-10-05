package anthropic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/djm56/kirsch/internal/provider"
)

// TestClientStreamsIncrementally: an event reaches the caller before the
// server has finished the response.
func TestClientStreamsIncrementally(t *testing.T) {
	gate, first := make(chan struct{}), make(chan struct{})
	rec := &recorder{}
	srv := serve(t, rec, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sseHead+
			block(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)+
			block(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"early"}}`))
		w.(http.Flusher).Flush()
		<-gate
		_, _ = io.WriteString(w, block(`{"type":"content_block_stop","index":0}`)+block(`{"type":"message_stop"}`))
	})
	c := newClient(t, srv, "opencode", &sleeps{})
	var once sync.Once
	errc := make(chan error, 1)
	go func() {
		errc <- c.Stream(context.Background(), simpleReq(), func(e provider.StreamEvent) {
			if e.Type == provider.EventTextDelta {
				once.Do(func() { close(first) })
			}
		})
	}()
	select {
	case <-first:
	case <-time.After(5 * time.Second):
		close(gate)
		t.Fatal("no event before the server finished: the response is buffered")
	}
	close(gate)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
}

// TestClientLargeResponse: a response far beyond the error-body limit decodes whole.
func TestClientLargeResponse(t *testing.T) {
	big := strings.Repeat("x", 200<<10)
	body := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"`+big+`"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + block(`{"type":"message_stop"}`)
	rec := &recorder{}
	srv := serve(t, rec, sse([]byte(body)))
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	if err != nil || types(evs) != "EventTextDelta,EventMessageDone" || len(evs[0].Text) != len(big) {
		t.Fatalf("err=%v events=%s", err, types(evs))
	}
}

// TestClientCancelledDuringRequest: a cancel while waiting for the response
// returns ctx.Err() as is, with no event and no retry.
func TestClientCancelledDuringRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release := make(chan struct{})
	rec := &recorder{}
	srv := serve(t, rec, func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	defer close(release)
	c := newClient(t, srv, "opencode", &sleeps{})
	var evs []provider.StreamEvent
	err := c.Stream(ctx, simpleReq(), func(e provider.StreamEvent) { evs = append(evs, e) })
	var pe *Error
	if !errors.Is(err, context.Canceled) || errors.As(err, &pe) || len(evs) != 0 || rec.count() != 1 {
		t.Fatalf("err=%v events=%d requests=%d", err, len(evs), rec.count())
	}
}

// TestClientCancelledMidStream: a cancel during the stream returns ctx.Err()
// as is, never wrapped as a stream failure.
func TestClientCancelledMidStream(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, sse(fixtureSSE(t)))
	c := newClient(t, srv, "opencode", &sleeps{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var evs []provider.StreamEvent
	err := c.Stream(ctx, simpleReq(), func(e provider.StreamEvent) {
		evs = append(evs, e)
		if e.Type == provider.EventTextDelta {
			cancel()
		}
	})
	var pe *Error
	if !errors.Is(err, context.Canceled) || errors.As(err, &pe) || types(evs) != "EventTextDelta" {
		t.Fatalf("err=%v events=%s", err, types(evs))
	}
}

// TestClientRetryAfterEdges: only plain integer seconds or an HTTP-date are
// honoured; anything else falls back to backoff; anything too long fails now.
func TestClientRetryAfterEdges(t *testing.T) {
	ok := sse(fixtureSSE(t))
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	far := time.Now().Add(2 * time.Hour).UTC().Format(http.TimeFormat)
	for _, c := range []struct {
		name   string
		after  string
		reqs   int
		kind   ErrorKind
		sleeps []time.Duration
	}{
		{"zero", "0", 2, "", []time.Duration{0}},
		{"negative", "-5", 2, "", []time.Duration{500 * time.Millisecond}},
		{"unit suffix", "5m", 2, "", []time.Duration{500 * time.Millisecond}},
		{"fraction", "1.5", 2, "", []time.Duration{500 * time.Millisecond}},
		{"overflow", "99999999999999999999", 1, KindRateLimited, nil},
		{"past date", past, 2, "", []time.Duration{0}},
		{"far date", far, 1, KindRateLimited, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			rec, sl := &recorder{}, &sleeps{}
			srv := serve(t, rec, status(429, "{}", "Retry-After", c.after), ok)
			evs, err := run(t, newClient(t, srv, "opencode", sl))
			if rec.count() != c.reqs || fmt.Sprint(sl.get()) != fmt.Sprint(c.sleeps) {
				t.Errorf("requests=%d sleeps=%v, want %d %v", rec.count(), sl.get(), c.reqs, c.sleeps)
			}
			if c.kind == "" {
				if err != nil {
					t.Fatalf("err = %v", err)
				}
				return
			}
			failure(t, evs, err, c.kind)
		})
	}
}

// TestClientTLSErrorNotRetried: a certificate failure is not transient.
func TestClientTLSErrorNotRetried(t *testing.T) {
	rec := &recorder{}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { rec.add(r) }))
	t.Cleanup(srv.Close)
	sl := &sleeps{}
	c, err := New(Options{
		Endpoint: "opencode", BaseURL: srv.URL + "/v1", Auth: "x-api-key", APIKey: testKey,
		KeyEnv: "OPENCODE_API_KEY", Version: "1.2.3", SessionID: "s", Sleep: sl.sleep,
	})
	if err != nil {
		t.Fatal(err)
	}
	evs, err := run(t, c)
	pe := failure(t, evs, err, KindUnavailable)
	if pe.Attempts != 1 || len(sl.get()) != 0 || rec.count() != 0 {
		t.Fatalf("attempts=%d sleeps=%v requests=%d", pe.Attempts, sl.get(), rec.count())
	}
}

// TestClientRedirectLocationHost: a relative Location names the origin host,
// and a huge host is clipped.
func TestClientRedirectLocationHost(t *testing.T) {
	long := strings.Repeat("a", 300) + ".example"
	for _, c := range []struct{ loc, want string }{
		{"/v1/other", `to host "127.0.0.1"`},
		{"https://" + long + "/", `to host "` + strings.Repeat("a", 100)},
	} {
		rec := &recorder{}
		srv := serve(t, rec, status(302, "", "Location", c.loc))
		_, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
		if err == nil || !strings.Contains(err.Error(), c.want) || len(err.Error()) > 400 {
			t.Errorf("Location %.40q: err (%d bytes) = %.200v", c.loc, len(fmt.Sprint(err)), err)
		}
	}
}

// TestClipKeepsUTF8: clipping never splits a rune.
func TestClipKeepsUTF8(t *testing.T) {
	s := strings.Repeat("é", 40)
	got := clip(s, 33)
	if !utf8.ValidString(got) || !strings.HasSuffix(got, "…") || len(got) > 33+len("…") {
		t.Fatalf("clip = %q", got)
	}
}

// TestDecodeClipRemainingSites: the error-type and invalid-JSON tool-id echo
// sites are clipped too.
func TestDecodeClipRemainingSites(t *testing.T) {
	long := strings.Repeat("t", 5000)
	_, err := decodeString(t, sseHead+block(`{"type":"error","error":{"type":"`+long+`","message":"m"}}`))
	if err == nil || len(err.Error()) > 700 {
		t.Fatalf("error type: err (%d bytes) = %.200v", len(fmt.Sprint(err)), err)
	}
	sse := sseHead +
		block(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"`+long+`","name":"n","input":{}}}`) +
		block(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"a\":"}}`) +
		block(`{"type":"content_block_stop","index":0}`) + sseTail
	_, err = decodeString(t, sse)
	if err == nil || len(err.Error()) > 200 || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("tool id: err (%d bytes) = %.200v", len(fmt.Sprint(err)), err)
	}
}
