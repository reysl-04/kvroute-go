package trace

import (
	"cmp"
	_ "embed" // enables the //go:embed directive below
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
)

//go:embed words.txt
var wordList string // words.txt is compiled into the binary, so the vocabulary can't change between machines

// words is the fixed vocabulary, one entry per line of words.txt. Every entry is
// lowercase letters only, so the simulator's regex tokenizer counts each as one token.
var words = strings.Fields(wordList)

// Record is one request in the trace and one line of the output JSONL.
// encoding/json writes fields in this declaration order, which keeps the output byte-stable.
type Record struct {
	RequestID string `json:"request_id"`  // e.g. s0042-t03
	SessionID string `json:"session_id"`  // e.g. s0042
	Turn      int    `json:"turn"`        // 0-based turn number within the session
	TOffsetMS int64  `json:"t_offset_ms"` // planned send time, in ms after the trace starts
	Prompt    string `json:"prompt"`      // full prompt text, including every earlier turn
	MaxTokens int    `json:"max_tokens"`  // output tokens to request
}

// Generate builds the trace described by cfg. The same cfg always returns the same
// records in the same order. The order of random draws below is part of the format:
// reordering them changes every trace generated from a given seed.
func Generate(cfg Config) ([]Record, error) {
	if err := cfg.Validate(); err != nil { // refuse to build anything from an invalid config
		return nil, err
	}
	r := rand.New(rand.NewPCG(cfg.Seed, 0)) // the one random source; every draw below comes from it

	// 1. Draw prefix_groups distinct prefixes, in group order.
	prefixes := make([]string, cfg.PrefixGroups)
	seen := make(map[string]bool, cfg.PrefixGroups) // only looked up, never iterated, so it can't affect order
	for g := range prefixes {
		p := randomWords(r, cfg.PrefixWords)
		for seen[p] { // a duplicate: draw again until it's new
			p = randomWords(r, cfg.PrefixWords)
		}
		seen[p] = true
		prefixes[g] = p
	}

	// 2. For each session, in order: draw its start time, its group, then one suffix per turn.
	recs := make([]Record, 0, cfg.Sessions*cfg.TurnsPerSession) // exact final size, so append never reallocates
	startSec := 0.0                                             // running arrival time, in seconds
	for s := range cfg.Sessions {
		startSec += r.ExpFloat64() / cfg.ArrivalRate           // exponential gaps with mean 1/rate = a Poisson process
		startMS := int64(math.Round(startSec * 1000))          // round once per session, so rounding errors don't add up
		sessionID := fmt.Sprintf("s%04d", s)                   // e.g. s0042
		var prompt strings.Builder                             // grows by one suffix per turn
		prompt.WriteString(prefixes[r.IntN(cfg.PrefixGroups)]) // assign the session a group, uniformly at random
		for k := range cfg.TurnsPerSession {
			prompt.WriteString(" ")
			prompt.WriteString(randomWords(r, cfg.SuffixWords)) // this turn's new suffix
			recs = append(recs, Record{
				RequestID: fmt.Sprintf("%s-t%02d", sessionID, k), // e.g. s0042-t03
				SessionID: sessionID,
				Turn:      k,
				TOffsetMS: startMS + int64(k)*cfg.GapMS, // turn k is sent at session_start + k*gap_ms
				Prompt:    prompt.String(),              // prefix + all suffixes so far
				MaxTokens: cfg.MaxTokens,
			})
		}
	}

	// 3. Sort by send time; request_id breaks ties. IDs are unique, so the order is total.
	slices.SortFunc(recs, func(a, b Record) int {
		return cmp.Or( // returns the first non-zero comparison
			cmp.Compare(a.TOffsetMS, b.TOffsetMS),
			strings.Compare(a.RequestID, b.RequestID),
		)
	})
	return recs, nil
}

// randomWords returns n words drawn uniformly from the vocabulary, joined by single spaces.
func randomWords(r *rand.Rand, n int) string {
	picked := make([]string, n)
	for i := range picked {
		picked[i] = words[r.IntN(len(words))]
	}
	return strings.Join(picked, " ")
}

// WriteJSONL writes the records as JSON Lines: one JSON object per line.
func WriteJSONL(w io.Writer, recs []Record) error {
	enc := json.NewEncoder(w) // Encode writes the object followed by "\n"
	for _, rec := range recs {
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	return nil
}
