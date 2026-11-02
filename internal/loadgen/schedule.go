package loadgen

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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
	fmt.Println(len(b))
	fmt.Printf("Request: %s, Offset: %d, Time Since: %v, worker: %s\n", rec.RequestID, rec.TOffsetMS, time.Since(planned), worker)
}
