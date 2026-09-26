package intent

import (
	"os"
	"path/filepath"
	"testing"
)

// The sibling in the bundle close: an impact a member gains in the window is
// judged under the lock against --impact, not overwritten by a stamp decided
// from the bytes read before it. A disagreement refuses the whole close.
func TestReconcileBundleJudgesAnImpactRecordedInTheWindow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", draftWithAC("itd-11", "beta"))
	planned, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "alpha-beta"})
	if err != nil {
		t.Fatal(err)
	}
	fired := landAtLockEntry(t, func() {
		if _, err := Plan(root, "itd-11", PlanOptions{Impact: "fix"}); err != nil {
			t.Errorf("the impact stamp landing in the window must succeed: %v", err)
		}
	})

	_, err = Reconcile(root, planned.Spec.ID, "additive", RemainderRequest{})
	if !*fired {
		t.Fatal("the bundle close never took the store lock: the seam never fired")
	}
	if err == nil {
		t.Fatal("a recorded impact that disagrees with --impact must refuse the close, not be overwritten")
	}
	if got := fmOf(t, root, plannedDir+"/itd-11-beta.md")["impact"].Value; got != "fix" {
		t.Errorf("the impact recorded in the window must survive, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, plannedDir, "itd-10-alpha.md")); err != nil {
		t.Errorf("no member may ship on a refusal: %v", err)
	}
}

// The same sibling in one intent's close: an impact the record gains in the
// window is judged under the lock, not overwritten by the stamp decided from
// the bytes read before it.
func TestReconcileJudgesAnImpactRecordedInTheWindow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	planned, err := Plan(root, "itd-10", PlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fired := landAtLockEntry(t, func() {
		if _, err := Plan(root, "itd-10", PlanOptions{Impact: "fix"}); err != nil {
			t.Errorf("the impact stamp landing in the window must succeed: %v", err)
		}
	})

	_, err = Reconcile(root, planned.Spec.ID, "additive", RemainderRequest{})
	if !*fired {
		t.Fatal("the close never took the store lock: the seam never fired")
	}
	if err == nil {
		t.Fatal("a recorded impact that disagrees with --impact must refuse the close, not be overwritten")
	}
	if got := fmOf(t, root, plannedDir+"/itd-10-alpha.md")["impact"].Value; got != "fix" {
		t.Errorf("the impact recorded in the window must survive, got %q", got)
	}
}
