package lifeboat

import (
	"fmt"
	"strings"
	"testing"
)

// probe_truncation_test.go — iss-133. WalkFiles has three bounds and they cut
// differently: only the whole-walk cap ends the walk; the depth cap prunes one
// chain and the per-directory bound truncates one listing, and the walk goes on
// past both. The adapters used to report every one of them as the walk cap with
// "the rest of the tree was not walked", which is false for the two that do not
// end the walk and names nothing about what was actually cut.

func joinedEvidence(ev Evidence) string {
	return strings.Join(append(append([]string(nil), ev.Sources...), ev.Searched...), "\n")
}

// TestWalkTruncationNamesTheOversizedDirectory: a single directory exceeding the
// per-directory bound is named, and the evidence does not claim the walk stopped.
func TestWalkTruncationNamesTheOversizedDirectory(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"src/a.go": "package src\n\n// TODO: finish\n"}
	for i := 0; i < 8; i++ {
		files[fmt.Sprintf("wide/f%02d.txt", i)] = "x\n"
	}
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	ctx.listCap = 5 // the one per-directory bound, forced low

	for _, ev := range []Evidence{
		convOpenQuestionsSource{}.Probe(ctx),
		convInternalsSource{}.Probe(ctx),
	} {
		all := joinedEvidence(ev)
		if !strings.Contains(all, "wide") || !strings.Contains(all, "more than 5 entries") {
			t.Errorf("evidence must name the directory read only in part and the bound it hit:\n%s", all)
		}
		if strings.Contains(all, "not walked") || strings.Contains(all, "walk cap") {
			t.Errorf("a per-directory truncation must not claim the walk stopped:\n%s", all)
		}
	}
}

// TestWalkTruncationNamesTheDepthCap: a chain pruned at the depth cap is reported
// as that, and the evidence does not claim the rest of the tree went unwalked.
func TestWalkTruncationNamesTheDepthCap(t *testing.T) {
	dir := t.TempDir()
	deep := strings.Repeat("d/", maxWalkDepth+2) + "leaf.txt"
	writeTree(t, dir, map[string]string{
		"src/a.go": "package src\n\n// TODO: finish\n",
		deep:       "x\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convOpenQuestionsSource{}.Probe(ctx)
	all := joinedEvidence(ev)
	if !strings.Contains(all, fmt.Sprintf("%d-level depth cap", maxWalkDepth)) {
		t.Errorf("evidence must say the depth cap pruned a chain:\n%s", all)
	}
	if strings.Contains(all, "not walked") || strings.Contains(all, "walk cap") {
		t.Errorf("a depth-pruned chain must not claim the walk stopped:\n%s", all)
	}
}

// TestWalkTruncationWholeWalkCapStillSaysStopped is the ok: side — the one bound
// that ends the walk still says the rest of the tree was not walked.
func TestWalkTruncationWholeWalkCapStillSaysStopped(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}
	for i := 0; i < 10; i++ {
		files[fmt.Sprintf("src/p%02d/a.go", i)] = "package p\n"
	}
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	all := joinedEvidence(convInternalsSource{}.probeLimited(ctx, 4))
	if !strings.Contains(all, "walk cap") || !strings.Contains(all, "not walked") {
		t.Errorf("the whole-walk cap must say the walk stopped there:\n%s", all)
	}
	if strings.Contains(all, "depth cap") || strings.Contains(all, "read only in part") {
		t.Errorf("only the bound that fired is named:\n%s", all)
	}
}
