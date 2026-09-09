package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/history"
)

// TestHistoryCaptureAcceptsWhatTheHooksAccept is the recovery-path guarantee.
//
// `history capture` exists to recover a session the automatic path did not
// store, so a transcript the hooks would accept must not be refused here. It
// used to be: the verb read through the JSON-operand cap (8 MiB, sized for
// registry/graveyard payloads) while the SessionEnd path read through the
// transcript cap (64 MiB), leaving every transcript between the two capturable
// automatically but unrecoverable by hand — the recovery verb bounded eight
// times tighter than the thing it recovers from (iss-2608231029040602).
//
// This is not hypothetical: the transcript that exposed it is 11.8 MB of an
// ordinary session, so the old bound refused real work rather than a pathology.
// The fixture is just over 8 MiB, which fails against the old constant and
// passes against the transcript cap.
func TestHistoryCaptureAcceptsWhatTheHooksAccept(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)

	// Just over the old 8 MiB operand cap, well under the 64 MiB transcript cap.
	const size = (8 << 20) + (1 << 20)
	line := `{"role":"assistant","text":"` + strings.Repeat("x", 512) + `"}` + "\n"
	var b strings.Builder
	for b.Len() < size {
		b.WriteString(line)
	}
	tp := filepath.Join(t.TempDir(), "big.jsonl")
	if err := os.WriteFile(tp, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewRootCommand()
	root.SetArgs([]string{"history", "capture", "--session", "big-session", tp})
	var out, errb strings.Builder
	root.SetOut(&out)
	root.SetErr(&errb)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	if err := root.Execute(); err != nil {
		t.Fatalf("capturing a %d-byte transcript failed: %v (%s)", b.Len(), err, errb.String())
	}
	recs, err := history.List(repo, rootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].SessionID != "big-session" {
		t.Fatalf("an over-8-MiB transcript must be recoverable by hand, got %+v", recs)
	}
}
