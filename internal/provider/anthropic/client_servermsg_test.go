package anthropic

import (
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// errBody builds an Anthropic-shaped error body around msg (already JSON-safe).
func errBody(msg string) string {
	return `{"type":"error","error":{"type":"api_error","message":"` + msg + `"}}`
}

// helperClient returns a Client holding testKey, for calling serverMessage.
func helperClient(t *testing.T) *Client {
	t.Helper()
	return &Client{key: testKey}
}

// TestServerMessage drives the helper directly: shapes, sanitising, cap, key.
func TestServerMessage(t *testing.T) {
	long := strings.Repeat("a", 500)
	// "a" then 200 two-byte runes: byte 300 is the second byte of the 150th "é",
	// so a cut at the cap lands mid-rune and must back up to byte 299.
	multi := "a" + strings.Repeat("é", 200)
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			"real 400", `{"type":"error","error":{"type":"invalid_request_error","message":"Upstream request failed: [invalid_request_error] \"thinking.type.between_tools\" is not supported for this model."}}`,
			`Upstream request failed: [invalid_request_error] "thinking.type.between_tools" is not supported for this model.`,
		},
		{
			"real api_error", `{"type":"error","error":{"type":"api_error","message":"Upstream request failed: Endpoint is unavailable."}}`,
			"Upstream request failed: Endpoint is unavailable.",
		},
		{"empty body", ``, ""},
		{"non-JSON", `<html>bad gateway</html>`, ""},
		{"different shape", `{"error":"bad request"}`, ""},
		{"no top-level type", `{"error":{"type":"invalid_request_error","message":"bad"}}`, ""},
		{"top-level type not error", `{"type":"message","error":{"message":"bad"}}`, ""},
		{"missing message", `{"type":"error","error":{"type":"api_error"}}`, ""},
		{"error is a string", `{"type":"error","error":"boom"}`, ""},
		{"ESC sequence", errBody(`a\u001b[31mred\u001b[0m b`), "a[31mred[0m b"},
		{"newline and tabs collapse", errBody(`one\n\n  two\t\tthree\r\nfour`), "one two three four"},
		{"DEL dropped", errBody("x\u007fy"), "xy"},
		{"C1 dropped", errBody("x\u009by"), "xy"},
		{"NEL (U+0085) becomes a space", errBody(`a\u0085b`), "a b"},
		{"bidi override U+202E dropped", errBody(`a\u202eb`), "ab"},
		{"bidi isolate U+2066 dropped", errBody(`a\u2066b`), "ab"},
		{"zero-width space U+200B dropped", errBody(`a\u200bb`), "ab"},
		{"leading whitespace trimmed", errBody(`\n  hi`), "hi"},
		{"trailing whitespace trimmed", errBody(`hi  \n`), "hi"},
		{"only control chars", errBody(`\u001b\u007f`), ""},
		{"key in message", errBody("bad key " + testKey + " here"), ""},
		{"key split by control char", errBody("bad key " + testKey[:6] + `\u001b` + testKey[6:]), ""},
		// The raw message holds U+200B inside the key, so only the check on the
		// sanitised text can catch it.
		{"key split by U+200B", errBody("bad key " + testKey[:6] + `\u200b` + testKey[6:]), ""},
		{"long message capped", errBody(long), strings.Repeat("a", maxServerMessage) + "…"},
		{"exactly at cap kept", errBody(strings.Repeat("a", maxServerMessage)), strings.Repeat("a", maxServerMessage)},
	}
	c := helperClient(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.serverMessage([]byte(tc.body)); got != tc.want {
				t.Fatalf("serverMessage = %q, want %q", got, tc.want)
			}
		})
	}
	t.Run("cap cut mid 2-byte rune", func(t *testing.T) {
		// multi is 401 bytes and the cap falls inside the 150th "é": the cut backs
		// up one byte, leaving "a" and 149 "é" (299 bytes) before the ellipsis.
		got := c.serverMessage([]byte(errBody(multi)))
		if want := "a" + strings.Repeat("é", 149) + "…"; got != want || !utf8.ValidString(got) {
			t.Fatalf("got %d bytes (valid=%v), want %d bytes %q", len(got), utf8.ValidString(got), len(want), want)
		}
	})
	t.Run("cap lands exactly on a 2-byte rune start", func(t *testing.T) {
		// 298 ASCII bytes, then two "é": the first fills bytes 298-299 and the
		// second starts at byte 300, the cap. Nothing is cut, so the first "é" stays.
		got := c.serverMessage([]byte(errBody(strings.Repeat("a", maxServerMessage-2) + "éé")))
		if want := strings.Repeat("a", maxServerMessage-2) + "é…"; got != want {
			t.Fatalf("got %q (%d bytes), want %q (%d bytes)", got, len(got), want, len(want))
		}
	})
	t.Run("cap lands exactly on a 3-byte rune start", func(t *testing.T) {
		// 297 ASCII bytes, then two "€": the first fills bytes 297-299 and the
		// second starts at byte 300, the cap.
		got := c.serverMessage([]byte(errBody(strings.Repeat("a", maxServerMessage-3) + "€€")))
		if want := strings.Repeat("a", maxServerMessage-3) + "€…"; got != want {
			t.Fatalf("got %q (%d bytes), want %q (%d bytes)", got, len(got), want, len(want))
		}
	})
	t.Run("key with internal whitespace", func(t *testing.T) {
		// The sanitiser collapses "ab  cd" to "ab cd", so only the check on the raw
		// message can see this key echoed.
		ws := &Client{key: "ab  cd"}
		if got := ws.serverMessage([]byte(errBody("echo ab  cd end"))); got != "" {
			t.Fatalf("serverMessage = %q, want empty", got)
		}
		if got := ws.serverMessage([]byte(errBody("echo ab cd end"))); got != "echo ab cd end" {
			t.Fatalf("control: serverMessage = %q, want the message kept", got)
		}
	})
	t.Run("cap cut inside a 3-byte rune", func(t *testing.T) {
		// 299 bytes of ASCII then a 3-byte rune straddling the cap.
		got := c.serverMessage([]byte(errBody(strings.Repeat("a", maxServerMessage-1) + "€€")))
		if !utf8.ValidString(got) || got != strings.Repeat("a", maxServerMessage-1)+"…" {
			t.Fatalf("got %q", got)
		}
	})
	// Cuts deeper inside a rune: clip must back up over every continuation byte,
	// not just one. pad ASCII bytes come first, so the rune starts at byte pad
	// and the cap (byte maxServerMessage) is its (maxServerMessage-pad+1)th byte.
	deep := []struct {
		name string
		pad  int
		r    string
	}{
		{"cap at third byte of a 3-byte rune", maxServerMessage - 2, "€"},
		{"cap at second byte of a 4-byte rune", maxServerMessage - 1, "😀"},
		{"cap at third byte of a 4-byte rune", maxServerMessage - 2, "😀"},
		{"cap at fourth byte of a 4-byte rune", maxServerMessage - 3, "😀"},
	}
	for _, d := range deep {
		t.Run(d.name, func(t *testing.T) {
			got := c.serverMessage([]byte(errBody(strings.Repeat("a", d.pad) + d.r + "z")))
			if want := strings.Repeat("a", d.pad) + "…"; got != want || !utf8.ValidString(got) {
				t.Fatalf("got %q (%d bytes, valid=%v), want %d ASCII bytes then the ellipsis", got, len(got), utf8.ValidString(got), d.pad)
			}
		})
	}
	t.Run("one byte over the cap is clipped", func(t *testing.T) {
		got := c.serverMessage([]byte(errBody(strings.Repeat("a", maxServerMessage+1))))
		if want := strings.Repeat("a", maxServerMessage) + "…"; got != want {
			t.Fatalf("got %d bytes, want %d bytes %q", len(got), len(want), want)
		}
	})
}

