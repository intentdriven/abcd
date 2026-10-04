package interview

// treewatch.go holds an AI-written interview's role to the paths its contract
// lets it change. The role runs with the person's own route in the checkout,
// and with no host session nobody watches its edits between the questions
// abcd draws, so abcd does: before each dispatch it reads the working tree's
// state, after the dispatch it reads it again, and a path whose state moved
// that the interview did not grant stops the interview, naming each.
//
// The state is git's own listing of what differs from HEAD (`git status
// --porcelain=v1 -z --untracked-files=all`, run through gitutil's isolated
// environment, as the reading assembler's dirty-path gate and the capture
// ledger's uncommitted marker read it), each listed path paired with its
// content's hash and mode, so a second edit to a file already changed before
// the dispatch moves its state too. Paths are relative to the repository's
// root; the local tier, where abcd keeps the turns and the records, is not
// watched.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxStatusBytes bounds the status listing; a tree whose listing exceeds it
// is refused rather than read in part.
const maxStatusBytes = 8 << 20

// treeState is each path git lists as differing from HEAD, outside the local
// tier, mapped to its status and its content's mode and hash.
type treeState map[string]string

// UnexpectedChangesError is an interview stopped because its role changed
// paths the interview does not let it change.
type UnexpectedChangesError struct {
	Role string
	// Paths are the paths changed, relative to the repository's root.
	Paths []string
}

func (e *UnexpectedChangesError) Error() string {
	return fmt.Sprintf("the %s changed %d path(s) this interview does not let it change, so the interview stopped: %s; "+
		"read each (git status, git diff) and restore what you did not ask for",
		e.Role, len(e.Paths), strings.Join(e.Paths, ", "))
}

// readTree reads the working tree's state under repo.
func readTree(repo string) (treeState, error) {
	out, err := gitutil.RunCapped(repo, maxStatusBytes, "--no-optional-locks", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, fmt.Errorf("interview: the working tree's state cannot be read, so the role's changes cannot be held to its contract: %w", err)
	}
	root, err := os.OpenRoot(repo)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	st := treeState{}
	add := func(status, p string) error {
		if p == localTierRel || strings.HasPrefix(p, localTierRel+"/") {
			return nil
		}
		h, err := contentState(root, strings.TrimSuffix(p, "/"))
		if err != nil {
			return err
		}
		st[p] = status + " " + h
		return nil
	}
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 4 {
			continue
		}
		status := rec[:2]
		if err := add(status, rec[3:]); err != nil {
			return nil, err
		}
		// A rename or copy carries its source as the following record.
		if status[0] == 'R' || status[0] == 'C' || status[1] == 'R' || status[1] == 'C' {
			i++
			if i < len(records) && records[i] != "" {
				if err := add(status+"<", records[i]); err != nil {
					return nil, err
				}
			}
		}
	}
	return st, nil
}

// contentState is rel's kind, mode and content hash: a link's target is
// hashed, never followed.
func contentState(root *os.Root, rel string) (string, error) {
	fi, err := root.Lstat(rel)
	if errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	h := sha256.New()
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		target, err := root.Readlink(rel)
		if err != nil {
			return "", err
		}
		h.Write([]byte(target))
	case fi.Mode().IsRegular():
		f, err := root.Open(rel)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%v %s", fi.Mode(), hex.EncodeToString(h.Sum(nil))), nil
}

// changedSince is every path whose state differs between before and after,
// sorted.
func changedSince(before, after treeState) []string {
	var out []string
	for p, s := range after {
		if before[p] != s {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
