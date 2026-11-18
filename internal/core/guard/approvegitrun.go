package guard

// approvegitrun.go is the approver's reading of the options of a git
// subcommand that hand git a command line to run, or name or pick the
// program it starts (dcn-2611160119487817): each is judged by the rule the
// matching setting is. A command line git runs through a shell is approved
// only when the line spells it out and the guard, reading it as a line of
// its own at the next depth (approveShellAt, through clean), clears it; a
// pager's value is judged as core.pager's is; an option that names or picks
// a program is refused as its setting is. The options are read as the
// installed git 2.52 parses them, each checked against its own page.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/shellquote"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// runKind is how one option's value is judged.
type runKind int

const (
	// runLine: a command line git runs through a shell.
	runLine runKind = iota
	// runPager: a pager, judged as core.pager's value is.
	runPager
	// runRefused: a program git starts, or a tool it picks, named by a
	// setting the approver refuses for any value.
	runRefused
	// runServer: git send-email's --smtp-server, a program only when it is a
	// path (git-send-email(1)).
	runServer
)

// runTarget is one option and how its value is judged; key is the setting
// it stands for, as a refusal names it.
type runTarget struct {
	kind runKind
	key  string
}

// runParser is one subcommand's option parser, as far as the approver reads
// it.
type runParser struct {
	// long maps each long option read to its target.
	long map[string]runTarget
	// short maps each short option letter read to its target.
	short map[byte]runTarget
	// own is every short letter the subcommand's own parser knows: a cluster
	// is read letter by letter and stops at one it does not (git difftool
	// hands such a word on to git diff whole).
	own string
	// takes are the letters of own that take a value, glued or the next
	// word; optional those that take one only glued (rebase -S, grep -O).
	takes, optional string
	// abbrev says a long option may be written abbreviated, as git's option
	// parser and Getopt::Long take one, unless the word is exactly another
	// option's name (exact).
	abbrev bool
	exact  []string
	// getopt says the parser is Perl's Getopt::Long, as git send-email's is:
	// a long option may be written with one dash, and in any case.
	getopt bool
	// prefixed says a word that starts with a long name is that option, as
	// git mergetool's `--tool*` pattern reads it, unless it is one of exact
	// (the name is read without any `=value`).
	prefixed bool
}

// gitRunParsers are the subcommands whose options the approver reads here,
// with what each page documents:
//   - git-rebase(1): -x/--exec <cmd> is "interpreted as one or more shell
//     commands";
//   - git-difftool(1): -x/--extcmd=<command> runs <command> through a
//     shell; -t/--tool=<tool> picks the tool, as diff.tool does;
//   - git-mergetool(1): -t/--tool=<tool> picks the tool, as merge.tool does;
//   - git-grep(1): -O[<pager>]/--open-files-in-pager[=<pager>] opens the
//     files in the pager, its value stuck to the option;
//   - git-send-email(1): --to-cmd, --cc-cmd, --header-cmd and --sendmail-cmd
//     name commands it runs through a shell, and --smtp-server, "for
//     backward compatibility", may name a sendmail-like program by its path;
//   - git-instaweb(1): -d/--httpd is "the HTTP daemon command-line that will
//     be executed", -b/--browser the browser it starts, and -m/--module-path
//     the modules the daemon loads, as instaweb.httpd, instaweb.browser and
//     instaweb.modulePath are.
//
// -g/--gui of difftool and mergetool picks the tool diff.guitool or
// merge.guitool names, each judged where the line sets it, so it is not read
// here. git submodule foreach, git bisect run and git filter-branch are read
// by their own readers below.
var gitRunParsers = map[string]runParser{
	"rebase": {
		long:  map[string]runTarget{"--exec": {runLine, ""}},
		short: map[byte]runTarget{'x': {runLine, ""}},
		own:   "qvnCfmiSxrsX", takes: "CxsX", optional: "Sr", abbrev: true,
	},
	"difftool": {
		long:  map[string]runTarget{"--extcmd": {runLine, ""}, "--tool": {runRefused, "diff.tool"}},
		short: map[byte]runTarget{'x': {runLine, ""}, 't': {runRefused, "diff.tool"}},
		own:   "gdytx", takes: "tx", abbrev: true,
	},
	"mergetool": {
		long:  map[string]runTarget{"--tool": {runRefused, "merge.tool"}},
		short: map[byte]runTarget{'t': {runRefused, "merge.tool"}},
		own:   "t", takes: "t", prefixed: true, exact: []string{"--tool-help"},
	},
	"grep": {
		long:  map[string]runTarget{"--open-files-in-pager": {runPager, "core.pager"}},
		short: map[byte]runTarget{'O': {runPager, "core.pager"}},
		own:   "viwaIrEGFPnhHlLzocCBApWfeqOm0123456789", takes: "CBAfem", optional: "O", abbrev: true,
	},
	"send-email": {
		long: map[string]runTarget{
			"--to-cmd": {runLine, ""}, "--cc-cmd": {runLine, ""}, "--header-cmd": {runLine, ""},
			"--sendmail-cmd": {runLine, ""}, "--smtp-server": {runServer, "sendemail.smtpServer"},
		},
		abbrev: true, getopt: true, exact: []string{"--to", "--cc"},
	},
	"instaweb": {
		long: map[string]runTarget{
			"--httpd": {runRefused, "instaweb.httpd"}, "--browser": {runRefused, "instaweb.browser"},
			"--module-path": {runRefused, "instaweb.modulePath"},
		},
		short: map[byte]runTarget{
			'd': {runRefused, "instaweb.httpd"}, 'b': {runRefused, "instaweb.browser"},
			'm': {runRefused, "instaweb.modulePath"},
		},
		own: "lpdbmh", takes: "pdbm", abbrev: true,
	},
}

