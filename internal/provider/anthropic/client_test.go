package anthropic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/telemetry"
)

var _ provider.Provider = (*Client)(nil)

const testKey = "sk-test-SECRET-0123456789"

// recorder captures what a test server received.
type recorder struct {
	mu     sync.Mutex
	reqs   []*http.Request
	bodies [][]byte
}

func (r *recorder) add(req *http.Request) {
	b, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
	r.bodies = append(r.bodies, b)
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.reqs)
}

func (r *recorder) get(i int) (*http.Request, []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reqs[i], r.bodies[i]
}

// serve answers the nth request with the nth handler; any extra request gets 418.
func serve(t *testing.T, rec *recorder, steps ...http.HandlerFunc) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		mu.Lock()
		i := n
		n++
		mu.Unlock()
		if i >= len(steps) {
			http.Error(w, "unexpected request", http.StatusTeapot)
			return
		}
		steps[i](w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func status(code int, body string, hdr ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i+1 < len(hdr); i += 2 {
			w.Header().Set(hdr[i], hdr[i+1])
		}
		w.WriteHeader(code)
		_, _ = io.WriteString(w, body)
	}
}

func sse(b []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(b)
	}
}

// hangup closes the connection without a response.
func hangup(w http.ResponseWriter, _ *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		panic("response writer cannot hijack")
	}
	conn, _, err := hj.Hijack()
	if err == nil {
		_ = conn.Close()
	}
}

func fixtureSSE(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtures, "minimax-m3/S1-text.sse"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sleeps records every wait instead of waiting.
type sleeps struct {
	mu sync.Mutex
	d  []time.Duration
}

func (s *sleeps) sleep(_ context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d = append(s.d, d)
	return nil
}

func (s *sleeps) get() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Duration(nil), s.d...)
}

func newClient(t *testing.T, srv *httptest.Server, endpoint string, sl *sleeps, mutate ...func(*Options)) *Client {
	t.Helper()
	o := Options{
		Endpoint: endpoint, BaseURL: srv.URL + "/v1", Auth: "x-api-key", APIKey: testKey,
		KeyEnv: "OPENCODE_API_KEY", Version: "1.2.3", SessionID: "sess-abc", Sleep: sl.sleep,
	}
	for _, m := range mutate {
		m(&o)
	}
	c, err := New(o)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func simpleReq() provider.Request {
	return provider.Request{Model: "minimax-m3", MaxTokens: 16, Messages: []provider.Message{userText("ping")}}
}

func run(t *testing.T, c *Client) ([]provider.StreamEvent, error) {
	t.Helper()
	var evs []provider.StreamEvent
	err := c.Stream(context.Background(), simpleReq(), func(e provider.StreamEvent) { evs = append(evs, e) })
	return evs, err
}

// failure asserts err is a *Error of kind k carried by exactly one trailing
// EventError, and that the key appears nowhere in it.
func failure(t *testing.T, evs []provider.StreamEvent, err error, k ErrorKind) *Error {
	t.Helper()
	var pe *Error
	if !errors.As(err, &pe) || pe.Kind != k {
		t.Fatalf("err = %v, want kind %s", err, k)
	}
	n := 0
	for _, e := range evs {
		if e.Type == provider.EventError {
			n++
		}
	}
	if n != 1 || evs[len(evs)-1].Type != provider.EventError || evs[len(evs)-1].Err != err {
		t.Fatalf("events = %s, want exactly one trailing EventError carrying err", types(evs))
	}
	if strings.Contains(err.Error(), testKey) {
		t.Fatal("error text contains the key")
	}
	return pe
}

// TestClientHeadersOpencode: the opencode endpoint gets its key, version,
// identity and session headers, on POST <base>/messages.
func TestClientHeadersOpencode(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, sse(fixtureSSE(t)))
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	if err != nil || types(evs) != "EventTextDelta,EventMessageDone" {
		t.Fatalf("err=%v events=%s", err, types(evs))
	}
	r, body := rec.get(0)
	want := map[string]string{
		"X-Api-Key": testKey, "Anthropic-Version": "2023-06-01", "User-Agent": "kirsch/1.2.3",
		"X-Opencode-Session": "sess-abc", "Content-Type": "application/json", "Accept": "text/event-stream",
	}
	for k, v := range want {
		if got := r.Header.Get(k); got != v {
			t.Errorf("header %s wrong (got %d bytes)", k, len(got))
		}
	}
	if r.Header.Get("Authorization") != "" || r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
		t.Errorf("method=%s path=%s authorization set=%v", r.Method, r.URL.Path, r.Header.Get("Authorization") != "")
	}
	if b := asJSON(t, body); b["model"] != "minimax-m3" || b["stream"] != true {
		t.Errorf("body = %v", b)
	}
}

