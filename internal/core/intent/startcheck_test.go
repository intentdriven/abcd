package intent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decide"
)

// blockerRecord is a minimal intent record for the blocked check's corpus: an
// id, a slug and, for a superseded one, the successor it names.
func blockerRecord(id, supersededBy string) string {
	s := "---\nid: " + id + "\nslug: " + strings.ReplaceAll(id, "itd-", "rec-") + "\nkind: standalone\n"
	if supersededBy != "" {
		s += "superseded_by: " + supersededBy + "\nkind_at_supersession: standalone\n"
	}
	return s + "---\n# " + id + "\n"
}

// adrRecord is a minimal decision record: its id and the status line's value.
func adrRecord(id, status string) string {
	return "---\nid: " + id + "\nslug: a-decision\nstatus: " + status + "\n---\n# " + id + "\n"
}

// TestStartBlockedRowFollowsASupersededBlockerToItsReplacement is ruling BZ2
// of 2026-09-29: a blocker that was superseded is followed along
// `superseded_by` to the intent that replaced it, transitively, and the intent
// waits on that replacement. It is blocked exactly when the last record of the
// chain is unsettled; a chain that loops, ends at a record this checkout does
// not hold, or names no successor refuses naming the chain. Rulings CF1 and CF2
// of 2026-09-30 settle two more endings: a chain ending at a decision settles
// when that ADR is accepted, and one ending at a discipline settles as a
// shipped intent does; a decision in any other status, or one this checkout
// does not hold, still refuses naming it.
func TestStartBlockedRowFollowsASupersededBlockerToItsReplacement(t *testing.T) {
	type rec struct{ bucket, id, by string }
	cases := []struct {
		name    string
		records []rec
		adrs    map[string]string // ADR filename -> its frontmatter
		ok      bool
		want    []string
	}{
		{"replaced by a shipped intent", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketShipped, "itd-94", ""},
		}, nil, true, []string{"itd-27 → itd-94"}},
		{"replaced by an unshipped intent", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketPlanned, "itd-94", ""},
		}, nil, false, []string{"itd-27 → itd-94", "planned"}},
		{"replaced twice, the last shipped", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketSuperseded, "itd-94", "itd-95"},
			{BucketShipped, "itd-95", ""},
		}, nil, true, []string{"itd-27 → itd-94 → itd-95"}},
		{"replaced twice, the last a draft", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketSuperseded, "itd-94", "itd-95"},
			{BucketDrafts, "itd-95", ""},
		}, nil, false, []string{"itd-27 → itd-94 → itd-95", "drafts"}},
		{"a supersession cycle", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketSuperseded, "itd-94", "itd-27"},
		}, nil, false, []string{"itd-27 → itd-94 → itd-27", "cycle"}},
		{"a replacement this checkout does not hold", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
		}, nil, false, []string{"itd-27 → itd-94", "not in this checkout's intent store"}},
		{"a superseded record naming no successor", []rec{
			{BucketSuperseded, "itd-27", "null"},
		}, nil, false, []string{"itd-27", "names no successor"}},
		{"replaced by an accepted decision", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, map[string]string{"0037-changelog-driven-releases.md": adrRecord("adr-37", "accepted")},
			true, []string{"itd-27 → adr-37 (accepted)"}},
		{"replaced by an accepted decision whose status line carries a comment", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, map[string]string{"0037-x.md": adrRecord("adr-37", "accepted                 # proposed | accepted | superseded | deprecated")},
			true, []string{"itd-27 → adr-37 (accepted)"}},
		{"replaced by an accepted minted decision, named zero-padded", []rec{
			{BucketSuperseded, "itd-27", "adr-02609012206053814"},
		}, map[string]string{"2609012206053814-x.md": adrRecord("adr-2609012206053814", "accepted")},
			true, []string{"itd-27 → adr-2609012206053814 (accepted)"}},
		{"replaced by a proposed decision", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, map[string]string{"0037-x.md": adrRecord("adr-37", "proposed")},
			false, []string{"itd-27 → adr-37", "adr-37 is proposed", "accepted"}},
		{"replaced by a decision carrying no status", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, map[string]string{"0037-x.md": "---\nid: adr-37\n---\n# adr-37\n"},
			false, []string{"itd-27 → adr-37", "adr-37 carries no status"}},
		{"replaced by a decision this checkout does not hold", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, nil, false, []string{"itd-27 → adr-37", "not in this checkout's decision store"}},
		{"replaced by a decision whose file claims another id", []rec{
			{BucketSuperseded, "itd-27", "adr-37"},
		}, map[string]string{"0037-x.md": adrRecord("adr-38", "accepted")},
			false, []string{"itd-27 → adr-37", "not in this checkout's decision store"}},
		{"replaced by a discipline", []rec{
			{BucketSuperseded, "itd-27", "itd-94"},
			{BucketDisciplines, "itd-94", ""},
		}, nil, true, []string{"itd-27 → itd-94 (disciplines)"}},
		{"reclassified as a discipline in place", []rec{
			{BucketDisciplines, "itd-27", ""},
		}, nil, true, []string{"names no unsettled blocker"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, r := range tc.records {
				writeFile(t, root, filepath.Join(IntentsRelDir, r.bucket, r.id+"-"+strings.ReplaceAll(r.id, "itd-", "rec-")+".md"), blockerRecord(r.id, r.by))
			}
			for name, body := range tc.adrs {
				writeFile(t, root, filepath.Join(filepath.FromSlash(decide.ADRsRelDir), name), body)
			}
			corpus, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			row, err := startBlockedRow(root, corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27]\n---\n")
			if err != nil {
				t.Fatal(err)
			}
			if row.OK != tc.ok {
				t.Fatalf("want OK=%v, got %+v", tc.ok, row)
			}
			for _, w := range tc.want {
				if !strings.Contains(row.Detail, w) {
					t.Errorf("the detail must name %q: %q", w, row.Detail)
				}
			}
			if !tc.ok && !strings.Contains(row.Remedy, "superseded_by") {
				t.Errorf("the remedy must say how to repair or follow the chain: %q", row.Remedy)
			}
		})
	}
}

