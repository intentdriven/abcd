package cli

import (
	"encoding/json"
	"reflect"
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

// TestOnlyUpdateCheckTouchesTheNetwork is the zero-network invariant (AC4,
// adr-38 tier 1): every implicit path reaches the disk only, and the network is
// touched exactly once, by the explicit `update --check` — which does what
// `version --check` did (itd-2609212130136102) and never constructs the
// updater that swaps a binary.
func TestOnlyUpdateCheckTouchesTheNetwork(t *testing.T) {
	var calls int32
	orig := newReleaseFetcher
	newReleaseFetcher = func() vintage.ReleaseFetcher { return recordingFetcher{&calls} }
	updaters := 0
	origUpdater := newUpdater
	newUpdater = func() updater { updaters++; return origUpdater() }
	t.Cleanup(func() { newReleaseFetcher = orig; newUpdater = origUpdater })

	// Every implicit path: none may fetch.
	runCLI(t, "--version")
	runCLI(t, "ahoy")
	if _, err := runCLIStdinErr(t, `{"cwd":"`+t.TempDir()+`"}`, "hook", "session-start"); err != nil {
		t.Fatalf("session-start hook errored: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("an implicit path fetched %d time(s); every path except --check must be disk-only", got)
	}

	// The explicit check fetches exactly once and names its source.
	out := runCLI(t, "update", "--check")
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("update --check fetched %d time(s), want exactly 1", got)
	}
	if updaters != 0 {
		t.Fatalf("update --check constructed the updater %d time(s); the check swaps nothing", updaters)
	}
	if !strings.Contains(string(out), checkSource) {
		t.Fatalf("check output did not name its source %q:\n%s", checkSource, out)
	}
	if !strings.Contains(string(out), "v99.0.0") {
		t.Fatalf("check output did not report the latest release:\n%s", out)
	}
}

// TestUpdateCheckNamesTheNextStep pins the line itd-130 promised: when an
// update is available, `update --check` says what to run next, and the verb it
// names depends on the install shape the update verb itself classifies — the
// swappable copy is pointed at `abcd update`, a plugin-root binary at the
// host's plugin update. The classification is disk-only; the check's single
// fetch stays the only network touch (iss-2609012111168872).
func TestUpdateCheckNamesTheNextStep(t *testing.T) {
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
	out := string(runCLI(t, "update", "--check"))
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
	if err := json.Unmarshal(runCLI(t, "update", "--check", "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Check.NextStep, "`abcd update`") {
		t.Errorf("--json must carry the next step additively, got check=%+v", got.Check)
	}

	resolveUpdateTarget = func() ahoy.UpdateTarget {
		return ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetPluginRoot}
	}
	out = string(runCLI(t, "update", "--check"))
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

// tagFetcher answers one fixed tag, counting the fetches.
type tagFetcher struct {
	calls *int32
	tag   string
}

func (f tagFetcher) LatestTag() (string, error) {
	atomic.AddInt32(f.calls, 1)
	return f.tag, nil
}

// TestUpdateCheckTrustsTheUpdatePathsRoots: the fetcher the check builds carries
// a root pool of its own, as the update path's does, rather than leaving the
// choice to the transport's default.
func TestUpdateCheckTrustsTheUpdatePathsRoots(t *testing.T) {
	transport := reflect.ValueOf(newReleaseFetcher()).Elem().FieldByName("client").Elem().FieldByName("Transport").Elem().Elem()
	tlsConfig := transport.FieldByName("TLSClientConfig")
	if tlsConfig.IsNil() {
		t.Fatal("the check's fetcher has no TLS configuration of its own")
	}
	if tlsConfig.Elem().FieldByName("RootCAs").IsNil() {
		t.Fatal("the check's fetcher pins no root pool")
	}
}

// TestUpdateCheckReadsOnlyAReleaseTag: a reported tag without the shape the
// update path holds a tag to is read as not determined, in text and --json, and
// is not shown; a tag with the shape is reported as before. Each check fetches
// once.
func TestUpdateCheckReadsOnlyAReleaseTag(t *testing.T) {
	var calls int32
	var tag string
	origFetcher := newReleaseFetcher
	newReleaseFetcher = func() vintage.ReleaseFetcher { return tagFetcher{&calls, tag} }
	origVersion := core.Version
	core.Version = "v0.1.0"
	origResolve := resolveUpdateTarget
	resolveUpdateTarget = func() ahoy.UpdateTarget {
		return ahoy.UpdateTarget{Path: "/x/abcd", ResolvedPath: "/x/abcd", Kind: ahoy.UpdateTargetFile}
	}
	t.Cleanup(func() {
		newReleaseFetcher = origFetcher
		core.Version = origVersion
		resolveUpdateTarget = origResolve
	})
	type report struct {
		Check struct {
			Latest   string `json:"latest"`
			Source   string `json:"source"`
			Verdict  string `json:"verdict"`
			NextStep string `json:"next_step"`
		} `json:"check"`
	}
	check := func() (string, report) {
		t.Helper()
		out := string(runCLI(t, "update", "--check"))
		var got report
		if err := json.Unmarshal(runCLI(t, "update", "--check", "--json"), &got); err != nil {
			t.Fatal(err)
		}
		return out, got
	}
	checks := int32(0)
	for _, tag = range []string{
		"-v1.0.0", "v1.0.0%0aupdate", "v1 0 0", "<b>v1.0.0</b>", "v1.0.0\x1b[31m", "\u202ev1.0.0",
		"v" + strings.Repeat("9", 64), ".hidden",
	} {
		out, got := check()
		checks += 2
		if !strings.Contains(out, "could not be determined") || strings.Contains(out, "latest:    ") || strings.Contains(out, "next:") {
			t.Errorf("%q: want a latest release that could not be determined, got:\n%s", tag, out)
		}
		if got.Check.Latest != "" || got.Check.NextStep != "" || got.Check.Source != checkSource ||
			!strings.Contains(got.Check.Verdict, "could not be determined") {
			t.Errorf("%q: --json check = %+v, want not determined with its source", tag, got.Check)
		}
	}
	for _, tag = range []string{"v99.0.0", "v99.0.0-rc.1", "v99.0.0+build.2"} {
		out, got := check()
		checks += 2
		if !strings.Contains(out, "update available: v0.1.0 -> "+tag) || got.Check.Latest != tag || got.Check.NextStep == "" {
			t.Errorf("%q: want it reported as the latest release, got check=%+v:\n%s", tag, got.Check, out)
		}
	}
	tag = "v0.1.0"
	checks += 2
	if out, got := check(); !strings.Contains(out, "up to date") || got.Check.Latest != tag {
		t.Errorf("%q: want up to date, got check=%+v:\n%s", tag, got.Check, out)
	}
	if got := atomic.LoadInt32(&calls); got != checks {
		t.Errorf("%d checks fetched %d time(s)", checks, got)
	}
}
