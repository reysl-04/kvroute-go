package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

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
	flag.Parse()
	if *tracePath == "" {
		return errors.New("--trace path is required")
	}

	records, err := loadgen.ReadTrace(*tracePath)
	if err != nil {
		return err
	}

	fmt.Printf("Load Gen successful. Loaded %d\n", len(records))
	return nil
}
