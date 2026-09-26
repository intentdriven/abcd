package capture

import (
	"path/filepath"
	"strings"

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
// It reads `git status --porcelain=v1 -z -uall`, whose NUL-terminated records
// carry each path verbatim; a rename's source record is skipped, since only the
// destination is a file in the ledger now.
func uncommittedLedgerPaths(repoRoot, issuesRoot string) (map[string]bool, bool) {
	if !fsutil.PathWithin(issuesRoot, repoRoot, false) {
		return nil, false
	}
	rel, err := filepath.Rel(repoRoot, issuesRoot)
	if err != nil {
		return nil, false
	}
	out, err := gitutil.RunCapped(repoRoot, maxStatusBytes,
		"status", "--porcelain=v1", "-z", "--untracked-files=all", "--", filepath.ToSlash(rel))
	if err != nil {
		return nil, false
	}
	set := map[string]bool{}
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 4 {
			continue
		}
		set[rec[3:]] = true
		if st := rec[:2]; st[0] == 'R' || st[0] == 'C' || st[1] == 'R' || st[1] == 'C' {
			i++
		}
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