// TestClientServerMessageReachesUser goes through Stream and an httptest
// server: what the operator reads for each status.
func TestClientServerMessageReachesUser(t *testing.T) {
	real400 := `{"type":"error","error":{"type":"invalid_request_error","message":"Upstream request failed: [invalid_request_error] \"thinking.type.between_tools\" is not supported for this model."}}`
	real5xx := `{"type":"error","error":{"type":"api_error","message":"Upstream request failed: Endpoint is unavailable."}}`
	billing := "payment required (HTTP 402): usage limit reached or billing not set up for this key"
	s503 := status(503, real5xx)
	s429 := status(429, errBody("Slow down."))
	lockoutAll := func(o *Options) {
		o.Lockout = func(st int, _ http.Header, _ []byte) bool { return st == http.StatusTooManyRequests }
	}
	cases := []struct {
		name    string
		steps   []http.HandlerFunc
		reqs    int
		kind    ErrorKind
		wantMsg string
		mutate  func(*Options)
	}{
		{
			"400 with message",
			[]http.HandlerFunc{status(400, real400)},
			1, KindBadRequest,
			`bad request: server said "Upstream request failed: [invalid_request_error] \"thinking.type.between_tools\" is not supported for this model."`, nil,
		},
		// The server text holds quotes and a backslash; strconv.Quote escapes both.
		{
			"400 message with quotes and backslash",
			[]http.HandlerFunc{status(400, errBody(`say \"hi\" \\ end`))},
			1, KindBadRequest,
			`bad request: server said "say \"hi\" \\ end"`, nil,
		},
		// Non-ASCII server text is shown as written: strconv.Quote keeps printable
		// runes, where strconv.QuoteToASCII would turn them into \u escapes.
		{
			"400 non-ASCII message kept plain",
			[]http.HandlerFunc{status(400, errBody("café 服务器繁忙"))},
			1, KindBadRequest,
			`bad request: server said "café 服务器繁忙"`, nil,
		},
		{"400 without message", []http.HandlerFunc{status(400, `{}`)}, 1, KindBadRequest, "bad request", nil},
		{"400 empty body", []http.HandlerFunc{status(400, ``)}, 1, KindBadRequest, "bad request", nil},
		{
			"402 with message",
			[]http.HandlerFunc{status(402, errBody("Monthly limit reached."))},
			1, KindBilling,
			billing + `: server said "Monthly limit reached."`, nil,
		},
		{"402 without body", []http.HandlerFunc{status(402, ``)}, 1, KindBilling, billing, nil},
		{"418 default branch", []http.HandlerFunc{status(418, errBody("I am a teapot."))}, 1, KindHTTP, `HTTP 418: server said "I am a teapot."`, nil},
		{"404 non-JSON body", []http.HandlerFunc{status(404, "nope")}, 1, KindHTTP, "HTTP 404", nil},
		{
			"5xx exhausted with message",
			[]http.HandlerFunc{s503, s503, s503, s503},
			4, KindUnavailable,
			`service unavailable: server said "Upstream request failed: Endpoint is unavailable."`, nil,
		},
		{
			"5xx exhausted without message",
			[]http.HandlerFunc{status(500, "boom"), status(500, "boom"), status(500, "boom"), status(500, "boom")},
			4, KindUnavailable,
			"service unavailable", nil,
		},
		// The last attempt's body is the one shown: a network error after a 503 leaves none.
		{"5xx then network error exhausted", []http.HandlerFunc{s503, hangup, hangup, hangup}, 4, KindUnavailable, "service unavailable", nil},
		{"429 exhausted with message", []http.HandlerFunc{s429, s429, s429, s429}, 4, KindRateLimited, `rate limited: server said "Slow down."`, nil},
		{"429 exhausted without body", []http.HandlerFunc{status(429, ``), status(429, ``), status(429, ``), status(429, ``)}, 4, KindRateLimited, "rate limited", nil},
		{"429 lockout with message", []http.HandlerFunc{s429}, 1, KindLockout, `rate limit lockout: server said "Slow down."`, lockoutAll},
		{"429 lockout without body", []http.HandlerFunc{status(429, ``)}, 1, KindLockout, "rate limit lockout", lockoutAll},
		// Retry-After beyond the maximum fails at once, still with the attempt's server text.
		{
			"429 retry-after seconds over max with message",
			[]http.HandlerFunc{status(429, errBody("Slow down."), "Retry-After", "120")},
			1, KindRateLimited,
			`retry-after exceeded maximum: server said "Slow down."`, nil,
		},
		{
			"429 retry-after seconds over max without body",
			[]http.HandlerFunc{status(429, ``, "Retry-After", "120")},
			1, KindRateLimited,
			"retry-after exceeded maximum", nil,
		},
		{
			"429 retry-after HTTP-date over max with message",
			[]http.HandlerFunc{status(429, errBody("Slow down."), "Retry-After", time.Now().Add(5*time.Minute).UTC().Format(http.TimeFormat))},
			1, KindRateLimited,
			`retry-after exceeded maximum: server said "Slow down."`, nil,
		},
		// MissingSessionID is a configuration error with fixed text; the server text is not shown.
		{
			"400 MissingSessionID: no server text",
			[]http.HandlerFunc{status(400, `{"type":"error","error":{"type":"MissingSessionID","message":"SERVERTEXT"}}`)},
			1, KindConfig,
			`provider "opencode" refused the request: MissingSessionID (configuration error, not retried)`, nil,
		},
		{
			"401 mirror: no server text",
			[]http.HandlerFunc{status(401, errBody("SERVERTEXT-401"))},
			1, KindAuth,
			`provider "opencode" rejected the API key (HTTP 401); set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY`, nil,
		},
		{
			"403 mirror: no server text",
			[]http.HandlerFunc{status(403, errBody("SERVERTEXT-403"))},
			1, KindAuth,
			`provider "opencode" rejected the API key (HTTP 403); set KIRSCH_OPENCODE_API_KEY or OPENCODE_API_KEY`, nil,
		},
		{
			"302 mirror: no server text",
			[]http.HandlerFunc{status(302, errBody("SERVERTEXT-302"), "Location", "https://example.test/x")},
			1, KindRedirect,
			`provider "opencode" redirected (HTTP 302) to host "example.test"; redirects are refused`, nil,
		},
		{"key in server message dropped", []http.HandlerFunc{status(400, errBody("echo "+testKey))}, 1, KindBadRequest, "bad request", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recorder{}
			srv := serve(t, rec, tc.steps...)
			var mutations []func(*Options)
			if tc.mutate != nil {
				mutations = append(mutations, tc.mutate)
			}
			evs, err := run(t, newClient(t, srv, "opencode", &sleeps{}, mutations...))
			pe := failure(t, evs, err, tc.kind)
			if pe.Message != tc.wantMsg {
				t.Errorf("Message = %q, want %q", pe.Message, tc.wantMsg)
			}
			if rec.count() != tc.reqs || pe.Attempts != tc.reqs {
				t.Errorf("requests = %d, Attempts = %d, want %d", rec.count(), pe.Attempts, tc.reqs)
			}
		})
	}
}

// TestHandleNonRetryable429NotLockout calls handleNonRetryable directly with
// isLockout=false. Stream never reaches that branch (isRetryable retries every
// 429 that is not a lockout, and the exhausted case is built in Stream), so this
// is the only way to cover the branch's own server-text handling; the entry
// point is the method, not Stream.
func TestHandleNonRetryable429NotLockout(t *testing.T) {
	c := helperClient(t)
	cases := []struct{ name, body, want string }{
		{"with message", errBody("Slow down."), `rate limited: server said "Slow down."`},
		{"without body", ``, "rate limited"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := c.handleNonRetryable(429, []byte(tc.body), http.Header{}, 1, false, nil)
			if e.Kind != KindRateLimited || e.Message != tc.want {
				t.Fatalf("kind=%s message=%q, want %s %q", e.Kind, e.Message, KindRateLimited, tc.want)
			}
		})
	}
}
