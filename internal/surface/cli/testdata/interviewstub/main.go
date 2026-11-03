// Command interviewstub is the claude runner the AI-written interviews' tests
// put on PATH (internal/surface/cli/written_interview_test.go), built once per
// test process. Each turn of an interview starts it, and a small program
// starts in milliseconds where the race-built test binary playing the same
// part took about a second before its first line ran.
package main

import (
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/surface/cli/testdata/interviewstub/stub"
)

func main() {
	dir := os.Getenv(stub.ScriptEnv)
	if !stub.Runs(dir, os.Args) {
		fmt.Fprintf(os.Stderr, "interviewstub: started without %s or claude's --print flag: %q\n", stub.ScriptEnv, os.Args[1:])
		os.Exit(2)
	}
	os.Exit(stub.Run(dir, os.Args, os.Stdout))
}
