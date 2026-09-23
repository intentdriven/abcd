// Command abcd-gen-surface writes the two artefacts derived from one walk of the
// abcd command tree: the committed compatibility snapshot, and the generated
// appendix at the end of every brief surface chapter (itd-147). It is the write
// half of both drift-checked artefacts: `go generate ./internal/surface/cli` runs
// it to refresh .abcd/development/release/surface.json and the appendices under
// .abcd/development/brief/04-surfaces/, and tests
// (internal/surface/cli/surface_test.go, brief_appendix_test.go) fail the build if
// either ever diverges from the tree. It holds no rendering logic of its own —
// the walk, the encoding and the composition live in cli.GenerateSurface and
// cli.SurfaceChapters, so the generator and the drift tests render
// byte-for-byte identically. A chapter whose markers are absent or malformed is
// refused by name and left untouched; the snapshot is written first either way.
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

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

	chapters, err := cli.SurfaceChapters(root)
	if err != nil {
		return err
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
	return nil
}