// gitRunSubcommands is every subcommand read here, for a subcommand the line
// does not spell out.
var gitRunSubcommands = []string{"rebase", "difftool", "mergetool", "grep", "send-email", "instaweb",
	"submodule", "bisect", "filter-branch"}

// gitRunRefusal reads one git subcommand's arguments, after its name, for an
// option that hands git a command line to run or names or picks the program
// it starts, and says why it is refused, if it is. clean judges a command
// line as the guard reads it at the next depth.
func gitRunRefusal(sub string, args []string, clean func(string) bool) (string, bool) {
	// A subcommand named with -- inside it (submodule--helper, difftool--helper,
	// web--browse, checkout--worker) is one of git's internal commands, which
	// git's other commands run for themselves. None is read here, and some run
	// a command line or a configured program, so each is refused.
	if !isUnknown(sub) && !strings.ContainsRune(sub, varMark) && strings.Index(sub, "--") > 0 {
		return fmt.Sprintf("git %s is one of git's internal commands, which the approver does not read", sub), true
	}
	subs := []string{sub}
	if isUnknown(sub) || strings.ContainsRune(sub, varMark) {
		subs = gitRunSubcommands
	}
	for _, s := range subs {
		var why string
		var ok bool
		switch s {
		case "submodule":
			why, ok = submoduleForeachRefusal(args, clean)
		case "bisect":
			why, ok = bisectRunRefusal(args, clean)
		case "filter-branch":
			why, ok = filterBranchRefusal(args, clean)
		default:
			p, known := gitRunParsers[s]
			if !known {
				continue
			}
			why, ok = p.refusal(args, clean)
		}
		if ok {
			return why, true
		}
	}
	return "", false
}

