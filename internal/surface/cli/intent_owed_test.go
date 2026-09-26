package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// intent_owed_test.go is the wiring proof for itd-2609150819445595: the owed
// fidelity reviews are listed by bare `abcd intent audit`, counted on bare
// `abcd intent`, and named as the next move by `abcd <itd-N>`, all three from
// the intent store's one reader of the review marker.

const cliShipped = ".abcd/development/intents/shipped"

func cliShippedWithNotes(id, notes string) string {
	return "---\nid: " + id + "\nslug: s\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n# s\n\n" +
		"## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n\n" + notes + "\n"
}

// owedRepo stages one shipped intent per review state, plus one with a
// duplicated marker (the first wins), and chdirs into it.
func owedRepo(t *testing.T) string {
	t.Helper()
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliShipped+"/itd-11-s.md", cliShippedWithNotes("itd-11",
		"<!-- abcd-review: OWED receipt=rcp-0000000000a1 -->\nFidelity review OWED (receipt rcp-0000000000a1)."))
	writeRepoFile(t, repo, cliShipped+"/itd-12-s.md", cliShippedWithNotes("itd-12",
		"<!-- abcd-review: INGESTED receipt=rcp-0000000000b2 -->\nFidelity review — receipt rcp-0000000000b2."))
	writeRepoFile(t, repo, cliShipped+"/itd-13-s.md", cliShippedWithNotes("itd-13",
		"<!-- abcd-review: DEAD_LETTER receipt=rcp-0000000000c3 -->\n"+
			"Fidelity review DEAD_LETTER (receipt rcp-0000000000c3): criterion ac-9 is unknown. "+
			"Raw payload retained at .abcd/.work.local/reviews/rcp-0000000000c3.deadletter.json. All criteria recorded INCONCLUSIVE."))
	writeRepoFile(t, repo, cliShipped+"/itd-14-s.md", cliShippedWithNotes("itd-14",
		"_Empty. Populated by intent-auditor when intent moves to shipped/._"))
	writeRepoFile(t, repo, cliShipped+"/itd-15-s.md", cliShippedWithNotes("itd-15",
		"<!-- abcd-review: OWED receipt=rcp-0000000000d4 -->\nFidelity review OWED (receipt rcp-0000000000d4).\n\n"+
			"<!-- abcd-review: INGESTED receipt=rcp-0000000000e5 -->\nFidelity review — receipt rcp-0000000000e5."))
	return repo
}

func TestIntentAuditBareListsOwedReviews(t *testing.T) {
	repo := owedRepo(t)
	before := snapshotTree(t, repo)

	out, err := runCLIErr(t, "intent", "audit")
	if err != nil {
		t.Fatalf("bare intent audit must exit 0: %v\n%s", err, out)
	}
	s := string(out)
	owed, dead, _ := strings.Cut(s, "\ndead-lettered (")
	for _, want := range []string{
		"owed 3",
		"itd-11", "rcp-0000000000a1", "abcd intent audit itd-11",
		"itd-14", "no receipt", "minted on re-emit", "abcd intent audit itd-14",
		"itd-15", "rcp-0000000000d4",
	} {
		if !strings.Contains(owed, want) {
			t.Errorf("owed section lacks %q:\n%s", want, s)
		}
	}
	for _, want := range []string{"itd-13", "rcp-0000000000c3", "criterion ac-9 is unknown", "unreviewed"} {
		if !strings.Contains(dead, want) {
			t.Errorf("dead-lettered section lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(owed, "itd-13") {
		t.Errorf("a dead-lettered intent is not owed:\n%s", s)
	}
	for _, never := range []string{"itd-12", "rcp-0000000000e5", ".work.local", "request.md"} {
		if strings.Contains(s, never) {
			t.Errorf("listing carries %q:\n%s", never, s)
		}
	}
	if after := snapshotTree(t, repo); len(after) != len(before) {
		t.Fatalf("bare intent audit wrote: %d files -> %d", len(before), len(after))
	} else {
		for p, c := range before {
			if after[p] != c {
				t.Fatalf("bare intent audit changed %s", p)
			}
		}
	}
}

func TestIntentAuditBareJSON(t *testing.T) {
	_ = owedRepo(t)
	out := runCLI(t, "intent", "audit", "--json")
	if strings.Contains(string(out), ".work.local") {
		t.Fatalf("--json carries a local-tier path:\n%s", out)
	}
	var got struct {
		Entries []struct {
			IntentID  string `json:"intent_id"`
			State     string `json:"state"`
			ReceiptID string `json:"receipt_id"`
			Reason    string `json:"reason"`
			ReEmit    string `json:"re_emit"`
		} `json:"entries"`
		Owed         int `json:"owed"`
		DeadLettered int `json:"dead_lettered"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("intent audit --json not JSON: %v\n%s", err, out)
	}
	if len(got.Entries) != 5 || got.Owed != 3 || got.DeadLettered != 1 {
		t.Fatalf("want 5 entries, owed 3, dead-lettered 1:\n%s", out)
	}
	byID := map[string]string{}
	for _, e := range got.Entries {
		byID[e.IntentID] = e.State + " " + e.ReceiptID
	}
	want := map[string]string{
		"itd-11": "OWED rcp-0000000000a1", "itd-12": "INGESTED rcp-0000000000b2",
		"itd-13": "DEAD_LETTER rcp-0000000000c3", "itd-14": "none ",
		"itd-15": "OWED rcp-0000000000d4",
	}
	for id, w := range want {
		if byID[id] != w {
			t.Errorf("%s = %q, want %q", id, byID[id], w)
		}
	}
}

func TestIntentBareCarriesOwedCount(t *testing.T) {
	_ = owedRepo(t)
	if s := string(runCLI(t, "intent")); !strings.Contains(s, "reviews owed 3") {
		t.Fatalf("bare intent lacks the owed count beside the buckets:\n%s", s)
	}
	var board struct {
		ReviewsOwed int `json:"reviews_owed"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "--json"), &board); err != nil {
		t.Fatal(err)
	}
	var listing struct {
		Owed int `json:"owed"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "audit", "--json"), &listing); err != nil {
		t.Fatal(err)
	}
	if board.ReviewsOwed != listing.Owed || board.ReviewsOwed != 3 {
		t.Fatalf("board owed %d, listing owed %d, want both 3", board.ReviewsOwed, listing.Owed)
	}
}

func TestRootDispatchNamesAnOwedReview(t *testing.T) {
	_ = owedRepo(t)
	s := string(runCLI(t, "itd-11"))
	for _, want := range []string{"fidelity review owed", "rcp-0000000000a1", "abcd intent audit itd-11"} {
		if !strings.Contains(s, want) {
			t.Errorf("abcd itd-11 lacks %q:\n%s", want, s)
		}
	}
}
