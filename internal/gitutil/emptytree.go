package gitutil

import "fmt"

// EmptyTree is the object name of the empty tree in the repository at root,
// for the repository's own object format (4b825dc6… under SHA-1, 6ef19b41…
// under SHA-256), written into its object store so a commit made on it can be
// pushed. It is the one place the name is computed: a reservation commit's
// tree is the empty tree, and the reader and the writer of those commits
// compare against this answer rather than a constant that holds for one
// format only.
//
// There are two forms because writing is the writer's need alone. EmptyTree
// is the writer's: a commit made on the tree needs the object in the store.
// EmptyTreeName is the reader's: a read only compares a tip's tree with the
// name, so it must leave the shared object store untouched (hash-object -w
// writes the object, or refreshes its file's time when it is already there).
//
// git hashes standard input, which the isolated command leaves empty, so the
// answer is the hash of zero bytes typed as a tree on every platform.
func EmptyTree(root string) (string, error) {
	return emptyTree(root, "hash-object", "-w", "-t", "tree", "--stdin")
}

// EmptyTreeName is EmptyTree's name without the write: the same answer, for a
// reader that compares against it and must leave .git/objects as it found
// it. A path that commits on the empty tree uses EmptyTree.
func EmptyTreeName(root string) (string, error) {
	return emptyTree(root, "hash-object", "-t", "tree", "--stdin")
}

// emptyTree runs args, a hash-object of empty standard input typed as a tree,
// and checks that git answered an object name.
func emptyTree(root string, args ...string) (string, error) {
	out, err := Run(root, args...)
	if err != nil {
		return "", fmt.Errorf("git could not name the empty tree: %w", err)
	}
	if !IsFullSHA(out) {
		return "", fmt.Errorf("git named the empty tree %q, which is not an object name", out)
	}
	return out, nil
}
