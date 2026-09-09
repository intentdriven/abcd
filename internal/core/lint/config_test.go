package lint

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// writeConfig writes a record-lint config JSON to a temp file and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "record-lint.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLoadConfigRejectsMissingSuccessor asserts a banned_tokens entry without a
// successor is rejected at load — the machine-readable old->new mapping is
// mandatory, not prose-only (iss-51).
func TestLoadConfigRejectsMissingSuccessor(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "banned_tokens": [
	    {"id":"t1","pattern":"foo","message":"no foo","severity":"blocker","allow_context":["ok"]}
	  ]
	}`)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted a banned_tokens entry with no successor; want rejection")
	}
}

// TestLoadConfigRejectsEmptyAllowContext asserts a banned_tokens entry with an
// empty allow_context is rejected at load — every ban must declare where the
// token is legitimately allowed (iss-51).
func TestLoadConfigRejectsEmptyAllowContext(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "banned_tokens": [
	    {"id":"t1","pattern":"foo","message":"no foo","severity":"blocker","successor":"bar","allow_context":[]}
	  ]
	}`)
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted a banned_tokens entry with empty allow_context; want rejection")
	}
}

// TestLoadConfigAcceptsWellFormedEntry asserts a fully-specified entry (successor
// present, allow_context non-empty) loads without error — the strict schema does
// not reject a valid ban.
func TestLoadConfigAcceptsWellFormedEntry(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "banned_tokens": [
	    {"id":"t1","pattern":"foo","message":"no foo","severity":"blocker","successor":"bar","allow_context":["ok"]}
	  ]
	}`)
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig rejected a well-formed entry: %v", err)
	}
}

// TestBannedTokenFindingCitesSuccessor asserts the rendered finding message for a
// banned token includes its declared successor — the finding tells the reader
// what to use instead (iss-51 decision c).
func TestBannedTokenFindingCitesSuccessor(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "banned_tokens": [
	    {"id":"t1","pattern":"oldpath/thing","message":"oldpath is retired","severity":"blocker","successor":"newpath/thing","allow_context":["historical"]}
	  ]
	}`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	root := t.TempDir()
	writeFile(t, root, "rec/bad.md", "see oldpath/thing here\n")

	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	var msg string
	for _, f := range fs {
		if f.RuleID == "t1" {
			msg = f.Message
		}
	}
	if msg == "" {
		t.Fatalf("expected a t1 finding: %+v", fs)
	}
	if !strings.Contains(msg, "newpath/thing") {
		t.Errorf("finding message does not cite the successor 'newpath/thing': %q", msg)
	}
}

// TestLoadConfigRefusesFIFO pins that a FIFO in the config's place returns
// immediately instead of blocking the open forever. The docs-lint/record-lint
// config is a committed, cross-repo-clonable trust boundary reachable through
// the session hooks; before the guarded read a planted FIFO wedged every verb
// that loads it (abcd docs lint / lint / ahoy / hook session-start / cite
// refresh) at exit 124.
func TestLoadConfigRefusesFIFO(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs-lint.json")
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := LoadConfig(path)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("LoadConfig accepted a FIFO config")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("LoadConfig blocked on a FIFO config (pre-fix behaviour)")
	}
}