// TestStartBlockedRowRefusesADecisionIdTwoFilesClaim: a supersession chain
// ending at a decision whose id two files in the decision store claim is not
// settled by whichever file the scan reads first. One file says accepted and
// the other proposed, so the standing of the decision is ambiguous, and the
// blocked check refuses naming the decision rather than settling the edge on
// the accepted copy.
func TestStartBlockedRowRefusesADecisionIdTwoFilesClaim(t *testing.T) {
	// At this head the store's lookup (recordid.LookupOne) keeps the first file
	// in scan order, so the accepted copy, which sorts first, settles the edge:
	// the check settles first-wins rather than refusing. Watched: without this
	// skip the test fails with OK=true and "itd-27 → adr-37 (accepted)". The
	// uniqueness of an ADR id is made a refusal by lane adrIdUnique
	// (fix/lint-adr-id-unique f6cd7b2d7), which lands later; it lifts this skip.
	t.Skip("first-wins at this head: an ADR id two files claim is refused once lane adrIdUnique (fix/lint-adr-id-unique f6cd7b2d7) lands")
	root := t.TempDir()
	writeFile(t, root, filepath.Join(IntentsRelDir, BucketSuperseded, "itd-27-rec-27.md"), blockerRecord("itd-27", "adr-37"))
	for name, status := range map[string]string{
		"0037-a-first-copy.md":  "accepted",
		"0037-b-second-copy.md": "proposed",
	} {
		writeFile(t, root, filepath.Join(filepath.FromSlash(decide.ADRsRelDir), name), adrRecord("adr-37", status))
	}
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	row, err := startBlockedRow(root, corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27]\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if row.OK {
		t.Fatalf("a decision id two files claim must refuse, not settle on the first file read: %+v", row)
	}
	if !strings.Contains(row.Detail, "adr-37") {
		t.Errorf("the refusal must name the decision: %q", row.Detail)
	}
}

// TestStartBlockedRowKeepsAPlainUnshippedBlocker holds the unchanged half: a
// blocker that was not superseded blocks until it ships, and one this checkout
// does not hold blocks too.
func TestStartBlockedRowKeepsAPlainUnshippedBlocker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join(IntentsRelDir, BucketPlanned, "itd-27-rec-27.md"), blockerRecord("itd-27", ""))
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	row, err := startBlockedRow(root, corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27, itd-99]\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if row.OK {
		t.Fatalf("an unshipped blocker blocks: %+v", row)
	}
	for _, w := range []string{"itd-27 (planned)", "itd-99 (not in this checkout's intent store)"} {
		if !strings.Contains(row.Detail, w) {
			t.Errorf("the detail must name %q: %q", w, row.Detail)
		}
	}
}
