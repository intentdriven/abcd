package cli

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// SurfaceChapters regenerates every brief surface chapter's appendix from the
// live command tree (itd-147). It walks the tree with commandSurface — the walk
// that builds the compatibility snapshot — so the snapshot and the chapters are
// derived from one traversal and cannot disagree about what the tree holds. The
// generator (cmd/abcd-gen-surface) writes each chapter's Want and the drift test
// compares it with Committed, so both come from this one call.
func SurfaceChapters(repoRoot string) ([]surface.RegeneratedChapter, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(surface.BriefSurfacesDir))
	return surface.RegenerateChapters(dir, commandSurface(NewRootCommand()))
}
