package guard

import (
	"path/filepath"
	"strings"
)

// The delete proof (iss-2610241239492159). A branch delete, local or on a
// remote, and a worktree removal lose work when their target holds any that
// is not kept elsewhere, and whether it does is a fact about the repository,
// not about the command's words. So the registry's delete entries only warn,
// and this file turns each delete the line makes into a block unless a
// DeleteProver, which the front door attaches, proves its target safe.
//
// It reads the delete entries as the trigger: only a line one of them fired
// on is read for targets, so a git command they do not name is never slowed
// by a proof. Every doubt blocks — a target the line does not fix, an option
// this reader does not know, a directory change before the delete, a delete
// inside a script the line runs, no prover attached, and a prover that cannot
// answer — because a delete that is wrongly refused costs a person one
// command, and one wrongly allowed costs the work.
//
// The trust model is the guard's: a careless agent, not a hostile one. The
// guard is a mistake filter, and this proof is what makes the mistake of
// deleting unfinished work one it catches.

// deleteEntryID is the reserved id a delete the guard could not prove safe is
// refused under.
const deleteEntryID = "git-delete-unproven"

// familyDelete is the family the refusal names.
const familyDelete = "git delete"

// deleteEntryIDs are the registry entries whose match marks a line as a
// delete: the bundled warns for a remote branch delete, a forced local branch
// delete and a worktree removal.
var deleteEntryIDs = map[string]bool{
	"git-push-delete":         true,
	"git-push-delete-refspec": true,
	"git-branch-force-delete": true,
	"git-branch-delete-force": true,
	"git-worktree-remove":     true,
}

// DeleteKind is what a DeleteTarget deletes.
type DeleteKind string

const (
	// DeleteRemoteBranch is a branch deleted on a remote: `git push <remote>
	// --delete <branch>` or `git push <remote> :<branch>`.
	DeleteRemoteBranch DeleteKind = "remote-branch"
	// DeleteLocalBranch is a local branch force-deleted: `git branch -D
	// <branch>` or `git branch -d -f <branch>`.
	DeleteLocalBranch DeleteKind = "local-branch"
	// DeleteWorktree is a worktree removed: `git worktree remove [--force]
	// <path>`.
	DeleteWorktree DeleteKind = "worktree"
)

// DeleteTarget is one thing a line deletes, as the line fixes it.
type DeleteTarget struct {
	Kind DeleteKind
	// Dir is the absolute directory git runs in: the command's working
	// directory, or the one `git -C` names.
	Dir string
	// Remote is the remote a remote branch is deleted on.
	Remote string
	// Branch is the branch deleted, without refs/heads/.
	Branch string
	// Path is the absolute path of the worktree removed.
	Path string
}

// String names the target for a refusal.
func (t DeleteTarget) String() string {
	switch t.Kind {
	case DeleteRemoteBranch:
		return "the branch " + t.Branch + " on " + t.Remote
	case DeleteLocalBranch:
		return "the local branch " + t.Branch
	default:
		return "the worktree " + t.Path
	}
}

// DeleteProof is a prover's answer for one target: safe, or the check that
// failed and what would settle it.
type DeleteProof struct {
	Safe bool
	// Failed is the check that did not pass, or could not answer.
	Failed string
	// Settle is what a person or agent can do to make the target safe.
	Settle string
}

// DeleteProver proves a delete's target safe: merged into the default branch,
// with no open pull request, in no worktree and held by no live reservation
// for a branch; clean, used by no live process and on a merged branch for a
// worktree. It is attached by the front door (ProvingDeletes), because it
// runs git and the forge client, which the core does not choose.
type DeleteProver interface {
	ProveDelete(t DeleteTarget) DeleteProof
}

// ProvingDeletes returns the registry with p proving its deletes. A registry
// loaded for a repository with no prover refuses every delete its entries
// name.
func (r Registry) ProvingDeletes(p DeleteProver) Registry {
	r.deletes = p
	return r
}

