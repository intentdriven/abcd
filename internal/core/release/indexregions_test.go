package release

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// cutTouchesGenerated names every generated index a cut would make stale: one
// whose document the cut writes, or whose source directory holds a file the cut
// writes. writes are the paths the release writer wrote in a fixture cut
// (fixtureCutWrites), so a write the cut gains is checked without anyone
// listing it here. A cut never runs the generator, so the answer must stay
// empty.
func cutTouchesGenerated(specs []lint.IndexSpec, writes []string) []string {
	var hits []string
	for _, s := range specs {
		if s.Mode != "generated" {
			continue
		}
		doc, dir := path.Clean(filepath.ToSlash(s.Doc)), path.Clean(filepath.ToSlash(s.Dir))
		for _, w := range writes {
			if doc == w || within(w, dir) {
				hits = append(hits, s.ID+" (cut writes "+w+")")
			}
		}
	}
	return hits
}

// within reports whether the slash-separated path w is dir or lies under it;
// the repository root (".") holds every path.
func within(w, dir string) bool {
	return dir == "." || w == dir || strings.HasPrefix(w, dir+"/")
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
	writes := fixtureCutWrites(t, repoRoot)
	if hits := cutTouchesGenerated(specs, writes); len(hits) > 0 {
		t.Fatalf("a cut writes what a generated index derives, so the cut must regenerate it: %v", hits)
	}

	// The check is not vacuous: a generated index over planned/ is caught.
	probe := []lint.IndexSpec{{ID: "planned-probe", Doc: "x.md", Dir: ".abcd/development/intents/" + intent.BucketPlanned, Mode: "generated"}}
	if len(cutTouchesGenerated(probe, writes)) == 0 {
		t.Fatal("a generated index over planned/ must be reported as one a cut makes stale")
	}
}

// fixtureCutWrites runs a full cut through the writer seam in a fixture
// repository and returns every repo-relative path the writer wrote. The fixture
// exercises every write a cut can make: a missed target moved to `next`, the
// outgoing page archived, the new page, the digest this repository declares
// (at the path it declares), and the changelog heading.
func fixtureCutWrites(t *testing.T, repoRoot string) []string {
	t.Helper()
	digest, err := declaredDigest(repoRoot)
	if err != nil {
		t.Fatalf("reading this repository's %s: %v", DigestKey, err)
	}
	r := pageRepo(t)
	r.Write(PageFile, outgoingPage)
	r.Write(plannedDir+"itd-95-due.md", "---\nid: itd-95\nimpact: additive\ntarget_release: v0.4.1\n---\n# Due\n")
	if digest != "" {
		r.Write(".abcd/config.json", `{"release":{"digest":"`+digest+`"}}`+"\n")
		r.Write(path.Dir(digest)+"/README.md", "the digest's directory\n")
	}
	r.Commit("a cut that makes every write")
	ops := newRecordingOps(r.Root())
	res, err := ingest(r.Root(), liveSurface(), marshalPage(t, "v0.4.1", pageEntries(), goodPage()), cutAt, ops)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if !res.Written || !res.Page.Written || res.Page.Archived == "" || len(res.Moved) == 0 {
		t.Fatalf("the fixture cut did not make every write (written=%v page=%v archived=%q moved=%v): %v",
			res.Written, res.Page.Written, res.Page.Archived, res.Moved, ops.log)
	}
	var writes []string
	for _, e := range ops.log {
		op, rel, _ := strings.Cut(e, " ")
		if op == "remove" {
			t.Fatalf("a completed cut removed %s: %v", rel, ops.log)
		}
		writes = append(writes, rel)
	}
	if digest != "" && !slices.Contains(writes, digest) {
		t.Fatalf("the fixture cut did not write the declared digest %s: %v", digest, ops.log)
	}
	return writes
}

// TestRegionCheckCoversEveryCutWrite is iss-2610150749068082: the region check
// must know every path the release writer writes, so a generated index over
// the directory of any of them is reported. The paths come from a cut the
// writer itself makes, not from a list kept beside it.
func TestRegionCheckCoversEveryCutWrite(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	writes := fixtureCutWrites(t, repoRoot)
	for _, w := range writes {
		probe := []lint.IndexSpec{{ID: "probe", Doc: "x.md", Dir: path.Dir(w), Mode: "generated"}}
		if len(cutTouchesGenerated(probe, writes)) == 0 {
			t.Errorf("the cut writes %s, but a generated index over %s is not reported as one a cut makes stale", w, path.Dir(w))
		}
	}
}
