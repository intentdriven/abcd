// Package abcdrepo names abcd's own repository: the one identity every verb
// that behaves differently in abcd's own checkout, or in a copy of it, reads.
// It is a standard-library leaf, so a light package (a lab, the report inbox)
// can import it without pulling in whatever else holds the rule it serves.
package abcdrepo

// RootCommit is abcd's own identity as git records it: the root commit of its
// repository. A report is promoted only in a checkout whose root commit is this
// one, and a lab holds its snapshot to the dual-binary gate only when the
// repository it studies is this one.
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
