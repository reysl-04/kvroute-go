package trace

import (
	"bytes"
	"strings"
	"testing"
)

// testConfig is small enough to run instantly but sets every field.
func testConfig() Config {
	return Config{
		Seed: 42, Sessions: 50, TurnsPerSession: 4, PrefixGroups: 3, PrefixWords: 200,
		SuffixWords: 30, GapMS: 1500, ArrivalRate: 5, MaxTokens: 64,
	}
}

// render generates cfg and returns the exact bytes tracegen would write.
func render(t *testing.T, cfg Config) []byte {
	t.Helper() // failures point at the caller's line, not this one
	recs, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteJSONL(&buf, recs); err != nil {
		t.Fatalf("WriteJSONL: %v", err)
	}
	return buf.Bytes()
}

func TestGenerateIsDeterministic(t *testing.T) {
	cfg := testConfig()
	first, second := render(t, cfg), render(t, cfg)
	if !bytes.Equal(first, second) {
		t.Fatal("same config produced different output")
	}

	// If the seed were accidentally ignored, the check above would still pass. Make sure it isn't.
	cfg.Seed++
	if bytes.Equal(first, render(t, cfg)) {
		t.Fatal("a different seed produced identical output")
	}
}

// Each turn must start with the previous turn's whole prompt. That repetition is
// the prefix reuse the cache-aware router is supposed to exploit.
func TestTurnsExtendPreviousPrompt(t *testing.T) {
	recs, err := Generate(testConfig())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	prev := map[string]string{} // session_id -> that session's latest prompt so far
	for _, rec := range recs {  // sorted by time, and turn k is always sent before turn k+1
		if p, ok := prev[rec.SessionID]; ok && !strings.HasPrefix(rec.Prompt, p+" ") {
			t.Fatalf("%s does not extend the previous turn's prompt", rec.RequestID)
		}
		prev[rec.SessionID] = rec.Prompt
	}
}

func TestGenerateRejectsOverBudget(t *testing.T) {
	cfg := testConfig()
	cfg.SuffixWords = 2000 // (200 + 4*2000)*1.5 + 64 = 12364, over the 8192 limit
	if _, err := Generate(cfg); err == nil {
		t.Fatal("expected a token-budget error, got nil")
	}
}
