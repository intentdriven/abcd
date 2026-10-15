package release

import (
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// cutWrites are the repo-relative paths a cut writes (write.go): the page, its
// archive directory, the changelog, and the planned intent records whose
// missed target it moves to `next` (intent.PlanTargetMoves, which refuses any
// record outside planned/ and rewrites only that frontmatter field).
var cutWrites = []string{
	PageFile,
	ArchiveDir,
	changelogFile,
	".abcd/development/intents/" + intent.BucketPlanned,
}

// cutTouchesGenerated names every generated index a cut would make stale: one
// whose document the cut writes, or whose source directory holds a file the cut
// writes. A cut never runs the generator, so the answer must stay empty.
func cutTouchesGenerated(specs []lint.IndexSpec) []string {
	var hits []string
	for _, s := range specs {
		if s.Mode != "generated" {
			continue
		}
		doc, dir := path.Clean(filepath.ToSlash(s.Doc)), path.Clean(filepath.ToSlash(s.Dir))
		for _, w := range cutWrites {
			if doc == w || strings.HasPrefix(doc, w+"/") || dir == w || strings.HasPrefix(w, dir+"/") {
				hits = append(hits, s.ID+" (cut writes "+w+")")
			}
		}
	}
	return hits
}

// TestCutLeavesGeneratedIndexesInSync holds the release roll to the generated
// index regions (iss-2608220150157511): no index the shipped record-lint config
// generates is written by a cut or derived from a directory a cut writes into,
// so a cut can never leave a region the index_drift gate then fails on, and
// the release flow needs no call into the generator. If a generated index ever
// comes to list what a cut writes, this fails, and the cut must then rerun the
// generator (lint.RegenerateIndexes) with its other writes.
func TestCutLeavesGeneratedIndexesInSync(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	cfg, err := lint.LoadConfig(filepath.Join(repoRoot, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	specs := cfg.Rules["index_drift"].Indexes
	generated := 0
	for _, s := range specs {
		if s.Mode == "generated" {
			generated++
		}
	}
	if generated == 0 {
		t.Fatal("the shipped config declares no generated index; this test would pass vacuously")
	}
	if hits := cutTouchesGenerated(specs); len(hits) > 0 {
		t.Fatalf("a cut writes what a generated index derives, so the cut must regenerate it: %v", hits)
	}

	// The check is not vacuous: a generated index over planned/ is caught.
	probe := []lint.IndexSpec{{ID: "planned-probe", Doc: "x.md", Dir: ".abcd/development/intents/planned", Mode: "generated"}}
	if len(cutTouchesGenerated(probe)) == 0 {
		t.Fatal("a generated index over planned/ must be reported as one a cut makes stale")
	}
}
