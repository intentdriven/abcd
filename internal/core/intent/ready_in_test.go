package intent

import (
	"reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// TestReadyInJudgesAsReadyDoes: ReadyIn, handed the stores a caller already
// loaded, returns exactly the verdict Ready loads them for — READY, refused for
// a missing spec, and refused as a draft — so a caller judging many intents
// loads each store once without changing a single check.
func TestReadyInJudgesAsReadyDoes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specsOpen+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	writeFile(t, root, plannedDir+"/itd-11-beta.md", plannedUnlinked("itd-11", "beta"))
	writeFile(t, root, draftsDir+"/itd-12-gamma.md", draftSeeded("itd-12", "gamma"))

	corpus, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := spec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"itd-10", "itd-11", "itd-12"} {
		want, err := Ready(root, id)
		if err != nil {
			t.Fatal(err)
		}
		it, ok := corpus.Lookup(id)
		if !ok {
			t.Fatalf("precondition: %s is in the corpus", id)
		}
		got, err := ReadyIn(root, store, it)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: ReadyIn = %+v\nwant Ready's %+v", id, got, want)
		}
	}
}
