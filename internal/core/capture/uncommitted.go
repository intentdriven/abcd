package capture

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxStatusBytes bounds the git status read below; a ledger whose uncommitted
// listing exceeds it is reported as unknown rather than read in part.
const maxStatusBytes = 8 << 20

// uncommittedLedgerPaths returns the repo-relative slash paths under the ledger
// that git reports as untracked or changed in this checkout, and ok=false when
// git cannot answer (no repository, git absent, a ledger outside the checkout),
// in which case nothing is marked: an unknown state is not reported as either
// (iss-2609100508570527).
//
// It reads gitutil.Status narrowed to the ledger, whose entries carry each path
// verbatim; a rename's source is not marked, since only the destination is a
// file in the ledger now.
func uncommittedLedgerPaths(repoRoot, issuesRoot string) (map[string]bool, bool) {
	if !fsutil.PathWithin(issuesRoot, repoRoot, false) {
		return nil, false
	}
	rel, err := filepath.Rel(repoRoot, issuesRoot)
	if err != nil {
		return nil, false
	}
	entries, err := gitutil.Status(repoRoot, maxStatusBytes, gitutil.StatusOptions{Pathspecs: []string{filepath.ToSlash(rel)}})
	if err != nil {
		return nil, false
	}
	set := map[string]bool{}
	for _, e := range entries {
		set[e.Path] = true
	}
	return set, true
}

// markUncommitted sets Uncommitted on each issue git reports as not committed.
// Issue paths must already be repo-relative (relativiseLedgerPaths).
func markUncommitted(set map[string]bool, issues []Issue) int {
	n := 0
	for i := range issues {
		if set[filepath.ToSlash(issues[i].Path)] {
			issues[i].Uncommitted = true
			n++
		}
	}
	return n
}
