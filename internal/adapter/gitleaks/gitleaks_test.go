package gitleaks

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// fakeRunner is an injected Runner that records whether it was invoked and
// returns a canned JSON report. No real gitleaks binary is spawned.
type fakeRunner struct {
	called  bool
	binPath string // the path the adapter asked it to execute
	report  string
	err     error
}

func (f *fakeRunner) Run(_ context.Context, binPath, _ /*text*/ string) ([]byte, error) {
	f.called = true
	f.binPath = binPath
	if f.err != nil {
		return nil, f.err
	}
	return []byte(f.report), nil
}

// foundLookPath returns a LookPath that resolves gitleaks to a real, executable
// regular file outside the test's repo root — the shape admitBinary admits —
// so the augmentation path is exercised without a real gitleaks binary (the
// injected fakeRunner never executes it).
func foundLookPath(t *testing.T) func(string) (string, error) {
	t.Helper()
	bin := writeExecutable(t, filepath.Join(t.TempDir(), "gitleaks"))
	return func(string) (string, error) { return bin, nil }
}

// missingLookPath is a LookPath that never finds the binary.
func missingLookPath(name string) (string, error) {
	return "", errors.New("exec: \"" + name + "\": not found")
}

// TestGateOffInvokesNothing is the load-bearing default-off guarantee: when the
// repo has not opted in (Enabled=false), Augment returns no findings, no error,
// and — critically — never touches the binary lookup or the runner. This is what
// makes the adapter zero-cost on the default path.
func TestGateOffInvokesNothing(t *testing.T) {
	runner := &fakeRunner{report: `[{"RuleID":"generic-api-key","Secret":"x"}]`}
	looked := false
	a := &Adapter{
		LookPath: func(s string) (string, error) { looked = true; return "/fake/gitleaks", nil },
		Runner:   runner,
	}

	findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: false}, "api_key = abcdef\n", "transcript")
	if err != nil {
		t.Fatalf("gate-off returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("gate-off returned %d findings, want 0", len(findings))
	}
	if runner.called {
		t.Error("gate-off invoked the runner; it must not spawn anything")
	}
	if looked {
		t.Error("gate-off performed a binary lookup; it must not")
	}
}

// TestConfiguredButAbsentLoudStages proves the loud-stage: opted in, but no
// gitleaks binary resolvable (not on PATH, no configured path). The adapter must
// return ErrConfiguredNotFound and name the opt-in, never a silent no-op.
func TestConfiguredButAbsentLoudStages(t *testing.T) {
	runner := &fakeRunner{}
	a := &Adapter{LookPath: missingLookPath, Runner: runner}

	_, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, "secret\n", "transcript")
	if err == nil {
		t.Fatal("configured-but-absent returned nil error; want a loud-stage error")
	}
	if !errors.Is(err, ErrConfiguredNotFound) {
		t.Fatalf("error is not ErrConfiguredNotFound: %v", err)
	}
	if !strings.Contains(err.Error(), "gitleaks configured but not found") {
		t.Fatalf("error message does not name the opt-in: %q", err.Error())
	}
	if runner.called {
		t.Error("runner was invoked despite the binary being absent")
	}
}

// TestConfiguredPathMissingLoudStages proves a configured path that does not
// point at a real file also loud-stages, without falling back to PATH.
func TestConfiguredPathMissingLoudStages(t *testing.T) {
	looked := false
	a := &Adapter{
		LookPath: func(s string) (string, error) { looked = true; return "/somewhere/gitleaks", nil },
		Runner:   &fakeRunner{},
	}
	_, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true, Path: "/no/such/gitleaks"}, "x\n", "transcript")
	if !errors.Is(err, ErrConfiguredNotFound) {
		t.Fatalf("configured-missing-path did not loud-stage: %v", err)
	}
	if looked {
		t.Error("a configured path must not fall back to PATH lookup")
	}
}

