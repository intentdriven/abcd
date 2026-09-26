package lint

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// crossStoreCfg builds a config with the cross-store rule armed and the ADR store
// declared, so the rule has a corpus to weigh an id claim against.
func crossStoreCfg() Config {
	return Config{
		Rules: map[string]RuleConfig{
			ruleCrossStoreIDClaim: {Enabled: true, Severity: severityBlocker},
			ruleRecordSchema: {Enabled: false, RecordStores: map[string]string{
				"adr": ".abcd/development/decisions/adrs",
				"itd": ".abcd/development/intents",
				"spc": ".abcd/development/specs",
				"iss": ".abcd/work/issues",
			}},
		},
		ExemptPaths: []string{".abcd/development/research/"},
	}
}

// writeTakenADR puts a real ADR in the ADR store, so its id is TAKEN. It is the
// sibling of writeADR (contextcurrency_test.go), which knobs the lifecycle
// frontmatter this rule never reads and carries no H1 to claim an id with.
func writeTakenADR(t *testing.T, root, file, id, title string) {
	t.Helper()
	writeFile(t, root, filepath.Join(".abcd", "development", "decisions", "adrs", file),
		"---\nid: "+id+"\nstatus: accepted\n---\n\n# "+title+"\n")
}

// probeNote is the iss-2608230752354926 probe, verbatim in shape: an H1 claiming a
// taken ADR id, a decision-shaped Status block, and no frontmatter at all.
const probeNote = "# ADR-23: Transport Agnostic Core (probe)\n" +
	"\n" +
	"## Status\n" +
	"\n" +
	"Accepted (locked)\n"

// TestCrossStoreIDClaimFlagsTheProbe is the criterion the issue's own probe
// states: the file passed record-lint clean, exit 0, zero findings. It must now
// produce a finding naming the outside-store id claim.
func TestCrossStoreIDClaimFlagsTheProbe(t *testing.T) {
	root := t.TempDir()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "Transport-agnostic core")
	writeFile(t, root, filepath.Join("research", "notes", "zz-recurrence-probe.md"), probeNote)

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 1 {
		t.Fatalf("expected exactly the probe to fire, got %d: %+v", n, fs)
	}
	if !hasFinding(fs, filepath.Join("research", "notes", "zz-recurrence-probe.md"), ruleCrossStoreIDClaim, 1) {
		t.Errorf("expected the finding on the claiming heading; got %+v", fs)
	}
	if !messageContains(fs, "adr-23") {
		t.Errorf("expected the message to name the claimed id; got %+v", fs)
	}
	// The severity is blocking, which is what makes record-lint exit non-zero.
	for _, f := range fs {
		if f.RuleID == ruleCrossStoreIDClaim && f.Severity != severityBlocker {
			t.Errorf("cross-store finding severity = %q, want blocker", f.Severity)
		}
	}
}

// TestCrossStoreIDClaimLeavesRealRecordsAlone: a genuine ADR sitting in its own
// store claims its own id with an accepted status, and must never fire.
func TestCrossStoreIDClaimLeavesRealRecordsAlone(t *testing.T) {
	root := t.TempDir()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "ADR-23: Transport-agnostic core")
	writeFile(t, root, filepath.Join(".abcd", "development", "decisions", "adrs", "0024-second.md"),
		"---\nid: adr-24\n---\n\n# ADR-24: Second\n\n## Status\n\nAccepted\n")

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("expected records inside their own store to be left alone; got %d: %+v", n, fs)
	}
}

// TestCrossStoreIDClaimNeedsBothSignals is the grandfathering criterion, stated
// as the fire condition rather than as a path list: an undated Phase 0 note
// whose FILENAME reads like a record id, and a note that names a taken id in
// prose, are each missing one half of the pair and neither fires.
func TestCrossStoreIDClaimNeedsBothSignals(t *testing.T) {
	root := t.TempDir()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "Transport-agnostic core")

	// Phase 0 note: the filename looks like an ordinal, the heading claims no id,
	// and there is no decision shape.
	writeFile(t, root, filepath.Join("notes", "01-harness-interface.md"),
		"# Harness interface\n\nA Phase 0 note. It mentions adr-23 in prose.\n")
	// A decision-shaped document that claims no id at all.
	writeFile(t, root, filepath.Join("notes", "meeting.md"),
		"# Wednesday review\n\n## Status\n\nAccepted\n")
	// An id-claiming heading with no decision shape anywhere in the body.
	writeFile(t, root, filepath.Join("notes", "adr-23-summary.md"),
		"# adr-23 in one paragraph\n\nA reading note, not a decision.\n")
	// The corpus's own commonest shape, and the one this rule must never fire on:
	// a design plan headed with the record it plans FOR, carrying a status of its
	// own that is not a record lifecycle state.
	writeFile(t, root, filepath.Join(".abcd", "development", "plans", "2026-07-11-adr-23-options.md"),
		"# adr-23 transport core — design options (STOP for sign-off)\n\n"+
			"**Status:** SIGNED OFF 2026-07-11. The maintainer chose Option C.\n")

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("expected neither half alone to fire; got %d: %+v", n, fs)
	}
}

