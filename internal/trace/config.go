// Package trace defines the synthetic trace format and its seeded generator.
package trace

import (
	"bytes"
	"fmt"
	"math"
	"os"

	"go.yaml.in/yaml/v3"
)

const (
	maxModelLen   = 8192  // must match max-model-len in deploy/sim/profile.yaml
	tokensPerWord = 1.5   // conservative words-to-tokens estimate used by the budget check
	maxSessions   = 10000 // session IDs are fixed-width s0000–s9999, so string order = numeric order
	maxTurns      = 100   // turn IDs are fixed-width t00–t99, for the same reason
)

// Config is one trace's parameters, read from a YAML file in traces/configs/.
type Config struct {
	Seed            uint64  `yaml:"seed"`              // seeds the single random source; same seed = same trace
	Sessions        int     `yaml:"sessions"`          // number of conversations
	TurnsPerSession int     `yaml:"turns_per_session"` // requests per conversation
	PrefixGroups    int     `yaml:"prefix_groups"`     // number of distinct shared prefixes
	PrefixWords     int     `yaml:"prefix_words"`      // words in each shared prefix
	SuffixWords     int     `yaml:"suffix_words"`      // new words each turn adds
	GapMS           int64   `yaml:"gap_ms"`            // time between consecutive turns of one session
	ArrivalRate     float64 `yaml:"arrival_rate"`      // new sessions per second (Poisson process)
	MaxTokens       int     `yaml:"max_tokens"`        // output tokens each request asks for
}

// LoadConfig reads a YAML config file. Unknown keys are an error, so a typo
// can't silently leave a field at zero (the simulator's YAML loader has that trap).
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path) // read the whole file into memory
	if err != nil {
		return Config{}, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data)) // a Decoder, because only it supports KnownFields
	dec.KnownFields(true)                         // reject keys that don't match a yaml tag above
	var cfg Config
	if err := dec.Decode(&cfg); err != nil { // fill cfg from the YAML; an empty file returns io.EOF
		return Config{}, fmt.Errorf("parse %s: %w", path, err) // %w keeps the original error inspectable
	}
	return cfg, nil
}

// Validate checks every field's range and the token budget. A missing key
// decodes as 0 and fails here, except seed, where 0 is a valid value.
func (c Config) Validate() error {
	switch {
	case c.Sessions < 1 || c.Sessions > maxSessions:
		return fmt.Errorf("sessions must be 1..%d, got %d", maxSessions, c.Sessions)
	case c.TurnsPerSession < 1 || c.TurnsPerSession > maxTurns:
		return fmt.Errorf("turns_per_session must be 1..%d, got %d", maxTurns, c.TurnsPerSession)
	case c.PrefixGroups < 1:
		return fmt.Errorf("prefix_groups must be >= 1, got %d", c.PrefixGroups)
	case c.PrefixWords < 1:
		return fmt.Errorf("prefix_words must be >= 1, got %d", c.PrefixWords)
	case c.SuffixWords < 1:
		return fmt.Errorf("suffix_words must be >= 1, got %d", c.SuffixWords)
	case c.GapMS < 1:
		return fmt.Errorf("gap_ms must be >= 1, got %d", c.GapMS)
	case !(c.ArrivalRate > 0): // written this way so NaN is rejected too
		return fmt.Errorf("arrival_rate must be > 0, got %v", c.ArrivalRate)
	case c.MaxTokens < 1:
		return fmt.Errorf("max_tokens must be >= 1, got %d", c.MaxTokens)
	}

	// Prefixes must be distinct, so the word list must allow at least prefix_groups different ones.
	if float64(c.PrefixGroups) > math.Pow(float64(len(words)), float64(c.PrefixWords)) {
		return fmt.Errorf("prefix_groups %d exceeds the %d^%d possible prefixes", c.PrefixGroups, len(words), c.PrefixWords)
	}

	// The last turn carries the prefix plus every turn's suffix. Computed in float64 so huge values can't overflow.
	longestWords := float64(c.PrefixWords) + float64(c.TurnsPerSession)*float64(c.SuffixWords)
	budget := longestWords*tokensPerWord + float64(c.MaxTokens)
	if budget >= maxModelLen {
		return fmt.Errorf("token budget (%d + %d*%d)*%.1f + %d = %.0f, must be < %d",
			c.PrefixWords, c.TurnsPerSession, c.SuffixWords, tokensPerWord, c.MaxTokens, budget, maxModelLen)
	}
	return nil
}
