package gitutil

import (
	"strconv"
	"strings"
)

// Worktree is one record of `git worktree list --porcelain`: a working tree
// this repository knows about, as git names it.
type Worktree struct {
	Path   string
	Head   string // the object name HEAD points at; "" for a bare entry
	Branch string // the full ref ("refs/heads/…"); "" when detached or bare
	Bare   bool
	// Detached is git's `detached` attribute: HEAD names a commit, not a branch.
	Detached bool
	// Locked is git's `locked` attribute, with its reason when one was given
	// ("" for a bare `locked`): git refuses to remove or prune the worktree.
	Locked     bool
	LockReason string
	// Prunable is git's `prunable` attribute, with its reason: git would drop
	// the entry on `git worktree prune` (most often its directory is gone).
	Prunable       bool
	PrunableReason string
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
// first record, and one this reader does not know, is ignored. A `locked` or
// `prunable` reason is git's own text: the newline form C-quotes a reason
// holding a newline or a quote, and it is unquoted here (reasonOf).
func ParseWorktreeList(out, sep string) []Worktree {
	var wts []Worktree
	p := porcelain{lines: sep == "\n", emit: func(w Worktree) { wts = append(wts, w) }}
	for _, field := range strings.Split(out, sep) {
		p.field(field)
	}
	p.end()
	return wts
}

// porcelain folds the attributes of a porcelain worktree listing, one at a
// time, into records, handing each to emit once the next record opens or the
// listing ends: the one parser ParseWorktreeList and EachWorktree share.
type porcelain struct {
	lines bool // the newline-separated form, whose line ends may carry a CR
	emit  func(Worktree)
	cur   *Worktree
}

func (p *porcelain) field(field string) {
	if p.lines {
		// A CRLF-emitting git's line end. In the -z form a trailing CR is
		// part of the path and stays.
		field = strings.TrimSuffix(field, "\r")
	}
	key, val, _ := strings.Cut(field, " ")
	if key == "worktree" {
		p.end()
		p.cur = &Worktree{Path: val}
		return
	}
	if p.cur == nil {
		return
	}
	switch key {
	case "HEAD":
		p.cur.Head = val
	case "branch":
		p.cur.Branch = val
	case "bare":
		p.cur.Bare = true
	case "detached":
		p.cur.Detached = true
	case "locked":
		p.cur.Locked, p.cur.LockReason = true, p.reasonOf(val)
	case "prunable":
		p.cur.Prunable, p.cur.PrunableReason = true, p.reasonOf(val)
	}
}

// reasonOf is a lock or prune reason as git meant it. The -z form never quotes,
// so its bytes are the reason. The newline form C-quotes a reason that holds a
// newline, a quote or a control byte (git-worktree(1)); git's C quoting is a
// subset of a Go string literal's escapes (\n, \t, \", \\ and three-digit
// octal), so strconv.Unquote reads it. An unquoted reason never holds a quote,
// so one led by a quote is quoted; one that does not unquote is kept as it
// stands rather than dropped.
func (p *porcelain) reasonOf(val string) string {
	if !p.lines || len(val) < 2 || val[0] != '"' || val[len(val)-1] != '"' {
		return val
	}
	if u, err := strconv.Unquote(val); err == nil {
		return u
	}
	return val
}

// end hands over the record being read, if any.
func (p *porcelain) end() {
	if p.cur != nil {
		p.emit(*p.cur)
		p.cur = nil
	}
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
	p := porcelain{lines: sep == '\n', emit: fn}
	n, err := streamFields(isolatedGit(root, args...), sep, maxWorktreeField, func(field string) error {
		p.field(field)
		return nil
	})
	if err != nil {
		return n, err
	}
	p.end()
	return n, nil
}
