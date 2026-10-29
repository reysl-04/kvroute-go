package loadgen

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/reysl-04/kvroute-go/internal/trace"
)

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

func Run(records []trace.Record) {
	start := time.Now()
	var wg sync.WaitGroup

	for _, record := range records {
		wg.Add(1)
		planned := start.Add(time.Duration(record.TOffsetMS) * time.Millisecond)
		time.Sleep(time.Until(planned))
		go fire(record, planned, &wg)
	}
	wg.Wait()

}

func fire(rec trace.Record, planned time.Time, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("Request: %s, Offset: %d, Time Since: %v\n", rec.RequestID, rec.TOffsetMS, time.Since(planned))
}
