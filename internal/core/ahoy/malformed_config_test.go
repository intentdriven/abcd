package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// conflictedConfig is the advisory's recipe: a config.json that carried the
// disclosure fence and the native-scanning opt-out until a merge left a
// conflict marker in it. It does not parse, and it is the user's data.
const conflictedConfig = `{"repo":{"visibility":"public"},"scan":{"native_secret_scanning":false},"meta":{"schema_version":1}
<<<<<<<
`

// conflictedRepo is an adoptable repo whose config.json is conflictedConfig and
// whose docs target is unreadable for the same reason.
func conflictedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath(repo), []byte(conflictedConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestInstallNeverRebuildsMalformedConfig is GHSA-mchq-gm34-3j34: a config
// that cannot be parsed is not rewritten by any install step — stepConfigValues
// already refused, stepVersionStamp must not rebuild it as a meta-only file
// behind that refusal — and the partial install says why.
func TestInstallNeverRebuildsMalformedConfig(t *testing.T) {
	setupHermetic(t)
	repo := conflictedRepo(t)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(configPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != conflictedConfig {
		t.Fatalf("install rewrote a config.json it could not parse:\n%s", after)
	}
	for _, w := range res.Writes {
		if strings.HasSuffix(w, filepath.Join(".abcd", "config.json")) || strings.HasSuffix(w, "config.json") {
			t.Fatalf("install reported writing config.json: %v", res.Writes)
		}
	}
	said := false
	for _, n := range res.Notes {
		if strings.Contains(n, "config.json") && strings.Contains(n, "could not be parsed") {
			said = true
		}
	}
	if !said {
		t.Fatalf("no note names the unparseable config.json; notes: %v", res.Notes)
	}
	if res.Status != "partial" {
		t.Fatalf("status = %q, want partial (the config is left for the operator to repair)", res.Status)
	}
}

// TestInstallPlantsNoMarkerOnMalformedConfig is the stepMarker sibling: the
// user's docs.target is unreadable, so no marker block may be planted — least
// of all into both files by the default.
func TestInstallPlantsNoMarkerOnMalformedConfig(t *testing.T) {
	setupHermetic(t)
	repo := conflictedRepo(t)
	adopt := true
	// No docs_target override: the value has to come from the file, which cannot be read.
	opts := InstallOptions{Adopt: &adopt, Yes: true}

	if _, err := Install(repo, opts, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(repo, name)); err == nil {
			t.Fatalf("install planted %s although docs.target could not be read", name)
		}
	}
}

// TestDetectMalformedConfigIsOneDiagnosticGap: detection on an unparseable
// config raises one non-resolvable config.malformed gap and nothing that would
// arm a rewrite (install_meta.missing, marker.missing, config.*_missing).
func TestDetectMalformedConfigIsOneDiagnosticGap(t *testing.T) {
	setupHermetic(t)
	repo := conflictedRepo(t)

	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	var malformed []Gap
	for _, g := range det.Gaps {
		switch g.ID {
		case "config.malformed":
			malformed = append(malformed, g)
		case "install_meta.missing", "marker.missing", "marker.outdated",
			"config.visibility_missing", "config.docs_target_missing",
			"config.oracle_backend_missing", "config.scan_deep_missing":
			t.Errorf("gap %s raised on an unparseable config; it would arm a rewrite of user data", g.ID)
		}
	}
	if len(malformed) != 1 {
		t.Fatalf("config.malformed gaps = %d, want exactly one: %+v", len(malformed), det.Gaps)
	}
	if malformed[0].Resolvable || !malformed[0].Required {
		t.Fatalf("config.malformed must be a required, non-resolvable diagnostic: %+v", malformed[0])
	}
	if !strings.Contains(malformed[0].Detail, "could not be parsed") {
		t.Fatalf("config.malformed detail = %q, want it to say the file could not be parsed", malformed[0].Detail)
	}
}

// TestInstallReportsAMalformedConfigWhenOtherwiseUpToDate is the sibling the
// GHSA-mchq-gm34-3j34 fix left open. That fix taught every apply step to refuse
// an unparseable config.json, but the idempotency early return sits BEFORE the
// first apply step: a repo that installed cleanly and only later acquired a
// merge marker raises exactly one gap (config.malformed), which is required and
// deliberately NOT resolvable, so `actionable` counts zero and the run returns
// already_up_to_date with no notes at all. The state that most needs the
// operator — a config abcd will no longer touch — was the quietest one abcd
// could report.
func TestInstallReportsAMalformedConfigWhenOtherwiseUpToDate(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("precondition: first install status = %q (remaining=%v), want clean", res.Status, res.Remaining)
	}

	// The merge that broke it. Everything else on disk is still correct.
	if err := os.WriteFile(configPath(repo), []byte(conflictedConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	res2, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Status != "partial" {
		t.Fatalf("re-install over a broken config: status = %q, want partial", res2.Status)
	}
	said := false
	for _, n := range res2.Notes {
		if strings.Contains(n, "config.json") && strings.Contains(n, "could not be parsed") {
			said = true
		}
	}
	if !said {
		t.Fatalf("no note names the unparseable config.json; notes: %v", res2.Notes)
	}
	after, err := os.ReadFile(configPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != conflictedConfig {
		t.Fatalf("the refusing path still rewrote the config:\n%s", after)
	}
}

// TestInstallSaysWhichOverrideAMalformedConfigDropped: overridesWouldChange
// reports "no change" for a config it cannot parse, which is right — nothing
// may be written — but it also means an explicit `--visibility public` is
// dropped on the floor. A flag the operator typed and abcd did not apply is
// said out loud, or the repo is left looking public when it is not.
func TestInstallSaysWhichOverrideAMalformedConfigDropped(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if res, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	} else if res.Status != "clean" {
		t.Fatalf("precondition: first install status = %q, want clean", res.Status)
	}
	if err := os.WriteFile(configPath(repo), []byte(conflictedConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := installOpts()
	opts.ValueOverrides["visibility"] = "public"
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "partial" {
		t.Fatalf("status = %q, want partial", res.Status)
	}
	said := false
	for _, n := range res.Notes {
		if strings.Contains(n, "visibility") {
			said = true
		}
	}
	if !said {
		t.Fatalf("no note says --visibility was not applied; notes: %v", res.Notes)
	}
}
