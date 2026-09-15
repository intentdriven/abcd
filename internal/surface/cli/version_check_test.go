package cli

import (
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

// recordingFetcher counts every LatestTag call, so a test can assert exactly
// which command paths reach the network.
type recordingFetcher struct{ calls *int32 }

func (r recordingFetcher) LatestTag() (string, error) {
	atomic.AddInt32(r.calls, 1)
	return "v99.0.0", nil
}

// TestOnlyVersionCheckTouchesTheNetwork is the zero-network invariant (AC4,
// adr-38 tier 1): every implicit path reaches the disk only, and the network is
// touched exactly once, by the explicit --check.
func TestOnlyVersionCheckTouchesTheNetwork(t *testing.T) {
	var calls int32
	orig := newReleaseFetcher
	newReleaseFetcher = func() vintage.ReleaseFetcher { return recordingFetcher{&calls} }
	t.Cleanup(func() { newReleaseFetcher = orig })

	// Every implicit path: none may fetch.
	runCLI(t, "version")
	runCLI(t, "ahoy")
	if _, err := runCLIStdinErr(t, `{"cwd":"`+t.TempDir()+`"}`, "hook", "session-start"); err != nil {
		t.Fatalf("session-start hook errored: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("an implicit path fetched %d time(s); every path except --check must be disk-only", got)
	}

	// The explicit check fetches exactly once and names its source.
	out := runCLI(t, "version", "--check")
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("version --check fetched %d time(s), want exactly 1", got)
	}
	if !strings.Contains(string(out), checkSource) {
		t.Fatalf("check output did not name its source %q:\n%s", checkSource, out)
	}
	if !strings.Contains(string(out), "v99.0.0") {
		t.Fatalf("check output did not report the latest release:\n%s", out)
	}
}

// TestVersionCheckNamesTheNextStep pins the line itd-130 promised: when an
// update is available, `version --check` says what to run next, and the verb it
// names depends on the install shape the update verb itself classifies — the
// swappable copy is pointed at `abcd update`, a plugin-root binary at the
// host's plugin update. The classification is disk-only; the check's single
// fetch stays the only network touch (iss-2609012111168872).
func TestVersionCheckNamesTheNextStep(t *testing.T) {
	var calls int32
	origFetcher := newReleaseFetcher
	newReleaseFetcher = func() vintage.ReleaseFetcher { return recordingFetcher{&calls} }
	origVersion := core.Version
	core.Version = "v0.1.0"
	origResolve := resolveUpdateTarget
	t.Cleanup(func() {
		newReleaseFetcher = origFetcher
		core.Version = origVersion
		resolveUpdateTarget = origResolve
	})

	resolveUpdateTarget = func() ahoy.UpdateTarget {
		return ahoy.UpdateTarget{Path: "/x/abcd", ResolvedPath: "/x/abcd", Kind: ahoy.UpdateTargetFile}
	}
	out := string(runCLI(t, "version", "--check"))
	if !strings.Contains(out, "update available: v0.1.0 -> v99.0.0") {
		t.Fatalf("expected the update verdict:\n%s", out)
	}
	if !strings.Contains(out, "next:      run `abcd update`") {
		t.Errorf("a swappable install must be pointed at the update verb:\n%s", out)
	}

	var got struct {
		Check struct {
			Verdict  string `json:"verdict"`
			NextStep string `json:"next_step"`
		} `json:"check"`
	}
	if err := json.Unmarshal(runCLI(t, "version", "--check", "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Check.NextStep, "`abcd update`") {
		t.Errorf("--json must carry the next step additively, got check=%+v", got.Check)
	}

	resolveUpdateTarget = func() ahoy.UpdateTarget {
		return ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetPluginRoot}
	}
	out = string(runCLI(t, "version", "--check"))
	if !strings.Contains(out, "next:      take a plugin update in the host") {
		t.Errorf("a plugin-root install must be pointed at the host's plugin update:\n%s", out)
	}
	if strings.Contains(out, "abcd update`") {
		t.Errorf("a plugin-root install must not be told to run the update verb:\n%s", out)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("three checks fetched %d time(s); classifying the install must add no network touch", got)
	}
}
