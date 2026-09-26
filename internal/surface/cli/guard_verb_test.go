package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// runGuard drives the guard verbs through the real exit-code mapping with stdin
// wired, and returns what a caller (a terminal or a host hook runner) would see:
// stdout, stderr, and the process exit code. The code is load-bearing on every
// path here — it is the whole protocol the hook speaks — so it is returned
// rather than asserted away.
func runGuard(stdin string, args ...string) (stdout, stderr string, code int) {
	root := NewRootCommand()
	root.SetArgs(args)
	var so, se bytes.Buffer
	root.SetOut(&so)
	root.SetErr(&se)
	root.SetIn(strings.NewReader(stdin))
	err := root.Execute()
	if err == nil {
		return so.String(), se.String(), 0
	}
	code = 1
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		code = coded.ExitCode()
	}
	// Mirror Run's error surface, so a test sees the diagnostic line a caller
	// sees rather than an error value the process never prints.
	if msg := scrubPaths(err); msg != "" {
		se.WriteString("abcd: " + msg + "\n")
	}
	return so.String(), se.String(), code
}

// guardRepo is a bare directory with a .abcd/ dir, so rulesRoot resolves there
// and the registry is the bundled default with no per-repo override. Chdir keeps
// the verb reading the fixture rather than the developer's own repo.
func guardRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return dir
}

