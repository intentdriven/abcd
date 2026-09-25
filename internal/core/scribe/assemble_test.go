package scribe

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/sessionkind"
)

const suppliedText = "Dispositions for this run, in the researcher's words.\n"

func assembleFixture(t *testing.T, f fixture, text string) AssembleResult {
	t.Helper()
	res, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: supply(t, text)})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return res
}

func readParked(t *testing.T, f fixture, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.repo, filepath.FromSlash(DefaultRunDir), fixtureRun, name))
	if err != nil {
		t.Fatalf("read the parked %s: %v", name, err)
	}
	return raw
}

// TestScribeContextIsLedgerAndSuppliedTextOnly is ac-1's exclusion half: the
// context holds ledger records and the supplied text, and no material from the
// shipped tree, the durable record outside the ledger, the local tier or the
// transcript store — each planted with a sentinel that names its class.
func TestScribeContextIsLedgerAndSuppliedTextOnly(t *testing.T) {
	f := newFixture(t, positionDetection, 2)
	res := assembleFixture(t, f, suppliedText)
	raw := readParked(t, f, ContextFileName)

	for rel, sentinel := range outsideSentinels {
		if strings.Contains(string(raw), sentinel) {
			t.Errorf("the scribe context carries %s, planted at %s", sentinel, rel)
		}
	}
	if strings.Contains(string(raw), "SENTINEL-TRANSCRIPT-STORE") {
		t.Error("the scribe context carries the transcript store's sentinel")
	}
	if len(res.Context.Ledger) == 0 {
		t.Fatal("the context carries no ledger record, so the exclusion half asserts nothing")
	}
	for _, e := range res.Context.Ledger {
		if !strings.HasPrefix(e.Path, capture.LedgerRelPath+"/") {
			t.Errorf("the context carries %s, outside the ledger", e.Path)
		}
	}
	if res.Context.Supplied.Dispositions != suppliedText {
		t.Errorf("the supplied text is %q, want it verbatim", res.Context.Supplied.Dispositions)
	}
	// The open issue is the ledger's other content, and it travels.
	if !contextHasPath(res.Context, ".abcd/work/issues/open/iss-1-an-open-issue.md") {
		t.Error("the context omits the issue ledger's open record")
	}
}

func contextHasPath(c Context, p string) bool {
	for _, e := range c.Ledger {
		if e.Path == p {
			return true
		}
	}
	return false
}