// TestLoadConfigRefusesSymlinkedLeaf pins that a symlinked config leaf is refused
// rather than followed. Before the fix a committed .abcd/docs-lint.json symlink
// pointing outside the repository was followed, so the linter ran its whole
// ruleset from a file the repository does not own and git does not track.
func TestLoadConfigRefusesSymlinkedLeaf(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.json")
	if err := os.WriteFile(outside, []byte(`{"roots":["elsewhere"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "docs-lint.json")
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err == nil {
		t.Fatalf("LoadConfig followed a symlinked config leaf and returned roots %v", cfg.Roots)
	}
}

// TestLoadConfigRefusesSymlinkedDir pins that a symlinked config DIRECTORY is
// refused before the leaf is touched, so a swapped .abcd cannot redirect the
// read at a config the repository does not own.
func TestLoadConfigRefusesSymlinkedDir(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "docs-lint.json"), []byte(`{"roots":["elsewhere"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(dir, ".abcd")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := LoadConfig(filepath.Join(linkDir, "docs-lint.json")); err == nil {
		t.Fatal("LoadConfig followed a symlinked config directory")
	}
}

// TestLoadConfigRefusesOversize pins that a config over the byte cap is refused
// before it is read, closing the /dev/zero-symlink OOM.
func TestLoadConfigRefusesOversize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs-lint.json")
	// Valid JSON, padded past the cap, so only the size guard can refuse it —
	// a malformed payload would fail json.Unmarshal even pre-fix (a vacuous
	// test). The padding lives in an exempt_paths entry the schema tolerates.
	pad := strings.Repeat("x", maxLintConfigBytes)
	body := `{"roots":["rec"],"exempt_paths":["` + pad + `"]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted an over-cap config")
	}
}

// TestLoadConfigRefusesEscapingRecordStore is the containment gate on the record
// stores. `.abcd/record-lint.json` is committed, so a pull request controls it;
// without this check a `"adr": "../outside/decisions"` reaches
// filepath.Join(repoRoot, …) in scanRecordStores and the gate reads and echoes
// frontmatter from outside the checkout into the CI log
// (iss-2608301308367566).
func TestLoadConfigRefusesEscapingRecordStore(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"record_schema": {"enabled": true, "severity": "blocker", "record_stores": {"adr": "../outside/decisions"}}}
	}`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig accepted a record_stores value that escapes the repository root")
	}
	if !strings.Contains(err.Error(), "../outside/decisions") {
		t.Errorf("the refusal must name the escaping value; got %q", err)
	}
}

// TestLoadConfigRefusesAbsoluteRecordStore is the same gate's other half: an
// absolute store root is refused rather than joined (filepath.Join(repoRoot,
// "/etc") is "/etc" only because Join cleans it — the value never reaches the
// repository at all).
func TestLoadConfigRefusesAbsoluteRecordStore(t *testing.T) {
	path := writeConfig(t, `{
	  "roots": ["rec"],
	  "rules": {"record_schema": {"enabled": true, "severity": "blocker", "record_stores": {"adr": "/etc"}}}
	}`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig accepted an absolute record_stores value")
	}
	if !strings.Contains(err.Error(), "/etc") {
		t.Errorf("the refusal must name the escaping value; got %q", err)
	}
}

// TestLoadConfigRefusesEscapingPathFields sweeps the SIBLINGS of record_stores:
// every other repo-relative path the config carries is joined onto the repo root
// the same way and read the same way, so a gate on one field only is a gate on
// the instance rather than on the pattern. Each field is exercised in both
// escaping shapes.
func TestLoadConfigRefusesEscapingPathFields(t *testing.T) {
	// body is a config template with %s standing in for the offending value.
	fields := []struct{ name, body string }{
		{"roots", `{"roots": ["%s"]}`},
		{"issues_dir", `{"roots":["rec"],"rules":{"issue_id_unique":{"enabled":true,"severity":"blocker","issues_dir":"%s"}}}`},
		{"commands_dir", `{"roots":["rec"],"rules":{"surface_coverage":{"enabled":true,"severity":"blocker","commands_dir":"%s"}}}`},
		{"skills_dir", `{"roots":["rec"],"rules":{"surface_coverage":{"enabled":true,"severity":"blocker","skills_dir":"%s"}}}`},
		{"changelog", `{"roots":["rec"],"rules":{"delivery_state":{"enabled":true,"severity":"blocker","changelog":"%s"}}}`},
		{"intents_root", `{"roots":["rec"],"rules":{"delivery_state":{"enabled":true,"severity":"blocker","intents_root":"%s"}}}`},
		{"registry", `{"roots":["rec"],"rules":{"persona_registry":{"enabled":true,"severity":"blocker","registry":"%s"}}}`},
		{"target", `{"roots":["rec"],"rules":{"context_status_free":{"enabled":true,"severity":"blocker","target":"%s"}}}`},
		{"receipts_dir", `{"roots":["rec"],"rules":{"receipt_gate":{"enabled":true,"severity":"blocker","receipts_dir":"%s"}}}`},
		{"runbook", `{"roots":["rec"],"rules":{"gate_lockstep":{"enabled":true,"severity":"blocker","runbook":"%s"}}}`},
		{"workflow", `{"roots":["rec"],"rules":{"gate_lockstep":{"enabled":true,"severity":"blocker","workflow":"%s"}}}`},
		{"glossary_dir", `{"roots":["rec"],"rules":{"forbidden_synonyms":{"enabled":true,"severity":"blocker","glossary_dir":"%s"}}}`},
		{"baseline", `{"roots":["rec"],"rules":{"citation_baseline":{"enabled":true,"severity":"blocker","baseline":"%s"}}}`},
		{"snapshot", `{"roots":["rec"],"rules":{"surface_coverage":{"enabled":true,"severity":"blocker","snapshot":"%s"}}}`},
		{"agents_dir", `{"roots":["rec"],"rules":{"agent_contract":{"enabled":true,"severity":"blocker","agents_dir":"%s"}}}`},
		{"intents_dir", `{"roots":["rec"],"rules":{"intent_lifecycle":{"enabled":true,"severity":"blocker","intents_dir":"%s"}}}`},
		{"specs_dir", `{"roots":["rec"],"rules":{"spec_lifecycle":{"enabled":true,"severity":"blocker","specs_dir":"%s"}}}`},
		{"index doc", `{"roots":["rec"],"rules":{"index_drift":{"enabled":true,"severity":"blocker","indexes":[{"id":"i","doc":"%s","dir":"d","entry":"^x$"}]}}}`},
		{"index dir", `{"roots":["rec"],"rules":{"index_drift":{"enabled":true,"severity":"blocker","indexes":[{"id":"i","doc":"d.md","dir":"%s","entry":"^x$"}]}}}`},
	}
	for _, f := range fields {
		for _, bad := range []string{"../outside", "/etc"} {
			path := writeConfig(t, strings.Replace(f.body, "%s", bad, 1))
			err := func() error { _, e := LoadConfig(path); return e }()
			if err == nil {
				t.Errorf("%s: LoadConfig accepted %q; the value is joined onto the repo root and read", f.name, bad)
				continue
			}
			if !strings.Contains(err.Error(), bad) {
				t.Errorf("%s: the refusal must name the escaping value %q; got %q", f.name, bad, err)
			}
		}
	}
}

// TestShippedConfigsStillLoad is the anti-vacuity guard on the gate above: this
// repository's own committed configs carry legitimately deep, nested paths (nine
// record stores, three of them inside a fourth), so a containment predicate that
// is too strict passes every refusal test while breaking the gate it protects.
func TestShippedConfigsStillLoad(t *testing.T) {
	for _, rel := range []string{".abcd/record-lint.json", ".abcd/docs-lint.json"} {
		cfg, err := LoadConfig(filepath.Join(repoRootFromPackage, rel))
		if err != nil {
			t.Fatalf("the repository's own %s must still load: %v", rel, err)
		}
		if len(cfg.Roots) == 0 {
			t.Fatalf("%s loaded with no roots; the fixture is not the config under test", rel)
		}
	}
}

// TestLoadConfigRefusesADotPathActionably pins the one legitimate-looking value
// the gate narrows: a bare "." is refused (ValidRelPath has no "." segment),
// while "" — the same location, the root the field is relative to — still loads.
// The two spelling the same thing is exactly why the refusal has to name the
// accepted one: a message that lists only "absolute, unclean, escaping" leaves
// the author of `{"roots": ["."]}` with no reading of what to write instead.
func TestLoadConfigRefusesADotPathActionably(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `{"roots": ["."]}`))
	if err == nil {
		t.Fatal("LoadConfig accepted a roots entry of \".\"; ValidRelPath refuses a \".\" segment")
	}
	if !strings.Contains(err.Error(), `an empty value`) {
		t.Errorf("the refusal must name the spelling that IS accepted, or it is unactionable; got %q", err)
	}
	if _, err := LoadConfig(writeConfig(t, `{"roots": [""]}`)); err != nil {
		t.Fatalf("an empty roots entry names the repository root and must still load: %v", err)
	}
}