// TestClientHeadersAnthropicBearer: bearer auth replaces x-api-key, and no
// opencode session header is sent to another endpoint.
func TestClientHeadersAnthropicBearer(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, sse(fixtureSSE(t)))
	c := newClient(t, srv, "anthropic", &sleeps{}, func(o *Options) { o.Auth = "bearer"; o.SessionID = "" })
	if _, err := run(t, c); err != nil {
		t.Fatal(err)
	}
	r, _ := rec.get(0)
	if r.Header.Get("Authorization") != "Bearer "+testKey || r.Header.Get("X-Api-Key") != "" ||
		r.Header.Get("X-Opencode-Session") != "" || r.Header.Get("Anthropic-Version") != "2023-06-01" ||
		r.Header.Get("User-Agent") != "kirsch/1.2.3" {
		t.Fatal("unexpected auth, session, version or user-agent header")
	}
}

// TestClientRetryMatrix: request counts, waits and error kinds per milestone
// Task 8.
func TestClientRetryMatrix(t *testing.T) {
	ok := sse(fixtureSSE(t))
	s500, s503 := status(500, "boom"), status(503, "busy")
	r429 := func(after string) http.HandlerFunc { return status(429, "{}", "Retry-After", after) }
	cases := []struct {
		name   string
		steps  []http.HandlerFunc
		reqs   int
		kind   ErrorKind // "" means success
		sleeps []time.Duration
	}{
		{"5xx then ok", []http.HandlerFunc{s500, s503, ok}, 3, "", []time.Duration{500 * time.Millisecond, time.Second}},
		{"5xx exhausted", []http.HandlerFunc{s500, s500, s500, s500}, 4, KindUnavailable, []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}},
		{"429 retry-after then ok", []http.HandlerFunc{r429("7"), ok}, 2, "", []time.Duration{7 * time.Second}},
		{"429 without retry-after", []http.HandlerFunc{status(429, "{}"), ok}, 2, "", []time.Duration{500 * time.Millisecond}},
		{"429 exhausted", []http.HandlerFunc{r429("1"), r429("1"), r429("1"), r429("1")}, 4, KindRateLimited, []time.Duration{time.Second, time.Second, time.Second}},
		{"429 retry-after too long", []http.HandlerFunc{r429("3600")}, 1, KindRateLimited, nil},
		{"401", []http.HandlerFunc{status(401, `{"error":{"type":"authentication_error"}}`)}, 1, KindAuth, nil},
		{"403", []http.HandlerFunc{status(403, "{}")}, 1, KindAuth, nil},
		{"400", []http.HandlerFunc{status(400, `{"error":{"type":"invalid_request_error","message":"bad"}}`)}, 1, KindBadRequest, nil},
		{"400 missing session", []http.HandlerFunc{status(400, `{"error":{"type":"MissingSessionID","message":"x"}}`)}, 1, KindConfig, nil},
		{"404", []http.HandlerFunc{status(404, "nope")}, 1, KindHTTP, nil},
		{"302", []http.HandlerFunc{status(302, "", "Location", "https://user:pw@evil.example:8443/steal?k=1")}, 1, KindRedirect, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec, sl := &recorder{}, &sleeps{}
			srv := serve(t, rec, c.steps...)
			evs, err := run(t, newClient(t, srv, "opencode", sl))
			if rec.count() != c.reqs {
				t.Errorf("requests = %d, want %d", rec.count(), c.reqs)
			}
			if got := sl.get(); fmt.Sprint(got) != fmt.Sprint(c.sleeps) {
				t.Errorf("sleeps = %v, want %v", got, c.sleeps)
			}
			if c.kind == "" {
				if err != nil || types(evs) != "EventTextDelta,EventMessageDone" {
					t.Fatalf("err=%v events=%s", err, types(evs))
				}
				return
			}
			if pe := failure(t, evs, err, c.kind); pe.Attempts != c.reqs {
				t.Errorf("Attempts = %d, want %d", pe.Attempts, c.reqs)
			}
		})
	}
}

