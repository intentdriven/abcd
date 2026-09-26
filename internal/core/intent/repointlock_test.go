package intent

import (
	"strings"
	"testing"
	"time"
)

// A record-moving verb's repoint rewrites every intent linking the moved path,
// and it does so under the store's lock: an intent writer arriving while the
// repoint runs — here a related-issue edge onto the linking draft — waits for
// the repoint's write instead of racing it, and the record ends up carrying
// both edits (iss-2609261254247117). The verb is plan; every record-moving verb
// repoints through the same helper.
func TestRepointHoldsTheStoreLockAgainstAnIntentWriter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md",
		"---\nid: itd-11\nslug: beta\nspec_id: null\nkind: null\n---\n# beta\n\nAfter [itd-10](itd-10-alpha.md).\n")

	const edge = "iss-2609261254247117"
	var landedEarly bool
	var writer chan error
	duringRepoint = func() {
		writer = make(chan error, 1)
		go func() {
			_, err := AddRelatedIssue(root, "itd-11", edge)
			writer <- err
		}()
		select {
		case err := <-writer:
			landedEarly = true
			writer <- err
		case <-time.After(300 * time.Millisecond):
		}
	}
	t.Cleanup(func() { duringRepoint = nil })

	res, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if writer == nil {
		t.Fatal("plan never reached its repoint")
	}
	if err := <-writer; err != nil {
		t.Fatal(err)
	}
	if landedEarly {
		t.Error("an intent writer landed while the repoint ran: the repoint does not hold the store's lock")
	}
	got := readRel(t, root, draftsDir+"/itd-11-beta.md")
	if !strings.Contains(got, "(../planned/itd-10-alpha.md)") || !strings.Contains(got, "related_issues: ["+edge+"]") {
		t.Errorf("the linking draft must carry both the repointed link and the concurrent edge:\n%s", got)
	}
	if res.RelinkError != "" || len(res.Relinked) != 1 {
		t.Errorf("plan must report the one rewrite: %+v %q", res.Relinked, res.RelinkError)
	}
}
