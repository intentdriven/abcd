package loop

import (
	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

// sourceBuild reports whether this abcd is a build of a source checkout's
// tree, go run's or scripts/abcd.sh's. A test replaces it.
var sourceBuild = func() bool {
	if _, ok := vintage.GoRunModule(); ok {
		return true
	}
	_, _, ok := vintage.CheckoutBuild()
	return ok
}

// newerWriterRemedy is the remedy of a refusal of a state file a newer abcd
// wrote, ending with keep, which tells the person not to remove it. An
// installed release is told to update. A source checkout's build is not:
// `abcd update` swaps the installed binary and leaves the checkout's tree, so
// the build that refused is the same on the next call (iss-2611010041545467).
// It is told to run the command from this directory with the scripts/abcd.sh of
// a newer checkout, which runs its verb in the caller's directory and so reads
// this checkout's state; a worktree is the copy to make, since the checkout
// itself may be one other sessions are working in.
func newerWriterRemedy(keep string) string {
	if sourceBuild() {
		return "this abcd is a build of a source checkout's tree, and `abcd update` does not change it: " +
			"run the command again from this directory with the scripts/abcd.sh of a checkout at least as new as the abcd that wrote it, " +
			"such as a worktree of origin/main (`git fetch origin`, then `git worktree add " + abcdhome.Display("worktrees/<root-sha>/<name>") + " origin/main`, " +
			"then `<that worktree>/scripts/abcd.sh <verb>` here); " + keep
	}
	return "run the newer abcd that wrote it (`abcd update` brings this one up to date); " + keep
}