// TestAugmentConvertsFindings proves the augmentation path with a fake gitleaks:
// a labelled high-entropy value that the native prefix set misses (iss-96's
// residue) is reported by the fake runner and converted into a scanner.Finding
// positioned exactly on the value, so scanner.Redact masks it out of the text.
func TestAugmentConvertsFindings(t *testing.T) {
	// A high-entropy value with a key name and delimiter — the shape the native
	// scanner misses and gitleaks' generic-api-key catches (iss-96 row 9). Built
	// at runtime, never a source literal (secret-shaped-fixtures-at-runtime).
	secret := testsecret.Synthetic(96, 40)
	text := "user: here is the key\n" +
		"api_key = " + secret + "\n" +
		"assistant: got it\n"

	// Canonical gitleaks JSON report shape (subset).
	runner := &fakeRunner{report: `[{"RuleID":"generic-api-key","Secret":"` + secret + `","Match":"api_key = ` + secret + `"}]`}
	a := &Adapter{LookPath: foundLookPath(t), Runner: runner}

	findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, text, "transcript")
	if err != nil {
		t.Fatalf("Augment: %v", err)
	}
	if !runner.called {
		t.Fatal("runner was not invoked on the opted-in, binary-present path")
	}
	if len(findings) != 1 {
		t.Fatalf("want exactly 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Line != 2 {
		t.Errorf("finding Line = %d, want 2", f.Line)
	}
	// Column is a 1-based byte offset; "api_key = " is 10 bytes, value starts at 11.
	if f.Column != 11 {
		t.Errorf("finding Column = %d, want 11", f.Column)
	}
	if f.Matched != secret {
		t.Errorf("finding Matched = %q, want the secret value", f.Matched)
	}
	if f.Severity != scanner.SeverityHardFail {
		t.Errorf("finding Severity = %q, want hard_fail", f.Severity)
	}
	if scanner.IsIdentityKind(f.Kind) {
		t.Errorf("finding Kind %q must be a secret kind (fingerprint-masked), not an identity kind", f.Kind)
	}

	// The augmentation is real: routed through the SAME redactor the native path
	// uses, the secret's raw value is gone from the text.
	redacted, changed := scanner.Redact(text, findings)
	if changed == 0 {
		t.Fatal("Redact changed nothing; the gitleaks finding did not fold into redaction")
	}
	if strings.Contains(redacted, secret) {
		t.Errorf("the secret survived redaction:\n%s", redacted)
	}
}

// TestAugmentEmptyReportNoFindings proves a clean scan (no leaks) yields no
// findings and no error — the common opted-in case.
func TestAugmentEmptyReportNoFindings(t *testing.T) {
	for _, rep := range []string{"", "[]", "null"} {
		a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{report: rep}}
		findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, "nothing here\n", "transcript")
		if err != nil {
			t.Fatalf("empty report %q: %v", rep, err)
		}
		if len(findings) != 0 {
			t.Fatalf("empty report %q produced %d findings", rep, len(findings))
		}
	}
}

// TestLoadConfigAbsentIsDisabled proves the default: no config file means not
// opted in, with no error.
func TestLoadConfigAbsentIsDisabled(t *testing.T) {
	repo := t.TempDir()
	cfg, err := LoadConfig(repo)
	if err != nil {
		t.Fatalf("LoadConfig on a repo with no config: %v", err)
	}
	if cfg.Enabled {
		t.Error("absent config reported Enabled=true; must default off")
	}
}

// TestLoadConfigEnabled proves an enabled config parses.
func TestLoadConfigEnabled(t *testing.T) {
	repo := t.TempDir()
	writeConfig(t, repo, `{"schema_version":1,"enabled":true,"path":"/opt/gitleaks"}`)
	cfg, err := LoadConfig(repo)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.Enabled {
		t.Error("enabled config reported Enabled=false")
	}
	if cfg.Path != "/opt/gitleaks" {
		t.Errorf("Path = %q, want /opt/gitleaks", cfg.Path)
	}
}

// TestLoadConfigInvalidJSONFailsClosed proves a present-but-broken config is an
// error, not a silent default-off — a repo that tried to arm the adapter and
// mistyped the JSON is told.
func TestLoadConfigInvalidJSONFailsClosed(t *testing.T) {
	repo := t.TempDir()
	writeConfig(t, repo, `{"enabled": true`) // truncated
	if _, err := LoadConfig(repo); err == nil {
		t.Fatal("invalid config JSON returned no error; must fail closed")
	}
}

func writeConfig(t *testing.T, repoRoot, body string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gitleaks.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAugmentReportsEveryOccurrenceOnALine pins the intra-line case: gitleaks
// reports the same secret twice on one line (a request echoed with its response,
// a retry log), and the adapter must yield one Finding per occurrence. A line
// search that never advances past its first hit yields two Findings at the same
// column, dedup collapses them, and the second occurrence survives sealLine.
func TestAugmentReportsEveryOccurrenceOnALine(t *testing.T) {
	secret := testsecret.Synthetic(97, 40)
	text := "sent " + secret + " got back " + secret + "\n"
	report := `[{"RuleID":"generic-api-key","Secret":"` + secret + `","StartColumn":6},` +
		`{"RuleID":"generic-api-key","Secret":"` + secret + `","StartColumn":56}]`
	a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{report: report}}

	findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, text, "transcript")
	if err != nil {
		t.Fatalf("Augment: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("want 2 findings (one per occurrence), got %d: %+v", len(findings), findings)
	}
	if findings[0].Column == findings[1].Column {
		t.Errorf("both findings sit at column %d; the second occurrence was never located", findings[0].Column)
	}
	redacted, _ := scanner.Redact(text, findings)
	if strings.Contains(redacted, secret) {
		t.Errorf("an occurrence survived redaction:\n%s", redacted)
	}
}

