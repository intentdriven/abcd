package gitleaks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// augValue is a value the native scanner does not see, so a finding for it came
// from the (fake) gitleaks run.
const augValue = "plumbob-harvest-quiet-lantern"

func TestAugmenterForAnUnarmedRepoIsNilAndInvokesNothing(t *testing.T) {
	looked := false
	a := &Adapter{LookPath: func(string) (string, error) { looked = true; return "", errors.New("no") }, Runner: &fakeRunner{}}
	if got := a.AugmenterFor(t.TempDir()); got != nil {
		t.Fatalf("an unarmed repository got an augmenter: %#v", got)
	}
	if looked {
		t.Error("an unarmed repository looked the binary up")
	}
}

func TestAugmenterNotFoundMatchesTheScannerSentinel(t *testing.T) {
	repo := t.TempDir()
	writeConfig(t, repo, `{"schema_version":1,"enabled":true}`)
	a := &Adapter{LookPath: missingLookPath, Runner: &fakeRunner{}}
	err := a.AugmenterFor(repo).Available()
	if !errors.Is(err, scanner.ErrAugmenterNotFound) || !errors.Is(err, ErrConfiguredNotFound) {
		t.Fatalf("Available() = %v, want the not-found sentinel of both packages", err)
	}
	if !strings.Contains(err.Error(), "gitleaks configured but not found") {
		t.Errorf("the error does not name the opt-in: %v", err)
	}
}

func TestAugmenterRefusalsAreNotAGap(t *testing.T) {
	for name, body := range map[string]string{
		"relative path": `{"schema_version":1,"enabled":true,"path":"bin/gitleaks"}`,
		"broken config": `{"enabled":`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			writeConfig(t, repo, body)
			a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{}}
			err := a.AugmenterFor(repo).Available()
			if err == nil || errors.Is(err, scanner.ErrAugmenterNotFound) {
				t.Fatalf("Available() = %v, want a refusal that is not the not-found gap", err)
			}
		})
	}
}

func TestAugmenterScanFailureIsSticky(t *testing.T) {
	repo := t.TempDir()
	writeConfig(t, repo, `{"schema_version":1,"enabled":true}`)
	a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{err: errors.New("boom")}}
	aug := a.AugmenterFor(repo)
	if err := aug.Available(); err != nil {
		t.Fatalf("armed augmenter unavailable before a run: %v", err)
	}
	if got := aug.Scan("text", "doc.md"); got != nil {
		t.Fatalf("a failed run returned findings: %v", got)
	}
	if err := aug.Available(); err == nil {
		t.Fatal("a failed run left the augmenter available")
	}
}

func TestAugmenterFindingsReachTheScanner(t *testing.T) {
	repo := t.TempDir()
	writeConfig(t, repo, `{"schema_version":1,"enabled":true}`)
	a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{
		report: `[{"RuleID":"generic-api-key","Secret":"` + augValue + `"}]`}}
	sc, err := scanner.New(repo, scanner.WithAugmenter(a.AugmenterFor(repo)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range sc.ScanText("key "+augValue, "doc.md") {
		if f.Matched == augValue && f.Kind == "gitleaks:generic-api-key" {
			return
		}
	}
	t.Fatal("the gitleaks finding did not reach ScanText")
}

// writeStub writes an executable shell script named gitleaks that stands in
// for the real binary. No real gitleaks is ever run.
func writeStub(t *testing.T, body string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gitleaks")
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin
}

// TestExecRunnerRunsInTheIsolatedEnvironment: the runner hands the binary the
// canonical isolated environment (gitutil.IsolatedEnv), not the raw parent one,
// so an inherited GIT_DIR never reaches it.
func TestExecRunnerRunsInTheIsolatedEnvironment(t *testing.T) {
	t.Setenv("GIT_DIR", "/nonexistent/abcd-gitdir")
	bin := writeStub(t, `rep=""
while [ $# -gt 0 ]; do
  if [ "$1" = "--report-path" ]; then rep="$2"; fi
  shift
done
printf '[{"RuleID":"env-%s","Secret":"x"}]' "${GIT_DIR:-scrubbed}" > "$rep"`)
	raw, err := execRunner{}.Run(context.Background(), bin, "x")
	if err != nil {
		t.Fatal(err)
	}
	reps, err := parseReport(raw)
	if err != nil || len(reps) != 1 {
		t.Fatalf("report %q: %v", raw, err)
	}
	if reps[0].RuleID != "env-scrubbed" {
		t.Fatalf("the binary saw the parent's GIT_DIR: %s", reps[0].RuleID)
	}
}

// TestExecRunnerFailureDoesNotEchoItsOutput: the binary's own output is
// untrusted and may carry what it found, so a failed run's error names the
// exit, never the output.
func TestExecRunnerFailureDoesNotEchoItsOutput(t *testing.T) {
	bin := writeStub(t, `echo "found `+augValue+`"; echo "found `+augValue+`" >&2; exit 3`)
	_, err := execRunner{}.Run(context.Background(), bin, "x")
	if err == nil {
		t.Fatal("a failing binary reported success")
	}
	if strings.Contains(err.Error(), augValue) {
		t.Fatalf("the error echoes the binary's output: %v", err)
	}
}
