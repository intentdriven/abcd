// Package abcdrepo names abcd's own repository: the one identity every verb
// that behaves differently in abcd's own checkout, or in a copy of it, reads.
// It is a standard-library leaf, so a light package (a lab, the report inbox)
// can import it without pulling in whatever else holds the rule it serves.
package abcdrepo

import (
	"os"
	"path/filepath"
)

// RootCommit is abcd's own identity as git records it: the root commit of its
// repository. A report is promoted only in a checkout whose root commit is this
// one, and a lab holds its snapshot to the dual-binary gate when the
// repository it studies is this one (or its snapshot looks like abcd's source
// tree, LooksLikeSourceTree).
//
// It is a pinned constant rather than something derived at run time. A root
// commit cannot change without rewriting every commit after it, so the value is
// as fixed as the module path, and a constant fails closed: a shallow clone, a
// rewritten history or an archive copy is refused rather than guessed at. The
// derivations on offer are weaker. The binary's embedded build revision is
// absent from a dirty or checkout-less build, and proves only that the checkout
// holds that one commit; the module path is text any repository can declare.
// TestAbcdRootCommitIsThisCheckouts (internal/core/report) holds the constant
// to the checkout the tests run in, so it cannot drift silently. A fork shares
// abcd's root commit and is abcd's code, so it counts as abcd.
const RootCommit = "488a0aa96ac5de805348635b27036addf15cddc2"

// LooksLikeSourceTree reports whether dir is laid out as abcd's source tree: it
// carries abcd's entry point, cmd/abcd/main.go, as a regular file. The curated
// plugin payload carries no cmd/, so its presence is the tell the binary
// resolution ladder relies on (internal/core/launch/commandladder_test.go); a
// lab reads it too, beside RootCommit, so a shallow or rewritten copy of abcd,
// whose root commit is another, is still held to the dual-binary gate. It is a
// layout check, not an identity: any repository can ship that file.
func LooksLikeSourceTree(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, "cmd", "abcd", "main.go"))
	return err == nil && fi.Mode().IsRegular()
}