// TestScribeContextCarriesTheRunsRecordsFromTheStore: the run's reading records
// come from the store, byte for byte, and not from a raw output supplied again.
func TestScribeContextCarriesTheRunsRecordsFromTheStore(t *testing.T) {
	f := newFixture(t, positionDetection, 2)
	res := assembleFixture(t, f, suppliedText)
	for _, item := range f.items {
		rel := path.Join(capture.LedgerRelPath, issueschema.ReadingsDir, fixtureRun, item+".md")
		want, err := os.ReadFile(filepath.Join(f.repo, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range res.Context.Ledger {
			if e.Path == rel {
				found = true
				if e.Text != string(want) {
					t.Errorf("%s travels as %q, not as the store holds it", rel, e.Text)
				}
			}
		}
		if !found {
			t.Errorf("the context omits the run's record %s", rel)
		}
	}
}

// TestScribeManifestNamesEveryPathPassed is ac-1's manifest half.
func TestScribeManifestNamesEveryPathPassed(t *testing.T) {
	f := newFixture(t, positionDetection, 2)
	res := assembleFixture(t, f, suppliedText)
	contextRaw := readParked(t, f, ContextFileName)
	m, err := DecodeManifest(readParked(t, f, ManifestFileName))
	if err != nil {
		t.Fatalf("the parked manifest does not decode: %v", err)
	}
	if len(m.Items) != len(res.Context.Ledger) {
		t.Fatalf("the manifest names %d items and the context carries %d", len(m.Items), len(res.Context.Ledger))
	}
	for i, e := range res.Context.Ledger {
		it := m.Items[i]
		if it.Path != e.Path || it.Bytes != len(e.Text) || it.SHA256 != sha([]byte(e.Text)) {
			t.Errorf("manifest item %d is %+v, and the context passed %s (%d bytes)", i, it, e.Path, len(e.Text))
		}
	}
	if m.ContextSHA256 != sha(contextRaw) || res.ContextSHA256 != m.ContextSHA256 {
		t.Errorf("the manifest's context hash %s is not the parked context's %s", m.ContextSHA256, sha(contextRaw))
	}
	def, _ := os.ReadFile(filepath.Join(f.repo, DefinitionPath))
	if m.DefinitionSHA256 != sha(def) {
		t.Errorf("the manifest's definition hash %s is not %s's", m.DefinitionSHA256, DefinitionPath)
	}
	if m.Supplied.DispositionsSHA256 != sha([]byte(suppliedText)) {
		t.Errorf("the manifest's supplied hash %s does not cover the supplied text", m.Supplied.DispositionsSHA256)
	}
	if !reflect.DeepEqual(m.AllowList, AllowList()) {
		t.Errorf("the manifest's allow list %q is not AllowList() %q", m.AllowList, AllowList())
	}
	if len(m.Exclusions) == 0 {
		t.Error("the manifest asserts no exclusion")
	}
	if m.Run != fixtureRun || m.ContextStamp != res.Context.ContextStamp {
		t.Errorf("the manifest names run %s stamped %s", m.Run, m.ContextStamp)
	}
}

// TestAssertAllowListFailsClosed: an item that reached the collection by any
// route other than the allow list is refused, and nothing is written.
func TestAssertAllowListFailsClosed(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	restore := collectHook
	t.Cleanup(func() { collectHook = restore })
	collectHook = func(repoRoot string) ([]LedgerEntry, error) {
		got, err := restore(repoRoot)
		return append(got, LedgerEntry{Path: ".abcd/development/brief/01-x.md", Text: "SENTINEL-BRIEF"}), err
	}
	_, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: supply(t, suppliedText)})
	if err == nil || !strings.Contains(err.Error(), ".abcd/development/brief/01-x.md") {
		t.Fatalf("an item outside the allow list was not refused by name: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(f.repo, filepath.FromSlash(DefaultRunDir))); !os.IsNotExist(statErr) {
		t.Fatalf("a refused assembly wrote its run directory (%v)", statErr)
	}
	// A prefix that is a SIBLING of an allowed directory is outside it too.
	if err := assertAllowList([]LedgerEntry{{Path: ".abcd/work/issues/openness/x.md"}}); err == nil {
		t.Error("a sibling of an allowed directory that shares its prefix passed the assertion")
	}
}

// TestAllowListIsDerivedFromLedgerDirs: the scribe's world is the ledger's own
// directory list, so a family the ledger declares later is in it the day its
// constant is.
func TestAllowListIsDerivedFromLedgerDirs(t *testing.T) {
	dirs := issueschema.LedgerDirs()
	got := AllowList()
	if len(got) != len(dirs) {
		t.Fatalf("AllowList() = %q, LedgerDirs() = %q", got, dirs)
	}
	for i, d := range dirs {
		if want := capture.LedgerRelPath + "/" + d; got[i] != want {
			t.Errorf("AllowList()[%d] = %q, want %q", i, got[i], want)
		}
	}
}

// TestScribeAssembleRefusesAnUncommittedRun: a run with no commit marker never
// happened, and a malformed run id is refused before any path is built from it.
func TestScribeAssembleRefusesAnUncommittedRun(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	if err := os.Remove(filepath.Join(f.repo, filepath.FromSlash(issueschema.ReadingsRecordDir), fixtureRun,
		issueschema.RunRecordFileName)); err != nil {
		t.Fatal(err)
	}
	_, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: supply(t, suppliedText)})
	if err == nil || !strings.Contains(err.Error(), fixtureRun) || !strings.Contains(err.Error(), "run.json") {
		t.Fatalf("an uncommitted run was not refused by name: %v", err)
	}
	for _, bad := range []string{"", "rdg-../x", "rdi-1", "rdg-1/../../etc"} {
		if _, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: bad, DispositionsPath: supply(t, suppliedText)}); err == nil {
			t.Errorf("run %q was not refused", bad)
		}
	}
	if _, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun}); err == nil {
		t.Error("an assembly with no supplied dispositions was not refused")
	}
}

// TestScribeContextCarriesThePerRunStamp is the scribe half of
// adr-2609021016275803.
func TestScribeContextCarriesThePerRunStamp(t *testing.T) {
	f := newFixture(t, positionDetection, 2)
	res := assembleFixture(t, f, suppliedText)
	p, ok := sessionkind.Parse(res.Context.ContextStamp)
	if !ok || p.Kind != sessionkind.Scribe || p.Run != fixtureRun {
		t.Fatalf("the context is stamped %q, want the scribe stamp of %s", res.Context.ContextStamp, fixtureRun)
	}
	ledger, err := encode(res.Context.Ledger)
	if err != nil {
		t.Fatal(err)
	}
	if want := sha(ledger)[:sessionkind.DigestLen]; p.Digest != want {
		t.Errorf("the stamp's digest is %s and the ledger array hashes to %s", p.Digest, want)
	}
	if found := sessionkind.Find(readParked(t, f, ContextFileName)); len(found) != 1 {
		t.Errorf("the parked context carries the stamps %q, want exactly its own", found)
	}
}

