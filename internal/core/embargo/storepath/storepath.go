// Package storepath names where a repository's embargo store lives
// (adr-2610120558328615). It stands apart from package embargo so a reader
// that must skip the store, the peers listing, can name it without importing
// the capture code embargo writes its private records through: capture imports
// site, site claim, and claim peers, so peers importing embargo is a cycle.
package storepath

import (
	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Name is the store's folder under the repository's level of the worktree
// store.
const Name = "embargo"

// Path is where the store of the repository at repoRoot lives under home:
// worktrees/<root-sha>/embargo in abcd's home, keyed on the root commit so
// every worktree of the repository finds the same store
// (adr-2610031751065746). The root commit is origin's default branch's, read
// as stored (gitutil.DefaultBranchRootCommit), not HEAD's: a store that is
// not found reads as nothing under embargo, so a checkout on an orphan branch
// or behind a replace ref must still find it. ok is false when the repository
// has no root commit to key it on.
func Path(home, repoRoot string) (p string, ok bool) {
	sha := gitutil.DefaultBranchRootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return "", false
	}
	return abcdhome.Path(home, "worktrees", sha, Name), true
}
