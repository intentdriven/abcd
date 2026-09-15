package ahoy

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// Managed reports whether the checkout rooted at root is one abcd manages: the
// exact question classify answers with ManagedRepo, exported as a cheap
// predicate for the two front doors that ask it on every render — the status
// verb the harness invokes on each refresh, and the bare board (spc-70).
//
// It is classify's strong-signal rule and nothing more: a marker block in
// CLAUDE.md or AGENTS.md at the root, or registration in the user-level
// history index by root commit. A bare `.abcd/` is not a managed signal on its
// own (iss-88), and a plain git checkout is not managed. No new heuristic lives
// here; TestManagedAgreesWithClassify holds the two to the same answer on
// every fixture.
//
// What it does NOT do is run the full detection pass. Detect walks PATH, reads
// the plugin root, probes guard health and the rest, and a status line that
// paid for that on every keystroke would be a status line people switch off.
// The marker is read first, because it settles the ordinary managed case with
// two file reads and no subprocess; git is asked for the root commit only when
// the marker is absent AND there is an index to look it up in. Zero network,
// zero writes.
func Managed(root string) bool {
	abs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if markerFileHasBlock(filepath.Join(abs, name)) {
			return true
		}
	}
	idx, err := loadHistoryIndex()
	if err != nil || idx == nil {
		return false
	}
	return indexHasRoot(idx, gitutil.RootCommit(abs))
}
