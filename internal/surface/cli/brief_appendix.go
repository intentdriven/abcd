package cli

import (
	"fmt"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/surface"
)

// SurfaceChapters regenerates every brief surface chapter's appendix from the
// live command tree (itd-147). It walks the tree with commandSurface — the walk
// that builds the compatibility snapshot — so the snapshot and the chapters are
// derived from one traversal and cannot disagree about what the tree holds. The
// generator (cmd/abcd-gen-surface) writes each chapter's Want and the drift test
// compares it with Committed, so both come from this one call.
//
// Which shipped surfaces are host-delegated is read from the record-lint
// config's surface_coverage host_delegated list, the one place it is declared,
// so an appendix calls a command host-delegated only when that list does
// (iss-2609302306003610).
func SurfaceChapters(repoRoot string) ([]surface.RegeneratedChapter, error) {
	cfg, err := lint.LoadConfig(filepath.Join(repoRoot, filepath.FromSlash(lint.DefaultRecordLintConfigPath)))
	if err != nil {
		return nil, fmt.Errorf("surface chapters: reading the host_delegated list from %s: %w", lint.DefaultRecordLintConfigPath, err)
	}
	dir := filepath.Join(repoRoot, filepath.FromSlash(surface.BriefSurfacesDir))
	return surface.RegenerateChapters(dir, commandSurface(NewRootCommand()), cfg.Rules["surface_coverage"].HostDelegated)
}
