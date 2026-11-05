// internal/sse/sse.go
package sse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
)

// Result is what one streamed response looked like, timing-wise.
type Result struct {
	FirstText  time.Time // when the first chunk with non-empty text arrived
	Done       time.Time // when "data: [DONE] arrived"
	TextChunks int       // chunks that carried text (stop chumk and usage chunk don't count)
}

// chunk holds only the field we need from a streamed completion chunk.
type chunk struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// payload returns what comes after "data:" on an SSE line, trimmed.
// ok is false for anything that isn't a data line (blank separators, comments).
func payload(line []byte) (p []byte, ok bool) {
	rest, ok := bytes.CutPrefix(line, []byte("data:"))
	if !ok {
		return nil, false
	}
	return bytes.TrimSpace(rest), true
}

// Read consumes an SSE response body until [DONE]
// It returns an error if the stream fails or ends without [DONE]
func Read(body io.Reader) (Result, error) {
	scanner := bufio.NewScanner(body)
	chunks := 0
	var firstText time.Time

	for scanner.Scan() {
		timestamp := time.Now()
		// slice into the scanner's buffer; overwritten by the next Scan
		buffer := scanner.Bytes()
		p, ok := payload(buffer)
		if !ok || len(p) == 0 {
			continue
		}

		if bytes.Equal(p, []byte("[DONE]")) {
			return Result{FirstText: firstText, Done: timestamp, TextChunks: chunks}, nil
		}

		var data chunk
		err := json.Unmarshal(p, &data)
		if err != nil {
			return Result{}, err
		}
		if len(data.Choices) > 0 && data.Choices[0].Text != "" {
			chunks++
			if firstText.IsZero() {
				firstText = timestamp
			}
		}
	}
	// Loop ended without [DONE]: either a read error, or the server closed early.
	if err := scanner.Err(); err != nil {
		return Result{}, err
	}
	return Result{}, errors.New("sse: stream ended without [DONE]")
}
