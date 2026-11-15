package gitutil

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// ErrObjectMismatch is an object whose content does not hash to the name it
// was read by, or is not of the type the reader asked for.
var ErrObjectMismatch = errors.New("an object does not hash to its name")

// TreeEntry is one entry of a verified tree that is not itself a tree: a file
// (100644 or 100755), a symbolic link (120000) or a submodule's commit
// (160000), with its object name.
type TreeEntry struct{ Mode, OID string }

// maxVerifiedTreeObjects and maxVerifiedTreeBytes bound one Tree walk: how
// many commit and tree objects it reads, and how many bytes of them in all.
const (
	maxVerifiedTreeObjects = 1 << 20
	maxVerifiedTreeBytes   = 64 << 20
)

// VerifiedObjects reads a repository's objects through one isolated `git
// cat-file --batch`, as stored (asStored), and re-hashes each against its
// name, in the repository's object format, before handing it on. git checks
// the hash of a commit and of a root tree when it parses them, but `git
// ls-tree`, `git archive` and `git cat-file` read any other object as its
// file holds it, so an object written over a subtree or a blob of a commit
// would otherwise put other files where the commit's belong. A Tree walk and
// the Blob reads after it share the one process.
type VerifiedObjects struct {
	newHash func() hash.Hash
	cmd     *exec.Cmd
	in      io.WriteCloser
	out     *bufio.Reader
}

// OpenVerifiedObjects starts the reader for the repository at root, git
// killed when ctx ends. Close ends it.
func OpenVerifiedObjects(ctx context.Context, root string) (*VerifiedObjects, error) {
	format, err := RunLimitedContext(ctx, root, 64, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, err
	}
	v := &VerifiedObjects{}
	switch f := strings.TrimSpace(format); f {
	case "sha1":
		v.newHash = sha1.New
	case "sha256":
		v.newHash = sha256.New
	default:
		return nil, fmt.Errorf("the repository's object format is %q, which no object can be checked against", f)
	}
	cmd := asStored(isolatedGitContext(ctx, root, "cat-file", "--batch"))
	if cmd.Err != nil {
		return nil, cmd.Err
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	v.cmd, v.in, v.out = cmd, in, bufio.NewReader(out)
	return v, nil
}

// Close ends git. A read may have stopped at a refusal with git still writing
// an answer nobody reads, so git is killed rather than waited for.
func (v *VerifiedObjects) Close() {
	_ = v.in.Close()
	_ = v.cmd.Process.Kill()
	_ = v.cmd.Wait()
}

// read reads the object named oid, which must be of type kind, and checks its
// content against oid. budget, when not nil, is what the caller may still
// read in bytes, and is spent.
func (v *VerifiedObjects) read(oid, kind string, budget *int) ([]byte, error) {
	if len(oid) != 2*v.newHash().Size() || strings.Trim(oid, "0123456789abcdef") != "" {
		return nil, fmt.Errorf("%q is not a full object name in the repository's format", oid)
	}
	if _, err := io.WriteString(v.in, oid+"\n"); err != nil {
		return nil, err
	}
	header, err := v.out.ReadString('\n')
	if err != nil {
		return nil, err
	}
	f := strings.Fields(header)
	if len(f) != 3 || f[0] != oid {
		return nil, fmt.Errorf("git cat-file: %q for %s", strings.TrimSpace(header), oid)
	}
	size, err := strconv.Atoi(f[2])
	if err != nil || size < 0 {
		return nil, fmt.Errorf("object %s has an unreadable size", short(oid))
	}
	if budget != nil {
		if *budget -= size; *budget < 0 {
			return nil, fmt.Errorf("the commit's trees exceed %d bytes", maxVerifiedTreeBytes)
		}
	}
	body := make([]byte, size+1) // the content and its newline
	if _, err := io.ReadFull(v.out, body); err != nil {
		return nil, err
	}
	body = body[:size]
	if f[1] != kind {
		return nil, fmt.Errorf("object %s is a %s, not the %s it is read as: %w", short(oid), f[1], kind, ErrObjectMismatch)
	}
	h := v.newHash()
	fmt.Fprintf(h, "%s %d\x00", kind, size)
	h.Write(body)
	if hex.EncodeToString(h.Sum(nil)) != oid {
		return nil, fmt.Errorf("object %s does not hash to its name (the object store was altered): %w", short(oid), ErrObjectMismatch)
	}
	return body, nil
}

// Blob reads the blob named oid, checked against its name.
func (v *VerifiedObjects) Blob(oid string) ([]byte, error) {
	return v.read(oid, "blob", nil)
}

// Tree reads commit and every tree under it, each checked against its name,
// and returns every entry that is not a tree by its slash-separated path. An
// entry named ".", ".." or holding a slash, a mode git does not write, or a
// tree that is not well formed is an error, as is a walk past
// maxVerifiedTreeObjects objects or maxVerifiedTreeBytes bytes.
func (v *VerifiedObjects) Tree(commit string) (map[string]TreeEntry, error) {
	rawLen := v.newHash().Size()
	objects, budget := 0, maxVerifiedTreeBytes
	read := func(oid, kind string) ([]byte, error) {
		if objects++; objects > maxVerifiedTreeObjects {
			return nil, fmt.Errorf("the commit holds more than %d trees", maxVerifiedTreeObjects)
		}
		return v.read(oid, kind, &budget)
	}
	body, err := read(commit, "commit")
	if err != nil {
		return nil, err
	}
	first, _, _ := strings.Cut(string(body), "\n")
	rootTree, ok := strings.CutPrefix(first, "tree ")
	if !ok || len(rootTree) != 2*rawLen {
		return nil, errors.New("the commit names no tree")
	}
	entries := map[string]TreeEntry{}
	type pending struct{ dir, oid string }
	queue := []pending{{"", rootTree}}
	for len(queue) > 0 {
		t := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		body, err := read(t.oid, "tree")
		if err != nil {
			return nil, err
		}
		for len(body) > 0 {
			sp := bytes.IndexByte(body, ' ')
			nul := bytes.IndexByte(body, 0)
			if sp <= 0 || nul <= sp+1 || len(body) < nul+1+rawLen {
				return nil, fmt.Errorf("tree %s is malformed", short(t.oid))
			}
			mode, name := string(body[:sp]), string(body[sp+1:nul])
			oid := hex.EncodeToString(body[nul+1 : nul+1+rawLen])
			body = body[nul+1+rawLen:]
			if name == "." || name == ".." || strings.Contains(name, "/") {
				return nil, fmt.Errorf("tree %s holds an entry named %q", short(t.oid), name)
			}
			p := name
			if t.dir != "" {
				p = t.dir + "/" + name
			}
			switch mode {
			case "40000":
				queue = append(queue, pending{p, oid})
			case "100644", "100755", "120000", "160000":
				entries[p] = TreeEntry{Mode: mode, OID: oid}
			default:
				return nil, fmt.Errorf("tree %s holds %s at an unknown mode %s", short(t.oid), p, mode)
			}
		}
	}
	return entries, nil
}

// short is an object name's first twelve characters, for a message.
func short(oid string) string {
	if len(oid) > 12 {
		return oid[:12]
	}
	return oid
}
