// Command count_helper is a tiny stdin/stdout bridge used only by
// scripts/calibrate.py. It exists so calibration's "base tokenizer count"
// is produced by calling this module's own GetEncoding(...).Count exactly
// as CountForModel does at runtime, rather than by a second
// reimplementation in Python that could quietly drift from the real
// behavior it's supposed to be measuring against.
//
// Not part of the public module surface: it lives outside internal/ only
// because `go run` needs a plain file path, not an importable package.
// It is invoked as `go run scripts/count_helper.go`, reads a JSON object
// {"encoding": "...", "texts": [...]} from stdin, and writes
// {"counts": [...]} to stdout.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	tokenizer "github.com/xorvus/tokenizer-go"
)

type request struct {
	Encoding string   `json:"encoding"`
	Texts    []string `json:"texts"`
}

type response struct {
	Counts []int `json:"counts"`
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "count_helper: decoding request: %v\n", err)
		os.Exit(1)
	}

	tok, err := tokenizer.GetEncoding(tokenizer.Encoding(req.Encoding))
	if err != nil {
		fmt.Fprintf(os.Stderr, "count_helper: GetEncoding(%q): %v\n", req.Encoding, err)
		os.Exit(1)
	}

	counts, err := tok.CountOrdinaryBatch(req.Texts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "count_helper: CountOrdinaryBatch: %v\n", err)
		os.Exit(1)
	}

	if err := json.NewEncoder(os.Stdout).Encode(response{Counts: counts}); err != nil {
		fmt.Fprintf(os.Stderr, "count_helper: encoding response: %v\n", err)
		os.Exit(1)
	}
}
