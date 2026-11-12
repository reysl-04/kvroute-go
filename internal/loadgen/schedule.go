package loadgen

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/reysl-04/kvroute-go/internal/sse"
	"github.com/reysl-04/kvroute-go/internal/trace"
)

// package-level JSON struct
type completionRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
	Stream    bool   `json:"stream"`
}

func ReadTrace(path string) ([]trace.Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	const maxCapacity = 1024 * 1024
	buf := make([]byte, 0, 64*64)

	scanner.Buffer(buf, maxCapacity)
	records := make([]trace.Record, 0)

	for scanner.Scan() {
		var result trace.Record
		err := json.Unmarshal(scanner.Bytes(), &result)

		if err != nil {
			return nil, err
		}
		records = append(records, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func Run(records []trace.Record, client *http.Client, workers []string, model string) {
	start := time.Now()
	var wg sync.WaitGroup

	for i, record := range records {
		worker := workers[i%len(workers)]
		wg.Add(1)
		planned := start.Add(time.Duration(record.TOffsetMS) * time.Millisecond)
		time.Sleep(time.Until(planned))
		go fire(record, planned, worker, &wg, client, model)
	}
	wg.Wait()

}

func fire(rec trace.Record, planned time.Time, worker string, wg *sync.WaitGroup, client *http.Client, model string) {
	defer wg.Done()

	body := completionRequest{
		Model:     model,
		Prompt:    rec.Prompt,
		MaxTokens: rec.MaxTokens,
		Stream:    true,
	}

	b, err := json.Marshal(body)
	if err != nil {
		log.Printf("%s: marshal: %v", rec.RequestID, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", worker+"/v1/completions", bytes.NewReader((b)))
	if err != nil {
		log.Printf("%s: %v", rec.RequestID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	sentAt := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("%s: %v", rec.RequestID, err)
		return
	}
	defer resp.Body.Close()

	// Error path: read the server's error message, log it, stop.
	// Must come before any draining: the body can only be read once.
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		log.Printf("%s: status %d: %s", rec.RequestID, resp.StatusCode, errBody)
		return
	}

	// Success path: read to the end (waits for the full stream,
	// lets the connection be reused). Replaced by the SSE scanner next step.
	// replace io.Copy(io.Discard, resp.Body) with sse

	result, err := sse.Read(resp.Body)
	if err != nil {
		log.Printf("%s: %v", rec.RequestID, err)
		return
	}

	if result.FirstText.IsZero() {
		log.Printf("%s: No text in response", rec.RequestID)
		return
	}

	ttft := result.FirstText.Sub(planned)
	e2e := result.Done.Sub(planned)
	sendLag := sentAt.Sub(planned)

	fmt.Printf("<%s> worker=%s, TTFT=%v, E2E=%v, SendLag=%v, Chunk=%v\n", rec.RequestID, worker, ttft, e2e, sendLag, result.TextChunks)
}
