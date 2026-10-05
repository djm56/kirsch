//go:build live

package anthropic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/djm56/kirsch/internal/config"
	"github.com/djm56/kirsch/internal/provider"
)

// TestLiveOpencodeSmoke sends one streamed, text-only request to the opencode
// endpoint through Kirsch's own adapter and checks that it completes and
// reports usage. It is compiled only with -tags live and is run by hand
// (npm run test:live); go test ./... and CI never build it. The key is read
// from the environment by config's own lookup and is never printed. The model
// name defaults to the opencode endpoint's configured default model and can be
// overridden by the KIRSCH_LIVE_MODEL environment variable, which is useful
// when the default model is disabled on the upstream endpoint. If
// KIRSCH_LIVE_MODEL is set but invalid (empty or containing whitespace or
// control characters), the test fails with a validation error before any
// network request is made.
func TestLiveOpencodeSmoke(t *testing.T) {
	ep := config.Defaults().Provider.Endpoints["opencode"]
	key, source := ep.ResolveKey(os.Getenv)
	if key == "" {
		t.Skipf("set KIRSCH_%s or %s to run the live smoke test", ep.APIKeyEnv, ep.APIKeyEnv)
	}
	model, modelSource, err := liveModel(os.LookupEnv, ep.Model)
	if err != nil {
		t.Fatalf("invalid model override: %v", err)
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	c, err := New(Options{
		Endpoint: "opencode", BaseURL: ep.BaseURL, Auth: ep.Auth, APIKey: key, KeyEnv: ep.APIKeyEnv,
		PromptCaching: ep.PromptCaching, Version: "smoke", SessionID: hex.EncodeToString(b[:]),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req := provider.Request{
		Model: model, MaxTokens: 256,
		Messages: []provider.Message{userText("Reply with the single word: pong")},
	}
	var text strings.Builder
	var done *provider.StreamEvent
	err = c.Stream(ctx, req, func(e provider.StreamEvent) {
		switch e.Type {
		case provider.EventTextDelta:
			text.WriteString(e.Text)
		case provider.EventMessageDone:
			ev := e
			done = &ev
		default:
			// Other events are not needed for the smoke check.
		}
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if done == nil || done.Usage == nil || done.Usage.InputTokens <= 0 || done.Usage.OutputTokens <= 0 {
		t.Fatalf("no usage reported: %+v", done)
	}
	if strings.TrimSpace(text.String()) == "" {
		t.Fatal("no text streamed")
	}
	t.Logf("endpoint=opencode model=%s model_source=%s key_source=%s text_bytes=%d stop=%s usage=%+v",
		model, modelSource, source, text.Len(), done.StopReason, *done.Usage)
}
