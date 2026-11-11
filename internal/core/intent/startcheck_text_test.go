package intent

import (
	"path/filepath"
	"strings"
	"testing"
)

// The blocked and open-questions rows quote record text agents and people
// write, so every value reaches a refusal masked for a terminal, clipped and
// counted (iss-2611101318042398): a spec's `blocked_by` values, a superseded
// record's `superseded_by`, a decision's status and an intent's own open
// questions alike.

// noRawControl fails when s carries an escape or a bell byte.
func noRawControl(t *testing.T, what, s string) {
	t.Helper()
	if strings.ContainsAny(s, "\x1b\x07") {
		t.Fatalf("%s quotes a terminal control raw: %q", what, s)
	}
}

func TestStartBlockedRowMasksAndCapsTheSpecsBlockers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join(IntentsRelDir, BucketPlanned, "itd-27-rec-27.md"), blockerRecord("itd-27", ""))
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("y", 200_000)
	values := []string{"\"\x1b]0;owned\x07itd-1\"", long}
	for i := 0; i < 10; i++ {
		values = append(values, "itd-27")
	}
	specContent := "---\nid: spc-1\nblocked_by: [" + strings.Join(values, ", ") + "]\n---\n# s\n"
	row, err := startBlockedRowWithSpec(root, corpus, "itd-10", "---\nid: itd-10\n---\n", "spc-1", specContent)
	if err != nil {
		t.Fatal(err)
	}
	if row.OK {
		t.Fatalf("an unsettled spec blocker blocks: %+v", row)
	}
	noRawControl(t, "the blocked row", row.Detail)
	if len(row.Detail) > 2000 {
		t.Fatalf("the blocked row is not capped: %d bytes", len(row.Detail))
	}
	if !strings.Contains(row.Detail, "and 7 more") || !strings.Contains(row.Detail, "(named by spc-1)") {
		t.Fatalf("the row names five edges, counts the rest and names the spec: %q", row.Detail)
	}
}

func TestStartBlockedRowMasksASupersededByValue(t *testing.T) {
	root := t.TempDir()
	evil := "\"\x1b[31m" + strings.Repeat("z", 5000) + "\""
	writeFile(t, root, filepath.Join(IntentsRelDir, BucketSuperseded, "itd-27-rec-27.md"), blockerRecord("itd-27", evil))
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	row, err := startBlockedRow(root, corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27]\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if row.OK || !strings.Contains(row.Detail, "names neither an intent nor a decision") {
		t.Fatalf("a successor that names nothing refuses: %+v", row)
	}
	noRawControl(t, "the blocked row", row.Detail)
	if len(row.Detail) > 1000 {
		t.Fatalf("a superseded_by value is not clipped: %d bytes", len(row.Detail))
	}
}

func TestStartBlockedRowMasksADecisionsStatus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, filepath.Join(IntentsRelDir, BucketSuperseded, "itd-27-rec-27.md"), blockerRecord("itd-27", "adr-37"))
	writeFile(t, root, filepath.Join(".abcd", "development", "decisions", "adrs", "0037-changelog-driven-releases.md"),
		adrRecord("adr-37", "\"\x1b[2J"+strings.Repeat("p", 5000)+"\""))
	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	row, err := startBlockedRow(root, corpus, "itd-10", "---\nid: itd-10\nblocked_by: [itd-27]\n---\n")
	if err != nil {
		t.Fatal(err)
	}
	if row.OK || !strings.Contains(row.Detail, "adr-37 is ") {
		t.Fatalf("a decision not accepted refuses naming it: %+v", row)
	}
	noRawControl(t, "the blocked row", row.Detail)
	if len(row.Detail) > 1000 {
		t.Fatalf("a decision's status is not clipped: %d bytes", len(row.Detail))
	}
}

func TestStartOpenQuestionsRowMasksAndCapsTheQuestions(t *testing.T) {
	content := "---\nid: itd-10\n---\n# T\n\n## Open Questions\n\n" +
		"- \x1b]0;owned\x07 which cache?\n" +
		"- " + strings.Repeat("w", 5000) + "?\n" +
		"- third?\n- fourth?\n- fifth?\n"
	row := startOpenQuestionsRow("itd-10", content)
	if row.OK {
		t.Fatalf("five open questions refuse: %+v", row)
	}
	noRawControl(t, "the open-questions row", row.Detail)
	if len(row.Detail) > 1000 {
		t.Fatalf("the open-questions row is not capped: %d bytes", len(row.Detail))
	}
	if !strings.Contains(row.Detail, "asks 5 open question(s)") || !strings.Contains(row.Detail, "and 2 more") || strings.Contains(row.Detail, "fourth") {
		t.Fatalf("the row counts every question and names three: %q", row.Detail)
	}
}

// TestStartOpenQuestionsRowWithholdsOnAnUnreadLayout: an intent whose text is
// not read whole asks questions nobody can count, so the row withholds
// readiness, as a spec's spec_body row does, and names the line and the
// construct, never the text there.
func TestStartOpenQuestionsRowWithholdsOnAnUnreadLayout(t *testing.T) {
	const head = "---\nid: itd-10\n---\n# T\n\n"
	cases := []struct{ name, body, want string }{
		{"an unclosed comment above the section", "<!-- \x1b[31m parked\n\n## Open Questions\n\n- which cache?\n", "line 6: nothing below this line is read (an HTML comment"},
		{"an unclosed fence below the section", "## Open Questions\n\n_None recorded yet._\n\n```\nsecret\n", "line 10: nothing below this line is read (a fenced block"},
		{"fences not read one way", "- item\n\n    ```\n    code\n  ```\n\n## Open Questions\n\n- which cache?\n\n```go\nx\n```\n", "line 11: the text from this line on is not read one way"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			row := startOpenQuestionsRow("itd-10", head+c.body)
			if row.OK {
				t.Fatalf("the row reads ready: %+v", row)
			}
			if !strings.Contains(row.Detail, c.want) || strings.Contains(row.Detail, "parked") || strings.Contains(row.Detail, "secret") {
				t.Fatalf("Detail = %q, want it to name %q and quote no record text", row.Detail, c.want)
			}
			noRawControl(t, "the open-questions row", row.Detail)
			if !strings.Contains(row.Remedy, "close the fence or HTML comment") {
				t.Fatalf("Remedy = %q", row.Remedy)
			}
		})
	}
}
