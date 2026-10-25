package guard

import (
	"path"
	"strings"
)

// BranchActKind is what a git command does to the branch a BranchAct names.
type BranchActKind string

const (
	// BranchCreate is a branch the command creates: `git worktree add
	// -b|-B <b>`, `git switch -c|-C <b>`, `git checkout -b|-B <b>`, `git
	// branch <b>` and `git branch -c|-C [<old>] <b>`. The recorder keeps it
	// only when no branch of that name exists yet.
	BranchCreate BranchActKind = "create"
	// BranchCheckout is a branch `git worktree add` checks out in a new
	// worktree, creating it when it does not exist: a bare `git worktree add
	// <path>` names the branch after the folder, and `git worktree add <path>
	// <commit-ish>` names the commit-ish.
	BranchCheckout BranchActKind = "checkout"
	// BranchRename is `git branch -m|-M [<old>] <new>`: a record of <old>
	// (the current branch when From is "") is carried to <new>.
	BranchRename BranchActKind = "rename"
)

// BranchAct is one branch a shell command creates, checks out into a new
// worktree or renames, as the created-branch recorder of the away mode reads
// it (spc-2610180914469489 step 3). It is a reading of the command line only:
// whether the branch exists is the recorder's question.
type BranchAct struct {
	Kind BranchActKind
	// Name is the branch the act leaves: the one created, checked out, or
	// renamed to.
	Name string
	// From is a rename's old name, "" for the current branch.
	From string
}

// BranchActs reads command for the branches it creates, checks out into a new
// worktree or renames, in the order written. It refuses nothing and decides
// nothing: a line the tokenizer cannot split, and a word a substitution or a
// variable prints, are read as no act at all.
func BranchActs(command string) []BranchAct {
	segs, err := tokenize(command)
	if err != nil {
		return nil
	}
	var out []BranchAct
	for _, s := range segs {
		out = append(out, branchActsIn(s.tokens)...)
	}
	return out
}

// gitValueOptions are git's global options that take their value as the next
// word.
var gitValueOptions = map[string]bool{
	"-C": true, "-c": true, "--git-dir": true, "--work-tree": true,
	"--namespace": true, "--config-env": true, "--super-prefix": true,
}

func branchActsIn(toks []string) []BranchAct {
	// A word nothing fixes (a substitution's output, a variable's value)
	// makes the whole command no act: the recorder only records, and a
	// branch it cannot name is one it does not record.
	for _, w := range toks {
		if isUnknown(w) {
			return nil
		}
	}
	for len(toks) > 0 && isAssignmentWord(toks[0]) {
		toks = toks[1:]
	}
	if len(toks) == 0 || path.Base(toks[0]) != "git" {
		return nil
	}
	i := 1
	for i < len(toks) && strings.HasPrefix(toks[i], "-") {
		if gitValueOptions[toks[i]] {
			i++
		}
		i++
	}
	if i >= len(toks) {
		return nil
	}
	sub, args := toks[i], toks[i+1:]
	switch sub {
	case "worktree":
		if len(args) > 0 && args[0] == "add" {
			return worktreeAddActs(args[1:])
		}
	case "switch":
		return createFlagActs(args, []string{"-c", "-C", "--create", "--force-create"})
	case "checkout":
		return createFlagActs(args, []string{"-b", "-B"})
	case "branch":
		return branchCommandActs(args)
	}
	return nil
}

// isAssignmentWord is a NAME=value word in command position.
func isAssignmentWord(w string) bool {
	eq := strings.IndexByte(w, '=')
	if eq <= 0 {
		return false
	}
	for j, r := range w[:eq] {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || j > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// flagValue reads a value-taking option at args[i] spelled one of names:
// separate (`-b x`), glued to a short name (`-bx`) or after `=` for a long one
// (`--create=x`). It returns the value and the index of the last word used.
func flagValue(args []string, i int, names []string) (string, int, bool) {
	a := args[i]
	for _, n := range names {
		switch {
		case a == n:
			if i+1 < len(args) {
				return args[i+1], i + 1, true
			}
			return "", i, false
		case strings.HasPrefix(n, "--") && strings.HasPrefix(a, n+"="):
			return a[len(n)+1:], i, true
		case !strings.HasPrefix(n, "--") && len(a) > len(n) && strings.HasPrefix(a, n):
			return a[len(n):], i, true
		}
	}
	return "", i, false
}

// createFlagActs is `git switch` or `git checkout`: a branch only when one of
// the create options names it.
func createFlagActs(args []string, names []string) []BranchAct {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			return nil
		}
		if v, _, ok := flagValue(args, i, names); ok {
			if plausibleBranch(v) {
				return []BranchAct{{Kind: BranchCreate, Name: v}}
			}
			return nil
		}
	}
	return nil
}

