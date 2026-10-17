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
// git hashes standard input, which the isolated command leaves empty, so the
// answer is the hash of zero bytes typed as a tree on every platform.
func EmptyTree(root string) (string, error) {
	out, err := Run(root, "hash-object", "-w", "-t", "tree", "--stdin")
	if err != nil {
		return "", fmt.Errorf("git could not name the empty tree: %w", err)
	}
	if !IsFullSHA(out) {
		return "", fmt.Errorf("git named the empty tree %q, which is not an object name", out)
	}
	return out, nil
}