// refusal reads args as p parses them, up to a `--`.
func (p runParser) refusal(args []string, clean func(string) bool) (string, bool) {
	for i := 0; i < len(args); i++ {
		w := args[i]
		if w == "--" {
			break
		}
		if !strings.HasPrefix(w, "-") {
			continue
		}
		if name, t, v, has, ok := p.longOption(w); ok {
			if !has && t.kind != runPager && i+1 < len(args) {
				i++
				v, has = args[i], true
			}
			if why, bad := t.judge(name, v, has, clean); bad {
				return why, true
			}
			continue
		}
		if isUnknown(w) {
			for letter := range p.short {
				if clusterCouldCarry(w, letter) {
					return "this git command carries an option the line does not spell out, which may hand git a command or a program it runs.", true
				}
			}
			continue
		}
		letter, glued, ok := p.shortOption(w)
		if !ok {
			continue
		}
		has := glued != ""
		if !has && !strings.ContainsRune(p.optional, rune(letter)) && i+1 < len(args) {
			i++
			glued, has = args[i], true
		}
		if t, target := p.short[letter]; target {
			if why, bad := t.judge("-"+string(letter), glued, has, clean); bad {
				return why, true
			}
		}
	}
	return "", false
}

// longOption reports which of p's long options the word w names, and the
// value glued to it after `=`. A word whose option name the line does not
// spell out is read as any option it can become, with a value the line does
// not fix.
func (p runParser) longOption(w string) (string, runTarget, string, bool, bool) {
	if p.getopt && !strings.HasPrefix(w, "--") && !isUnknown(w) {
		w = "-" + w
	}
	if !strings.HasPrefix(w, "--") && !isUnknown(w) {
		return "", runTarget{}, "", false, false
	}
	n, v, glued := strings.Cut(w, "=")
	if isUnknown(n) {
		for _, name := range optionNames(p.long) {
			if longFlagIs(w, name) {
				return name, p.long[name], unknownText, true, true
			}
		}
		return "", runTarget{}, "", false, false
	}
	if p.getopt {
		n = strings.ToLower(n)
	}
	for _, name := range optionNames(p.long) {
		var match bool
		switch {
		case p.prefixed:
			match = strings.HasPrefix(n, name)
			for _, e := range p.exact {
				match = match && n != e
			}
		case p.abbrev:
			match = n == name || len(n) > 2 && strings.HasPrefix(name, n) && !containsString(p.exact, n)
		default:
			match = n == name
		}
		if match {
			return name, p.long[name], v, glued, true
		}
	}
	return "", runTarget{}, "", false, false
}

// shortOption reads a short-option cluster as git's option parser does,
// letter by letter: the first letter that takes a value, and the text glued
// after it. A letter the parser does not know ends the reading.
func (p runParser) shortOption(w string) (byte, string, bool) {
	if p.getopt || len(w) < 2 || w[0] != '-' || w[1] == '-' {
		return 0, "", false
	}
	for i := 1; i < len(w); i++ {
		c := w[i]
		switch {
		case strings.IndexByte(p.own, c) < 0:
			return 0, "", false
		case strings.IndexByte(p.takes, c) >= 0 || strings.IndexByte(p.optional, c) >= 0:
			return c, w[i+1:], true
		}
	}
	return 0, "", false
}

// judge says why an option given value v (has: given one at all) is
// refused, if it is.
func (t runTarget) judge(opt, v string, has bool, clean func(string) bool) (string, bool) {
	o := termsafe.Sanitize(opt)
	if !has {
		// No value: a pager option opens git's own pager, which is judged
		// where the line sets it; any other git refuses.
		return "", false
	}
	if isUnknown(v) || strings.ContainsRune(v, varMark) {
		return "this git command hands git, with " + o + ", a value the line does not spell out, which may be a command or a program git runs.", true
	}
	switch t.kind {
	case runLine:
		if clean == nil || !clean(v) {
			return "this git command hands git, with " + o + ", a command line git runs through a shell, which the guard did not clear when it read it as a line of its own.", true
		}
	case runPager:
		if clean == nil || !clean(v) {
			return "this git command names, with " + o + ", a pager git runs (as " + t.key + " does), which the guard did not clear when it read it as a line of its own.", true
		}
	case runRefused:
		return "this git command names, with " + o + ", the program git starts or the tool it picks (as " + t.key + " does), which the guard does not read.", true
	case runServer:
		if s := strings.TrimSpace(v); strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") || strings.HasPrefix(s, "~") {
			return "this git command names, with " + o + ", a path, which git send-email runs as a sendmail-like program (as " + t.key + " does with one), that the guard does not read.", true
		}
	}
	return "", false
}

