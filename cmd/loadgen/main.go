package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/reysl-04/kvroute-go/internal/loadgen"
)

func main() {
	if err := run(); err != nil { // keep main thin: all the work and error paths live in run
		fmt.Fprintln(os.Stderr, "loadgen:", err)
		os.Exit(1) // a non-zero exit code tells scripts and CI that it failed
	}
}

func run() error {
	tracePath := flag.String("trace", "", "trace JSONL to replay")
	workersFlag := flag.String("workers", "http://localhost:8001,http://localhost:8002,http://localhost:8003,http://localhost:8004", "comma-separated worker URLs")
	model := flag.String("model", "sim-8b", "model used")

	flag.Parse()
	if *tracePath == "" {
		return errors.New("--trace path is required")
	}
	if *workersFlag == "" {
		return errors.New("--workers url is required")
	}

	workers := strings.Split(*workersFlag, ",")

	// fmt.Println(*model, workers, len(workers))
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = 64
	client := &http.Client{Transport: t}

	records, err := loadgen.ReadTrace(*tracePath)
	if err != nil {
		return err
	}

	loadgen.Run(records[:20], client, workers, *model)

	fmt.Printf("Load Gen successful. Loaded %d\n", len(records))
	return nil
}
