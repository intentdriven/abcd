package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The surface half of the outbound gate. The core primitive is proven in
// internal/adapter/scanner; what these hold is that the front door exists, that
// its exit codes are the ones a CI gate branches on, and that it does not
// republish the leak it reports.

// A synthetic opaque session id. It is built here rather than imported from the
// scanner's test helpers (different package) and is checked for the property that
// makes it exercise the detector: base62 carrying both a digit and an upper-case
// letter.
const testOutboundSessionID = "qNs22jeg43nekhIpcwrcSr"

// A commit message carrying a live session URL must be REFUSED at exit 1 — the
// gap that let one reach three commit messages and two pull-request bodies of a
// managed public repo (iss-2609061438431625).
func TestLintOutboundRefusesASessionURLOnStdin(t *testing.T) {
	msg := "fix: the walk skips a record family\n\n" +
		"https://agent-host.dev/code/session_" + testOutboundSessionID + "\n\n" +
		"Assisted-by: Claude:claude-opus-5\n"

	var stdout, stderr bytes.Buffer
	code := runOutbound(t, msg, &stdout, &stderr, "lint", "outbound", "--label", "commit-message")
	if code != 1 {
		t.Fatalf("want exit 1 for a refused artefact, got %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "commit-message") {
		t.Errorf("the verdict must name the artefact it judged:\n%s", stdout.String())
	}
	// The gate must not print the leak back out. A CI log on a public repository
	// is public text, so echoing the session URL would publish the live handle the
	// gate exists to catch.
	if strings.Contains(stdout.String()+stderr.String(), testOutboundSessionID) {
		t.Errorf("the verdict republished the session id it was refusing:\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
}

// A clean commit message passes at exit 0. The gate runs on every commit of every
// pull request, so a false red is the failure that gets it switched off.
func TestLintOutboundPassesACleanArtefact(t *testing.T) {
	msg := "fix: the regime operator-surface walk skips the readings record family\n\n" +
		"Assisted-by: Claude:claude-opus-5\n"

	var stdout, stderr bytes.Buffer
	code := runOutbound(t, msg, &stdout, &stderr, "lint", "outbound", "--label", "commit-message")
	if code != 0 {
		t.Fatalf("want exit 0 for a clean artefact, got %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "✓") {
		t.Errorf("a clean artefact must say so:\n%s", stdout.String())
	}
}

// Reading the artefact from a FILE is the shape the commit half of the CI gate
// uses: it writes each message to a temp file rather than piping, so one unreadable
// artefact cannot be mistaken for an empty one.
func TestLintOutboundReadsAFilePositional(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.md")
	body := "Closes the gate.\n\n🤖 Generated with [Some Tool](https://sometool.dev)\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runOutbound(t, "", &stdout, &stderr, "lint", "outbound", "--label", "pr-body", path)
	if code != 1 {
		t.Fatalf("want exit 1 for a body carrying a tool footer, got %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
}

// An EMPTY artefact is a FAULT (exit 2), never a pass. "Nothing was checked" and
// "what was checked is clean" must not look the same to a gate's caller: that
// equivalence is how a misrouted pipe becomes a green tick.
func TestLintOutboundRefusesAnEmptyArtefactAsAFault(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runOutbound(t, "   \n", &stdout, &stderr, "lint", "outbound")
	if code != 2 {
		t.Fatalf("want exit 2 for an empty artefact, got %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
}

// A degraded scanner configuration exits 2, not 1. A caller that reads 1 as "the
// text is bad" must not be handed 1 when the check never ran at all.
func TestLintOutboundExitsTwoOnADegradedScannerConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config", "pii.json"), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runOutbound(t, "Ordinary text.\n", &stdout, &stderr, "lint", "outbound", "--root", root)
	if code != 2 {
		t.Fatalf("want exit 2 when the scan could not run, got %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
}

// --json emits ONE document carrying the findings and the policy, and the refusal
// arrives as the exit status alone (the report is already the machine-readable
// output, so Run adds no error envelope on top of it).
func TestLintOutboundJSONReportIsOneDocument(t *testing.T) {
	msg := "fix: something\n\nhttps://agent-host.dev/code/session_" + testOutboundSessionID + "\n"

	var stdout, stderr bytes.Buffer
	code := runOutbound(t, msg, &stdout, &stderr, "lint", "outbound", "--label", "pr-body", "--json")
	if code != 1 {
		t.Fatalf("want exit 1, got %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	// The report IS the machine-readable output, so the refusal must arrive as the
	// exit status alone: no second message anywhere.
	if strings.TrimSpace(stderr.String()) != "" {
		t.Errorf("a --json run wrote a second verdict to stderr:\n%s", stderr.String())
	}
	var report struct {
		Label    string `json:"label"`
		Policy   string `json:"policy"`
		Findings []struct {
			Kind    string `json:"kind"`
			Matched string `json:"matched"`
		} `json:"findings"`
	}
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	if err := dec.Decode(&report); err != nil {
		t.Fatalf("--json output is not one JSON document: %v\nstdout: %q", err, stdout.String())
	}
	if dec.More() {
		t.Errorf("--json emitted more than one document; a report plus an envelope is two verdicts:\n%s", stdout.String())
	}
	if report.Label != "pr-body" {
		t.Errorf("report label: want pr-body, got %q", report.Label)
	}
	if report.Policy == "" {
		t.Errorf("the report must carry the policy that refused it:\n%s", stdout.String())
	}
	if len(report.Findings) == 0 {
		t.Fatalf("the report carries no findings:\n%s", stdout.String())
	}
	// Finding.MarshalJSON masks the matched span. Proven here rather than trusted,
	// because this is the surface a CI job archives.
	for _, f := range report.Findings {
		if strings.Contains(f.Matched, testOutboundSessionID) {
			t.Errorf("the serialised finding carries the raw session id: %q", f.Matched)
		}
	}
}

// runOutbound runs the verb with stdin bound to the given text, from a scratch
// working directory so the ambient repository's own configuration plays no part.
func runOutbound(t *testing.T, stdin string, stdout, stderr *bytes.Buffer, args ...string) int {
	t.Helper()
	// Only bind a scratch cwd when --root was not given: the --root cases name
	// their own directory and must not be redirected.
	hasRoot := false
	for _, a := range args {
		if a == "--root" {
			hasRoot = true
		}
	}
	if !hasRoot {
		t.Chdir(t.TempDir())
	}
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetIn(strings.NewReader(stdin))
	err := root.Execute()
	if err == nil {
		return 0
	}
	code := 1
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		code = coded.ExitCode()
	}
	// Mirror Run's error surface, so a test sees the diagnostic line a caller sees
	// rather than an error value the process never prints. An empty message means
	// the command already rendered its report and only the code propagates — which
	// is exactly the claim TestLintOutboundJSONReportIsOneDocument depends on, and
	// why that test also asserts stderr stayed empty.
	if msg := scrubPaths(err); msg != "" {
		stderr.WriteString("abcd: " + msg + "\n")
	}
	return code
}