// deleteSignal is the refusal of the deletes v's line makes that the prover
// does not prove safe, or false when the line makes none or every one is
// proven.
func (r Registry) deleteSignal(v verdicts) (payloadSignal, bool) {
	// A registry with no repository behind it (the bundled defaults on their
	// own) has no delete to prove, as for the shared stash (stash.go): its
	// entries speak with their own tier. Every front door loads one for a
	// repository, and there a delete no prover proves is refused.
	if r.worktrees == nil {
		return payloadSignal{}, false
	}
	line, script := matchedDelete(v)
	if !line && !script {
		return payloadSignal{}, false
	}
	if script {
		return deleteRefusal([]string{"a script the line runs deletes a branch or a worktree, and a delete inside a script cannot be proven"}, []string{
			"Run the delete as its own command, so its targets can be proven.",
		}), true
	}
	dir := ""
	if r.files != nil {
		dir = r.files.dir
	}
	targets, read, why := deleteTargets(v.segs, dir)
	for id, seg := range v.hitSeg {
		if why == "" && deleteEntryIDs[id] && !read[seg] {
			why = "the delete's targets could not be read from the line"
		}
	}
	if why != "" {
		return deleteRefusal([]string{why}, []string{
			"Write the delete with every target spelled out, one git command, no directory change before it and no variable in it, so its targets can be proven.",
		}), true
	}
	if r.deletes == nil {
		return deleteRefusal([]string{"no proof of the delete's targets can be made here"}, nil), true
	}
	var failed, settle []string
	for _, t := range targets {
		p := r.deletes.ProveDelete(t)
		if p.Safe {
			continue
		}
		failed = append(failed, t.String()+": "+p.Failed)
		if p.Settle != "" && !containsString(settle, p.Settle) {
			settle = append(settle, p.Settle)
		}
	}
	if len(failed) == 0 {
		return payloadSignal{}, false
	}
	return deleteRefusal(failed, settle), true
}

// matchedDelete reports whether a delete entry fired on the line itself, and
// whether one fired on a script the line runs.
func matchedDelete(v verdicts) (line, script bool) {
	for _, ids := range [][]string{v.blockers, v.unnamedBlockers, v.warns, v.unnamedWarns} {
		for _, id := range ids {
			if deleteEntryIDs[id] {
				line = true
			}
		}
	}
	for _, s := range v.signals {
		for _, id := range s.also {
			if deleteEntryIDs[id] {
				script = true
			}
		}
	}
	return line, script
}

// deleteRefusal is the block itself: what failed, what settles it, and the
// person's one-command way past it.
func deleteRefusal(failed, settle []string) payloadSignal {
	successor := strings.Join(settle, " ")
	if successor != "" {
		successor += " "
	}
	successor += "If the person decides it is safe anyway, they can allow this one exact command, once, by running " +
		"`abcd guard allow --once --command '<the command>'` themselves; an agent cannot."
	return payloadSignal{
		id:      deleteEntryID,
		verdict: VerdictBlock,
		family:  familyDelete,
		reason: "This deletes work the guard could not prove is kept elsewhere (" + strings.Join(failed, "; ") +
			"), and a deleted branch or worktree takes with it whatever only it held.",
		successor: successor,
	}
}

// deleteTargets reads every delete the line's segments make. read holds the
// index of each segment that is a deleting git command read whole, even one
// that deletes nothing the proof judges (`git branch -D -r`); why is non-empty when a
// segment may delete something the reading cannot name, and the line is then
// refused whole.
func deleteTargets(segs []segment, dir string) (targets []DeleteTarget, read map[int]bool, why string) {
	if dir == "" || !filepath.IsAbs(dir) {
		return nil, nil, "the directory the command runs in is not known"
	}
	for _, s := range segs {
		if len(s.tokens) > 0 && (s.tokens[0] == "cd" || s.tokens[0] == "pushd" || s.tokens[0] == "popd") {
			return nil, nil, "the line changes directory, so which repository it deletes in is not fixed"
		}
	}
	read = map[int]bool{}
	for si, s := range segs {
		for _, site := range sitesNamed(s, "git") {
			for _, w := range s.tokens[:site.idx] {
				if strings.Contains(w, "GIT_") {
					return nil, nil, "git's repository is set from the environment (" + w + ")"
				}
			}
			ts, ok, w := deletesIn(s.tokens[site.idx+1:], dir)
			if w != "" {
				return nil, nil, w
			}
			if ok {
				read[si] = true
			}
			targets = append(targets, ts...)
		}
	}
	return targets, read, ""
}

