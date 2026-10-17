package guard

// approve.go answers the permission prompts of an agent abcd started without
// a terminal (iss-2610161612203555, dcn-2610161624025493). The runner hands
// the harness a prompt tool that reaches this file, so every prompt the
// harness would put to a person is put to the guard instead: a shell command
// the guard reads whole and clears is approved, and everything else is
// refused with the guard's reason, so the agent can rewrite the command.
//
// The bar is higher than the hook's. The hook stands in front of a person who
// still answers the host's own prompt, so a warn there is shown and runs, and
// a guard that cannot answer fails open. Here nobody answers after the guard,
// so the guard's allow is the approval itself, and only a CLEAN allow
// approves: no entry and no signal, a warn included; no note about the check;
// the registry the repository declares, in force and switched on; every
// program the line runs named by the line, not by a variable, a substitution
// or a glob; no inline code handed to a non-shell interpreter, whose
// payload the guard does not read; and, read from the guard's own parse of
// the line (approvepaths.go), no write or removal whose target the line does
// not fix, none that reaches a path the harness protects, and no git command
// that writes configuration outside the repository's own file or sets a key
// that starts a program. The guard is still a mistake filter, not a
// boundary (the check verb's help says what an allow does not see), and this
// widens nothing past what it reads.

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/shellquote"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The reserved ids an approval's refusal speaks under when no registry entry
// or reading signal does. No registry entry may claim them.
const (
	// unnamedProgramEntryID: a program the line runs is named by a variable,
	// a substitution or a glob, so the guard cannot say which program runs.
	unnamedProgramEntryID = "program-name-unread"
	// inlineCodeEntryID: inline code handed to a non-shell interpreter
	// (python -c, node -e), a payload the guard does not read.
	inlineCodeEntryID = "inline-code-unread"
	// checkNoteEntryID: the check left a note (a script that does not exist),
	// so it did not read everything the line names.
	checkNoteEntryID = "check-note"
	// registryEntryID: the registry in force is not the one the repository
	// declares, or is switched off.
	registryEntryID = "registry-not-in-force"
	// promptToolEntryID: the prompt is for a tool other than the shell.
	promptToolEntryID = "prompt-not-shell"
	// promptInputEntryID: the shell prompt's input carries no command the
	// guard can read.
	promptInputEntryID = "prompt-input-unread"
	// protectedPathEntryID: the line writes or removes a path the harness
	// protects (repository state, start-up files, its own configuration).
	protectedPathEntryID = "protected-path"
	// criticalRemovalEntryID: a removal, a move or a recursive change whose
	// target is, or may be, a critical path: the root, a top-level directory,
	// the home directory or one of its direct children, the working tree or
	// a directory above it, or another checkout of the repository.
	criticalRemovalEntryID = "critical-path-removal"
	// sandboxEntryID: the shell call asks to run outside the host's sandbox.
	sandboxEntryID = "sandbox-off"
	// unfixedPathEntryID: a write or a removal whose target the line does
	// not fix: a substitution's output, a variable's value, a glob, or a
	// relative path after a change of directory the line leaves unknown.
	unfixedPathEntryID = "path-not-fixed"
	// gitConfigWriteEntryID: a git command that writes configuration outside
	// the repository's own file, or sets a key that starts a program.
	gitConfigWriteEntryID = "git-config-write"
	// outsideRemovalEntryID: a removal whose fixed target lies outside the
	// launched working tree, the run's own directories and the temporary
	// directory.
	outsideRemovalEntryID = "removal-outside-tree"
	// groupBrokenEntryID: the command, placed inside the group that runs it
	// from the working tree, would close that group before its end or leave
	// the text unparsable as one group (a stray } or ), a closing keyword
	// with nothing open, a backslash at its end), so part of it could run
	// outside the change of directory.
	groupBrokenEntryID = "tree-group-broken"
)

// approveFoldsCase reports whether the filesystem folds case, so a protected
// path matches in any case; a variable so a test can take either branch.
var approveFoldsCase = fsutil.CaseFoldingFS

// Approval is abcd's answer to one permission prompt.
type Approval struct {
	// Allow is true when the guard approves the call.
	Allow bool
	// EntryID names what refused it: a registry entry, a reading's signal or
	// one of the reserved ids above. Empty on an allow.
	EntryID string
	// Reason is the refusal the agent is handed: what was refused, why, and
	// what to run instead. Empty on an allow.
	Reason string
}

