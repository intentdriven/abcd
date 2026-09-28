package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// iss-2609261215168271: a shipped/ twin at the second member's destination is
// refused before the first member moves, as PlanBundle refuses a planned/ twin:
// the refusal is a pre-flight, not a rollback.
func TestReconcileBundleRefusesADestinationTwinBeforeAnyMove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta", Impact: "additive"})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, shippedDir+"/itd-11-beta.md", "---\nid: itd-11\nslug: beta\nspec_id: null\nkind: standalone\nimpact: fix\n---\n# twin\n")
	before := readRec(t, root, plannedDir+"/itd-10-alpha.md")

	_, err = Reconcile(root, planned.Spec.ID, "", RemainderRequest{})
	if err == nil {
		t.Fatal("a shipped/ twin at a member's destination must refuse the close")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite existing") || !strings.Contains(err.Error(), "nothing moved") || strings.Contains(err.Error(), "put back") {
		t.Errorf("the refusal must be the pre-flight's, before anything moved, not a rollback: %v", err)
	}
	if readRec(t, root, plannedDir+"/itd-10-alpha.md") != before {
		t.Error("the first member must stay planned, byte-identical")
	}
	if _, err := os.Stat(filepath.Join(root, planned.Spec.Path)); err != nil {
		t.Errorf("the shared spec must stay open: %v", err)
	}
}
