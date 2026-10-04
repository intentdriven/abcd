// Package abcdhome is the one place in abcd's Go code that spells the name of
// the user-level home: the folder abcd keeps under the person's home directory
// for its machine-scoped records (the trusted-roots and rules.json
// declarations, the path-entry record, the credential and routing files, the
// transcript, worktree, lab and sources stores, the run logs).
//
// Every other package reaches that folder through Rel, Path or Display, and
// TestOnlyTheHomeResolverNamesTheHome holds them to it, so the folder's name is
// one constant (spc-2610031309233367, step 2). The repository tier, a
// project's own `.abcd/` beside its sources, is a different folder that shares
// the name; it is not spelled here and nothing here reaches it.
//
// The package is a leaf that imports only the standard library, so
// internal/core, internal/surface and cmd can all import it without an edge
// back. It changes only abcd's own folder: the computer's search settings are
// never named in code (adr-2610030720195401), which
// TestNoCodeNamesTheSearchSettings holds.
package abcdhome

import (
	"path/filepath"
	"strings"
)

// name is the home folder abcd keeps under the person's home directory.
const name = ".abcd"

// Rel is the slash path, relative to the person's home directory, of leaf
// below abcd's home: Rel("trusted-roots") is ".abcd/trusted-roots", and Rel()
// is the home folder itself. It is the form the home-scope primitives in
// internal/fsutil take as their rel. The leaves are joined as written and never
// cleaned, so Rel(x) is byte-for-byte the ".abcd/" + x it replaces, and a leaf
// that is not clean ("../x", "") reaches fsutil.ValidRelPath unaltered and is
// refused there as it was before: every guard a reader applies judges exactly
// the bytes it judged when the name was spelled at the call site.
func Rel(leaf ...string) string {
	if len(leaf) == 0 {
		return name
	}
	return name + "/" + strings.Join(leaf, "/")
}

// Path is the filesystem path of leaf below abcd's home inside home, the
// person's home directory: filepath.Join(home, ".abcd", leaf...) in the
// platform's separators, cleaned as filepath.Join cleans.
func Path(home string, leaf ...string) string {
	return filepath.Join(home, filepath.FromSlash(Rel(leaf...)))
}

// Display is leaf below abcd's home in the tilde form a message names it by,
// "~/.abcd/trusted-roots", so a line a person pastes into a shell works and no
// message carries the caller's home path. Like Rel it joins the leaves as
// written, so a placeholder or a trailing slash a message shows survives:
// Display("worktrees/<root-sha>/<name>/") keeps its final slash, and Display()
// is the home folder itself.
func Display(leaf ...string) string {
	return "~/" + Rel(leaf...)
}