// optionNames is m's option names in order, so a reading names the same
// option every time.
func optionNames(m map[string]runTarget) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// wordsLine judges the words of a command git runs as git runs them, and
// says why they are refused, if they are: every word spelled out, and the
// line clean clears, read both ways git may build it, the first word as
// shell text and the rest as its operands, and every word as one operand.
func wordsLine(what string, words []string, clean func(string) bool) (string, bool) {
	if len(words) == 0 {
		return "", false
	}
	for _, w := range words {
		if isUnknown(w) || strings.ContainsRune(w, varMark) {
			return "this git command hands git, with " + what + ", a word the line does not spell out, in a command git runs.", true
		}
	}
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = shellquote.Single(w)
	}
	first := strings.Join(append([]string{words[0]}, quoted[1:]...), " ")
	if clean == nil || !clean(first) || !clean(strings.Join(quoted, " ")) {
		return "this git command hands git, with " + what + ", a command git runs through a shell, which the guard did not clear when it read it as a line of its own.", true
	}
	return "", false
}

// submoduleForeachRefusal reads `git submodule [--quiet] foreach
// [--recursive] <command>` (git-submodule(1): "Evaluates an arbitrary shell
// command"): the words after foreach's options are the command, the first
// run as shell text and the rest handed to it. A subcommand word the line
// does not spell out may be foreach.
func submoduleForeachRefusal(args []string, clean func(string) bool) (string, bool) {
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") && !isUnknown(args[i]) {
		i++
	}
	if i >= len(args) {
		return "", false
	}
	if isUnknown(args[i]) || strings.ContainsRune(args[i], varMark) {
		return "this git submodule names its subcommand with a word the line does not spell out, which may be foreach, which runs a command.", true
	}
	if args[i] != "foreach" {
		return "", false
	}
	i++
	for i < len(args) && (args[i] == "-q" || args[i] == "--quiet" || args[i] == "--recursive") {
		i++
	}
	return wordsLine("submodule foreach", args[i:], clean)
}

// bisectRunRefusal reads `git bisect run <cmd> [<arg>...]` (git-bisect(1)):
// every word after run is the command git runs. A subcommand word the line
// does not spell out may be run.
func bisectRunRefusal(args []string, clean func(string) bool) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	if isUnknown(args[0]) || strings.ContainsRune(args[0], varMark) {
		return "this git bisect names its subcommand with a word the line does not spell out, which may be run, which runs a command.", true
	}
	if args[0] != "run" {
		return "", false
	}
	return wordsLine("bisect run", args[1:], clean)
}

// filterBranchCommands are git filter-branch's options whose value is a
// command it evaluates in a shell (git-filter-branch(1)): --setup, and every
// filter but --subdirectory-filter, which names a directory.
var filterBranchCommands = []string{"--setup", "--env-filter", "--tree-filter", "--index-filter",
	"--parent-filter", "--msg-filter", "--commit-filter", "--tag-name-filter"}

// filterBranchRefusal reads git filter-branch's options as its own parser
// does: each option word whole, every one but -f, --force,
// --remap-to-ancestor and --prune-empty taking the next word as its value,
// up to `--` or the first word that is not an option.
func filterBranchRefusal(args []string, clean func(string) bool) (string, bool) {
	for i := 0; i < len(args); i++ {
		w := args[i]
		switch {
		case w == "--":
			return "", false
		case w == "-f" || w == "--force" || w == "--remap-to-ancestor" || w == "--prune-empty":
			continue
		case isUnknown(w) || strings.ContainsRune(w, varMark):
			return "this git filter-branch carries a word the line does not spell out where it reads its options, which may be a filter git runs.", true
		case !strings.HasPrefix(w, "-"):
			return "", false
		}
		if i+1 >= len(args) {
			return "", false
		}
		i++
		if containsString(filterBranchCommands, w) {
			if why, bad := (runTarget{kind: runLine}).judge(w, args[i], true, clean); bad {
				return why, true
			}
		}
	}
	return "", false
}
