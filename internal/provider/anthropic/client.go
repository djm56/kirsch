package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
	"github.com/djm56/kirsch/internal/telemetry"
)

// ErrorKind refines a provider failure. Every failure is a provider_error to
// the user; Kind says which.
type ErrorKind string

const (
	KindUnavailable ErrorKind = "unavailable"  // 5xx or transient network error on every attempt, or any non-transient network error
	KindRateLimited ErrorKind = "rate_limited" // 429 on every attempt, or Retry-After beyond maxRetryAfter
	KindLockout     ErrorKind = "lockout"      // 429 the Lockout hook recognised
	KindAuth        ErrorKind = "auth"         // 401 or 403
	KindConfig      ErrorKind = "config"       // gateway policy error such as MissingSessionID
	KindBadRequest  ErrorKind = "bad_request"  // any other 400
	KindRedirect    ErrorKind = "redirect"     // any 3xx; never followed
	KindHTTP        ErrorKind = "http"         // any other non-2xx, or a 2xx other than 200
	KindStream      ErrorKind = "stream"       // failure while decoding a 200 response
)

// Error is every failure Stream returns, apart from cancellation.
type Error struct {
	Kind     ErrorKind
	Status   int    // HTTP status of the last response; 0 when there was none
	Attempts int    // requests sent
	Message  string // safe to display; never contains the key
	Err      error  // underlying cause; may be nil
}

// Error returns "provider_error: <Kind>: <Message>".
func (e *Error) Error() string {
	return fmt.Sprintf("provider_error: %s: %s", e.Kind, e.Message)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// Options configures a Client. The caller resolves the key (config.ResolveKey).
type Options struct {
	Endpoint      string // endpoint name; "opencode" adds x-opencode-session
	BaseURL       string // must pass config.ValidateBaseURL
	Auth          string // "x-api-key" or "bearer"
	APIKey        string // never logged, never in an error
	KeyEnv        string // unprefixed variable name, for the onboarding message
	PromptCaching bool
	Version       string                                            // Kirsch version; User-Agent is kirsch/<Version>
	SessionID     string                                            // required when Endpoint == "opencode"
	HTTPClient    *http.Client                                      // nil: a new client; never mutated; a Transport that follows redirects bypasses the refusal
	Log           *telemetry.Logger                                 // nil: disabled
	Sleep         func(ctx context.Context, d time.Duration) error  // nil: a timer that honours ctx; a non-cancellation error from Sleep fails the stream as KindUnavailable
	Lockout       func(status int, h http.Header, body []byte) bool // nil: no lockout signal known
}

// Client is a provider.Provider over one Messages endpoint.
type Client struct {
	endpoint      string
	baseURL       string
	auth          string
	key           string
	keyEnv        string
	promptCaching bool
	version       string
	sessionID     string
	httpClient    *http.Client
	log           *telemetry.Logger
	sleep         func(context.Context, time.Duration) error
	lockout       func(int, http.Header, []byte) bool
}

var _ provider.Provider = (*Client)(nil)

// New constructs a Client or returns an error. No error text may contain the key.
func New(o Options) (*Client, error) {
	if err := config.ValidateBaseURL(o.BaseURL); err != nil {
		return nil, err
	}
	if o.Auth != "x-api-key" && o.Auth != "bearer" {
		return nil, fmt.Errorf("auth must be x-api-key or bearer, got %q", o.Auth)
	}
	if o.APIKey == "" {
		return nil, errors.New("api_key is required")
	}
	if o.Endpoint == "opencode" && o.SessionID == "" {
		return nil, errors.New("session_id is required for opencode endpoint")
	}
	if o.Version == "" {
		return nil, errors.New("version is required")
	}

	httpClient := o.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	} else {
		clientCopy := *httpClient
		httpClient = &clientCopy
	}

	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &Client{
		endpoint:      o.Endpoint,
		baseURL:       strings.TrimSuffix(o.BaseURL, "/"),
		auth:          o.Auth,
		key:           o.APIKey,
		keyEnv:        o.KeyEnv,
		promptCaching: o.PromptCaching,
		version:       o.Version,
		sessionID:     o.SessionID,
		httpClient:    httpClient,
		log:           o.Log,
		sleep:         o.Sleep,
		lockout:       o.Lockout,
	}, nil
}

