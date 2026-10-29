package loadgen

import (
	"bufio"
	"encoding/json"
	"os"

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
