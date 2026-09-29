// Command flushprobe is a non-test binary that reports whether fsutil.Flush
// reaches the flush. TestAShippedBinaryFlushesWithTheOptInSet builds it with
// go build and runs it with the test opt-in set, because a shipped binary is
// the one place the opt-in must do nothing.
//
// The flush is observed by its error: Flush on a closed file returns
// os.ErrClosed when it calls File.Sync, and nil when the gate skipped the call.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/fsutil"
)

func main() {
	f, err := os.CreateTemp("", "flushprobe-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	closeErr := f.Close()
	_ = os.Remove(f.Name())
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, closeErr)
		os.Exit(2)
	}
	switch err := fsutil.Flush(f); {
	case errors.Is(err, os.ErrClosed):
		fmt.Println("flushed")
	case err == nil:
		fmt.Println("skipped")
	default:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