const (
	maxAttempts   = 4
	baseBackoff   = 500 * time.Millisecond
	maxRetryAfter = 60 * time.Second
	maxErrorBody  = 64 << 10
)

var retryAfterDigitsOnly = regexp.MustCompile(`^[0-9]+$`)

// fail reports err as the stream's single failure: it emits one EventError
// carrying err and returns err — unless ctx is done, in which case it emits
// nothing and returns ctx.Err(). Every failure path goes through it.
func (c *Client) fail(ctx context.Context, onEvent func(provider.StreamEvent), err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	onEvent(provider.StreamEvent{Type: provider.EventError, Err: err})
	return err
}

// Stream sends req to the endpoint and reports the response through onEvent.
//
// Events arrive in stream order. On success the last event is MessageDone and
// Stream returns nil. On any failure other than cancellation, Stream emits
// exactly one EventError whose Err is the *Error it returns, and nothing after
// it. On cancellation Stream returns ctx.Err() unwrapped and emits nothing.
//
// Retries: at most maxAttempts (4) requests. A 5xx or a transient network error
// (EOF, connection reset or refused, timeout, temporary DNS failure) waits
// baseBackoff (500ms), doubled per retry. A 429 waits for Retry-After, given as
// whole seconds or an HTTP-date; any other form falls back to the backoff, and
// a wait beyond maxRetryAfter (60s) fails at once. A 429 the Lockout hook
// recognises is not retried. Nothing is retried once a 200 response has begun
// to stream. Any other network error, including TLS and certificate failures,
// fails at once.
//
// Failures carry an ErrorKind; see its constants. A 2xx other than 200 is
// KindHTTP. Redirects are refused through CheckRedirect on a copy of the
// caller's http.Client; a caller-supplied Transport that follows redirects
// itself bypasses that guard.
func (c *Client) Stream(ctx context.Context, req provider.Request, onEvent func(provider.StreamEvent)) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	reqBody, err := EncodeRequest(req, EncodeOptions{PromptCaching: c.promptCaching})
	if err != nil {
		e := &Error{Kind: KindBadRequest, Attempts: 0, Message: "request not sent: " + err.Error(), Err: err}
		return c.fail(ctx, onEvent, e)
	}
	var lastStatus int
	var lastNetErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		status, headers, _, retErr, retry := c.streamAttempt(ctx, attempt, reqBody, onEvent)
		lastStatus = status
		lastNetErr = nil
		if retErr != nil && isNetworkError(retErr) {
			lastNetErr = retErr
		}
		if !retry {
			return retErr
		}
		if attempt < maxAttempts {
			wait, waitErr := c.retryWait(status, headers, attempt)
			if waitErr != nil {
				if e, isErr := waitErr.(*Error); isErr && e.Attempts == 0 {
					e.Attempts = attempt
				}
				return c.fail(ctx, onEvent, waitErr)
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logRetry(attempt, status, wait)
			if err := c.doSleep(ctx, wait); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				e := &Error{Kind: KindUnavailable, Status: status, Attempts: attempt, Message: "retry wait failed", Err: err}
				return c.fail(ctx, onEvent, e)
			}
		}
	}
	var e *Error
	if lastStatus == 429 {
		e = &Error{Kind: KindRateLimited, Status: lastStatus, Attempts: maxAttempts, Message: "rate limited"}
	} else {
		e = &Error{Kind: KindUnavailable, Status: lastStatus, Attempts: maxAttempts, Message: "service unavailable", Err: lastNetErr}
	}
	return c.fail(ctx, onEvent, e)
}

