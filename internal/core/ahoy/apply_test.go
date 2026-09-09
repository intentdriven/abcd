package ahoy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/identity"
)

// stubPrompter answers Prompt from a fixed map (falling back to def) and Confirm
// with a fixed boolean. It is the seam for exercising interactive install paths.
type stubPrompter struct {
	answers map[string]string
	confirm bool
}

func (s stubPrompter) Confirm(string) bool { return s.confirm }

func (s stubPrompter) Prompt(key string, _ []string, def string) string {
	if v, ok := s.answers[key]; ok {
		return v
	}
	return def
}

// TestStepConfigValuesRejectsInvalidDocsTarget guards that a typo'd interactive
// docs_target answer is never persisted (which would plant markers in both
// files and re-emit the gap forever). stepConfigValues must return nil, exactly
// as it already does for an invalid visibility.
func TestStepConfigValuesRejectsInvalidDocsTarget(t *testing.T) {
	dir := t.TempDir()
	a := &applyCtx{
		cwd:      dir,
		approved: map[GapCategory]bool{ConfigChange: true},
		gapPresent: map[string]bool{
			"config.visibility_missing":     true,
			"config.docs_target_missing":    true,
			"config.oracle_backend_missing": true,
		},
		prompter: stubPrompter{answers: map[string]string{
			"visibility":  "private",
			"docs_target": "clade_md", // typo — not in docsTargetChoices
		}},
	}
	if cfg := a.stepConfigValues(); cfg != nil {
		t.Fatalf("stepConfigValues persisted an invalid docs_target: %+v", cfg)
	}
	if _, err := os.Stat(configPath(dir)); err == nil {
		t.Errorf("invalid docs_target was written to config.json")
	}
}