// TestScribeAssembleWritesNothingBesideTheRun: assembly parks in the local tier
// and touches nothing in the durable record or the ledger, so an assembled and
// never ingested session leaves no trace beside the run.
func TestScribeAssembleWritesNothingBesideTheRun(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	durable := treeDigest(t, filepath.Join(f.repo, ".abcd", "development"))
	work := treeDigest(t, filepath.Join(f.repo, ".abcd", "work"))
	assembleFixture(t, f, suppliedText)
	if treeDigest(t, filepath.Join(f.repo, ".abcd", "development")) != durable {
		t.Error("assembly changed the durable record")
	}
	if treeDigest(t, filepath.Join(f.repo, ".abcd", "work")) != work {
		t.Error("assembly changed the working tier")
	}
	entries, err := os.ReadDir(filepath.Join(f.repo, filepath.FromSlash(DefaultRunDir), fixtureRun))
	if err != nil || len(entries) != 2 {
		t.Fatalf("the parked run directory holds %v (%v), want the context and the manifest", entries, err)
	}

	// A dry run writes nothing at all, and a second assembly into the occupied
	// default directory is refused rather than mixing two sessions' evidence.
	dry, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun,
		DispositionsPath: supply(t, suppliedText), DryRun: true})
	if err != nil || dry.Written {
		t.Fatalf("dry run: %+v, %v", dry, err)
	}
	if _, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun,
		DispositionsPath: supply(t, suppliedText)}); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("a second assembly into an occupied directory: %v", err)
	}
}

// TestScribeAssembleRefusesAnOutDirAReadingCanReach: a context parked where the
// reading include table reaches it would hand the next reading the ledger.
func TestScribeAssembleRefusesAnOutDirAReadingCanReach(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	_, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun,
		DispositionsPath: supply(t, suppliedText), OutDir: "docs/scribe", OutDirLabel: "docs/scribe"})
	if err == nil || !strings.Contains(err.Error(), "docs/scribe") {
		t.Fatalf("an out dir the include table reaches was not refused: %v", err)
	}
	out := filepath.Join(t.TempDir(), "session")
	res, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun,
		DispositionsPath: supply(t, suppliedText), OutDir: out})
	if err != nil || !res.Written {
		t.Fatalf("an out dir outside the repository: %+v, %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(out, ContextFileName)); err != nil {
		t.Fatalf("the context did not land under --out: %v", err)
	}
}

// TestScribeContextCarriesNoHomePath: the supplied text is the researcher's own
// and may name their home; the context names none.
func TestScribeContextCarriesNoHomePath(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	assembleFixture(t, f, "See "+f.home+"/notes/run.md and "+f.repo+"/x.md for what I meant.\n")
	raw := string(readParked(t, f, ContextFileName))
	for _, leak := range []string{f.home, f.repo} {
		if strings.Contains(raw, leak) {
			t.Errorf("the context names the absolute path %s", leak)
		}
	}
	if !strings.Contains(raw, "~/notes/run.md") {
		t.Errorf("the home path was dropped rather than redacted to ~:\n%s", raw)
	}
}

// TestScribeAssembleRefusesASymlinkedLedgerDirectory: a link inside the allow
// list is a route out of it, refused as capture's own readers refuse one.
func TestScribeAssembleRefusesASymlinkedLedgerDirectory(t *testing.T) {
	f := newFixture(t, positionDetection, 1)
	outside := t.TempDir()
	writeFile(t, outside, "leak.md", "SENTINEL-OUTSIDE-THE-REPO\n")
	link := filepath.Join(f.repo, filepath.FromSlash(capture.LedgerRelPath), issueschema.SurprisesDir)
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := Assemble(AssembleRequest{RepoRoot: f.repo, Run: fixtureRun, DispositionsPath: supply(t, suppliedText)})
	if err == nil || !errors.Is(err, ErrSymlink) {
		t.Fatalf("a symlinked ledger directory was not refused: %v", err)
	}
}
