package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/gittest"
)

// targetedShippable is shippableRepo (a ready cut to v0.4.1) with three
// planned intents naming a release: one at the release being cut, one at
// `next`, and one at a later release the cut does not reach.
func targetedShippable(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shippableRepo(t)
	r.Write(plannedDir+"itd-91-due.md", "---\nid: itd-91\nimpact: additive\ntarget_release: v0.4.1\n---\n# Due\n")
	r.Write(plannedDir+"itd-92-next.md", "---\nid: itd-92\nimpact: additive\ntarget_release: next\n---\n# Next\n")
	r.Write(plannedDir+"itd-93-later.md", "---\nid: itd-93\nimpact: additive\ntarget_release: v0.5.0\n---\n# Later\n")
	r.Commit("three planned intents name a release")
	return r
}

func readRel(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestIngestMovesAMissedTargetToNext is itd-2609212103572513 criterion 3 as
// the product thinker ruled it on 2026-09-29 (BS1): given the cut is written,
// when the changelog is rolled, then each unshipped target the cut reaches
// becomes `next` (whatever the following release is numbered) in the same
// change, and the changelog names the move. A target past the cut is left.
func TestIngestMovesAMissedTargetToNext(t *testing.T) {
	r := targetedShippable(t)
	root := r.Root()
	ops := newRecordingOps(root)
	res, err := ingest(root, liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt, ops)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if !res.Written {
		t.Fatalf("nothing was written; refusals = %v", refusalKinds(res.Cut))
	}

	if got := readRel(t, root, plannedDir+"itd-91-due.md"); got != "---\nid: itd-91\nimpact: additive\ntarget_release: next\n---\n# Due\n" {
		t.Errorf("the missed version target must become `next`:\n%s", got)
	}
	if got := readRel(t, root, plannedDir+"itd-92-next.md"); !strings.Contains(got, "target_release: next\n") {
		t.Errorf("a `next` target stays `next`:\n%s", got)
	}
	if got := readRel(t, root, plannedDir+"itd-93-later.md"); !strings.Contains(got, "target_release: v0.5.0\n") {
		t.Errorf("a target past the cut is not moved:\n%s", got)
	}
	// The record rewrite is one of the cut's writes, ahead of the heading the
	// tagging workflow reads.
	want := []string{"replace " + plannedDir + "itd-91-due.md", "replace " + PageFile, "replace " + changelogFile}
	if strings.Join(ops.log, "|") != strings.Join(want, "|") {
		t.Errorf("writes = %v, want %v", ops.log, want)
	}

	var moved []string
	for _, m := range res.Moved {
		moved = append(moved, m.ID+"="+m.From)
	}
	if strings.Join(moved, ",") != "itd-91=v0.4.1,itd-92=next" {
		t.Errorf("Moved = %v", moved)
	}

	note := "Targeted and not shipped in this release, so each targets the next release (`next`): " +
		"itd-91 (targeted v0.4.1), itd-92 (targeted `next`)."
	log := readChangelog(t, root)
	if !strings.Contains(log, "## [0.4.1] - 2026-07-21\n\n"+sectionNotice+"\n\n"+note+"\n\n### Added\n") {
		t.Errorf("the dated section must name the move under its notice, ahead of every change-type heading:\n%s", log)
	}
	if !changelog.IsTargetMoveNote(note) {
		t.Error("the note written is not the one the readers pass over")
	}
}

// A cut that fails after the record rewrite puts the record back with the
// rest of the tree, and a later refusal's undo (the ship verb's payload
// render) does the same.
func TestTheMoveRollsBackWithTheCut(t *testing.T) {
	r := targetedShippable(t)
	root := r.Root()
	before := treeDigest(t, root)
	if _, err := ingest(root, liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt, newRecordingOps(root, "replace "+changelogFile)); err == nil {
		t.Fatal("ingest succeeded through an injected failure")
	}
	if after := treeDigest(t, root); after != before {
		t.Error("a failed cut left the moved target behind")
	}

	res, err := Ingest(root, liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err != nil || !res.Written {
		t.Fatalf("Ingest: %v (written=%v)", err, res.Written)
	}
	if failures := res.Undo.Apply(root); len(failures) > 0 {
		t.Fatalf("undo: %v", failures)
	}
	if after := treeDigest(t, root); after != before {
		t.Error("the cut's undo left the moved target behind")
	}
}

// A cut with no target reached writes no note and touches no record.
func TestIngestWithNoMissedTargetWritesNoNote(t *testing.T) {
	r := shippableRepo(t)
	r.Write(plannedDir+"itd-93-later.md", "---\nid: itd-93\nimpact: additive\ntarget_release: v0.5.0\n---\n# Later\n")
	r.Commit("a target past the cut")
	res, err := Ingest(r.Root(), liveSurface(), marshalPayload(t, "v0.4.1", goodEntries()), cutAt)
	if err != nil || !res.Written {
		t.Fatalf("Ingest: %v", err)
	}
	if len(res.Moved) != 0 || strings.Contains(readChangelog(t, r.Root()), "Targeted and not shipped") {
		t.Errorf("no target reached, no move: %+v", res.Moved)
	}
}
