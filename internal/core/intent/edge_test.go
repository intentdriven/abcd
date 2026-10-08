package intent

import (
	"strings"
	"testing"
)

// edge_test.go — iss-2610040758579292. The build gate reads an intent's
// `blocked_by` edges, and before this verb nothing wrote them: a downstream
// project hand-typed every edge between its intents.

// TestEdgeWritesTheBlockedByEdgeTheBuildGateThenRefusesOn is the issue's
// remedy as one story: linking two fixture intents writes the edge, and the
// build gate's blocked check then refuses the blocked intent; dropping the edge
// releases it.
func TestEdgeWritesTheBlockedByEdgeTheBuildGateThenRefusesOn(t *testing.T) {
	root := t.TempDir()
	rel := plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nbuilds_on: []\n---\n# alpha\n")
	writeFile(t, root, draftsDir+"/itd-27-beta.md", draftWithAC("itd-27", "beta"))

	gate := func() StartRow {
		t.Helper()
		corpus, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		row, err := startBlockedRow(root, corpus, "itd-10", readIntent(t, root, rel))
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	if row := gate(); !row.OK {
		t.Fatalf("before the edge the record is not blocked: %+v", row)
	}

	res, err := Edge(root, EdgeRequest{ID: "itd-10", BlockedBy: []string{"itd-27"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.IntentID != "itd-10" || res.Path != rel || strings.Join(res.BlockedBy, ",") != "itd-27" || len(res.BuildsOn) != 0 {
		t.Fatalf("EdgeResult = %+v", res)
	}
	got := readIntent(t, root, rel)
	if !containsLine(frontmatterLines(t, got), "blocked_by: [itd-27]") {
		t.Fatalf("the edge must be written inline:\n%s", got)
	}
	if !containsLine(frontmatterLines(t, got), "builds_on: []") {
		t.Fatalf("an untouched list keeps its line:\n%s", got)
	}
	if row := gate(); row.OK || !strings.Contains(row.Detail, "itd-27 (drafts)") {
		t.Fatalf("the build gate must refuse the blocked intent naming its blocker: %+v", row)
	}

	// Linking the same edge again is a no-op; unblocking it releases the gate
	// and leaves the list empty rather than dropping the key.
	if _, err := Edge(root, EdgeRequest{ID: "itd-10", BlockedBy: []string{"itd-27"}}); err != nil {
		t.Fatal(err)
	}
	if got := readIntent(t, root, rel); strings.Count(got, "itd-27") != 1 {
		t.Fatalf("a repeated edge must collapse:\n%s", got)
	}
	if _, err := Edge(root, EdgeRequest{ID: "itd-10", Unblock: []string{"itd-27"}}); err != nil {
		t.Fatal(err)
	}
	if !containsLine(frontmatterLines(t, readIntent(t, root, rel)), "blocked_by: []") {
		t.Fatalf("the emptied list is written as []:\n%s", readIntent(t, root, rel))
	}
	if row := gate(); !row.OK {
		t.Fatalf("dropping the edge releases the gate: %+v", row)
	}
}

// TestEdgeWritesBuildsOnAndKeepsABlockSequenceABlock: builds_on is the second
// edge the verb writes, and a list spelled as a block sequence stays one.
func TestEdgeWritesBuildsOnAndKeepsABlockSequenceABlock(t *testing.T) {
	root := t.TempDir()
	rel := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, "---\nid: itd-10\nslug: alpha\nbuilds_on:\n  - itd-3\nkind: null\n---\n# alpha\n")
	writeFile(t, root, shippedDir+"/itd-3-gamma.md", "---\nid: itd-3\nslug: gamma\nkind: standalone\n---\n# gamma\n")
	writeFile(t, root, draftsDir+"/itd-4-delta.md", draftWithAC("itd-4", "delta"))

	res, err := Edge(root, EdgeRequest{ID: "itd-10", BuildsOn: []string{"itd-4"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.BuildsOn, ",") != "itd-3,itd-4" {
		t.Fatalf("builds_on after the write = %v", res.BuildsOn)
	}
	want := "---\nid: itd-10\nslug: alpha\nbuilds_on:\n  - itd-3\n  - itd-4\nkind: null\n---\n# alpha\n"
	if got := readIntent(t, root, rel); got != want {
		t.Fatalf("the block sequence must gain one item and nothing else change:\n--- want\n%s--- got\n%s", want, got)
	}
	if _, err := Edge(root, EdgeRequest{ID: "itd-10", DropBuildsOn: []string{"itd-3", "itd-4"}}); err != nil {
		t.Fatal(err)
	}
	want = "---\nid: itd-10\nslug: alpha\nbuilds_on: []\nkind: null\n---\n# alpha\n"
	if got := readIntent(t, root, rel); got != want {
		t.Fatalf("an emptied block sequence is written as []:\n--- want\n%s--- got\n%s", want, got)
	}
	// An absent key is added.
	if _, err := Edge(root, EdgeRequest{ID: "itd-10", BlockedBy: []string{"itd-4"}}); err != nil {
		t.Fatal(err)
	}
	if !containsLine(frontmatterLines(t, readIntent(t, root, rel)), "blocked_by: [itd-4]") {
		t.Fatalf("an absent key is inserted:\n%s", readIntent(t, root, rel))
	}
}

// TestEdgeRefusesWithNothingWritten: an id no intent store holds, the subject
// naming itself, a malformed id, an unknown subject, a removal of an edge the
// record does not carry, and a call with nothing to do — each refused, the
// record byte-identical.
func TestEdgeRefusesWithNothingWritten(t *testing.T) {
	root := t.TempDir()
	rel := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, "---\nid: itd-10\nslug: alpha\nblocked_by: [itd-27]\nbuilds_on: []\n---\n# alpha\n")
	writeFile(t, root, draftsDir+"/itd-27-beta.md", draftWithAC("itd-27", "beta"))
	before := readIntent(t, root, rel)

	cases := []struct {
		name string
		req  EdgeRequest
		want []string
	}{
		{"an unknown blocker", EdgeRequest{ID: "itd-10", BlockedBy: []string{"itd-99"}}, []string{"--blocked-by itd-99", "not found in the intent store"}},
		{"an unknown builds_on target", EdgeRequest{ID: "itd-10", BuildsOn: []string{"itd-99"}}, []string{"--builds-on itd-99", "not found in the intent store"}},
		{"an unknown target beside a known one", EdgeRequest{ID: "itd-10", BlockedBy: []string{"itd-27", "itd-99"}}, []string{"itd-99", "not found"}},
		{"the record itself", EdgeRequest{ID: "itd-10", BlockedBy: []string{"itd-10"}}, []string{"names the record itself"}},
		{"a malformed target", EdgeRequest{ID: "itd-10", BuildsOn: []string{"iss-3"}}, []string{"must match itd-N"}},
		{"an unknown subject", EdgeRequest{ID: "itd-99", BlockedBy: []string{"itd-27"}}, []string{"itd-99 not found"}},
		{"an edge the record does not carry", EdgeRequest{ID: "itd-10", DropBuildsOn: []string{"itd-27"}}, []string{"not in itd-10's builds_on", "(none)"}},
		{"nothing to do", EdgeRequest{ID: "itd-10"}, []string{"nothing to do"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Edge(root, tc.req)
			if err == nil {
				t.Fatal("want a refusal")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("the refusal must name %q: %v", w, err)
				}
			}
			if got := readIntent(t, root, rel); got != before {
				t.Fatalf("a refusal wrote the record:\n%s", got)
			}
		})
	}
}