// TestAugmentMultiLineFindingIsRedactedPerLine is the GHSA-j7v5-q7x6-v3rp
// multi-line limb. gitleaks' default private-key rule, and any custom rule, can
// report a value spanning lines; the adapter must split it into one finding per
// line so scanner.Redact (which seals by line) masks every line, rather than
// skip the report and store the whole block verbatim. The fixture is
// deliberately NOT PEM-shaped: a native PEM-body rule must not be able to mask
// this regression.
func TestAugmentMultiLineFindingIsRedactedPerLine(t *testing.T) {
	l1 := testsecret.Synthetic(98, 40)
	l2 := testsecret.Synthetic(99, 40)
	text := "user: key\n" + l1 + "\n" + l2 + "\nassistant: ok\n"
	report := `[{"RuleID":"custom-multiline","Secret":"` + l1 + `\n` + l2 + `","Match":"` + l1 + `\n` + l2 + `"}]`
	a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{report: report}}

	findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, text, "transcript")
	if err != nil {
		t.Fatalf("Augment: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("want 2 findings (one per line of the multi-line value), got %d: %+v", len(findings), findings)
	}
	want := []struct {
		line    int
		matched string
	}{{2, l1}, {3, l2}}
	for i, w := range want {
		if findings[i].Line != w.line || findings[i].Column != 1 || findings[i].Matched != w.matched {
			t.Errorf("finding %d = line %d col %d %q, want line %d col 1 %q",
				i, findings[i].Line, findings[i].Column, findings[i].Matched, w.line, w.matched)
		}
	}
	redacted, _ := scanner.Redact(text, findings)
	if strings.Contains(redacted, l1) || strings.Contains(redacted, l2) {
		t.Errorf("a line of the multi-line secret survived redaction:\n%s", redacted)
	}
}

// TestAugmentFallsBackToMatchWhenSecretIsNotVerbatim is the GHSA-j7v5-q7x6-v3rp
// non-verbatim limb. A rule that reports a decoded credential while the
// transcript holds the encoded form must fall back to Match — the bytes gitleaks
// actually saw — instead of locating nothing and dropping the finding.
func TestAugmentFallsBackToMatchWhenSecretIsNotVerbatim(t *testing.T) {
	decoded := "svc-user:" + testsecret.Synthetic(100, 24)
	encoded := base64.StdEncoding.EncodeToString([]byte(decoded))
	match := "Authorization: Basic " + encoded
	text := "user: curl -H \"" + match + "\" https://example.invalid/\nassistant: ok\n"
	report := `[{"RuleID":"basic-auth","Secret":"` + decoded + `","Match":"` + match + `"}]`
	a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{report: report}}

	findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, text, "transcript")
	if err != nil {
		t.Fatalf("Augment: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding positioned on Match, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Line != 1 || f.Column != strings.Index(text, match)+1 || f.Matched != match {
		t.Errorf("finding = line %d col %d %q, want line 1 col %d %q", f.Line, f.Column, f.Matched, strings.Index(text, match)+1, match)
	}
	redacted, _ := scanner.Redact(text, findings)
	if strings.Contains(redacted, encoded) {
		t.Errorf("the encoded credential survived redaction:\n%s", redacted)
	}
}

// TestAugmentUnlocatableReportFailsClosed pins the fail-closed contract for a
// report the adapter cannot place: neither Secret nor Match occurs in the text.
// The package header promises "never a silent no-op"; a report that yields zero
// findings must be an error the store refuses the write on, not a dropped
// finding with the record asserting cleanliness.
func TestAugmentUnlocatableReportFailsClosed(t *testing.T) {
	ghost := testsecret.Synthetic(101, 40)
	report := `[{"RuleID":"custom","Secret":"` + ghost + `","Match":"` + ghost + `"}]`
	a := &Adapter{LookPath: foundLookPath(t), Runner: &fakeRunner{report: report}}

	findings, err := a.Augment(context.Background(), t.TempDir(), Config{Enabled: true}, "nothing here\n", "transcript")
	if err == nil {
		t.Fatalf("an unlocatable report returned no error (findings=%+v); the adapter dropped it silently", findings)
	}
	if !errors.Is(err, ErrFindingNotLocated) {
		t.Fatalf("error is not ErrFindingNotLocated: %v", err)
	}
	if strings.Contains(err.Error(), ghost) {
		t.Errorf("the error message carries the reported value verbatim: %q", err.Error())
	}
}