// TestStepConfigValuesRefusesToClobberMalformedConfig proves a malformed
// config.json is left untouched: rebuilding it from the four install values would
// destroy whatever the user had. stepConfigValues must return nil (partial) and
// the file bytes must be unchanged.
func TestStepConfigValuesRefusesToClobberMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(configPath(dir)), 0o755); err != nil {
		t.Fatal(err)
	}
	const malformed = "{ this is not valid json, but the user's file }\n"
	if err := os.WriteFile(configPath(dir), []byte(malformed), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &applyCtx{
		cwd:      dir,
		approved: map[GapCategory]bool{ConfigChange: true},
		gapPresent: map[string]bool{
			"config.visibility_missing":     true,
			"config.docs_target_missing":    true,
			"config.oracle_backend_missing": true,
		},
		prompter: stubPrompter{answers: map[string]string{
			"visibility":     "private",
			"docs_target":    docsTargetDefault,
			"oracle_backend": oracleBackendDefault,
		}},
	}
	if cfg := a.stepConfigValues(); cfg != nil {
		t.Fatalf("stepConfigValues acted on a malformed config: %+v", cfg)
	}
	got, err := os.ReadFile(configPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != malformed {
		t.Errorf("malformed config.json was overwritten:\n got: %q\nwant: %q", got, malformed)
	}
}

// writeValidConfig seeds a fully-valid .abcd/config.json for the four install
// values, so stepConfigValues loads them as already-valid (no config gaps).
func writeValidConfig(t *testing.T, dir, visibility, docsTarget, oracleBackend string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(configPath(dir)), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{
		"repo":   map[string]any{"visibility": visibility},
		"docs":   map[string]any{"target": docsTarget},
		"oracle": map[string]any{"backend": oracleBackend},
	}
	if err := writeConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
}

// TestStepConfigValuesExplicitOverrideForcesAlreadyValidSlot is the iss-107
// detector: a repo whose visibility is already persisted (and valid, so no
// config gap) must still honour an explicitly-passed --visibility override,
// overwriting the value and echoing the change — not silently no-op. This
// exercises the --yes/non-interactive path (RefusingPrompter never consulted
// because every slot is already valid).
func TestStepConfigValuesExplicitOverrideForcesAlreadyValidSlot(t *testing.T) {
	dir := t.TempDir()
	writeValidConfig(t, dir, "private", "both", "host-delegated")

	a := &applyCtx{
		cwd:        dir,
		approved:   map[GapCategory]bool{}, // no config gap => category not approved
		gapPresent: map[string]bool{},      // no config.*_missing gaps
		overrides:  map[string]string{"visibility": "public"},
		prompter:   RefusingPrompter{},
		autoYes:    true,
	}

	cfg := a.stepConfigValues()
	if cfg == nil {
		t.Fatal("stepConfigValues returned nil for a valid config")
	}
	if cfg.Visibility != "public" {
		t.Errorf("visibility override dropped: got %q, want public", cfg.Visibility)
	}
	// The change must be persisted to config.json.
	m, err := readConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := stringVal(subMap(m, "repo"), "visibility"); v != "public" {
		t.Errorf("config.json visibility = %q, want public", v)
	}
	// The change must be echoed to the user.
	if len(a.changes) == 0 {
		t.Errorf("no change echoed for an explicit override")
	}
}

// TestStepConfigValuesNoOverrideLeavesValidValue guards the other side of the
// contract: with NO override flag, an already-valid persisted value is left
// untouched (a silent no-op — no write, no echoed change).
func TestStepConfigValuesNoOverrideLeavesValidValue(t *testing.T) {
	dir := t.TempDir()
	writeValidConfig(t, dir, "private", "both", "host-delegated")
	before, err := os.ReadFile(configPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	a := &applyCtx{
		cwd:        dir,
		approved:   map[GapCategory]bool{},
		gapPresent: map[string]bool{},
		overrides:  nil, // no override
		prompter:   RefusingPrompter{},
		autoYes:    true,
	}

	cfg := a.stepConfigValues()
	if cfg == nil || cfg.Visibility != "private" {
		t.Fatalf("already-valid visibility not preserved: %+v", cfg)
	}
	if len(a.changes) != 0 {
		t.Errorf("a no-op re-install echoed a change: %v", a.changes)
	}
	after, err := os.ReadFile(configPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("no-op re-install rewrote config.json")
	}
}

// TestInstallAdoptsIdentityPinInteractively guards that the advisory identity
// pin can be adopted through a later interactive `ahoy install`, as the gap's
// fix hint advertises — the "already_up_to_date" early return must not short
// -circuit it (iss-62).
func TestInstallAdoptsIdentityPinInteractively(t *testing.T) {
	setupHermetic(t)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	idMustGit(t, repo, "config", "user.name", "Alex Reppel")
	idMustGit(t, repo, "config", "user.email", "alex@example.com")

	// First install (--yes) resolves the actionable gaps but deliberately leaves
	// the advisory identity pin un-adopted.
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := identity.LoadPin(repo); ok {
		t.Fatal("precondition: --yes must not pin the current identity")
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "git_identity.unpinned") {
		t.Fatalf("expected advisory git_identity.unpinned gap after --yes install: %+v", det.Gaps)
	}
	if len(actionable(det.Gaps)) != 0 {
		t.Fatalf("precondition: repo should be otherwise clean, got %+v", actionable(det.Gaps))
	}

	// Interactive re-install with a confirming prompter must fall through the
	// early return and adopt the pin rather than reporting already_up_to_date.
	if _, err := Install(repo, InstallOptions{}, stubPrompter{confirm: true}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := identity.LoadPin(repo)
	if err != nil || !ok {
		t.Fatalf("interactive install did not adopt the identity pin: ok=%v err=%v", ok, err)
	}
	if got.Name != "Alex Reppel" || got.Email != "alex@example.com" {
		t.Fatalf("wrong pin written: %+v", got)
	}
}

// trufflehogOnPath plants an executable named trufflehog as the only entry on
// PATH, so the conditional scan_deep slot in stepConfigValues is reached: its
// gate is private visibility + trufflehog present + scan.deep unset.
func trufflehogOnPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "trufflehog"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// scanDeepCollectCtx builds the collect-missing applyCtx that reaches the
// conditional scan_deep slot: the three preceding slots are answered validly,
// so the only thing under test is what the scan_deep answer does.
func scanDeepCollectCtx(dir, answer string) *applyCtx {
	return &applyCtx{
		cwd:      dir,
		approved: map[GapCategory]bool{ConfigChange: true},
		gapPresent: map[string]bool{
			"config.visibility_missing":     true,
			"config.docs_target_missing":    true,
			"config.oracle_backend_missing": true,
			"config.scan_deep_missing":      true,
		},
		prompter: stubPrompter{answers: map[string]string{
			"visibility":     "private",
			"docs_target":    docsTargetDefault,
			"oracle_backend": oracleBackendDefault,
			"scan_deep":      answer,
		}},
	}
}

// persistedScanDeep reports the scan.deep value written to config.json, and
// whether the file carries one at all.
func persistedScanDeep(t *testing.T, dir string) (val, present bool) {
	t.Helper()
	m, err := readConfig(dir)
	if err != nil || m == nil {
		return false, false
	}
	return boolVal(subMap(m, "scan"), "deep")
}

// TestStepConfigValuesRejectsUnparseableScanDeep is the iss-2609090642031172
// detector. scan_deep is collected through the same prompter as docs_target and
// oracle_backend, and the prompter returns the typed line verbatim — so an
// answer outside the choice set must abandon the install exactly as a typo'd
// docs_target does. It must NEVER resolve to a weaker scan than the one the
// operator asked for: "yes" is a person saying yes to deep secret scanning, and
// silently recording it as scan.deep=false is a security downgrade with no
// diagnostic.
func TestStepConfigValuesRejectsUnparseableScanDeep(t *testing.T) {
	for _, answer := range []string{"yes", "y", "Yes", "true ", "TRUE", "1", "clade"} {
		t.Run(answer, func(t *testing.T) {
			trufflehogOnPath(t)
			dir := t.TempDir()
			a := scanDeepCollectCtx(dir, answer)

			cfg := a.stepConfigValues()
			if v, present := persistedScanDeep(t, dir); present && !v {
				t.Errorf("answer %q silently DISABLED the deep secret scan: config.json holds scan.deep=false", answer)
			}
			if cfg != nil {
				got := "unset"
				if cfg.ScanDeep != nil {
					got = fmt.Sprintf("%v", *cfg.ScanDeep)
				}
				t.Fatalf("stepConfigValues accepted an out-of-set scan_deep answer %q: ScanDeep=%s (want nil — partial install)", answer, got)
			}
			if _, err := os.Stat(configPath(dir)); err == nil {
				t.Errorf("an out-of-set scan_deep answer %q was written to config.json", answer)
			}
		})
	}
}

// TestStepConfigValuesHonoursExplicitScanDeepAnswers guards the other side: the
// two answers the prompt actually offers still mean what they say. An explicit
// false disables the deep scan deliberately — a legitimate answer, not a typo —
// and an explicit true enables it.
func TestStepConfigValuesHonoursExplicitScanDeepAnswers(t *testing.T) {
	for _, tc := range []struct {
		answer string
		want   bool
	}{{"true", true}, {"false", false}} {
		t.Run(tc.answer, func(t *testing.T) {
			trufflehogOnPath(t)
			dir := t.TempDir()
			a := scanDeepCollectCtx(dir, tc.answer)

			cfg := a.stepConfigValues()
			if cfg == nil {
				t.Fatalf("stepConfigValues refused the valid scan_deep answer %q", tc.answer)
			}
			if cfg.ScanDeep == nil || *cfg.ScanDeep != tc.want {
				t.Fatalf("scan_deep %q => ScanDeep %v, want %v", tc.answer, cfg.ScanDeep, tc.want)
			}
			v, present := persistedScanDeep(t, dir)
			if !present || v != tc.want {
				t.Errorf("config.json scan.deep = %v (present=%v), want %v", v, present, tc.want)
			}
		})
	}
}

// TestApplyScanDeepOverrideRejectsUnparseableOverride holds the guarded override
// path where it already stands: an override outside the choice set is a no-op on
// an already-set slot, never a coercion to false, and a valid override still
// flips the slot and echoes the change.
func TestApplyScanDeepOverrideRejectsUnparseableOverride(t *testing.T) {
	on := true
	ic := &InstallConfig{ScanDeep: &on}
	a := &applyCtx{overrides: map[string]string{"scan_deep": "yes"}}
	if a.applyScanDeepOverride(ic) {
		t.Errorf("an out-of-set scan_deep override forced the slot")
	}
	if ic.ScanDeep == nil || !*ic.ScanDeep {
		t.Errorf("an out-of-set scan_deep override weakened the slot: %v", ic.ScanDeep)
	}
	if len(a.changes) != 0 {
		t.Errorf("an out-of-set scan_deep override echoed a change: %v", a.changes)
	}

	b := &applyCtx{overrides: map[string]string{"scan_deep": "false"}}
	if !b.applyScanDeepOverride(ic) {
		t.Fatalf("a valid scan_deep override was dropped")
	}
	if ic.ScanDeep == nil || *ic.ScanDeep {
		t.Errorf("a deliberate scan_deep=false override did not disable the scan: %v", ic.ScanDeep)
	}
	if len(b.changes) != 1 {
		t.Errorf("a forced scan_deep override echoed %v, want one change", b.changes)
	}
}

// TestStepConfigValuesScanDeepDefaultOnRefusedPrompt pins the answer a
// non-interactive run gets: the prompter is never consulted, so the slot resolves
// to the default the interactive prompt displays as its bracketed answer. The
// default is the one thing a refusing prompter can persist, so it is asserted
// rather than inferred.
func TestStepConfigValuesScanDeepDefaultOnRefusedPrompt(t *testing.T) {
	trufflehogOnPath(t)
	dir := t.TempDir()
	writeValidConfig(t, dir, "private", "both", "host-delegated")

	a := &applyCtx{
		cwd:        dir,
		approved:   map[GapCategory]bool{ConfigChange: true},
		gapPresent: map[string]bool{"config.scan_deep_missing": true},
		prompter:   RefusingPrompter{},
		autoYes:    true,
	}
	cfg := a.stepConfigValues()
	if cfg == nil {
		t.Fatal("stepConfigValues returned nil for an already-valid config with only scan.deep unset")
	}
	want := scanDeepDefault == "true"
	if cfg.ScanDeep == nil || *cfg.ScanDeep != want {
		t.Fatalf("refused scan_deep prompt => ScanDeep %v, want %v (scanDeepDefault %q)", cfg.ScanDeep, want, scanDeepDefault)
	}
	if want {
		t.Errorf("scanDeepDefault is %q: a non-interactive run must not enable an opt-in deep scan nobody asked for", scanDeepDefault)
	}
	v, present := persistedScanDeep(t, dir)
	if !present || v != want {
		t.Errorf("config.json scan.deep = %v (present=%v), want %v", v, present, want)
	}
}