// TestCrossStoreIDClaimUntakenIDDoesNotFire pins the baseline half: the claim is
// weighed against what the corpus HOLDS, so a decision-shaped note claiming an id
// no record has taken is not this rule's finding.
func TestCrossStoreIDClaimUntakenIDDoesNotFire(t *testing.T) {
	root := t.TempDir()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "Transport-agnostic core")
	writeFile(t, root, filepath.Join("research", "notes", "zz-probe.md"),
		strings.Replace(probeNote, "ADR-23", "ADR-99", 1))

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("expected an untaken id to be out of scope; got %d: %+v", n, fs)
	}
}

// TestCrossStoreIDClaimReachesEveryStore pins that the detector is not
// ADR-shaped: an intent id claimed outside the intents store fires the same way.
func TestCrossStoreIDClaimReachesEveryStore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join(".abcd", "development", "intents", "planned", "itd-7-real.md"),
		"---\nid: itd-7\n---\n\n# The real intent\n")
	writeFile(t, root, filepath.Join("docs", "notes", "duplicate.md"),
		"# itd-7: a second intent with the same handle\n\nStatus: Accepted\n")

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 1 {
		t.Fatalf("expected the outside-store intent claim to fire, got %d: %+v", n, fs)
	}
	if !messageContains(fs, "itd-7") {
		t.Errorf("expected the message to name the claimed id; got %+v", fs)
	}
}

// TestCrossStoreIDClaimIgnoresFencedStatus pins that a fenced example — a
// document QUOTING the shape, which is how this rule gets documented — is not
// read as the document's own decision shape.
func TestCrossStoreIDClaimIgnoresFencedStatus(t *testing.T) {
	root := t.TempDir()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "Transport-agnostic core")
	writeFile(t, root, filepath.Join("docs", "explainer.md"),
		"# ADR-23 is the example this page explains\n\n```\n## Status\n\nAccepted\n```\n")

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("expected a fenced example not to count as a decision shape; got %d: %+v", n, fs)
	}
}

// TestCrossStoreIDClaimIgnoresFrontmatterStatus: a `status:` frontmatter key is
// ordinary record metadata on a great many documents, and reading it as the
// decision shape made the fire condition "an H1 claiming a taken id, plus almost
// any record-shaped file" — a verbatim copy of a real ADR fired with no Status
// section in it at all. The signal is a Status SECTION, not a metadata key.
func TestCrossStoreIDClaimIgnoresFrontmatterStatus(t *testing.T) {
	root := t.TempDir()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "Transport-agnostic core")
	writeFile(t, root, filepath.Join("notes", "copy.md"),
		"---\nid: adr-23\nstatus: accepted\n---\n\n# ADR-23: Transport-agnostic core\n\nA copy with no Status section.\n")

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("frontmatter metadata is not a decision section; got %d: %+v", n, fs)
	}
}

// TestCrossStoreIDClaimSkipsUntrackedFiles: in a git repository the candidate set
// is the TRACKED files. A bare walk read the gitignored local tier AGENTS.md
// tells agents to default to for scratch output, and a `git worktree` made inside
// the checkout, turning both into blockers over files the repository does not
// carry.
func TestCrossStoreIDClaimSkipsUntrackedFiles(t *testing.T) {
	repo := gittest.NewRepo(t)
	root := repo.Root()
	writeTakenADR(t, root, "0023-transport-agnostic-core.md", "adr-23", "Transport-agnostic core")
	writeFile(t, root, ".gitignore", ".abcd/.work.local/\n")
	repo.Commit("seed")

	// The gitignored local tier, and a sibling checkout's record tree.
	writeFile(t, root, filepath.Join(".abcd", ".work.local", "scratch", "oracle-output.md"), probeNote)
	writeFile(t, root, filepath.Join("wt", "x", ".abcd", "development", "decisions", "adrs", "0023-copy.md"), probeNote)

	fs, err := Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("untracked content is not a claim on an id; got %d: %+v", n, fs)
	}

	// The same content, TRACKED, is a claim. Staged explicitly rather than with
	// `add -A`, so the sibling checkout above stays untracked — which is what it
	// is in a real tree.
	writeFile(t, root, filepath.Join("research", "notes", "zz-recurrence-probe.md"), probeNote)
	repo.Git("add", "research/notes/zz-recurrence-probe.md")
	repo.Git("commit", "-m", "add the probe")
	fs, err = Lint(crossStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != 1 {
		t.Fatalf("expected the tracked probe to fire, got %d: %+v", n, fs)
	}
}

