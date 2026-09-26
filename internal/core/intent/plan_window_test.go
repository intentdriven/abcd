package intent

import (
	"strings"
	"testing"
)

// The sibling in Plan: a draft's kind changed in the window must not be
// overwritten from the kind the corpus held before the lock. Plan's result must
// be what it would be had the reclassify run first: the kind the record carries
// when Plan reads it under the lock.
func TestPlanKeepsAKindChangedInTheWindow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", strings.Replace(draftWithAC("itd-11", "beta"), "kind: null\n", "kind: bundle-member\nbundle: pair\n", 1))
	fired := landAtLockEntry(t, func() {
		if _, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindBundleMember, Bundle: "pair", Date: "2026-09-26"}); err != nil {
			t.Errorf("the reclassify landing in the window must succeed: %v", err)
		}
	})

	res, err := Plan(root, "itd-10", PlanOptions{})
	if !*fired {
		t.Fatal("Plan never took the store lock: the seam never fired")
	}
	if err != nil {
		t.Fatal(err)
	}
	f := fmOf(t, root, plannedDir+"/itd-10-alpha.md")
	if f["kind"].Value != KindBundleMember || f["bundle"].Value != "pair" {
		t.Fatalf("Plan must keep the kind the record carries under the lock: kind=%q bundle=%q", f["kind"].Value, f["bundle"].Value)
	}
	if res.Intent.Kind != KindBundleMember {
		t.Errorf("the result must report the kind written, got %q", res.Intent.Kind)
	}
}