// TestGuardCheckBlocksHazard is AC 2's blocker half at the CLI front door: a
// command matching a blocker entry is refused, the refusal names the safe
// successor and the plain-language why, and the exit code says "refused" (1 —
// an expected outcome), never 0.
func TestGuardCheckBlocksHazard(t *testing.T) {
	guardRepo(t)
	stdout, _, code := runGuard("", "guard", "check", "--command", "cd scratch && rm -rf *")

	if code == 0 {
		t.Errorf("a blocker match must exit non-zero; got 0 (stdout %q)", stdout)
	}
	if !strings.Contains(stdout, "rm-rf-after-cd-chain") {
		t.Errorf("the refusal must name the entry; stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "absolute path") {
		t.Errorf("the refusal must carry the safe successor; stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "the delete still runs") {
		t.Errorf("the refusal must carry the plain-language why; stdout = %q", stdout)
	}
}

// TestGuardCheckAllowsIncidentCapture is AC 3 at the front door: this repo's own
// incident-capture command names a hazard inside a quoted argument and must run.
// It is the known-good fixture the whole matching design exists for.
func TestGuardCheckAllowsIncidentCapture(t *testing.T) {
	guardRepo(t)
	capture := `abcd capture "agent ran cd scratch && rm -rf * — one failed cd from disaster"`
	stdout, stderr, code := runGuard("", "guard", "check", "--command", capture)

	if code != 0 {
		t.Errorf("the incident-capture command must be allowed; exit %d (stdout %q stderr %q)", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "allow") {
		t.Errorf("an allow must say so; stdout = %q", stdout)
	}
}

// TestGuardCheckWarnPasses is AC 2's warn half: a warn-tier match passes (exit 0)
// with the warning rendered, so the command runs and the lesson still lands.
func TestGuardCheckWarnPasses(t *testing.T) {
	guardRepo(t)
	stdout, _, code := runGuard("", "guard", "check", "--command", "git reset --hard origin/main")

	if code != 0 {
		t.Errorf("a warn match must exit 0; got %d (stdout %q)", code, stdout)
	}
	if !strings.Contains(stdout, "warn") || !strings.Contains(stdout, "git-reset-hard") {
		t.Errorf("the warning must be rendered and name its entry; stdout = %q", stdout)
	}
}

// TestGuardCheckReadsStdin covers the second input channel: with no --command the
// candidate is read from stdin, so a caller can pipe a command line in.
func TestGuardCheckReadsStdin(t *testing.T) {
	guardRepo(t)
	stdout, _, code := runGuard("git push --force origin main\n", "guard", "check")

	if code == 0 {
		t.Errorf("a blocker read from stdin must still refuse; exit 0 (stdout %q)", stdout)
	}
	if !strings.Contains(stdout, "git-push-force") {
		t.Errorf("stdin candidate must be evaluated; stdout = %q", stdout)
	}
}

// TestGuardCheckJSON holds the machine surface: --json emits the core decision
// verbatim, so another front door reads the same fields the text render shows.
func TestGuardCheckJSON(t *testing.T) {
	guardRepo(t)
	stdout, _, code := runGuard("", "guard", "check", "--json", "--command", "cd scratch && rm -rf *")

	if code == 0 {
		t.Errorf("--json must not change the exit code; got 0")
	}
	var got struct {
		Verdict   string `json:"verdict"`
		EntryID   string `json:"entry_id"`
		Successor string `json:"successor"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("--json must emit a JSON decision, got %q (%v)", stdout, err)
	}
	if got.Verdict != "block" || got.EntryID != "rm-rf-after-cd-chain" || got.Successor == "" {
		t.Errorf("decision JSON is incomplete: %+v", got)
	}
}

// TestGuardCheckUnparsableCommandIsAFault separates "refused" from "broken": a
// command line the tokenizer cannot split is a structural fault (exit 2), never
// a silent allow that would leave the caller believing the guard had cleared it.
func TestGuardCheckUnparsableCommandIsAFault(t *testing.T) {
	guardRepo(t)
	_, stderr, code := runGuard("", "guard", "check", "--command", `rm -rf "unterminated`)

	if code != 2 {
		t.Errorf("an unparsable command must be a fault (exit 2), got %d", code)
	}
	if !strings.Contains(stderr, "guard") {
		t.Errorf("the fault must be named on stderr; stderr = %q", stderr)
	}
}

// TestGuardCheckEmptyFlagDoesNotFallThroughToStdin pins the channel choice: an
// explicitly empty --command is an empty question, answered immediately. Falling
// through to stdin would leave the verb waiting on a terminal that never answers.
func TestGuardCheckEmptyFlagDoesNotFallThroughToStdin(t *testing.T) {
	guardRepo(t)
	_, stderr, code := runGuard("cd scratch && rm -rf *\n", "guard", "check", "--command", "")

	if code != 2 {
		t.Errorf("an empty --command is a usage fault (exit 2), got %d", code)
	}
	if !strings.Contains(stderr, "--command is empty") {
		t.Errorf("the fault must say the flag was empty, not read stdin; stderr = %q", stderr)
	}
}

// TestGuardCheckOversizedCandidateIsAFault closes the quietest way to get a
// clearance you did not earn: pad a command past the stdin cap and the guard used
// to answer on the prefix it happened to read. A candidate that does not fit is
// a candidate that was not checked.
func TestGuardCheckOversizedCandidateIsAFault(t *testing.T) {
	guardRepo(t)
	padded := strings.Repeat("# padding\n", (maxGuardStdinBytes/10)+1) + "cd scratch && rm -rf *\n"
	stdout, stderr, code := runGuard(padded, "guard", "check")

	if code != 2 {
		t.Errorf("a truncated candidate must be a fault (exit 2), got %d (stdout %q)", code, stdout)
	}
	if strings.Contains(stdout, "allow") {
		t.Errorf("a prefix must never be rendered as a clearance; stdout = %q", stdout)
	}
	if !strings.Contains(stderr, "too long") {
		t.Errorf("the fault must say the candidate did not fit; stderr = %q", stderr)
	}
}

// TestGuardCheckRefusesToAnswerFromADisabledRegistry closes the scriptable half
// of the kill switch. A CI job or script using this verb as a gate got a bare
// `allow`, exit 0, indistinguishable from a real clearance, for a command an
// entry blocks. A disabled registry means nothing was evaluated, which is the
// verb's own definition of a fault — not an answer.
func TestGuardCheckRefusesToAnswerFromADisabledRegistry(t *testing.T) {
	dir := guardRepo(t)
	commitGuardConfig(t, dir, `{"schema_version":1,"disabled":true,"entries":{}}`)
	stdout, stderr, code := runGuard("", "guard", "check", "--command", "cd scratch && rm -rf *")

	if code != 2 {
		t.Errorf("a disabled registry checked nothing and must be a fault (exit 2), got %d", code)
	}
	if strings.Contains(stdout, "allow") {
		t.Errorf("a disabled registry must never render as a clearance; stdout = %q", stdout)
	}
	if !strings.Contains(stderr, guard.RepoRelPath) {
		t.Errorf("the fault must name the file that turned the guard off; stderr = %q", stderr)
	}
}

// TestGuardCheckMalformedRepoConfigIsAFault holds the same line for the config:
// a broken .abcd/guard.json must be loud, never a fallback to "no guard".
func TestGuardCheckMalformedRepoConfigIsAFault(t *testing.T) {
	dir := guardRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runGuard("", "guard", "check", "--command", "ls")

	if code != 2 {
		t.Errorf("a malformed .abcd/guard.json must be a fault (exit 2), got %d", code)
	}
	if !strings.Contains(stderr, "guard") {
		t.Errorf("the fault must be named on stderr; stderr = %q", stderr)
	}
}

// A synthetic block over registry warns appends the winner LAST in Matches
// (blockers, warns, synthetics — guard.Decision's documented order), so the
// render must select "also matched" entries by id, not by position: indexing
// Matches[1:] dropped the matched warn and echoed the winner twice (iss-346).
func TestGuardCheckAlsoMatchedListsTheNonWinners(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	stdout, _, code := runGuard("", "guard", "check", "--command", `git clean -fd && env -S "a$b"`)
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (block maps to 1 on check)\n%s", code, stdout)
	}
	if !strings.Contains(stdout, "also matched: git-clean") {
		t.Fatalf("also-matched line lost the matched warn entry:\n%s", stdout)
	}
	if strings.Contains(stdout, "also matched: execute-string-uninspectable") {
		t.Fatalf("also-matched line echoes the winner:\n%s", stdout)
	}
}
