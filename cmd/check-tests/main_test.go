package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/testreuse"
)

// TestARecipeThatIsNotAGoTestIsRefused: the recipe writes the go test command out
// whole after the stage, so anything else is a recipe mistake, which fails the
// stage there rather than passing over no tests.
func TestARecipeThatIsNotAGoTestIsRefused(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"unit"},
		{"unit", "go"},
		{"unit", "go", "vet", "./..."},
		{"-unknown", "unit", "go", "test", "./..."},
		{"unit", "go", "test", "-count=2", "./..."},
	} {
		var out, errb bytes.Buffer
		code := run(context.Background(), args, t.TempDir(), t.TempDir(), &out, &errb)
		if code != testreuse.ExitUsage {
			t.Errorf("%q exited %d, want %d: %s", args, code, testreuse.ExitUsage, errb.String())
		}
		if !strings.Contains(errb.String(), "check-tests") && !strings.Contains(errb.String(), "flag") {
			t.Errorf("%q says nothing useful: %s", args, errb.String())
		}
	}
}
