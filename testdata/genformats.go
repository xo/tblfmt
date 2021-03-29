// +build ignore

package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	for _, opts := range formatOpts() {
		opts = opts
	}
	return nil
}

// formatOpts returns the format options.
func formatOpts() []map[string]string {
	return []map[string]string{
		map[string]string{
			"name": "tiny",
			"set":  "tiny",
		},
		map[string]string{
			"name": "big",
			"set": "big",
			"seed": "",
		}
	}
}