// streamAttempt makes one request. Returns (status, headers, body, error, shouldRetry).
// Emits error events for fatal failures.
func (c *Client) streamAttempt(ctx context.Context, attempt int, reqBody []byte, onEvent func(provider.StreamEvent)) (int, http.Header, []byte, error, bool) {
	resp, err := c.doRequest(ctx, reqBody)
	if err != nil {
		if ctx.Err() != nil {
			return 0, nil, nil, ctx.Err(), false
		}
		if !c.isRetryable(0, err, false) {
			e := &Error{Kind: KindUnavailable, Status: 0, Attempts: attempt, Message: "request failed: " + clip(err.Error(), 200), Err: err}
			return 0, nil, nil, c.fail(ctx, onEvent, e), false
		}
		return 0, nil, nil, err, true
	}
	if resp.StatusCode == 200 {
		defer func() { _ = resp.Body.Close() }()
		err := c.decodeStream(ctx, resp.Body, attempt, onEvent)
		return 200, resp.Header, nil, err, false
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	_ = resp.Body.Close()
	if readErr != nil && ctx.Err() == nil {
		c.logErrorBodyReadFailed(readErr, resp.StatusCode)
	}
	if ctx.Err() != nil {
		return resp.StatusCode, resp.Header, body, ctx.Err(), false
	}
	var isLockout bool
	if resp.StatusCode == 429 && c.lockout != nil {
		isLockout = c.lockout(resp.StatusCode, resp.Header, body)
	}
	if !c.isRetryable(resp.StatusCode, nil, isLockout) {
		e := c.handleNonRetryable(resp.StatusCode, body, resp.Header, attempt, isLockout, reqBody)
		return resp.StatusCode, resp.Header, body, c.fail(ctx, onEvent, e), false
	}
	return resp.StatusCode, resp.Header, body, nil, true
}

// doRequest makes one HTTP request and returns the response unread, or an error.
// The caller must close the response body.
func (c *Client) doRequest(ctx context.Context, reqBody []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("User-Agent", "kirsch/"+c.version)
	if c.auth == "x-api-key" {
		httpReq.Header.Set("x-api-key", c.key)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+c.key)
	}
	if c.endpoint == "opencode" {
		httpReq.Header.Set("x-opencode-session", c.sessionID)
	}
	return c.httpClient.Do(httpReq)
}

// isRetryable checks if we should retry this response. For 429s, pass the
// pre-computed lockout verdict; lockout checks are done once per attempt.
func (c *Client) isRetryable(status int, err error, isLockout bool) bool {
	if status >= 500 {
		return true
	}
	if status == 429 {
		return !isLockout
	}
	if isNetworkError(err) {
		return true
	}
	return false
}

// retryWait calculates the wait time before the next attempt. A 429's Retry-After
// is parsed as integer seconds (failing if > 60s or unparseable), then as an HTTP date
// (waiting until that time, or failing if the wait exceeds 60s or is unparseable).
// Other statuses get exponential backoff. A wait that fails returns KindRateLimited.
func (c *Client) retryWait(status int, headers http.Header, attempt int) (time.Duration, error) {
	if status == 429 {
		if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
			// Try parsing as integer seconds (digits only).
			if retryAfterDigitsOnly.MatchString(retryAfter) {
				secs, err := strconv.ParseInt(retryAfter, 10, 64)
				if err == nil && secs <= 60 {
					return time.Duration(secs) * time.Second, nil
				}
				return 0, &Error{Kind: KindRateLimited, Status: status, Attempts: 0, Message: "retry-after exceeded maximum"}
			}
			// Try parsing as HTTP date.
			if t, err := http.ParseTime(retryAfter); err == nil {
				wait := time.Until(t)
				if wait < 0 {
					wait = 0
				}
				if wait > maxRetryAfter {
					return 0, &Error{Kind: KindRateLimited, Status: status, Attempts: 0, Message: "retry-after exceeded maximum"}
				}
				return wait, nil
			}
		}
	}
	return baseBackoff * time.Duration(1<<(attempt-1)), nil
}

// logRetry logs a retry attempt.
func (c *Client) logRetry(attempt int, status int, wait time.Duration) {
	if c.log == nil {
		return
	}
	c.log.Debug("provider retry", slog.Int("attempt", attempt), slog.Int("status", status), slog.String("wait", wait.String()))
}

