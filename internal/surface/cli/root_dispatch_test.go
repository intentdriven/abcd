package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/record"
)

// TestRootDispatchJSONContract is the spc-26 surface AC: `abcd <id> --json`
// renders the Description fields for a record found in its store.
func TestRootDispatchJSONContract(t *testing.T) {
	_ = captureLedgerRepo(t)

	capOut := runCLI(t, "capture", "a dispatchable observation", "--json")
	var minted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}

	out := runCLI(t, minted.ID, "--json")
	var d struct {
		ID        string   `json:"id"`
		Family    string   `json:"family"`
		Status    string   `json:"status"`
		Path      string   `json:"path"`
		NextMoves []string `json:"next_moves"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatalf("dispatch output not JSON: %v\n%s", err, out)
	}
	if d.ID != minted.ID || d.Family != "issue" || d.Status != "open" {
		t.Fatalf("unexpected description: %+v", d)
	}
	if d.Path == "" || filepath.IsAbs(d.Path) {
		t.Fatalf("path must be repo-relative: %+v", d)
	}
	if len(d.NextMoves) == 0 || !strings.Contains(strings.Join(d.NextMoves, " "), "capture promote "+minted.ID) {
		t.Fatalf("open issue next moves wrong: %v", d.NextMoves)
	}
}

// TestRootDispatchUnknownIDFaults: a shape-matching id in no store exits
// non-zero with a diagnostic naming the id, and never renders success.
func TestRootDispatchUnknownIDFaults(t *testing.T) {
	_ = captureLedgerRepo(t)
	out, err := runCLIErr(t, "itd-42")
	if err == nil {
		t.Fatalf("dispatch of an absent id must fail:\n%s", out)
	}
	if !strings.Contains(err.Error(), "itd-42") {
		t.Fatalf("fault must name the id, got: %v", err)
	}
}

// TestRootNonIDPositionalUnchanged is the golden test: a positional that does
// not match the record-id shape reproduces the pre-change unknown-command
// error byte-for-byte, exit code 2.
func TestRootNonIDPositionalUnchanged(t *testing.T) {
	_ = captureLedgerRepo(t)
	for _, arg := range []string{"nonsense", "status", "iss-", "itd-x", "plan-3", "adr-40-slug"} {
		_, err := runCLIErr(t, arg)
		if err == nil {
			t.Fatalf("positional %q must stay on the unknown-command path", arg)
		}
		want := "unknown command \"" + arg + "\" for \"abcd\""
		if err.Error() != want {
			t.Fatalf("error for %q = %q, want %q (byte-for-byte)", arg, err.Error(), want)
		}
		var exitErr *exitError
		if !errors.As(err, &exitErr) || exitErr.Code != 2 {
			t.Fatalf("positional %q must exit 2, got %v", arg, err)
		}
	}
}

// TestRootDispatchZeroWrites: describing a record performs zero writes — the
// tree is byte-identical after the run.
func TestRootDispatchZeroWrites(t *testing.T) {
	repo := captureLedgerRepo(t)
	capOut := runCLI(t, "capture", "watch for stray writes", "--json")
	var minted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}

	before := map[string]string{}
	_ = filepath.Walk(repo, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		b, _ := os.ReadFile(path)
		before[path] = string(b)
		return nil
	})

	runCLI(t, minted.ID)
	runCLI(t, minted.ID, "--json")

	n := 0
	_ = filepath.Walk(repo, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		n++
		b, _ := os.ReadFile(path)
		if before[path] != string(b) {
			t.Fatalf("dispatch mutated %s", path)
		}
		return nil
	})
	if n != len(before) {
		t.Fatalf("dispatch changed the file count: %d -> %d", len(before), n)
	}
}

// TestNextMoveVerbsResolveInLiveTree is the anti-drift test the spec walks
// into the ACs: every verb path the next-move table can recommend must
// resolve to a registered command in the cobra tree — a rename breaks this
// test instead of shipping stale advice.
func TestNextMoveVerbsResolveInLiveTree(t *testing.T) {
	root := NewRootCommand()
	for _, path := range record.RecommendedVerbPaths() {
		cmd := root
		for _, part := range strings.Fields(path) {
			var next bool
			for _, sub := range cmd.Commands() {
				if sub.Name() == part {
					cmd, next = sub, true
					break
				}
			}
			if !next {
				t.Fatalf("recommended verb path %q does not resolve at %q in the live tree", path, part)
			}
		}
	}
}
