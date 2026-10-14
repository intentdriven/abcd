package decisions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// record renders a well-formed decision record the way the step-2 writer will:
// frontmatter id, slug and date, then a body that opens with the date.
func record(id, slug, date, text string) string {
	return "---\nid: " + id + "\nslug: " + slug + "\ndate: " + date + "\n---\n" + date + " — " + text + "\n"
}

func writeRecord(t *testing.T, root, name, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(RelDir), name)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestListOrdersByIDAndReadsAnAbsentFolderAsEmpty pins the folder reader
// (spc-2610110931381316, step 1): a repository with no decisions folder has no
// decisions, which is a state and not a fault; a folder's records come back in
// id order whatever order they were written in, each with the fields its file
// carries; and the folder's README is not a record.
func TestListOrdersByIDAndReadsAnAbsentFolderAsEmpty(t *testing.T) {
	root := t.TempDir()
	got, skipped, err := List(root)
	if err != nil || len(got) != 0 || len(skipped) != 0 {
		t.Fatalf("List on an absent folder = %v, %v, %v; want empty and no error", got, skipped, err)
	}

	// Written out of id order on purpose: the latest first.
	writeRecord(t, root, "dcn-2610141200005555-the-third.md", record("dcn-2610141200005555", "the-third", "2026-10-14", "the third"))
	writeRecord(t, root, "dcn-2609150000000001-the-first.md", record("dcn-2609150000000001", "the-first", "2026-09-15", "the first"))
	writeRecord(t, root, "dcn-2609150000000002-the-second.md", record("dcn-2609150000000002", "the-second", "2026-09-15", "the second"))
	writeRecord(t, root, "README.md", "# Decisions\n\nOne file per decision.\n")

	got, skipped, err = List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(skipped) != 0 {
		t.Errorf("nothing in this folder should be skipped; got %+v", skipped)
	}
	var ids []string
	for _, d := range got {
		ids = append(ids, d.ID)
	}
	want := []string{"dcn-2609150000000001", "dcn-2609150000000002", "dcn-2610141200005555"}
	if strings.Join(ids, " ") != strings.Join(want, " ") {
		t.Fatalf("List order = %v, want %v", ids, want)
	}
	first := got[0]
	if first.Slug != "the-first" || first.Date != "2026-09-15" ||
		first.Path != RelDir+"/dcn-2609150000000001-the-first.md" ||
		first.Text != "2026-09-15 — the first" {
		t.Errorf("first decision = %+v", first)
	}
}

// TestParseRefusesARecordItCannotTrust pins the reader's half of record-lint's
// readerFailsClosed declaration for the dcn store: a record missing a required
// property, one whose date disagrees with its id's stamp or with the date its
// body opens with, one stating a key twice, and one whose filename and
// frontmatter disagree are each refused, and List skips the refused file and
// names it rather than dropping it in silence.
func TestParseRefusesARecordItCannotTrust(t *testing.T) {
	const name = "dcn-2610141200001234-a-ruling.md"
	const rel = RelDir + "/" + name
	good := record("dcn-2610141200001234", "a-ruling", "2026-10-14", "a ruling")
	if d, err := Parse(rel, []byte(good)); err != nil || d.ID != "dcn-2610141200001234" {
		t.Fatalf("Parse(good) = %+v, %v", d, err)
	}
	for label, content := range map[string]string{
		"missing date":        "---\nid: dcn-2610141200001234\nslug: a-ruling\n---\n2026-10-14 — a ruling\n",
		"blank slug":          "---\nid: dcn-2610141200001234\nslug: \"\"\ndate: 2026-10-14\n---\n2026-10-14 — a ruling\n",
		"date against stamp":  record("dcn-2610141200001234", "a-ruling", "2026-10-13", "a ruling"),
		"date against body":   "---\nid: dcn-2610141200001234\nslug: a-ruling\ndate: 2026-10-14\n---\n2026-10-13 — a ruling\n",
		"body opens undated":  "---\nid: dcn-2610141200001234\nslug: a-ruling\ndate: 2026-10-14\n---\na ruling\n",
		"duplicate key":       "---\nid: dcn-2610141200001234\nslug: a-ruling\ndate: 2026-10-14\ndate: 2026-10-13\n---\n2026-10-14 — a ruling\n",
		"id against filename": record("dcn-2610141200009999", "a-ruling", "2026-10-14", "a ruling"),
		"slug against name":   record("dcn-2610141200001234", "another-ruling", "2026-10-14", "a ruling"),
		"no frontmatter":      "2026-10-14 — a ruling\n",
	} {
		if d, err := Parse(rel, []byte(content)); err == nil {
			t.Errorf("%s: Parse accepted %+v", label, d)
		}
	}

	root := t.TempDir()
	writeRecord(t, root, name, record("dcn-2610141200001234", "a-ruling", "2026-10-13", "a ruling"))
	got, skipped, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 || len(skipped) != 1 || skipped[0].Path != rel || skipped[0].Reason == "" {
		t.Errorf("List over a refused record = %+v, skipped %+v; want it skipped and named", got, skipped)
	}
}