// deletesIn reads one git command's arguments for the deletes it makes, and
// whether it was a deleting subcommand read whole.
func deletesIn(args []string, dir string) ([]DeleteTarget, bool, string) {
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		switch a := args[i]; {
		case a == "-C":
			if i+1 >= len(args) || isUnknown(args[i+1]) {
				return nil, false, "git -C names a directory the line does not fix"
			}
			d := args[i+1]
			if !filepath.IsAbs(d) {
				d = filepath.Join(dir, d)
			}
			dir = filepath.Clean(d)
			i += 2
		case a == "-c":
			i += 2
		case a == "--no-pager", a == "-P", a == "--no-optional-locks", a == "--no-replace-objects",
			a == "--literal-pathspecs", a == "--no-advice":
			i++
		default:
			if mayDelete(args[i:]) {
				return nil, false, "git's option " + a + " is one this reading does not know"
			}
			return nil, false, ""
		}
	}
	if i >= len(args) {
		return nil, false, ""
	}
	sub, rest := args[i], args[i+1:]
	if isUnknown(sub) {
		if mayDelete(args[i:]) {
			return nil, false, "git's subcommand is not fixed by the line"
		}
		return nil, false, ""
	}
	switch sub {
	case "push":
		ts, why := pushDeletes(rest, dir)
		return ts, why == "", why
	case "branch":
		ts, why := branchDeletes(rest, dir)
		return ts, why == "", why
	case "worktree":
		ts, why := worktreeRemoves(rest, dir)
		return ts, why == "", why
	}
	return nil, false, ""
}

// mayDelete reports whether args could still reach one of the deleting
// subcommands, for an option or subcommand the reading cannot place.
func mayDelete(args []string) bool {
	for _, a := range args {
		if isUnknown(a) || a == "push" || a == "branch" || a == "worktree" {
			return true
		}
	}
	return false
}

// longOption is the option of git <sub> a long spelling names: the exact
// option, or the one it is an unambiguous prefix of, or "" for none.
func longOption(sub, arg string) string {
	name, _, _ := strings.Cut(arg, "=")
	var hit string
	for _, o := range gitLongOptions[sub] {
		if o == name {
			return o
		}
		if strings.HasPrefix(o, name) {
			if hit != "" {
				return ""
			}
			hit = o
		}
	}
	return hit
}

// pushValueOptions are git push's long options that take the next word as
// their value when written without `=`.
var pushValueOptions = map[string]bool{"--repo": true, "--receive-pack": true, "--exec": true, "--push-option": true}

// pushDeletes reads `git push`'s arguments: the remote, then the refspecs, each
// deleted under --delete and each written `:<branch>` deleted without it.
func pushDeletes(args []string, dir string) ([]DeleteTarget, string) {
	del := false
	var ops []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if isUnknown(a) {
			return nil, "a word of the push is not fixed by the line"
		}
		switch {
		case a == "--":
			ops = append(ops, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "--"):
			name, _, hasValue := strings.Cut(a, "=")
			o := longOption("push", a)
			if o == "" {
				// An ambiguous prefix of --delete reads as it (`--d`), the
				// fail-closed side; any other unknown spelling is refused.
				if len(name) > 2 && strings.HasPrefix("--delete", name) {
					del = true
					continue
				}
				return nil, "git push's option " + a + " is one this reading does not know"
			}
			switch {
			case o == "--delete":
				del = true
			case o == "--no-delete":
				del = false
			case pushValueOptions[o] && !hasValue:
				i++
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			for j := 1; j < len(a); j++ {
				switch a[j] {
				case 'd':
					del = true
				case 'o':
					if j == len(a)-1 {
						i++
					}
					j = len(a)
				case 'v', 'q', 'n', 'f', 'u', '4', '6':
				default:
					return nil, "git push's option -" + string(a[j]) + " is one this reading does not know"
				}
			}
		default:
			ops = append(ops, a)
		}
	}
	if len(ops) == 0 {
		if del {
			return nil, "the push names no remote to delete on"
		}
		return nil, ""
	}
	remote, refspecs := ops[0], ops[1:]
	var out []DeleteTarget
	for _, rs := range refspecs {
		name := rs
		if !del {
			src, dst, ok := strings.Cut(strings.TrimPrefix(rs, "+"), ":")
			if !ok || src != "" {
				continue
			}
			name = dst
		}
		b, why := branchOperand(name)
		if why != "" {
			return nil, why
		}
		out = append(out, DeleteTarget{Kind: DeleteRemoteBranch, Dir: dir, Remote: remote, Branch: b})
	}
	if del && len(out) == 0 {
		return nil, "the push deletes but names no branch"
	}
	return out, ""
}

