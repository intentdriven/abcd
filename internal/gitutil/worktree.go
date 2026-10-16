package gitutil

import "strings"

// Worktree is one record of `git worktree list --porcelain`: a working tree
// this repository knows about, as git names it.
type Worktree struct {
	Path   string
	Head   string // the object name HEAD points at; "" for a bare entry
	Branch string // the full ref ("refs/heads/…"); "" when detached or bare
	Bare   bool
}

// MaxWorktreeListing is the cap a caller listing a repository's worktrees
// passes ListWorktrees when it has no reason for another: a listing is a few
// hundred bytes a worktree, and 16 MiB bounds a planted repository without
// refusing a real one.
const MaxWorktreeListing = 16 << 20

// ListWorktrees lists every working tree of the repository at root, the main
// one first (git documents that order), with its output capped at maxBytes as
// RunCapped caps it.
//
// It asks for the NUL-separated form first, which is the only one that carries
// a path holding a newline intact, and falls back to the newline-separated form
// when git refuses it: `-z` arrived in git 2.36, and a git older than that (the
// 2.34 an LTS distribution still ships) answers "unknown switch" and exit 129.
// Falling back on any refusal rather than on that exact message keeps the
// fallback independent of how a given git words it; a failure the older form
// shares (not a repository, the cap exceeded) fails there too and is returned.
// The newline form is exact for every path without a newline in it, which is
// the only limit it carries.
func ListWorktrees(root string, maxBytes int) ([]Worktree, error) {
	if out, err := RunCapped(root, maxBytes, "worktree", "list", "--porcelain", "-z"); err == nil {
		return ParseWorktreeList(out, "\x00"), nil
	}
	out, err := RunCapped(root, maxBytes, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(out, "\n"), nil
}

// ParseWorktreeList parses the porcelain output of `git worktree list`, whose
// attributes are separated by sep: "\x00" for the -z form, "\n" otherwise. A
// record opens at its `worktree <path>` attribute; an attribute before the
// first record, and one this reader does not use (locked, prunable, detached),
// is ignored.
func ParseWorktreeList(out, sep string) []Worktree {
	var wts []Worktree
	for _, field := range strings.Split(out, sep) {
		if sep == "\n" {
			// A CRLF-emitting git's line end. In the -z form a trailing CR is
			// part of the path and stays.
			field = strings.TrimSuffix(field, "\r")
		}
		key, val, _ := strings.Cut(field, " ")
		if key == "worktree" {
			wts = append(wts, Worktree{Path: val})
			continue
		}
		if len(wts) == 0 {
			continue
		}
		cur := &wts[len(wts)-1]
		switch key {
		case "HEAD":
			cur.Head = val
		case "branch":
			cur.Branch = val
		case "bare":
			cur.Bare = true
		}
	}
	return wts
}

// maxWorktreeField bounds one field of the streamed listing: a path, an
// object name, or a lock or prune reason. A path is a few KiB at most; the
// bound is far past that, so only a planted reason meets it.
const maxWorktreeField = 1 << 20

// EachWorktree streams every working tree of the repository at root to fn,
// the main one first, as ListWorktrees parses them, without capping the
// listing's total length: a repository with many worktrees is read whole.
// One field past maxWorktreeField is an error, never a record cut short.
//
// It asks for the NUL-separated form first and falls back to the
// newline-separated form, as ListWorktrees does, only when git refused the
// first before handing over any field; a failure partway is returned.
func EachWorktree(root string, fn func(Worktree)) error {
	n, err := eachWorktree(root, 0, fn)
	if err == nil || n > 0 {
		return err
	}
	_, err = eachWorktree(root, '\n', fn)
	return err
}

func eachWorktree(root string, sep byte, fn func(Worktree)) (int, error) {
	args := []string{"worktree", "list", "--porcelain"}
	if sep == 0 {
		args = append(args, "-z")
	}
	var cur *Worktree
	n, err := streamFields(isolatedGit(root, args...), sep, maxWorktreeField, func(field string) error {
		if sep == '\n' {
			field = strings.TrimSuffix(field, "\r")
		}
		key, val, _ := strings.Cut(field, " ")
		if key == "worktree" {
			if cur != nil {
				fn(*cur)
			}
			cur = &Worktree{Path: val}
			return nil
		}
		if cur == nil {
			return nil
		}
		switch key {
		case "HEAD":
			cur.Head = val
		case "branch":
			cur.Branch = val
		case "bare":
			cur.Bare = true
		}
		return nil
	})
	if err != nil {
		return n, err
	}
	if cur != nil {
		fn(*cur)
	}
	return n, nil
}