// doSleep waits for the given duration. If Sleep returns an error, that error
// is returned immediately with no retry.
func (c *Client) doSleep(ctx context.Context, d time.Duration) error {
	if c.sleep != nil {
		return c.sleep(ctx, d)
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// handleNonRetryable handles non-retryable responses.
func (c *Client) handleNonRetryable(status int, bodyBytes []byte, headers http.Header, attempt int, isLockout bool, reqBody []byte) *Error {
	switch status {
	case 401, 403:
		msg := fmt.Sprintf("provider %q rejected the API key (HTTP %d); set KIRSCH_%s or %s", c.endpoint, status, c.keyEnv, c.keyEnv)
		return &Error{Kind: KindAuth, Status: status, Attempts: attempt, Message: msg}
	case 400:
		if c.isMissingSessionID(bodyBytes) {
			return &Error{Kind: KindConfig, Status: status, Attempts: attempt, Message: fmt.Sprintf("provider %q refused the request: MissingSessionID (configuration error, not retried)", c.endpoint)}
		}
		c.logBadRequest(string(reqBody), string(bodyBytes))
		return &Error{Kind: KindBadRequest, Status: status, Attempts: attempt, Message: "bad request"}
	case 429:
		if isLockout {
			return &Error{Kind: KindLockout, Status: status, Attempts: attempt, Message: "rate limit lockout"}
		}
		return &Error{Kind: KindRateLimited, Status: status, Attempts: attempt, Message: "rate limited"}
	default:
		if status >= 300 && status < 400 {
			baseURL, _ := url.Parse(c.baseURL + "/messages")
			host := c.getLocationHost(baseURL, headers)
			return &Error{Kind: KindRedirect, Status: status, Attempts: attempt, Message: fmt.Sprintf("provider %q redirected (HTTP %d) to host %q; redirects are refused", c.endpoint, status, host)}
		}
		return &Error{Kind: KindHTTP, Status: status, Attempts: attempt, Message: fmt.Sprintf("HTTP %d", status)}
	}
}

// isMissingSessionID checks if the error is MissingSessionID.
func (c *Client) isMissingSessionID(bodyBytes []byte) bool {
	var msg struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &msg); err != nil {
		return false
	}
	return msg.Error.Type == "MissingSessionID"
}

// getLocationHost extracts and clips the host from a Location header, resolving
// relative URLs against the origin. A missing or unparseable host returns "unknown".
func (c *Client) getLocationHost(baseURL *url.URL, headers http.Header) string {
	locationURL := headers.Get("Location")
	if locationURL == "" {
		return "unknown"
	}
	u, err := baseURL.Parse(locationURL)
	if err != nil {
		return "unknown"
	}
	host := u.Hostname()
	if host == "" {
		return "unknown"
	}
	return clip(host, 255)
}

// logBadRequest logs a 400 response with request and response body.
func (c *Client) logBadRequest(reqBody, respBody string) {
	if c.log == nil {
		return
	}
	c.log.Debug("provider bad request", slog.String("endpoint", c.endpoint), slog.Int("status", 400),
		slog.String("request_body", reqBody), telemetry.Content("response_body", respBody))
}

// logErrorBodyReadFailed logs a failure to read an error response body.
func (c *Client) logErrorBodyReadFailed(err error, status int) {
	if c.log == nil {
		return
	}
	c.log.Debug("provider error body read failed", slog.String("endpoint", c.endpoint), slog.Int("status", status), slog.String("error", err.Error()))
}

// decodeStream decodes a 200 response body. Wraps EventError to mark stream
// failures as KindStream. Returns ctx.Err() directly if context is cancelled.
func (c *Client) decodeStream(ctx context.Context, body io.Reader, attempt int, onEvent func(provider.StreamEvent)) error {
	var lastStreamErr error
	wrappedEvent := func(e provider.StreamEvent) {
		if e.Type == provider.EventError {
			if se, ok := e.Err.(*StreamError); ok {
				wrappedErr := &Error{Kind: KindStream, Status: 200, Attempts: attempt, Message: se.Message, Err: se}
				lastStreamErr = wrappedErr
				e.Err = wrappedErr
			} else {
				wrappedErr := &Error{Kind: KindStream, Status: 200, Attempts: attempt, Message: e.Err.Error(), Err: e.Err}
				lastStreamErr = wrappedErr
				e.Err = wrappedErr
			}
		}
		onEvent(e)
	}
	err := Decode(ctx, body, wrappedEvent)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if lastStreamErr != nil {
			return lastStreamErr
		}
		if se, ok := err.(*StreamError); ok {
			return &Error{Kind: KindStream, Status: 200, Attempts: attempt, Message: se.Message, Err: se}
		}
		return &Error{Kind: KindStream, Status: 200, Attempts: attempt, Message: err.Error(), Err: err}
	}
	return nil
}

// isNetworkError checks if an error is a transient network error worth retrying.
// io.EOF, io.ErrUnexpectedEOF, syscall ECONNRESET/ECONNREFUSED, net.Error with
// Timeout(), and *net.DNSError with IsTemporary or IsTimeout are transient.
// TLS and certificate errors return false (not transient).
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && (dnsErr.IsTemporary || dnsErr.IsTimeout) {
		return true
	}
	return false
}