// branchOperand is a deleted ref as a branch name, or why it is not one this
// proof can judge.
func branchOperand(ref string) (string, string) {
	b := strings.TrimPrefix(ref, "refs/heads/")
	switch {
	case isUnknown(ref):
		return "", "a deleted branch is not fixed by the line"
	case b == "":
		return "", "a delete names an empty ref"
	case strings.HasPrefix(b, "refs/"):
		return "", ref + " is not a branch, and only a branch delete can be proven safe"
	case strings.ContainsAny(b, "*?[\\") || strings.HasPrefix(b, "-"):
		return "", ref + " is not a branch name the line fixes"
	}
	return b, ""
}

// branchDeletes reads `git branch`'s arguments for a forced delete: -D, or
// -d with -f. A delete of remote-tracking refs (-r) removes only this copy's
// record of another repository's branch, so it is no target.
func branchDeletes(args []string, dir string) ([]DeleteTarget, string) {
	del, force, remotes := false, false, false
	var ops []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if isUnknown(a) {
			return nil, "a word of the branch command is not fixed by the line"
		}
		switch {
		case a == "--":
			ops = append(ops, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "--"):
			switch longOption("branch", a) {
			case "--delete":
				del = true
			case "--force":
				force = true
			case "--remotes":
				remotes = true
			case "--quiet", "--verbose":
			default:
				return nil, "git branch's option " + a + " is one this reading does not know"
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			for j := 1; j < len(a); j++ {
				switch a[j] {
				case 'd':
					del = true
				case 'D':
					del, force = true, true
				case 'f':
					force = true
				case 'r':
					remotes = true
				case 'q', 'v':
				default:
					return nil, "git branch's option -" + string(a[j]) + " is one this reading does not know"
				}
			}
		default:
			ops = append(ops, a)
		}
	}
	if !del || !force || remotes {
		return nil, ""
	}
	var out []DeleteTarget
	for _, op := range ops {
		b, why := branchOperand(op)
		if why != "" {
			return nil, why
		}
		out = append(out, DeleteTarget{Kind: DeleteLocalBranch, Dir: dir, Branch: b})
	}
	if len(out) == 0 {
		return nil, "the branch delete names no branch"
	}
	return out, ""
}

// worktreeRemoves reads `git worktree remove [--force] <path>`.
func worktreeRemoves(args []string, dir string) ([]DeleteTarget, string) {
	if len(args) == 0 || args[0] != "remove" {
		if len(args) > 0 && isUnknown(args[0]) {
			return nil, "git worktree's subcommand is not fixed by the line"
		}
		return nil, ""
	}
	var ops []string
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case isUnknown(a):
			return nil, "the worktree removed is not fixed by the line"
		case a == "--":
			ops = append(ops, args[i+1:]...)
			i = len(args)
		case a == "-f" || a == "-ff" || (len(a) > 2 && strings.HasPrefix("--force", a)):
		case strings.HasPrefix(a, "-"):
			return nil, "git worktree remove's option " + a + " is one this reading does not know"
		default:
			ops = append(ops, a)
		}
	}
	if len(ops) != 1 {
		return nil, "git worktree remove names no single worktree"
	}
	p := ops[0]
	if strings.ContainsAny(p, "*?[") {
		return nil, "the worktree removed is a pattern, not a path the line fixes"
	}
	if strings.HasPrefix(p, "~") {
		return nil, "the worktree removed starts with ~, which the reading does not expand"
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return []DeleteTarget{{Kind: DeleteWorktree, Dir: dir, Path: filepath.Clean(p)}}, ""
}