// refusal builds a refusal in the one shape every refusal takes.
func refusal(id, why, successor string) Approval {
	return Approval{EntryID: id, Reason: fmt.Sprintf(
		"Refused by the abcd guard (%s), which answers the permission prompts of an agent abcd started without a terminal: %s Run instead: %s",
		id, strings.TrimSpace(why), strings.TrimSpace(successor))}
}

// ApprovePrompt answers a permission prompt for tool with input, the tool's
// parameters as the harness sent them. Only the shell is judged; a prompt for
// any other tool is refused, since nobody is there to approve it, and so is a
// shell call that asks to run outside the host's sandbox. On an allow the
// second value is the input to run: the command judged and its description,
// re-encoded, and nothing else the call carried, so a repeated key can never
// run a command other than the one the guard read and no field the guard did
// not read rides along on its approval.
func ApprovePrompt(ld Loaded, files *Files, tool string, input json.RawMessage) (Approval, json.RawMessage) {
	if !strings.EqualFold(tool, "Bash") {
		name := termsafe.Sanitize(tool)
		if len(name) > 64 || name == "" {
			name = "this tool"
		}
		return refusal(promptToolEntryID,
			"this agent runs without a terminal, so nobody can approve its "+name+" call, and abcd answers a prompt only for a shell command its guard can check.",
			"stay within the tools the brief grants, or say in the receipt that the task needs "+name+" beyond them."), nil
	}
	var fields map[string]json.RawMessage
	var command string
	if json.Unmarshal(input, &fields) != nil || json.Unmarshal(fields["command"], &command) != nil || strings.TrimSpace(command) == "" {
		return refusal(promptInputEntryID,
			"the shell call carries no command string the guard can read, so nothing was checked.",
			"the same call with its command as one plain string."), nil
	}
	if raw, ok := fields["dangerouslyDisableSandbox"]; ok {
		if v := strings.TrimSpace(string(raw)); v != "false" && v != "null" {
			return refusal(sandboxEntryID,
				"the shell call asks to run outside the host's sandbox, which only a person can agree to.",
				"the same command without dangerouslyDisableSandbox, or say in the receipt that the task needs the sandbox lifted."), nil
		}
	}
	if files == nil {
		files = NewFiles("")
	}
	a := approveShell(ld, files, command)
	if !a.Allow {
		return a, nil
	}
	run, intact := runFromIntact(files.dir, command)
	if !intact {
		return refusal(groupBrokenEntryID,
			"the command closes the group that runs it from the working tree before its end, or leaves that group unparsable (a } or ) with nothing open for it to close, a closing keyword such as fi, done or esac with nothing open, a } after a command's words, which zsh reads as a close, or a backslash at its very end), so part of it could run outside the working tree.",
			"the same command with every group, if, loop and case it opens closed inside it, and no backslash at its end."), nil
	}
	out := map[string]string{"command": run}
	var description string
	if json.Unmarshal(fields["description"], &description) == nil && description != "" {
		out["description"] = description
	}
	updated, err := json.Marshal(out)
	if err != nil {
		return refusal(promptInputEntryID, "the shell call's input could not be written back.", "the same call again."), nil
	}
	return a, updated
}

// runFrom is command as the host runs it once approved: from dir, the
// working tree it was judged for. The host keeps the shell's directory from
// one call to the next, so without the change an approved command would run
// wherever the previous one left the shell. The command sits on lines of its
// own inside a group the change of directory guards, so when the directory
// cannot be entered no part of it runs, not even one after `||` or `;`, and
// a here-document that ends it, or a comment at its end, closes before the
// group does. A change of directory inside the command still holds for the
// rest of its line.
func runFrom(dir, command string) string {
	if !filepath.IsAbs(dir) {
		return command
	}
	return "cd -- " + shellquote.Single(dir) + " && {\n" + command + "\n}"
}

// approveShell answers a prompt for one shell command line, read against ld's
// registry with files as the directory it runs in.
func approveShell(ld Loaded, files *Files, command string) Approval {
	return approveShellAt(ld, files, command, 0)
}

