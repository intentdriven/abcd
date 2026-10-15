// Command abcd-gen-surface writes the three artefacts derived from one walk of
// the abcd command tree: the committed compatibility snapshot, the description
// of every plugin command page (each verb's sentence, itd-2609212113220149), and
// the generated appendix at the end of every brief surface chapter (itd-147). It
// is the write half of those drift-checked artefacts: `go generate
// ./internal/surface/cli` runs it to refresh
// .abcd/development/release/surface.json, the commands/*.md descriptions and the
// appendices under .abcd/development/brief/04-surfaces/, and tests
// (internal/surface/cli/surface_test.go, sentences_test.go,
// brief_appendix_test.go) fail the build if any ever diverges from the tree. It holds no rendering logic of its own —
// the walk, the encoding and the composition live in cli.GenerateSurface and
// cli.SurfaceChapters, so the generator and the drift tests render
// byte-for-byte identically. The snapshot is written first. A chapter that
// cannot be regenerated — its markers absent or malformed, or no register row
// naming it — is refused by name and left untouched, every other chapter is
// still written, and the run then exits 1 with each refusal, so a lane landing
// behind another never has its whole regeneration blocked by one chapter.
//
// It also writes the record's generated index regions (iss-2608220150157511):
// every index the index_drift rule in .abcd/record-lint.json declares in
// generated mode — the command list in the surfaces register, the drafts bench
// in the out-of-scope chapter, the ADR index — rendered by lint.RegenerateIndexes,
// the same function the rule compares against, so a conflicted region is
// resolved by rerunning this generator over the merged tree.
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/surface/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "abcd-gen-surface:", err)
		os.Exit(1)
	}
}

// run does the whole job so every failure path returns an error to one reporter,
// rather than each step repeating the exit dance.
func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := fsutil.ModuleRoot(cwd)
	if err != nil {
		return err
	}
	data, err := cli.GenerateSurface(root)
	if err != nil {
		return err
	}
	dest := filepath.Join(root, filepath.FromSlash(cli.SurfaceSnapshotPath))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", cli.SurfaceSnapshotPath)

	// The plugin pages' descriptions are each verb's sentence from the surface
	// manifest (itd-2609212113220149). A page that cannot carry one is an
	// error naming it, and nothing is written for any page.
	pages, err := cli.SentencePages(root)
	if err != nil {
		return err
	}
	for _, p := range pages {
		if p.Committed == p.Want {
			continue
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p.File)), []byte(p.Want), 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", p.File)
	}

	// The record's generated index regions are derived from the filesystem, not
	// the command tree, and written before the chapters so a refused chapter
	// never leaves them stale.
	if err := writeIndexes(root); err != nil {
		return err
	}

	chapters, refused := cli.SurfaceChapters(root)
	if chapters == nil && refused != nil {
		return refused
	}
	for _, ch := range chapters {
		if ch.Committed == ch.Want {
			continue
		}
		rel := path.Join(surface.BriefSurfacesDir, ch.File)
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(ch.Want), 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", rel)
	}
	return refused
}

// writeIndexes rewrites every generated index region the record-lint config
// declares, and only the documents whose bytes change.
func writeIndexes(root string) error {
	cfg, err := lint.LoadConfig(filepath.Join(root, ".abcd", "record-lint.json"))
	if err != nil {
		return err
	}
	rewrites, err := lint.RegenerateIndexes(root, cfg)
	if err != nil {
		return err
	}
	for _, rw := range rewrites {
		if rw.Committed == rw.Want {
			continue
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rw.Doc)), []byte(rw.Want), 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", rw.Doc)
	}
	return nil
}
