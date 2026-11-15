package gittest

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// ForgeCommitGraphTree rewrites the root tree that the commit-graph file of the
// repository whose git directory is gitDir records for commit, to tree, and
// recomputes the file's trailing checksum, so the graph is well formed and
// claims a tree the commit does not store. The graph must already be written
// (`git commit-graph write --reachable`) and list commit; a git that reads the
// graph then takes tree as the commit's tree.
func ForgeCommitGraphTree(t *testing.T, gitDir, commit, tree string) {
	t.Helper()
	path := filepath.Join(gitDir, "objects", "info", "commit-graph")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("gittest: %v", err)
	}
	if len(b) < 8 || string(b[:4]) != "CGPH" {
		t.Fatalf("gittest: %s is not a commit-graph", path)
	}
	h := sha1.Size
	if b[5] == 2 {
		h = sha256.Size
	}
	chunks := map[string]uint64{}
	for i := 0; i < int(b[6]); i++ {
		e := b[8+12*i:]
		chunks[string(e[:4])] = binary.BigEndian.Uint64(e[4:12])
	}
	oidl, cdat := chunks["OIDL"], chunks["CDAT"]
	if oidl == 0 || cdat == 0 {
		t.Fatalf("gittest: %s has no OIDL or CDAT chunk", path)
	}
	want, err := hex.DecodeString(commit)
	if err != nil {
		t.Fatalf("gittest: commit %q: %v", commit, err)
	}
	treeID, err := hex.DecodeString(tree)
	if err != nil || len(treeID) != h {
		t.Fatalf("gittest: tree %q is not an object id of the graph's format (%v)", tree, err)
	}
	n := int(binary.BigEndian.Uint32(b[oidl-4 : oidl])) // the fanout's last entry
	for i := 0; i < n; i++ {
		if !bytes.Equal(b[int(oidl)+i*h:int(oidl)+(i+1)*h], want) {
			continue
		}
		copy(b[int(cdat)+i*(h+16):], treeID)
		var sum []byte
		if h == sha256.Size {
			s := sha256.Sum256(b[:len(b)-h])
			sum = s[:]
		} else {
			s := sha1.Sum(b[:len(b)-h])
			sum = s[:]
		}
		copy(b[len(b)-h:], sum)
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatalf("gittest: %v", err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatalf("gittest: %v", err)
		}
		return
	}
	t.Fatalf("gittest: %s does not list %s", path, commit)
}
