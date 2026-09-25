package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// intent_condition_cli_test.go — spc-2609020626046252's wiring proof: the
// condition verb is reachable through the front door in both forms, the read
// form writes nothing, and the write form's refusals exit 2 with nothing
// written.

const cliConditionItem = "rdi-2609011200000001"

// conditionVerbRepo stages the shipped, conditioned intent and one reading item
// whose constraint_in_play cites cite.
func conditionVerbRepo(t *testing.T, cite string) (root, rel string) {
	t.Helper()
	root = intentTestRepo(t)
	rel = ".abcd/development/intents/shipped/itd-10-alpha.md"
	writeRepoFile(t, root, rel, conditionedIntent)
	writeRepoFile(t, root, ".abcd/work/issues/readings/rdg-2609011200000009/"+cliConditionItem+".md",
		"---\nschema_version: 1\nid: "+cliConditionItem+"\nrun: rdg-2609011200000009\nposition: detection\n"+
			"regime: registrative\ntension: two readings\nconstraint_in_play: \""+cite+"\"\nwhy_a_tension: both cannot hold\n---\n")
	return root, rel
}

func TestIntentConditionReadFormWritesNothing(t *testing.T) {
	root, rel := conditionVerbRepo(t, auditConditionID)
	before, _ := os.ReadFile(filepath.Join(root, rel))
	out := string(runCLI(t, "intent", "condition", "itd-10"))
	if !strings.Contains(out, auditConditionID+" — untested (no block)") {
		t.Fatalf("read form render:\n%s", out)
	}
	if after, _ := os.ReadFile(filepath.Join(root, rel)); string(after) != string(before) {
		t.Fatal("the read form wrote the record")
	}
	// A write flag on the read form is refused rather than ignored.
	if _, err := runCLIErr(t, "intent", "condition", "itd-10", "--disposition", "falsified"); err == nil || exitCodeOf(err) != 2 {
		t.Fatalf("a write flag without a condition id: exit = %d (%v)", exitCodeOf(err), err)
	}
}

func TestIntentConditionJSONCarriesStanding(t *testing.T) {
	root, rel := conditionVerbRepo(t, "holds elsewhere cond-2608300000000001")
	before, _ := os.ReadFile(filepath.Join(root, rel))

	// A refusal exits 2 and writes nothing.
	if _, err := runCLIErr(t, "intent", "condition", "itd-10", auditConditionID,
		"--disposition", "narrowed", "--occasioned-by", cliConditionItem,
		"--grounds", "the detection pass named the index bound"); err == nil ||
		!strings.Contains(err.Error(), "states no narrowing") || exitCodeOf(err) != 2 {
		t.Fatalf("narrowed without --narrowing: exit = %d (%v)", exitCodeOf(err), err)
	}
	if after, _ := os.ReadFile(filepath.Join(root, rel)); string(after) != string(before) {
		t.Fatal("a refused write changed the record")
	}

	text := string(runCLI(t, "intent", "condition", "itd-10", auditConditionID,
		"--disposition", "narrowed", "--narrowing", "holds below 2k records",
		"--occasioned-by", cliConditionItem, "--grounds", "the detection pass named the index bound"))
	for _, want := range []string{
		"abcd intent condition — " + auditConditionID + " on itd-10: narrowed, occasioned by " + cliConditionItem,
		auditConditionID + " — narrowed (from condition " + cliConditionItem,
		"cites cond-2608300000000001, not " + auditConditionID,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("write render lacks %q:\n%s", want, text)
		}
	}

	var view struct {
		IntentID     string `json:"intent_id"`
		Dispositions []struct {
			ConditionID string `json:"condition_id"`
			Disposition string `json:"disposition"`
			Narrowing   string `json:"narrowing"`
			Occasion    string `json:"occasion"`
			Date        string `json:"date"`
		} `json:"dispositions"`
		Standing []struct {
			ConditionID string `json:"condition_id"`
			Disposition string `json:"disposition"`
			Source      string `json:"source"`
			Occasion    string `json:"occasion"`
		} `json:"standing"`
	}
	out := runCLI(t, "intent", "condition", "itd-10", "--json")
	if err := json.Unmarshal(out, &view); err != nil {
		t.Fatalf("--json: %v\n%s", err, out)
	}
	if len(view.Dispositions) != 1 || view.Dispositions[0].Narrowing != "holds below 2k records" ||
		view.Dispositions[0].Occasion != cliConditionItem || view.Dispositions[0].Date == "" {
		t.Fatalf("dispositions = %+v", view.Dispositions)
	}
	if len(view.Standing) != 1 || view.Standing[0].Disposition != "narrowed" ||
		view.Standing[0].Source != "condition "+cliConditionItem {
		t.Fatalf("standing = %+v", view.Standing)
	}
}