// approveShellAt is approveShell at depth: a command a git setting on the
// line runs (commandKeys) is judged one level down, and one that sets such a
// command again is not judged further, and refused.
func approveShellAt(ld Loaded, files *Files, command string, depth int) Approval {
	switch ld.Posture {
	case LoadUnavailable:
		return refusal(registryEntryID, "no hazard registry is loaded, so nothing was checked.",
			"nothing until the registry loads; say so in the receipt.")
	case LoadRepoDropped:
		what := "it did not load"
		if ld.Err != nil {
			what = termsafe.Sanitize(ld.Err.Error())
		}
		return refusal(registryEntryID,
			"the repository's "+RepoRelPath+" is not in force ("+what+"), so the guard cannot check against the registry the repository declares.",
			"nothing until the repository's guard file loads; say so in the receipt.")
	}
	r := ld.Registry
	if r.Disabled {
		return refusal(registryEntryID, "the hazard registry is switched off in "+RepoRelPath+", so nothing was checked.",
			"nothing while the registry is off; say so in the receipt.")
	}
	if files == nil {
		files = NewFiles("")
	}
	r = r.ReadingFrom(files)
	if len(command) > maxCommandBytes {
		return fromDecision(syntheticDecision(VerdictBlock, commandTooLongSignal(), nil))
	}
	// The reading keeps the line's segments with the states it placed them
	// in, so the checks below read the line's writes from the same model.
	rc := newReadCtx(r.files)
	rc.record = true
	v, err := r.judge(command, rc, nil)
	switch {
	case errors.Is(err, ErrUnparsableCommand):
		return fromDecision(UnparsableDecision(err))
	case err != nil:
		return refusal(unparsableEntryID, "the guard could not check this command line.",
			"a plainer command line, one command at a time.")
	}
	if d := r.decide(v); d.Verdict != VerdictAllow {
		return fromDecision(d)
	}
	if len(rc.notes) > 0 {
		return refusal(checkNoteEntryID, termsafe.Sanitize(strings.Join(rc.notes, " ")),
			"a command whose every script exists and is a file the guard can read.")
	}
	facts := factsFor(r.files)
	placed := placedLine(rc.line, v.segs)
	for _, p := range placed {
		if p.seg.parenWord {
			return parenRefusal()
		}
	}
	for _, p := range placed {
		for _, st := range p.states {
			target, unplacedName, ok := criticalRemoval(p.seg, st, facts)
			switch {
			case ok && unplacedName:
				return unfixedRefusal(pathUse{remove: true, rel: true, written: target})
			case ok:
				return refusal(criticalRemovalEntryID,
					"this removes, moves or recursively changes "+termsafe.Sanitize(target)+", which is or may be the filesystem root, a top-level directory, the home directory or a directory directly in it, the working tree or a directory above it, or another checkout of the repository.",
					"change the files by their full paths below the working tree, spelled out.")
			}
		}
	}
	vf := r.gitValueFlags()
	// wrote is set once a segment of the line writes or removes a path: a
	// command a git setting runs after that, and a glob removal after that,
	// are judged against files the line may have changed (a directory it may
	// have made a link), so they are not judged on their own.
	wrote := false
	for _, p := range placed {
		wroteBefore := wrote
		// A command a git setting runs is judged as run from the working
		// tree, so only where the reading placed git there.
		states := p.states
		clean := func(v string) bool {
			if strings.TrimSpace(v) == "" {
				return true // no pager
			}
			if depth > 0 || wrote {
				return false
			}
			for _, st := range states {
				if !st.dirOK || st.dir != r.files.dir {
					return false
				}
			}
			return approveShellAt(ld, files, v, depth+1).Allow
		}
		if a, ok := gitConfigRefusal(p.seg, vf, clean); ok {
			return a
		}
		for _, st := range p.states {
			for _, u := range pathUses(rc, p.seg, st) {
				wrote = true
				if u.bound && !u.ok {
					// Its starting point is the critical check's to judge.
					continue
				}
				if u.ok && u.glob != "" {
					// A glob removal inside the working tree is approved
					// unless a protected name could match it: its directory
					// judged where it leads, through symlinks, and only when
					// no earlier part of the line writes or removes a file,
					// which could make that directory a link first. A
					// pattern holding a / (a wildcard in a directory part,
					// build/*/*, or a trailing /) lists directories the line
					// never names, any of which may be a link out of the
					// tree, so it is refused too.
					if wroteBefore || strings.Contains(u.glob, "/") || !facts.withinResolved(u.path, facts.tree, true) {
						return unfixedRefusal(pathUse{remove: true, written: u.written})
					}
					if name, ok := facts.globReaches(u); ok {
						return protectedRefusal(name)
					}
					continue
				}
				if !u.ok {
					return unfixedRefusal(u)
				}
				if facts.outsideRemoval(u) {
					name := termsafe.Sanitize(fsutil.DisplayPath(u.path))
					return refusal(outsideRemovalEntryID,
						"this removes "+name+", which lies outside the working tree the agent was started in, the run's own directories and the temporary directory, so abcd does not approve it unattended.",
						"remove only files inside the working tree, or say in the receipt which file outside it should go.")
				}
				if u.bound {
					continue
				}
				if name, ok := facts.protects(u); ok {
					return protectedRefusal(name)
				}
			}
		}
	}
	for _, s := range v.segs {
		for _, a := range commandSites(s) {
			tok := s.tokens[a.idx]
			if strings.ContainsRune(tok, unknownMark) || strings.ContainsRune(tok, varMark) || s.globAt(a.idx) {
				return refusal(unnamedProgramEntryID,
					"a variable, a substitution or a glob stands where the program's name goes, so the guard cannot say which program runs.",
					"the same command with the program's name spelled out.")
			}
			if name, ok := inlineCode(s.tokens, a.idx); ok {
				return refusal(inlineCodeEntryID,
					"this hands "+termsafe.Sanitize(name)+" inline code, which the guard does not read, so it has not checked what runs.",
					"write the code to a file and run the file, or do the work with shell commands the guard reads.")
			}
			if name, ok := otherCode(s, a.idx); ok {
				return refusal(inlineCodeEntryID,
					"this hands "+termsafe.Sanitize(name)+" code to run (on its standard input, in an awk program or sed script that runs a command, in a tar option that runs a program, or in a vim command holding a !), which the guard does not read, so it has not checked what runs.",
					"write the code to a file and run the file, or do the work with shell commands the guard reads.")
			}
		}
	}
	return Approval{Allow: true}
}

