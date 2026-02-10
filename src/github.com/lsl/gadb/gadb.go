package gadb

import (
	"fmt"
	"os"
)

// Gadb is the main entry point for gadb
// It determines whether to run in REPL mode (no args) or normal mode (with args)
func Gadb() {
	args := os.Args[1:]

	// No arguments - enter interactive REPL mode
	if len(args) == 0 {
		devices := readDevices()
		if err := RunREPL(devices); err != nil {
			fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Has arguments - run in normal mode (backward compatible)
	if err := RunNormalMode(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