// reframeStoreCfg arms the cross-store rule with the reframe store declared
// and NO record held anywhere, so an outside-store reframe is judged on its
// own shape rather than against a taken id.
func reframeStoreCfg() Config {
	return Config{
		Rules: map[string]RuleConfig{
			ruleCrossStoreIDClaim: {Enabled: true, Severity: severityBlocker},
			ruleRecordSchema: {Enabled: false, RecordStores: map[string]string{
				"iss": ".abcd/work/issues",
				"rfm": ".abcd/work/issues/reframes",
			}},
		},
	}
}

// reframeLookAlike is a reframe record in its written shape: warm grounds in
// the frontmatter and a body, exactly what the reading exclusion keeps out by
// the store's PATH.
const reframeLookAlike = "---\nschema_version: 1\nid: rfm-9\noccasioned_by: rdi-11\n" +
	"construal_before: " + "0000000000000000000000000000000000000000000000000000000000000000" + "\n" +
	"grounds: why the frame moved\n---\n\nThe warm body.\n"

// A reframe record is excluded from every cold reading by the path of its
// store, so a reframe-shaped file anywhere else reaches a reading with its
// grounds and body. The rule names every one of the three shapes that make a
// file a reframe outside its store — the rfm-N name, an rfm id in the
// frontmatter, and the reframe's own keys — and leaves the store, a nested
// tree's own store, and a page that only mentions a reframe alone.
func TestCrossStoreFlagsAReframeOutsideItsStore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join(".abcd", "work", "issues", "reframes", "rfm-1.md"), reframeLookAlike)
	writeFile(t, root, filepath.Join("evals", "testdata", "repo", ".abcd", "work", "issues", "reframes", "rfm-1.md"), reframeLookAlike)
	writeFile(t, root, filepath.Join("docs", "reframes.md"), "# Reframes\n\nSee rfm-9 for the shape.\n")

	planted := map[string]string{
		filepath.Join(".abcd", "development", "brief", "01-product", "rfm-9.md"):  "# A chapter\n\nNo frontmatter, only the name.\n",
		filepath.Join(".abcd", "development", "brief", "01-product", "notes.md"):  "---\nid: RFM-3\n---\n\n# Notes\n",
		filepath.Join(".abcd", "development", "brief", "01-product", "moved.md"):  "---\noccasioned_by: dsp-5\nscope_before: abc\n---\n\n# Moved\n",
		filepath.Join(".abcd", "development", "brief", "01-product", "record.md"): reframeLookAlike,
	}
	for rel, body := range planted {
		writeFile(t, root, rel, body)
	}

	fs, err := Lint(reframeStoreCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleCrossStoreIDClaim); n != len(planted) {
		t.Fatalf("expected the %d planted reframes and nothing else, got %d: %+v", len(planted), n, fs)
	}
	for rel := range planted {
		found := false
		for _, f := range fs {
			if f.RuleID == ruleCrossStoreIDClaim && filepath.ToSlash(f.File) == filepath.ToSlash(rel) {
				found = true
				if f.Severity != severityBlocker || !strings.Contains(f.Message, ".abcd/work/issues/reframes") ||
					!strings.Contains(f.Message, "cold reading") {
					t.Errorf("%s: finding = %+v, want a blocker naming the store and the reading it would reach", rel, f)
				}
			}
		}
		if !found {
			t.Errorf("%s: no finding; got %+v", rel, fs)
		}
	}

	// With no reframe store declared, the arm has nothing to weigh a reframe
	// against and names nothing.
	cfg := reframeStoreCfg()
	rs := cfg.Rules[ruleRecordSchema]
	rs.RecordStores = map[string]string{"iss": ".abcd/work/issues"}
	cfg.Rules[ruleRecordSchema] = rs
	if fs, err = Lint(cfg, root); err != nil {
		t.Fatal(err)
	} else if n := countRule(fs, ruleCrossStoreIDClaim); n != 0 {
		t.Fatalf("no reframe store declared, yet %d finding(s): %+v", n, fs)
	}
}
