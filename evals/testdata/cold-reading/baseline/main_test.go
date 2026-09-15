package main

import "testing"

// The shipped tree's tests are the largest single material class and are
// admitted identically to source, which is why the include table labels them
// apart by a basename SUFFIX rather than by an extension. Without a file of
// this shape in the corpus, deleting that suffix row leaves every fixture
// manifest byte-identical but for one item's kind, and nothing was reading the
// kind (iss-2608312019547974).
//
// It is corpus rather than a test: nothing under testdata/ is built by the Go
// toolchain, and this file exists to be ASSEMBLED.
func TestFixtureTestFileIsCorpusAndNeverRuns(t *testing.T) {
	t.Fatal("the fixture's own test file is corpus for the assembler and is never built")
}