// worktreeAddValueOptions are `git worktree add`'s options that take the next
// word, beside -b and -B.
var worktreeAddValueOptions = map[string]bool{"--reason": true}

func worktreeAddActs(args []string) []BranchAct {
	var operands []string
	created := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		if v, j, ok := flagValue(args, i, []string{"-b", "-B"}); ok {
			created, i = v, j
			continue
		}
		switch {
		case a == "-d" || a == "--detach" || a == "--orphan":
			return nil
		case worktreeAddValueOptions[a]:
			i++
		case strings.HasPrefix(a, "-"):
		default:
			operands = append(operands, a)
		}
	}
	if created != "" {
		if plausibleBranch(created) {
			return []BranchAct{{Kind: BranchCreate, Name: created}}
		}
		return nil
	}
	var name string
	switch len(operands) {
	case 1:
		name = path.Base(strings.TrimRight(operands[0], "/"))
	case 2:
		name = operands[1]
	default:
		return nil
	}
	if !plausibleBranch(name) {
		return nil
	}
	return []BranchAct{{Kind: BranchCheckout, Name: name}}
}

// branchPassOptions are `git branch` options a creation may carry; any other
// option makes the command something else (a listing, a delete, an upstream).
var branchPassOptions = map[string]bool{
	"-f": true, "--force": true, "-t": true, "--track": true, "--no-track": true,
	"-q": true, "--quiet": true, "--create-reflog": true,
	"--recurse-submodules": true, "--no-recurse-submodules": true,
}

func branchCommandActs(args []string) []BranchAct {
	var operands []string
	move, copyOp := false, false
	for i, a := range args {
		if a == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		switch {
		case a == "-m" || a == "-M" || a == "--move":
			move = true
		case a == "-c" || a == "-C" || a == "--copy":
			copyOp = true
		case branchPassOptions[a] || strings.HasPrefix(a, "--track="):
		case strings.HasPrefix(a, "-"):
			return nil
		default:
			operands = append(operands, a)
		}
	}
	for _, o := range operands {
		if !plausibleBranch(o) {
			return nil
		}
	}
	switch {
	case move && copyOp:
		return nil
	case move && len(operands) == 1:
		return []BranchAct{{Kind: BranchRename, Name: operands[0]}}
	case move && len(operands) == 2:
		return []BranchAct{{Kind: BranchRename, From: operands[0], Name: operands[1]}}
	case copyOp && len(operands) == 1:
		return []BranchAct{{Kind: BranchCreate, Name: operands[0]}}
	case copyOp && len(operands) == 2:
		return []BranchAct{{Kind: BranchCreate, Name: operands[1]}}
	case move || copyOp:
		return nil
	case len(operands) == 1 || len(operands) == 2:
		return []BranchAct{{Kind: BranchCreate, Name: operands[0]}}
	}
	return nil
}

// plausibleBranch is a word that can name a branch: not empty, not an option,
// at most 255 bytes, with no whitespace, control character, `..`, or any of
// the characters git refuses in a ref name.
func plausibleBranch(b string) bool {
	if b == "" || len(b) > 255 || strings.HasPrefix(b, "-") || strings.Contains(b, "..") ||
		strings.HasSuffix(b, "/") || strings.HasSuffix(b, ".lock") || b == "@" {
		return false
	}
	for _, r := range b {
		if r <= ' ' || r == 0x7f || strings.ContainsRune("~^:?*[\\", r) {
			return false
		}
	}
	return true
}
