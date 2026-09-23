package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// implementRepo stands up a committed repository under a temporary HOME — the
// run state lands under that HOME, never the real ~/.abcd — and changes into it.
// It returns the HOME and the run directory the verbs should use.
func implementRepo(t *testing.T) (home, runDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	// This repository's own preset file, so the second session's reading-corpus
	// bound is derived as the live run derives it.
	presets, err := os.ReadFile(filepath.Join(implementPkgDir, "..", "..", "..", ".abcd", "config", "reading-presets.json"))
	if err != nil {
		t.Fatal(err)
	}
	repo.Write(".abcd/config/reading-presets.json", string(presets))
	repo.Commit("init")
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
	t.Chdir(repo.Root())
	return home, filepath.Join(home, ".abcd", "runs", sha)
}

// implementCLI runs one invocation and returns its exit code and streams.
func implementCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// mustImplement runs an invocation that must succeed.
func mustImplement(t *testing.T, args ...string) string {
	t.Helper()
	code, out, errOut := implementCLI(t, args...)
	if code != 0 {
		t.Fatalf("abcd %s exited %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, out, errOut)
	}
	return out
}

// refusalEnvelope asserts a --json refusal on stdout with the given exit code
// and returns its message.
func refusalEnvelope(t *testing.T, want int, args ...string) string {
	t.Helper()
	code, out, errOut := implementCLI(t, args...)
	if code != want {
		t.Fatalf("abcd %s exited %d, want %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, want, out, errOut)
	}
	var env struct {
		Abcd     string `json:"abcd"`
		Error    string `json:"error"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil || env.Abcd != "error" || env.ExitCode != want {
		t.Fatalf("refusal is not a JSON envelope on stdout (%v): %q", err, out)
	}
	if strings.TrimSpace(errOut) != "" {
		t.Fatalf("a --json refusal wrote prose to stderr: %q", errOut)
	}
	return env.Error
}

// TestImplementBareRendersAndCreatesNothing: the bare verb is a read-only render
// of a run nobody has started, and leaves the store uncreated.
func TestImplementBareRendersAndCreatesNothing(t *testing.T) {
	home, _ := implementRepo(t)
	out := mustImplement(t, "implement", "--json")
	var st struct {
		Dir      string            `json:"dir"`
		Window   *json.RawMessage  `json:"window"`
		Sessions []json.RawMessage `json:"sessions"`
		Claims   []json.RawMessage `json:"claims"`
	}
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatalf("bare --json: %v\n%s", err, out)
	}
	if !strings.HasPrefix(st.Dir, "~/.abcd/runs/") || st.Sessions == nil || st.Claims == nil || st.Window != nil {
		t.Fatalf("bare render = %+v", st)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bare implement created ~/.abcd: %v", err)
	}
	if code, _, _ := implementCLI(t, "implement", "report"); code != 0 {
		t.Fatalf("report on an empty run exited %d", code)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("report created ~/.abcd: %v", err)
	}
}

// TestImplementTwoSessionsShareARun drives the whole surface the way two
// sessions would: join, a window, a claim each, the contention and the bounds,
// the log, the release, and the comparison derived from what they wrote.
func TestImplementTwoSessionsShareARun(t *testing.T) {
	_, runDir := implementRepo(t)
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	mustImplement(t, "implement", "join", "--session", "beta", "--role", "second", "--model", "opus", "--json")
	mustImplement(t, "implement", "mode", "claim", "--session", "alpha", "--window", "1", "--json")

	out := mustImplement(t, "implement", "claim", "itd-1", "--session", "alpha", "--lane", "one", "--json")
	var granted struct {
		Claim struct{ Record, Session, Lane string }
	}
	if err := json.Unmarshal([]byte(out), &granted); err != nil || granted.Claim.Session != "alpha" {
		t.Fatalf("claim --json = %q (%v)", out, err)
	}
	if _, err := os.Stat(filepath.Join(runDir, "claims", "itd-1.json")); err != nil {
		t.Fatalf("the claim file is not in the run state: %v", err)
	}

	// Contention: exit 3, naming the holder.
	if msg := refusalEnvelope(t, 3, "implement", "claim", "itd-1", "--session", "beta", "--lane", "two", "--json"); !strings.Contains(msg, "alpha") {
		t.Fatalf("contention message does not name the holder: %q", msg)
	}
	// The bounds: one lane, never the release.
	mustImplement(t, "implement", "claim", "itd-2", "--session", "beta", "--lane", "two", "--json")
	refusalEnvelope(t, 2, "implement", "claim", "itd-3", "--session", "beta", "--lane", "three", "--json")
	refusalEnvelope(t, 2, "implement", "check", "release", "--session", "beta", "--json")
	if msg := refusalEnvelope(t, 2, "implement", "check", "lane", "--session", "beta", "--path", "internal/surface/cli/reading.go", "--json"); !strings.Contains(msg, "is in the reading corpus") {
		t.Fatalf("corpus refusal = %q; want the corpus named as the reason", msg)
	}
	mustImplement(t, "implement", "check", "lane", "--session", "beta", "--path", "internal/surface/cli/implement.go", "--json")
	mustImplement(t, "implement", "check", "review", "--session", "beta", "--json")
	mustImplement(t, "implement", "check", "release", "--session", "alpha", "--json")

	mustImplement(t, "implement", "log", "lane_open", "--session", "beta", "--field", "lane=two", "--field", "record=itd-2", "--json")
	mustImplement(t, "implement", "log", "backoff", "--session", "beta", "--field", "on=queue", "--field", "reason=queue busy", "--field", "minutes=4", "--json")
	mustImplement(t, "implement", "log", "lane_close", "--session", "beta", "--field", "lane=two", "--field", "outcome=merged", "--json")
	refusalEnvelope(t, 2, "implement", "log", "claim", "--session", "beta", "--json")
	refusalEnvelope(t, 2, "implement", "log", "stop", "--session", "beta", "--field", "noequals", "--json")

	mustImplement(t, "implement", "release", "itd-2", "--session", "beta", "--json")
	refusalEnvelope(t, 2, "implement", "release", "itd-1", "--session", "beta", "--json")
	mustImplement(t, "implement", "leave", "--session", "beta", "--reason", "window", "--json")

	out = mustImplement(t, "implement", "--json")
	if !strings.Contains(out, `"record": "itd-1"`) || strings.Contains(out, `"session": "beta"`) {
		t.Fatalf("bare render after the window:\n%s", out)
	}

	out = mustImplement(t, "implement", "report", "--json")
	var rep struct {
		Unparsed []json.RawMessage
		Modes    []struct {
			Mode              string
			Collisions        int
			LanesLanded       int     `json:"lanes_landed"`
			SecondLanesLanded int     `json:"second_session_lanes_landed"`
			BackoffMinutes    float64 `json:"backoff_minutes"`
			Refusals          int
		}
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("report --json: %v\n%s", err, out)
	}
	// The joins precede the window line, so they fall in an "unset" window unless
	// they share its second; everything the test counts is after it.
	if len(rep.Unparsed) != 0 {
		t.Fatalf("report = %s", out)
	}
	var claim = rep.Modes[len(rep.Modes)-1]
	for _, m := range rep.Modes[:len(rep.Modes)-1] {
		if m.Mode != "unset" {
			t.Fatalf("unexpected mode %q in %s", m.Mode, out)
		}
	}
	if claim.Mode != "claim" || claim.Collisions != 1 || claim.LanesLanded != 1 || claim.SecondLanesLanded != 1 ||
		claim.BackoffMinutes != 4 || claim.Refusals != 3 {
		t.Fatalf("claim mode = %+v", claim)
	}
}

// TestImplementWritersRefuseMalformedInvocations: every writer refuses what it
// does not recognise at exit 2, as a JSON envelope, with nothing written.
func TestImplementWritersRefuseMalformedInvocations(t *testing.T) {
	home, _ := implementRepo(t)
	for _, args := range [][]string{
		{"implement", "claim", "itd-1", "--lane", "l", "--json"},
		{"implement", "join", "--session", "alpha", "--role", "third", "--json"},
		{"implement", "mode", "pairs", "--session", "alpha", "--json"},
		{"implement", "check", "ship", "--session", "alpha", "--json"},
		{"implement", "log", "stop", "--json"},
		{"implement", "report", "--date", "yesterday", "--json"},
		{"implement", "report", "--date", "2026-09-23", "--log", "x.jsonl", "--json"},
	} {
		refusalEnvelope(t, 2, args...)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused invocation created ~/.abcd: %v", err)
	}
	// Unjoined: a session no run knows is refused before anything is created —
	// no run directory, no lock, no log.
	for _, args := range [][]string{
		{"implement", "log", "stop", "--session", "ghost", "--json"},
		{"implement", "claim", "itd-1", "--session", "ghost", "--lane", "l", "--json"},
		{"implement", "release", "itd-1", "--session", "ghost", "--json"},
		{"implement", "check", "lane", "--session", "ghost", "--json"},
		{"implement", "mode", "claim", "--session", "ghost", "--json"},
		{"implement", "leave", "--session", "ghost", "--json"},
	} {
		refusalEnvelope(t, 2, args...)
		if _, err := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("abcd %s created ~/.abcd: %v", strings.Join(args, " "), err)
		}
	}
	// In a run that exists, the ghost still writes nothing.
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	refusalEnvelope(t, 2, "implement", "claim", "itd-1", "--session", "ghost", "--lane", "l", "--json")
	out := mustImplement(t, "implement", "--json")
	if strings.Contains(out, "itd-1") || strings.Contains(out, "ghost") {
		t.Fatalf("an unjoined claim left state behind:\n%s", out)
	}
}

// TestImplementRefusesOutsideACheckout: with no repository there is no root
// commit to key a run on, so nothing is written anywhere.
func TestImplementRefusesOutsideACheckout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	gittest.Env(t)
	t.Chdir(t.TempDir())
	refusalEnvelope(t, 2, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	refusalEnvelope(t, 2, "implement", "--json")
	if _, err := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refusal outside a checkout created ~/.abcd: %v", err)
	}
}

// TestImplementReportReadsANamedLog: --log derives the comparison from a log
// kept anywhere, the run's hand-kept file included.
func TestImplementReportReadsANamedLog(t *testing.T) {
	implementRepo(t)
	pkgFixture := filepath.Join(implementPkgDir, "..", "..", "core", "implement", "testdata", "run-log.jsonl")
	out := mustImplement(t, "implement", "report", "--log", pkgFixture, "--json")
	var rep struct {
		Leader string
		Modes  []struct{ Mode string }
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil || rep.Leader != "claim" || len(rep.Modes) != 4 {
		t.Fatalf("report --log = %+v (%v)\n%s", rep, err, out)
	}
	text := mustImplement(t, "implement", "report", "--log", pkgFixture)
	if !strings.Contains(text, "split-roles") || !strings.Contains(text, "most lanes landed per wall-clock hour: claim") {
		t.Fatalf("report text:\n%s", text)
	}
}

// implementPkgDir is this package's directory, captured before any test changes
// the working directory.
var implementPkgDir = func() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}()