// TestClientErrorMessages: what the user reads names the fix, and never the
// redirect target's userinfo, port or path.
func TestClientErrorMessages(t *testing.T) {
	for _, c := range []struct {
		step     http.HandlerFunc
		want     []string
		mustNot  []string
		wantCode int
	}{
		{status(401, "{}"), []string{`provider "opencode"`, "401", "KIRSCH_OPENCODE_API_KEY", "OPENCODE_API_KEY"}, nil, 401},
		{status(302, "", "Location", "https://user:pw@evil.example:8443/steal?k=1"), []string{"302", `"evil.example"`}, []string{"user", "pw@", "steal", "8443"}, 302},
		{status(400, `{"error":{"type":"MissingSessionID"}}`), []string{"MissingSessionID"}, nil, 400},
	} {
		rec := &recorder{}
		srv := serve(t, rec, c.step)
		_, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
		var pe *Error
		if !errors.As(err, &pe) || pe.Status != c.wantCode {
			t.Fatalf("err = %v", err)
		}
		for _, w := range c.want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("%q lacks %q", err, w)
			}
		}
		for _, n := range c.mustNot {
			if strings.Contains(err.Error(), n) {
				t.Errorf("%q contains %q", err, n)
			}
		}
	}
}

// TestClientRedirectNotFollowed: the redirect target never receives a request,
// so the key never reaches it.
func TestClientRedirectNotFollowed(t *testing.T) {
	target := &recorder{}
	evil := serve(t, target, sse(fixtureSSE(t)))
	rec := &recorder{}
	srv := serve(t, rec, status(307, "", "Location", evil.URL+"/v1/messages"))
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	if pe := failure(t, evs, err, KindRedirect); pe.Status != 307 {
		t.Fatalf("status = %d", pe.Status)
	}
	if target.count() != 0 || rec.count() != 1 {
		t.Fatalf("redirect target got %d requests, origin %d", target.count(), rec.count())
	}
}

// TestNewDoesNotMutateHTTPClient: the caller's client keeps its own policy.
func TestNewDoesNotMutateHTTPClient(t *testing.T) {
	hc := &http.Client{}
	rec := &recorder{}
	srv := serve(t, rec, sse(fixtureSSE(t)))
	newClient(t, srv, "opencode", &sleeps{}, func(o *Options) { o.HTTPClient = hc })
	if hc.CheckRedirect != nil {
		t.Fatal("caller's http.Client was modified")
	}
}

// TestClientBadRequestLogsBodyNotKey: a 400 puts the request and response in
// the debug log; the key goes nowhere.
func TestClientBadRequestLogsBodyNotKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	log, err := telemetry.New(telemetry.Options{Path: path, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	srv := serve(t, rec, status(400, `{"error":{"type":"invalid_request_error","message":"bad"}}`))
	c := newClient(t, srv, "opencode", &sleeps{}, func(o *Options) { o.Log = log })
	if _, err := run(t, c); err == nil {
		t.Fatal("no error")
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `\"model\":\"minimax-m3\"`) || !strings.Contains(string(b), "invalid_request_error") {
		t.Fatalf("debug log lacks the request or response body:\n%s", b)
	}
	if strings.Contains(string(b), testKey) {
		t.Fatal("debug log contains the key")
	}
}

// TestClientStreamErrorWrapped: a failure inside a 200 stream is a KindStream
// *Error that still unwraps to the decoder's error, and is not retried.
func TestClientStreamErrorWrapped(t *testing.T) {
	rec := &recorder{}
	body := sseHead + block(`{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`)
	srv := serve(t, rec, sse([]byte(body)))
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	pe := failure(t, evs, err, KindStream)
	var se *StreamError
	if !errors.As(pe, &se) || se.Type != "overloaded_error" || rec.count() != 1 {
		t.Fatalf("err = %v, requests %d", err, rec.count())
	}
}

// TestClientNetworkErrorRetried: a dropped connection is retried; four drops
// are KindUnavailable.
func TestClientNetworkErrorRetried(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, hangup, hangup, sse(fixtureSSE(t)))
	evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}))
	if err != nil || types(evs) != "EventTextDelta,EventMessageDone" || rec.count() != 3 {
		t.Fatalf("err=%v events=%s requests=%d", err, types(evs), rec.count())
	}
	rec2 := &recorder{}
	srv2 := serve(t, rec2, hangup, hangup, hangup, hangup)
	evs, err = run(t, newClient(t, srv2, "opencode", &sleeps{}))
	failure(t, evs, err, KindUnavailable)
	if rec2.count() != 4 {
		t.Fatalf("requests = %d", rec2.count())
	}
}

