package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// landAtLockEntry arms the lock seam so that the FIRST acquisition of the store
// lock is preceded by land, a whole verb run to completion in the window
// between the verb under test's early reads and its critical section. The seam
// disarms itself before land runs, so land's own acquisition passes through,
// and it reports whether it fired at all.
func landAtLockEntry(t *testing.T, land func()) *bool {
	t.Helper()
	fired := false
	beforeIntentMintLock = func() {
		beforeIntentMintLock = nil
		fired = true
		land()
	}
	t.Cleanup(func() { beforeIntentMintLock = nil })
	return &fired
}

// iss-2609261215159796: a bundle name another plan takes in the window before
// this plan's critical section is refused under the lock, not merged into a
// second spec. Two plans of disjoint drafts under one name cannot both succeed.
func TestPlanBundleRefusesANameTakenInTheWindow(t *testing.T) {
	root := t.TempDir()
	for _, d := range []struct{ id, slug string }{{"itd-10", "alpha"}, {"itd-11", "beta"}, {"itd-12", "gamma"}, {"itd-13", "delta"}} {
		writeFile(t, root, draftsDir+"/"+d.id+"-"+d.slug+".md", draftWithAC(d.id, d.slug))
	}
	fired := landAtLockEntry(t, func() {
		if _, err := PlanBundle(root, []string{"itd-12", "itd-13"}, BundleOptions{Bundle: "same-name"}); err != nil {
			t.Errorf("the plan landing in the window must succeed: %v", err)
		}
	})

	_, err := PlanBundle(root, []string{"itd-10", "itd-11"}, BundleOptions{Bundle: "same-name"})
	if !*fired {
		t.Fatal("PlanBundle never took the store lock: the seam never fired")
	}
	if err == nil {
		t.Fatal("a bundle name taken in the window must be refused; two plans named one bundle with two specs")
	}
	if !strings.Contains(err.Error(), "already carried by") {
		t.Errorf("the refusal must say the name is taken: %v", err)
	}
	if n := specCount(t, root); n != 1 {
		t.Errorf("one bundle name, one spec: found %d", n)
	}
	for _, rel := range []string{"itd-10-alpha.md", "itd-11-beta.md"} {
		if _, err := os.Stat(filepath.Join(root, draftsDir, rel)); err != nil {
			t.Errorf("%s must stay a draft: %v", rel, err)
		}
	}
}

// iss-2609261215159796, the reclassify half: the other record naming a bundle
// leaves it in the window, so joining it would make a bundle of one nobody
// planned. The locked judgement refuses it.
func TestReclassifyJoinRefusesABundleItsLastNamerLeftInTheWindow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, draftsDir+"/itd-11-beta.md", strings.Replace(draftWithAC("itd-11", "beta"), "kind: null\n", "kind: bundle-member\nbundle: pair\n", 1))
	before := readRec(t, root, draftsDir+"/itd-10-alpha.md")
	fired := landAtLockEntry(t, func() {
		if _, err := Reclassify(root, "itd-11", ReclassifyRequest{Kind: KindStandalone, Date: "2026-09-26"}); err != nil {
			t.Errorf("the reclassify landing in the window must succeed: %v", err)
		}
	})

	_, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindBundleMember, Bundle: "pair", Date: "2026-09-26"})
	if !*fired {
		t.Fatal("Reclassify never took the store lock: the seam never fired")
	}
	if err == nil {
		t.Fatal("joining a bundle whose last other namer left in the window must be refused")
	}
	if readRec(t, root, draftsDir+"/itd-10-alpha.md") != before {
		t.Error("a refused join must write nothing")
	}
}

// iss-2609261215159796, the survivor half: of a bundle of three, one member is
// superseded in the window, so this supersession leaves the third alone. Its
// history must say so, which it can only if the survivor set is computed under
// the lock.
func TestReclassifySupersessionNamesASurvivorLeftAloneInTheWindow(t *testing.T) {
	root := t.TempDir()
	for _, d := range []struct{ id, slug string }{{"itd-10", "alpha"}, {"itd-11", "beta"}, {"itd-12", "gamma"}, {"itd-20", "successor"}} {
		writeFile(t, root, draftsDir+"/"+d.id+"-"+d.slug+".md", draftWithAC(d.id, d.slug))
	}
	if _, err := PlanBundle(root, []string{"itd-10", "itd-11", "itd-12"}, BundleOptions{Bundle: "trio", Impact: "additive"}); err != nil {
		t.Fatal(err)
	}
	fired := landAtLockEntry(t, func() {
		if _, err := Reclassify(root, "itd-11", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "absorbed", Date: "2026-09-26"}); err != nil {
			t.Errorf("the supersession landing in the window must succeed: %v", err)
		}
	})

	res, err := Reclassify(root, "itd-10", ReclassifyRequest{Kind: KindSuperseded, By: "itd-20", Reason: "absorbed", Date: "2026-09-26"})
	if !*fired {
		t.Fatal("Reclassify never took the store lock: the seam never fired")
	}
	if err != nil {
		t.Fatal(err)
	}
	if res.Survivor != "itd-12" {
		t.Errorf("the member left alone must be named the survivor, got %q", res.Survivor)
	}
	if !strings.Contains(readRec(t, root, plannedDir+"/itd-12-gamma.md"), "bundle trio now has one member") {
		t.Error("the survivor's record must state the bundle now has one member")
	}
	for _, fnd := range recordLintFindings(t, root) {
		t.Errorf("the tree must be record-lint clean: %s:%d [%s] %s", fnd.File, fnd.Line, fnd.RuleID, fnd.Message)
	}
}
