package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// `abcd reserve --sweep` (iss-2610142200389875): the front door onto
// internal/core/reservesweep, whose judgement is proved there. These tests
// prove the verb drives it: a dry run by default that deletes nothing, --apply
// that deletes, its --json, and the usage refusals.

func sweepAt(t *testing.T, at time.Time) {
	t.Helper()
	reserveSweepNow = func() time.Time { return at }
	t.Cleanup(func() { reserveSweepNow = nil })
}

type sweepJSON struct {
	Mode          string `json:"mode"`
	OlderThanDays int    `json:"older_than_days"`
	Spent         int    `json:"spent"`
	Deleted       int    `json:"deleted"`
	Branches      []struct {
		Record  string `json:"record"`
		Branch  string `json:"branch"`
		Kind    string `json:"kind"`
		Spent   bool   `json:"spent"`
		Reason  string `json:"reason"`
		Outcome string `json:"outcome"`
	} `json:"branches"`
}

func TestReserveSweepIsADryRunUntilApplied(t *testing.T) {
	w := newReserveWorld(t)
	dir := w.clone(t, "copy", "1234+octo@users.noreply.github.com")
	if code, _, stderr := reserveRun(t, dir, "reserve", "iss-7", "--footprint", "x.go"); code != 0 {
		t.Fatalf("reserve: %d %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, dir, "reserve", "iss-7", "--release"); code != 0 {
		t.Fatalf("release: %d %s", code, stderr)
	}
	tip := w.remoteTip(t, "iss-7")
	sweepAt(t, time.Now().Add(8*24*time.Hour))

	code, stdout, stderr := reserveRun(t, dir, "reserve", "--sweep", "--json")
	if code != 0 {
		t.Fatalf("dry run: exit %d, %s", code, stderr)
	}
	var doc sweepJSON
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("dry run --json: %v\n%s", err, stdout)
	}
	if doc.Mode != "dry_run" || doc.OlderThanDays != 7 || doc.Spent != 1 || doc.Deleted != 0 || len(doc.Branches) != 1 ||
		doc.Branches[0].Branch != "abcd-reserved/iss-7" || doc.Branches[0].Reason != "released" || doc.Branches[0].Kind != "plain_tombstone" {
		t.Fatalf("dry run = %+v; want iss-7 listed spent as released and nothing deleted", doc)
	}
	if w.remoteTip(t, "iss-7") != tip {
		t.Fatal("a dry run moved or deleted the branch")
	}

	// A plain release does not wait the age, so a longer one still lists it.
	code, stdout, _ = reserveRun(t, dir, "reserve", "--sweep", "--older-than", "9", "--json")
	doc = sweepJSON{}
	if code != 0 || json.Unmarshal([]byte(stdout), &doc) != nil || doc.Spent != 1 || doc.Deleted != 0 || doc.Branches[0].Reason != "released" {
		t.Fatalf("a nine-day dry run: exit %d\n%s; want iss-7 still spent as released, nothing deleted", code, stdout)
	}

	code, stdout, stderr = reserveRun(t, dir, "reserve", "--sweep", "--apply")
	if code != 0 || !strings.Contains(stdout, "deleted") {
		t.Fatalf("--apply: exit %d\n%s%s", code, stdout, stderr)
	}
	if w.remoteTip(t, "iss-7") != "" {
		t.Fatal("--apply left the spent branch on origin")
	}
	if strings.Contains(stdout+stderr, w.home) {
		t.Fatalf("the render names the home directory:\n%s%s", stdout, stderr)
	}
}

func TestReserveSweepRefusesWhatItDoesNotTake(t *testing.T) {
	w := newReserveWorld(t)
	dir := w.clone(t, "copy", "1234+octo@users.noreply.github.com")
	for _, args := range [][]string{
		{"reserve", "--sweep", "iss-7"},
		{"reserve", "--sweep", "--release"},
		{"reserve", "--sweep", "--holder"},
		{"reserve", "--sweep", "--footprint", "x.go"},
		{"reserve", "--sweep", "--older-than", "0"},
		{"reserve", "iss-7", "--release", "--apply"},
		{"reserve", "--holder", "--older-than", "3"},
	} {
		code, _, stderr := reserveRun(t, dir, args...)
		if code != 2 || !strings.Contains(stderr, "nothing was") {
			t.Fatalf("%v: exit %d, %q; want exit 2 saying nothing was done", args, code, stderr)
		}
	}
}