// TestClientLockoutHook: a 429 the hook recognises is not retried.
func TestClientLockoutHook(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, status(429, `{"error":{"type":"usage_limit"}}`, "Retry-After", "1"))
	c := newClient(t, srv, "opencode", &sleeps{}, func(o *Options) {
		o.Lockout = func(st int, _ http.Header, body []byte) bool {
			return st == http.StatusTooManyRequests && strings.Contains(string(body), "usage_limit")
		}
	})
	evs, err := run(t, c)
	failure(t, evs, err, KindLockout)
	if rec.count() != 1 {
		t.Fatalf("requests = %d", rec.count())
	}
}

// TestClientCancelledDuringBackoff: cancellation while waiting returns
// ctx.Err() with no event and no further request.
func TestClientCancelledDuringBackoff(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, status(503, "busy"), status(503, "busy"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := newClient(t, srv, "opencode", &sleeps{}, func(o *Options) {
		o.Sleep = func(ctx context.Context, _ time.Duration) error { cancel(); return ctx.Err() }
	})
	var evs []provider.StreamEvent
	err := c.Stream(ctx, simpleReq(), func(e provider.StreamEvent) { evs = append(evs, e) })
	if !errors.Is(err, context.Canceled) || len(evs) != 0 || rec.count() != 1 {
		t.Fatalf("err=%v events=%d requests=%d", err, len(evs), rec.count())
	}
}

// TestClientRetryAfterDate: an HTTP-date Retry-After waits until that time.
func TestClientRetryAfterDate(t *testing.T) {
	rec, sl := &recorder{}, &sleeps{}
	when := time.Now().Add(5 * time.Second).UTC().Format(http.TimeFormat)
	srv := serve(t, rec, status(429, "{}", "Retry-After", when), sse(fixtureSSE(t)))
	if _, err := run(t, newClient(t, srv, "opencode", sl)); err != nil {
		t.Fatal(err)
	}
	if d := sl.get(); len(d) != 1 || d[0] < 3*time.Second || d[0] > 5*time.Second {
		t.Fatalf("sleeps = %v", d)
	}
}

// TestNewRefuses: unusable options never produce a client, and no refusal
// echoes the key.
func TestNewRefuses(t *testing.T) {
	base := Options{
		Endpoint: "opencode", BaseURL: "https://opencode.ai/zen/go/v1", Auth: "x-api-key",
		APIKey: testKey, KeyEnv: "OPENCODE_API_KEY", Version: "1", SessionID: "s",
	}
	for name, mut := range map[string]func(*Options){
		"http non-loopback":   func(o *Options) { o.BaseURL = "http://example.com/v1" },
		"userinfo":            func(o *Options) { o.BaseURL = "https://u:p@example.com/v1" },
		"bad auth":            func(o *Options) { o.Auth = "basic" },
		"no key":              func(o *Options) { o.APIKey = "" },
		"opencode no session": func(o *Options) { o.SessionID = "" },
		"no version":          func(o *Options) { o.Version = "" },
	} {
		o := base
		mut(&o)
		c, err := New(o)
		if err == nil || c != nil {
			t.Errorf("%s: accepted", name)
		}
		if err != nil && strings.Contains(err.Error(), testKey) {
			t.Errorf("%s: error contains the key", name)
		}
	}
	if _, err := New(base); err != nil {
		t.Fatalf("valid options refused: %v", err)
	}
}
