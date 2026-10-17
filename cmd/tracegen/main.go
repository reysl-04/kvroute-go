// Command tracegen writes a seeded synthetic trace as JSONL.
//
//	go run ./cmd/tracegen --config traces/configs/synth.yaml --out traces/synth.jsonl
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/reysl-04/kvroute-go/internal/trace"
)

func main() {
	if err := run(); err != nil { // keep main thin: all the work and error paths live in run
		fmt.Fprintln(os.Stderr, "tracegen:", err)
		os.Exit(1) // a non-zero exit code tells scripts and CI that it failed
	}
}

func run() error {
	configPath := flag.String("config", "", "trace config YAML (traces/configs/*.yaml)")
	outPath := flag.String("out", "", "output JSONL path")
	flag.Parse() // the flag package accepts both -config and --config
	if *configPath == "" || *outPath == "" {
		return errors.New("both --config and --out are required")
	}

	cfg, err := trace.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	recs, err := trace.Generate(cfg) // validates first; on failure we return before --out is created
	if err != nil {
		return err
	}

	f, err := os.Create(*outPath) // only now touch the output file
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f) // buffers writes, so each line isn't its own system call
	werr := trace.WriteJSONL(w, recs)
	if werr == nil {
		werr = w.Flush() // push whatever is still sitting in the buffer to the file
	}
	if err := errors.Join(werr, f.Close()); err != nil { // report a write error and a close error together
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %d records to %s\n", len(recs), *outPath)
	return nil
}
