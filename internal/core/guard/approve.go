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
// or a glob; and no inline code handed to a non-shell interpreter, whose
// payload the guard does not read. The guard is still a mistake filter, not a
// boundary (the check verb's help says what an allow does not see), and this
// widens nothing past what it reads.

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

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
	// protectedPathEntryID: the line names a path the harness protects from
	// writes (repository state, start-up files, its own configuration).
	protectedPathEntryID = "protected-path"
	// criticalRemovalEntryID: an rm or rmdir whose target is, or may be, a
	// critical path: the root, a top-level directory, the home directory, the
	// working tree or a directory above it.
	criticalRemovalEntryID = "critical-path-removal"
)

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
// any other tool is refused, since nobody is there to approve it. On an allow
// the second value is the input to run, re-encoded from what was judged, so a
// repeated key can never run a command other than the one the guard read.
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
	a := approveShell(ld, files, command)
	if !a.Allow {
		return a, nil
	}
	updated, err := json.Marshal(fields)
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
	if files != nil {
		r = r.ReadingFrom(files)
	}
	if len(command) > maxCommandBytes {
		return fromDecision(syntheticDecision(VerdictBlock, commandTooLongSignal(), nil))
	}
	var rc *readCtx
	if r.files != nil {
		rc = newReadCtx(r.files)
	}
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
	if rc != nil && len(rc.notes) > 0 {
		return refusal(checkNoteEntryID, termsafe.Sanitize(strings.Join(rc.notes, " ")),
			"a command whose every script exists and is a file the guard can read.")
	}
	if name, ok := protectedPath(command); ok {
		return refusal(protectedPathEntryID,
			"this line names "+name+", a path the harness never lets a command write without a person's yes, so abcd does not approve it unattended.",
			"leave "+name+" alone, or say in the receipt that the task needs it changed.")
	}
	var dir, home string
	if r.files != nil {
		dir, home = r.files.dir, r.files.home
	}
	for i, s := range v.segs {
		if target, ok := criticalRemoval(s, v.segs[:i], dir, home); ok {
			return refusal(criticalRemovalEntryID,
				"this removes "+termsafe.Sanitize(target)+", which is or may be the filesystem root, a top-level directory, the home directory, the working tree or a directory above it.",
				"remove the files by their full paths below the working tree, spelled out.")
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

// protectedNames are the paths the harness protects from writes (its own
// documentation, "Protected paths", read 2026-10-16): in the mode that refused
// every prompt a write there was refused, so naming one anywhere on the line,
// a redirection's target included, is refused here too. The match is on the
// text as written, so a quoted mention counts: the refusal errs toward a
// person.
var protectedNames = regexp.MustCompile(`(^|[^A-Za-z0-9_.-])(` +
	`\.git|\.config/git|\.vscode|\.idea|\.husky|\.cargo|\.devcontainer|\.yarn|\.mvn|\.claude|` +
	`\.gitconfig|\.gitmodules|\.bashrc|\.bash_profile|\.bash_login|\.bash_aliases|\.bash_logout|` +
	`\.zshrc|\.zprofile|\.zshenv|\.zlogin|\.zlogout|\.profile|\.envrc|` +
	`\.npmrc|\.yarnrc|\.yarnrc\.yml|\.pnp\.cjs|\.pnp\.loader\.mjs|\.pnpmfile\.cjs|bunfig\.toml|\.bunfig\.toml|` +
	`\.bazelrc|\.bazelversion|\.bazeliskrc|\.pre-commit-config\.yaml|\.?lefthook\.ya?ml|` +
	`gradle-wrapper\.properties|maven-wrapper\.properties|\.devcontainer\.json|\.ripgreprc|pyrightconfig\.json|` +
	`\.mcp\.json|\.claude\.json` +
	`)($|[^A-Za-z0-9_.-])`)

// protectedPath names the first protected path the line spells.
func protectedPath(command string) (string, bool) {
	m := protectedNames.FindStringSubmatch(command)
	if m == nil {
		return "", false
	}
	return m[2], true
}

// criticalRemoval reports an rm or rmdir in s whose target is, or may be, a
// critical path, and the target as written. before is the segments ahead of s,
// read for a directory change that leaves a relative target's directory
// unknown.
func criticalRemoval(s segment, before []segment, dir, home string) (string, bool) {
	for _, a := range commandSites(s) {
		if b := filepath.Base(s.tokens[a.idx]); b != "rm" && b != "rmdir" {
			continue
		}
		moved := precededByCD(before, s.chain)
		flags := true
		for j := a.idx + 1; j < len(s.tokens); j++ {
			op := s.tokens[j]
			if flags && op == "--" {
				flags = false
				continue
			}
			if flags && strings.HasPrefix(op, "-") && op != "-" {
				continue
			}
			if criticalTarget(op, s.globAt(j), moved, dir, home) {
				return strings.NewReplacer(string(unknownMark), "$…", string(varMark), "$…").Replace(op), true
			}
		}
	}
	return "", false
}

// criticalTarget reports whether one removal operand is, or may be, a critical
// path. An operand a variable or a substitution fills may be any path, and so
// may a relative one after the line changed directory; a glob is judged by the
// directory it expands in.
func criticalTarget(op string, globbed, moved bool, dir, home string) bool {
	if strings.ContainsRune(op, unknownMark) || strings.ContainsRune(op, varMark) {
		return true
	}
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
				return criticalDir(p, dir, home) || p == filepath.Clean(dir)
			}
		}
	}
	return criticalDir(p, dir, home)
}

// criticalDir reports whether p, a clean absolute path, is the root, a
// top-level directory, the home directory, or the working tree or a
// directory above it.
func criticalDir(p, dir, home string) bool {
	if p == "/" || strings.Count(p, "/") == 1 {
		return true
	}
	if home != "" && p == filepath.Clean(home) {
		return true
	}
	if dir != "" {
		d := filepath.Clean(dir)
		if p == d || strings.HasPrefix(d+"/", p+"/") {
			return true
		}
	}
	return false
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