// protectedRefusal is the refusal of a write or removal reaching name, a
// protected path.
func protectedRefusal(name string) Approval {
	name = termsafe.Sanitize(fsutil.DisplayPath(name))
	return refusal(protectedPathEntryID,
		"this line writes or removes "+name+", a path the harness never lets a command change without a person's yes, so abcd does not approve it unattended.",
		"leave "+name+" alone, or say in the receipt that the task needs it changed.")
}

// fromDecision is the refusal a guard verdict that is not a clean allow
// gives: its id, its why and its successor.
func fromDecision(d Decision) Approval {
	return refusal(d.EntryID, termsafe.Sanitize(d.Why), termsafe.Sanitize(d.Successor))
}

// criticalRemoval reports a command in s that removes, moves or recursively
// changes a path that is, or may be, a critical one, and the path as written:
// an rm or rmdir of one; a find rooted at home or above that deletes (-delete,
// or -exec rm); a move of one; an rsync that deletes into one; and a recursive
// chmod, chown or chgrp of one. st is the shell state the reading placed s
// in: a relative target is read in the directory st fixes, and where st
// leaves the directory unknown the target is reported with unplaced set, a
// name the line does not place rather than a critical path.
func criticalRemoval(s segment, st *shellState, f approveFacts) (target string, unplaced, ok bool) {
	for _, a := range commandSites(s) {
		var ops []int
		critical := f.criticalDir
		switch b := filepath.Base(s.tokens[a.idx]); b {
		case "rm", "rmdir":
			ops = operandsFrom(s, a.idx+1, nil)
		case "find":
			ops, critical = findDeletes(s, a.idx+1), f.homeOrAbove
		case "mv":
			ops = operandsFrom(s, a.idx+1, []string{"-S", "--suffix", "-t", "--target-directory"})
			if !flagGiven(s, a.idx+1, "-t", "--target-directory") && len(ops) > 0 {
				ops = ops[:len(ops)-1] // the destination stays
			}
		case "rsync":
			if !flagGiven(s, a.idx+1, "--del") && !flagPrefixGiven(s, a.idx+1, "--delete") {
				continue
			}
			if all := operandsFrom(s, a.idx+1, rsyncValueFlags); len(all) > 1 && !remoteSpec(s.tokens[all[len(all)-1]]) {
				ops = all[len(all)-1:]
			}
		case "chmod", "chown", "chgrp":
			if !flagGiven(s, a.idx+1, "--recursive") && !clusterHas(s, a.idx+1, 'R') {
				continue
			}
			if ops = operandsFrom(s, a.idx+1, nil); len(ops) > 0 && !flagPrefixGiven(s, a.idx+1, "--reference") {
				ops = ops[1:] // the mode, or the owner
			}
		default:
			continue
		}
		for _, j := range ops {
			op := s.tokens[j]
			crit, unplacedName := criticalTarget(op, s.globAt(j), st, f, critical)
			if crit || unplacedName {
				return strings.NewReplacer(string(unknownMark), "$…", string(varMark), "$…").Replace(op), !crit, true
			}
		}
	}
	return "", false, false
}

