package release

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordingOps is the writer seam under test: it performs every operation for
// real through the production ops, logs the order, and fails the ones named in
// failOn ("op rel").
type recordingOps struct {
	real   fileOps
	log    []string
	failOn map[string]bool
}

func newRecordingOps(root string, failOn ...string) *recordingOps {
	m := map[string]bool{}
	for _, f := range failOn {
		m[f] = true
	}
	return &recordingOps{real: osOps{root: root}, failOn: m}
}

func (o *recordingOps) do(op, rel string, fn func() error) error {
	key := op + " " + rel
	o.log = append(o.log, key)
	if o.failOn[key] {
		// A failure is injected once per key, so the rollback that follows can
		// perform the same operation for real unless the test fails it too.
		delete(o.failOn, key)
		return errors.New("injected failure: " + key)
	}
	return fn()
}

func (o *recordingOps) createExclusive(rel string, data []byte) error {
	return o.do("create", rel, func() error { return o.real.createExclusive(rel, data) })
}

func (o *recordingOps) replace(rel string, data []byte) error {
	return o.do("replace", rel, func() error { return o.real.replace(rel, data) })
}

func (o *recordingOps) remove(rel string) error {
	return o.do("remove", rel, func() error { return o.real.remove(rel) })
}

const outgoingPage = "# Release 0.3.9 (2026-06-01)\n\nAn earlier release. (itd-12)\n"

// withOutgoingPage is pageRepo carrying a previous release page, so a cut moves
// it to the archive.
func withOutgoingPage(t *testing.T) (root string) {
	t.Helper()
	r := pageRepo(t)
	r.Write(PageFile, outgoingPage)
	r.Commit("the previous release page")
	return r.Root()
}

var archiveRel = ArchiveDir + "/0.3.9.md"

// TestCutWritesArchivePageThenHeading pins the order: the archive, then the
// page, then the changelog heading the tagging workflow reads, last.
func TestCutWritesArchivePageThenHeading(t *testing.T) {
	root := withOutgoingPage(t)
	ops := newRecordingOps(root)
	res, err := ingest(root, liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt, ops)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	want := []string{"create " + archiveRel, "replace " + PageFile, "replace " + changelogFile}
	if strings.Join(ops.log, "|") != strings.Join(want, "|") {
		t.Errorf("writes = %v, want %v", ops.log, want)
	}
	if !res.Written || !res.Page.Written {
		t.Errorf("written=%v page=%v, want both", res.Written, res.Page.Written)
	}
}

// TestCutRollsBackEveryEarlierWrite fails each step in turn and asserts the tree
// is byte-identical to what it was, and the error says it was rolled back.
func TestCutRollsBackEveryEarlierWrite(t *testing.T) {
	for _, step := range []string{"create " + archiveRel, "replace " + PageFile, "replace " + changelogFile} {
		t.Run(step, func(t *testing.T) {
			root := withOutgoingPage(t)
			before := treeDigest(t, root)
			res, err := ingest(root, liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt, newRecordingOps(root, step))
			if err == nil {
				t.Fatal("ingest succeeded through an injected failure")
			}
			if !strings.Contains(err.Error(), "rolled back") {
				t.Errorf("error %q does not say the steps were rolled back", err)
			}
			if res.Written || res.Page.Written {
				t.Error("a failed cut reported a write")
			}
			if after := treeDigest(t, root); after != before {
				t.Error("a failed cut left the tree changed")
			}
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ArchiveDir))); err == nil {
				t.Error("the archive directory the cut created was left behind")
			}
		})
	}
}

// TestFirstCutRollbackRemovesThePage: with no outgoing page, rolling back the
// page means removing the one the cut wrote.
func TestFirstCutRollbackRemovesThePage(t *testing.T) {
	r := pageRepo(t)
	root := r.Root()
	ops := newRecordingOps(root, "replace "+changelogFile)
	if _, err := ingest(root, liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt, ops); err == nil {
		t.Fatal("ingest succeeded through an injected failure")
	}
	if _, err := os.Stat(filepath.Join(root, PageFile)); !os.IsNotExist(err) {
		t.Errorf("RELEASE.md survived a first cut's rollback (stat err = %v)", err)
	}
	if strings.Contains(strings.Join(ops.log, "|"), "create ") {
		t.Errorf("a first cut archived something: %v", ops.log)
	}
}

