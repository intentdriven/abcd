package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/sessionkind"
)

// history_separation_surface_test.go — the front door onto the session
// separation check (adr-2609021016275803, spc-2609020626045177): `history
// separation` renders the report, and `history list` ends with its one line.

// TestHistorySeparationRendersUnobservedOnAnEmptyStore: a store with nothing in
// it says the property is unobserved, and never reads as clean.
func TestHistorySeparationRendersUnobservedOnAnEmptyStore(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"history", "separation"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history separation exited %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "unobserved") {
		t.Fatalf("an empty store rendered %q, want it unobserved", stdout.String())
	}

	stdout.Reset()
	if code := Run([]string{"history", "separation", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history separation --json exited %d: %s", code, stderr.String())
	}
	var rep history.SeparationReport
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("--json is not a report: %v\n%s", err, stdout.String())
	}
	if !rep.Unobserved || rep.Reason == "" {
		t.Fatalf("--json reported %+v, want unobserved with a reason", rep)
	}
}

// TestHistorySeparationNamesABreachAndExitsOne: a retained transcript holding
// both stamps of one run is named by session, and the verb exits 1 — a found
// breach is a finding, not a render.
func TestHistorySeparationNamesABreachAndExitsOne(t *testing.T) {
	repo, rootSHA := sessionEndRepo(t)
	t.Chdir(repo)
	digest := strings.Repeat("c", 64)
	reading, _ := sessionkind.Stamp(sessionkind.Reading, "rdg-5", digest)
	scribe, _ := sessionkind.Stamp(sessionkind.Scribe, "rdg-5", digest)
	if _, err := history.Capture(repo, rootSHA, []byte("tool: "+reading+"\ntool: "+scribe+"\n"),
		history.CaptureMeta{SessionID: "sess-held-both", Kind: "native"}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"history", "separation"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("a breach exited %d, want 1\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if out := stdout.String(); !strings.Contains(out, "sess-held-both") || !strings.Contains(out, "rdg-5") {
		t.Fatalf("the breach render %q does not name the session and the run", out)
	}

	// The list's text render ends with the same line, and its JSON stays an
	// array so a consumer of it is untouched.
	stdout.Reset()
	if code := Run([]string{"history", "list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history list exited %d: %s", code, stderr.String())
	}
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "session separation") || !strings.Contains(last, "sess-held-both") {
		t.Fatalf("history list ends with %q, want the separation line", last)
	}
	stdout.Reset()
	if code := Run([]string{"history", "list", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("history list --json exited %d: %s", code, stderr.String())
	}
	var records []history.Record
	if err := json.Unmarshal(stdout.Bytes(), &records); err != nil {
		t.Fatalf("history list --json is no longer an array: %v\n%s", err, stdout.String())
	}
}