// operandsFrom is the indexes of s's operands from index from, stepping
// options (and the value of each in valueFlags); a `--` ends the options.
func operandsFrom(s segment, from int, valueFlags []string) []int {
	return writerArgs{s: s, from: from}.operandIdx(valueFlags)
}

// flagGiven reports whether one of flags stands among s's words from index
// from, alone or with its value glued on after `=`.
func flagGiven(s segment, from int, flags ...string) bool {
	for _, t := range s.tokens[from:] {
		for _, f := range flags {
			if t == f || strings.HasPrefix(t, f+"=") {
				return true
			}
		}
	}
	return false
}

// flagPrefixGiven reports whether a word from index from starts with prefix:
// one of a family of long options (--delete, --delete-after …).
func flagPrefixGiven(s segment, from int, prefix string) bool {
	for _, t := range s.tokens[from:] {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// clusterHas reports whether a bundled short-flag word from index from holds
// the letter c; a word the line does not fix may.
func clusterHas(s segment, from int, c rune) bool {
	for _, t := range s.tokens[from:] {
		if isUnknown(t) || isShortCluster(t) && strings.ContainsRune(t[1:], c) {
			return true
		}
	}
	return false
}

// rsyncValueFlags are rsync's options that take their value as the next word.
var rsyncValueFlags = []string{"-e", "--rsh", "-f", "--filter", "--exclude", "--include", "--exclude-from",
	"--include-from", "--files-from", "-T", "--temp-dir", "--compare-dest", "--copy-dest", "--link-dest",
	"-B", "--block-size", "--log-file", "--password-file", "--chmod", "--chown", "--rsync-path", "--partial-dir"}

// remoteSpec reports whether an rsync operand names another host
// (host:path, or rsync://), which no local check judges.
func remoteSpec(op string) bool {
	if strings.Contains(op, "://") {
		return true
	}
	colon := strings.IndexByte(op, ':')
	return colon > 0 && !strings.Contains(op[:colon], "/")
}

// findDeletes is the starting points of a find that deletes what it finds
// (-delete, or -exec / -execdir / -ok / -okdir running rm, rmdir or unlink),
// or nil for one that does not. With no starting point named, find starts
// in ".".
func findDeletes(s segment, from int) []int {
	if !findDeleting(s, from) {
		return nil
	}
	return findStarts(s, from)
}

// findDeleting reports whether the find whose words start at from deletes
// what it finds: -delete, or -exec / -execdir / -ok / -okdir running rm,
// rmdir or unlink, or a command the line does not fix.
func findDeleting(s segment, from int) bool {
	for i := from; i < len(s.tokens); i++ {
		switch s.tokens[i] {
		case "-delete":
			return true
		case "-exec", "-execdir", "-ok", "-okdir":
			if i+1 < len(s.tokens) {
				switch filepath.Base(s.tokens[i+1]) {
				case "rm", "rmdir", "unlink":
					return true
				}
				if isUnknown(s.tokens[i+1]) {
					return true
				}
			}
		}
	}
	return false
}

// findStarts is a find's starting points, from index from.
func findStarts(s segment, from int) []int {
	var starts []int
	i := from
	for i < len(s.tokens) && (s.tokens[i] == "-H" || s.tokens[i] == "-L" || s.tokens[i] == "-P" || strings.HasPrefix(s.tokens[i], "-O") || s.tokens[i] == "-D") {
		if s.tokens[i] == "-D" {
			i++
		}
		i++
	}
	for ; i < len(s.tokens); i++ {
		t := s.tokens[i]
		if !isUnknown(t) && (strings.HasPrefix(t, "-") || t == "(" || t == "!" || t == ")") {
			break
		}
		starts = append(starts, i)
	}
	return starts
}

// criticalTarget reports whether one operand is, or may be, a path critical
// judges critical. An operand a variable or a substitution fills may be any
// path; a relative one is read in the directory st fixes, and where st leaves
// that directory unknown it is reported unplaced instead; a glob is judged by
// the directory it expands in.
func criticalTarget(op string, globbed bool, st *shellState, f approveFacts, critical func(string) bool) (crit, unplaced bool) {
	if strings.ContainsRune(op, unknownMark) || strings.ContainsRune(op, varMark) {
		return true, false
	}
	dir, home := f.dir, f.home
	p := op
	switch {
	case p == "~" || strings.HasPrefix(p, "~/"):
		if home == "" {
			return true, false
		}
		p = home + strings.TrimPrefix(p, "~")
	case strings.HasPrefix(p, "~"):
		return true, false
	case !filepath.IsAbs(p):
		if st == nil || !st.dirOK || st.dir == "" {
			return false, true
		}
		p = filepath.Join(st.dir, p)
	}
	p = filepath.Clean(p)
	if globbed {
		// The directory the glob lists: the path up to its first globbed
		// component.
		parts := strings.Split(p, string(filepath.Separator))
		for i, c := range parts {
			if strings.ContainsAny(c, "*?[") {
				p = filepath.Clean(string(filepath.Separator) + filepath.Join(parts[:i]...))
				return critical(p) || critical(fsutil.RealExistingPath(p)) || p == filepath.Clean(dir), false
			}
		}
	}
	// As written and with its existing directories resolved: the checkouts
	// git lists are in their resolved spelling.
	return critical(p) || critical(fsutil.RealExistingPath(p)), false
}

// criticalDir reports whether p, a clean absolute path, is the root, a
// top-level directory, the home directory or one of its direct children, the
// working tree or a directory above it, or within or above another checkout
// of the repository.
func (f approveFacts) criticalDir(p string) bool {
	if f.homeOrAbove(p) {
		return true
	}
	if f.home != "" && fsutil.PathWithin(filepath.Dir(p), filepath.Clean(f.home), f.fold) {
		if filepath.Dir(p) == filepath.Clean(f.home) || fsutil.FoldPath(filepath.Dir(p), f.fold) == fsutil.FoldPath(filepath.Clean(f.home), f.fold) {
			return true
		}
	}
	if f.dir != "" && fsutil.PathWithin(f.dir, p, f.fold) {
		return true
	}
	for _, o := range f.others {
		if fsutil.PathWithin(o, p, f.fold) || (fsutil.PathWithin(p, o, f.fold) && (f.dir == "" || !fsutil.PathWithin(p, f.dir, f.fold))) {
			return true
		}
	}
	return false
}

// homeOrAbove reports whether p, a clean absolute path, is the root, a
// top-level directory, or the home directory or a directory above it.
func (f approveFacts) homeOrAbove(p string) bool {
	if p == "/" || strings.Count(p, "/") == 1 {
		return true
	}
	return f.home != "" && fsutil.PathWithin(filepath.Clean(f.home), p, f.fold)
}

// interpreters are the non-shell interpreters whose inline code the guard does
// not read, by the program name a line spells (its base name), with the flags
// that hand one inline code. A short flag also counts inside a bundled cluster
// (python -Bc, perl -ne), and a long one in its --flag=value form. sub, when
// set, is a subcommand that takes inline code instead (deno eval).
var interpreters = []struct {
	name  *regexp.Regexp
	short string
	long  []string
	sub   string
	// word, when set, judges a flag word on its own: PowerShell takes a
	// parameter by any prefix that names it alone, and bundles none.
	word func(low string) bool
	// module, when set, is the short flag that names a module to run, after
	// which every word is the module's (python -m).
	module string
	// plain are the interpreter's flags that take no value, and plainShort
	// the letters of a bundled cluster of them: a script named right after
	// one is the script. After any other flag the next word may be that
	// flag's value, so it is not taken for the script.
	plain      []string
	plainShort string
	// glued are the short flags that take their value glued on (perl
	// -Ilib): such a word is a flag and its value, so the next word may be
	// the script.
	glued string
}{
	{name: regexp.MustCompile(`^(python|pypy)[0-9.]*$`), short: "c", module: "m", plainShort: "bBdEhiIOPqsSuvVx", glued: "WX"},
	{name: regexp.MustCompile(`^(node|nodejs|bun)$`), short: "ep", long: []string{"--eval", "--print"},
		plain: []string{"--inspect", "--inspect-brk", "--inspect-wait", "--enable-source-maps", "--no-warnings",
			"--trace-warnings", "--trace-deprecation", "--throw-deprecation", "--pending-deprecation", "--no-deprecation",
			"--watch", "--test", "--expose-gc", "--preserve-symlinks", "--abort-on-uncaught-exception",
			"--experimental-vm-modules", "--experimental-strip-types", "--no-experimental-strip-types", "-i", "--interactive"}},
	{name: regexp.MustCompile(`^deno$`), sub: "eval"},
	{name: regexp.MustCompile(`^perl[0-9.]*$`), short: "eE", glued: "IMm"},
	{name: regexp.MustCompile(`^ruby[0-9.]*$`), short: "e", glued: "Ir"},
	{name: regexp.MustCompile(`^php[0-9.]*$`), short: "r"},
	{name: regexp.MustCompile(`^(osascript|lua[0-9.]*|luajit|Rscript)$`), short: "e"},
	{name: regexp.MustCompile(`^(?i:pwsh|powershell)(\.exe)?$`), word: pwshCode},
}

// scriptAfter reports whether an operand right after prev is the script an
// interpreter runs rather than prev's value: prev is the interpreter, a word
// that is not a flag (a flag's value), a flag with its value glued on after
// `=` or glued to its letter (one of glued's), or one of the flags that take
// no value (plain, or a cluster of plainShort's letters).
func scriptAfter(plain []string, plainShort, glued, prev string) bool {
	switch {
	case !strings.HasPrefix(prev, "-"), strings.Contains(prev, "="), containsString(plain, prev):
		return true
	case len(prev) > 2 && prev[1] != '-' && strings.ContainsRune(glued, rune(prev[1])):
		return true
	case plainShort != "" && shortCluster.MatchString(prev):
		for _, c := range prev[1:] {
			if !strings.ContainsRune(plainShort, c) {
				return false
			}
		}
		return true
	}
	return false
}

// pwshCode reports whether low, a lower-cased word, is PowerShell's -Command
// or -EncodedCommand by any prefix PowerShell takes for it (-c, -e and -ec
// are its own short names).
func pwshCode(low string) bool {
	switch low {
	case "-c", "-e", "-ec":
		return true
	}
	return len(low) >= 3 && (strings.HasPrefix("-command", low) || strings.HasPrefix("-encodedcommand", low))
}

// shortCluster is a bundled short-flag word: a dash and up to five letters.
var shortCluster = regexp.MustCompile(`^-[A-Za-z]{1,5}$`)

// inlineCode reports whether the command at tokens[idx] is an interpreter
// handed inline code, and the interpreter's name. A word after it that holds
// a substitution's output or a variable's value may be any flag, so it counts
// as one that hands code: spell the script's path, and the run is read. The
// words after the script's name, or after python's -m module, are the
// script's, so the reading stops there; a word is taken for the script only
// where it cannot be the value of the flag before it: right after the
// interpreter, after a flag given with `=`, after a flag that takes no value
// (plain), or after a word that was itself a flag's value.
func inlineCode(tokens []string, idx int) (string, bool) {
	name := filepath.Base(tokens[idx])
	for _, in := range interpreters {
		if !in.name.MatchString(name) {
			continue
		}
		for i, tok := range tokens[idx+1:] {
			if isUnknown(tok) || strings.ContainsRune(tok, varMark) {
				return name, true
			}
			if in.module != "" && strings.HasPrefix(tok, "-"+in.module) {
				return "", false // the module, and the words that are its own
			}
			if !strings.HasPrefix(tok, "-") && tok != "" && scriptAfter(in.plain, in.plainShort, in.glued, tokens[idx+i]) &&
				!(in.sub != "" && i == 0 && tok == in.sub) {
				return "", false // the script, and the words that are its own
			}
			if in.word != nil && in.word(strings.ToLower(tok)) {
				return name, true
			}
			if in.sub != "" && i == 0 && tok == in.sub {
				return name, true
			}
			low := strings.ToLower(tok)
			for _, l := range in.long {
				if low == l || strings.HasPrefix(low, l+"=") {
					return name, true
				}
			}
			if in.short != "" && shortCluster.MatchString(tok) && strings.ContainsAny(tok[1:], in.short) {
				return name, true
			}
		}
		return "", false
	}
	return "", false
}