// TestCutReportsAFailedRollback: when an undo step fails too, the report says so
// and names what to recover by hand.
func TestCutReportsAFailedRollback(t *testing.T) {
	root := withOutgoingPage(t)
	ops := newRecordingOps(root, "replace "+changelogFile, "remove "+archiveRel)
	_, err := ingest(root, liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt, ops)
	if err == nil {
		t.Fatal("ingest succeeded through an injected failure")
	}
	for _, want := range []string{"THE ROLLBACK FAILED", "recover by hand", archiveRel} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not say %q", err, want)
		}
	}
}

// TestArchiveNeverOverwrites: an archive page already standing at the target is
// a structural stop, found before anything is written.
func TestArchiveNeverOverwrites(t *testing.T) {
	r := pageRepo(t)
	r.Write(PageFile, outgoingPage)
	r.Write(archiveRel, "an archived page nobody may clobber\n")
	r.Commit("an archive collision")
	before := treeDigest(t, r.Root())
	_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt)
	if err == nil || !strings.Contains(err.Error(), "0.3.9.md") {
		t.Fatalf("err = %v, want a refusal naming the archive collision", err)
	}
	var refusal *PayloadRefusal
	if errors.As(err, &refusal) {
		t.Error("an archive collision is a structural stop, not a payload refusal to recompose")
	}
	if after := treeDigest(t, r.Root()); after != before {
		t.Error("an archive collision changed the working tree")
	}
}

// TestOutgoingPageWithoutAHeadingStops: an outgoing page whose heading does not
// name a release cannot be archived under a version, so the cut stops.
func TestOutgoingPageWithoutAHeadingStops(t *testing.T) {
	r := pageRepo(t)
	r.Write(PageFile, "a hand-written page\n")
	r.Commit("a page with no release heading")
	before := treeDigest(t, r.Root())
	_, err := Ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt)
	if err == nil || !strings.Contains(err.Error(), PageFile) {
		t.Fatalf("err = %v, want a structural refusal naming %s", err, PageFile)
	}
	var refusal *PayloadRefusal
	if errors.As(err, &refusal) {
		t.Error("an unreadable outgoing page is a stop, not a payload refusal")
	}
	if after := treeDigest(t, r.Root()); after != before {
		t.Error("the refused cut changed the working tree")
	}
}

// TestFixesOnlyCutLeavesThePageAlone: an empty press-release set writes the
// changelog heading and neither the archive nor the page, and says why.
func TestFixesOnlyCutLeavesThePageAlone(t *testing.T) {
	for _, tc := range []struct {
		name       string
		page       string
		wantReason string
	}{
		{"a page stands", outgoingPage, "RELEASE.md stays on 0.3.9"},
		{"no page yet", "", "no RELEASE.md exists yet"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := releasedRepo(t)
			r.Write("CHANGELOG.md", baseChangelog)
			if tc.page != "" {
				r.Write(PageFile, tc.page)
			}
			r.Record(resolvedDir+"iss-51-crash.md", "iss-51", "fix")
			r.Commit("a fix alone")
			entries := []ChangelogEntry{{Section: SectionFixed, Records: []string{"iss-51"}, Text: "the crash is gone."}}
			ops := newRecordingOps(r.Root())
			res, err := ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", entries, nil), cutAt, ops)
			if err != nil {
				t.Fatalf("ingest: %v", err)
			}
			if !res.Written || res.Page.Written {
				t.Fatalf("written=%v page=%v, want the changelog only", res.Written, res.Page.Written)
			}
			if strings.Join(ops.log, "|") != "replace "+changelogFile {
				t.Errorf("writes = %v, want the changelog alone", ops.log)
			}
			for _, want := range []string{"No release page written", "no user-facing intent shipped", tc.wantReason} {
				if !strings.Contains(res.Page.Reason, want) {
					t.Errorf("reason %q does not say %q", res.Page.Reason, want)
				}
			}
			if tc.page != "" {
				if got := readPage(t, r.Root()); got != tc.page {
					t.Errorf("RELEASE.md changed on a fixes-only cut: %q", got)
				}
			}
		})
	}
}

// TestUndoPlanRestoresEverything is the undo the ship verb calls when a later
// step refuses after the ingest wrote: the page comes back, the archive goes, and
// the changelog is restored.
func TestUndoPlanRestoresEverything(t *testing.T) {
	root := withOutgoingPage(t)
	before := treeDigest(t, root)
	res, err := Ingest(root, liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if failures := res.Undo.Apply(root); len(failures) > 0 {
		t.Fatalf("undo failed: %v", failures)
	}
	if after := treeDigest(t, root); after != before {
		t.Error("the undo did not restore the tree")
	}
}
