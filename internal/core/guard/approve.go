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
	a := approveShell(ld, files, command)
	if !a.Allow {
		return a, nil
	}
	out := map[string]string{"command": command}
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

// approveShell answers a prompt for one shell command line, read against ld's
// registry with files as the directory it runs in.
func approveShell(ld Loaded, files *Files, command string) Approval {
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
	for i, s := range v.segs {
		if target, ok := criticalRemoval(s, v.segs[:i], facts); ok {
			return refusal(criticalRemovalEntryID,
				"this removes, moves or recursively changes "+termsafe.Sanitize(target)+", which is or may be the filesystem root, a top-level directory, the home directory or a directory directly in it, the working tree or a directory above it, or another checkout of the repository.",
				"change the files by their full paths below the working tree, spelled out.")
		}
	}
	vf := r.gitValueFlags()
	for _, p := range placedLine(rc.line, v.segs) {
		if a, ok := gitConfigRefusal(p.seg, vf); ok {
			return a
		}
		for _, st := range p.states {
			for _, u := range pathUses(rc, p.seg, st) {
				if !u.ok {
					return unfixedRefusal(u)
				}
				if name, ok := facts.protects(u); ok {
					name = termsafe.Sanitize(fsutil.DisplayPath(name))
					return refusal(protectedPathEntryID,
						"this line writes or removes "+name+", a path the harness never lets a command change without a person's yes, so abcd does not approve it unattended.",
						"leave "+name+" alone, or say in the receipt that the task needs it changed.")
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
		}
	}
	return Approval{Allow: true}
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
// chmod, chown or chgrp of one. before is the segments ahead of s, read for a
// directory change that leaves a relative target's directory unknown.
func criticalRemoval(s segment, before []segment, f approveFacts) (string, bool) {
	moved := precededByCD(before, s.chain)
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
			if criticalTarget(op, s.globAt(j), moved, f, critical) {
				return strings.NewReplacer(string(unknownMark), "$…", string(varMark), "$…").Replace(op), true
			}
		}
	}
	return "", false
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
	deletes := false
	for i := from; i < len(s.tokens); i++ {
		switch s.tokens[i] {
		case "-delete":
			deletes = true
		case "-exec", "-execdir", "-ok", "-okdir":
			if i+1 < len(s.tokens) {
				switch filepath.Base(s.tokens[i+1]) {
				case "rm", "rmdir", "unlink":
					deletes = true
				}
				if isUnknown(s.tokens[i+1]) {
					deletes = true
				}
			}
		}
	}
	if !deletes {
		return nil
	}
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
// path, and so may a relative one after the line changed directory; a glob is
// judged by the directory it expands in.
func criticalTarget(op string, globbed, moved bool, f approveFacts, critical func(string) bool) bool {
	if strings.ContainsRune(op, unknownMark) || strings.ContainsRune(op, varMark) {
		return true
	}
	dir, home := f.dir, f.home
	p := op
	switch {
	case p == "~" || strings.HasPrefix(p, "~/"):
		if home == "" {
			return true
		}
		p = home + strings.TrimPrefix(p, "~")
	case strings.HasPrefix(p, "~"):
		return true
	case !filepath.IsAbs(p):
		if moved || dir == "" {
			return true
		}
		p = filepath.Join(dir, p)
	}
	p = filepath.Clean(p)
	if globbed {
		// The directory the glob lists: the path up to its first globbed
		// component.
		parts := strings.Split(p, string(filepath.Separator))
		for i, c := range parts {
			if strings.ContainsAny(c, "*?[") {
				p = filepath.Clean(string(filepath.Separator) + filepath.Join(parts[:i]...))
				return critical(p) || critical(fsutil.RealExistingPath(p)) || p == filepath.Clean(dir)
			}
		}
	}
	// As written and with its existing directories resolved: the checkouts
	// git lists are in their resolved spelling.
	return critical(p) || critical(fsutil.RealExistingPath(p))
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
}{
	{name: regexp.MustCompile(`^(python|pypy)[0-9.]*$`), short: "c"},
	{name: regexp.MustCompile(`^(node|nodejs|bun)$`), short: "ep", long: []string{"--eval", "--print"}},
	{name: regexp.MustCompile(`^deno$`), sub: "eval"},
	{name: regexp.MustCompile(`^perl[0-9.]*$`), short: "eE"},
	{name: regexp.MustCompile(`^ruby[0-9.]*$`), short: "e"},
	{name: regexp.MustCompile(`^php[0-9.]*$`), short: "r"},
	{name: regexp.MustCompile(`^(osascript|lua[0-9.]*|luajit|Rscript)$`), short: "e"},
	{name: regexp.MustCompile(`^(?i:pwsh|powershell)(\.exe)?$`), word: pwshCode},
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
// as one that hands code: spell the script's path, and the run is read.
func inlineCode(tokens []string, idx int) (string, bool) {
	name := filepath.Base(tokens[idx])
	for _, in := range interpreters {
		if !in.name.MatchString(name) {
			continue
		}
		for i, tok := range tokens[idx+1:] {
			if strings.ContainsRune(tok, unknownMark) || strings.ContainsRune(tok, varMark) {
				return name, true
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
