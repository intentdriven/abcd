package gitutil

import (
	"os/exec"
	"slices"
	"strings"
)

// contentprograms.go is where an isolated git command asks the check
// SetBeforeHookedGit set (iss-2611131308266006, fix round 5, ruling 1). The
// isolated environment pins hooks and the other programs ExecPins names off,
// but not content filters, textconv and external diff programs, or merge
// drivers, which are keyed on names the repository chooses. So what decides
// whether an isolated command asks is what it can start, not which runner
// built it: isolatedGit and isolatedGitContext ask when
// LeavesContentProgramsOn says the command can start one of those, and a
// command that blanks them (FilterOverrides, --no-filters, --no-textconv
// with --no-ext-diff, MergeDriverOverrides) or reads no working-tree content
// and no blob through a program does not ask. A status that does not carry
// the filter blanks asks in any repository, one that names no filter
// included: the command line cannot show that none is named. A subcommand
// the table does not know asks (fix round 6).

// globalValued are git's global options that take their value as the next
// argument, before the subcommand.
var globalValued = []string{"-c", "-C", "--git-dir", "--work-tree", "--namespace", "--config-env", "--super-prefix", "--attr-source"}

// Subcommand is the git subcommand args run, past git's global options and
// their values; "" when there is none.
func Subcommand(args []string) string {
	if i := subcommandIndex(args); i >= 0 {
		return args[i]
	}
	return ""
}

// subcommandIndex is the index of args' subcommand, -1 for none.
func subcommandIndex(args []string) int {
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case slices.Contains(globalValued, a):
			i++
		case strings.HasPrefix(a, "-"):
		default:
			return i
		}
	}
	return -1
}

// gitArgs is one command line read for what it can start: the settings
// given before the subcommand (each `-c` value), the subcommand, the options
// and revisions after it up to "--", and whether paths follow a "--".
type gitArgs struct {
	settings []string
	verb     string
	opts     []string
	revs     []string
	paths    bool
}

func readGitArgs(args []string) gitArgs {
	var g gitArgs
	i := subcommandIndex(args)
	for j := 0; j < len(args) && (i < 0 || j < i); j++ {
		if args[j] == "-c" && j+1 < len(args) {
			g.settings = append(g.settings, args[j+1])
			j++
		}
	}
	if i < 0 {
		return g
	}
	g.verb = args[i]
	for j, a := range args[i+1:] {
		if a == "--" {
			g.paths = j+i+2 < len(args)
			break
		}
		if strings.HasPrefix(a, "-") {
			g.opts = append(g.opts, a)
		} else {
			g.revs = append(g.revs, a)
		}
	}
	return g
}

// has reports whether any option is one of names, alone or as name=value.
func (g gitArgs) has(names ...string) bool {
	return slices.ContainsFunc(g.opts, func(o string) bool {
		for _, n := range names {
			if o == n || strings.HasPrefix(o, n+"=") {
				return true
			}
		}
		return false
	})
}

// hasAbbrev reports whether any option is name, name=value, or a shorter
// prefix of name past "--", which git's option parser takes as name when no
// other option of the subcommand shares it. others are options of their own
// that are prefixes of name (--text of --textconv), which are not it. A
// prefix git would refuse as ambiguous is counted too: that only asks more.
func (g gitArgs) hasAbbrev(name string, others ...string) bool {
	return slices.ContainsFunc(g.opts, func(o string) bool {
		o, _, _ = strings.Cut(o, "=")
		return len(o) > 2 && strings.HasPrefix(o, "--") && strings.HasPrefix(name, o) && !slices.Contains(others, o)
	})
}

// hasShort reports whether any option gives the one-letter option c, alone
// or in a bundle of short options (-tm). A bundle whose letter c is another
// option's value is counted too: that only asks more.
func (g gitArgs) hasShort(c byte) bool {
	return slices.ContainsFunc(g.opts, func(o string) bool {
		return len(o) > 1 && o[0] == '-' && o[1] != '-' && strings.IndexByte(o[1:], c) >= 0
	})
}

// hasPrefix reports whether any option starts with one of prefixes.
func (g gitArgs) hasPrefix(prefixes ...string) bool {
	return slices.ContainsFunc(g.opts, func(o string) bool {
		for _, p := range prefixes {
			if strings.HasPrefix(o, p) {
				return true
			}
		}
		return false
	})
}

// filtersBlanked reports whether the command carries FilterOverrides'
// settings, which blank every content filter the repository named when they
// were read.
func (g gitArgs) filtersBlanked() bool {
	return slices.ContainsFunc(g.settings, func(s string) bool {
		return strings.HasPrefix(s, "filter.") &&
			(strings.HasSuffix(s, ".clean=") || strings.HasSuffix(s, ".smudge=") || strings.HasSuffix(s, ".process="))
	})
}

// driversBuiltin reports whether the command carries MergeDriverOverrides'
// settings, which make every merge driver git's built-in text merge.
func (g gitArgs) driversBuiltin() bool {
	return slices.ContainsFunc(g.settings, func(s string) bool { return strings.HasPrefix(s, "merge.default=") })
}

// diffProgramsOff reports whether textconv and external diff programs are
// both switched off on the command line.
func (g gitArgs) diffProgramsOff() bool { return g.has("--no-textconv") && g.has("--no-ext-diff") }

// diffProgramsAsked reports whether the command asks for a textconv or an
// external diff program by name, which plumbing does only when told.
func (g gitArgs) diffProgramsAsked() bool {
	return g.hasAbbrev("--textconv", "--text") || g.hasAbbrev("--ext-diff")
}

// patchy reports whether the command writes a patch or counts lines of
// content (a stat), or searches content (pickaxe, line log): what a textconv
// program is run for. A merge's diff (-m, -c, --cc, --dd, --remerge-diff,
// --diff-merges=) and --binary each count as a patch.
func (g gitArgs) patchy() bool {
	return g.has("-p", "-u", "--patch", "--patch-with-stat", "--patch-with-raw", "--check", "--compact-summary",
		"-m", "-c", "--cc", "--dd", "--remerge-diff", "--binary") ||
		g.hasPrefix("--stat", "--numstat", "--shortstat", "--dirstat", "-U", "--unified", "--word-diff", "--color-words", "--function-context", "--diff-merges") ||
		slices.ContainsFunc(g.opts, func(o string) bool {
			return !strings.HasPrefix(o, "--") && (strings.HasPrefix(o, "-L") || strings.HasPrefix(o, "-G") || strings.HasPrefix(o, "-S"))
		})
}

// noPatch reports whether the command's output names files or objects
// alone, with nothing written of their content.
func (g gitArgs) noPatch() bool {
	return g.has("-s", "--no-patch", "--name-only", "--name-status", "--raw", "--quiet") && !g.patchy()
}

// twoTrees reports whether a diff compares two revisions, so that no side
// is the working tree.
func (g gitArgs) twoTrees() bool {
	return len(g.revs) >= 2 || slices.ContainsFunc(g.revs, func(r string) bool { return strings.Contains(r, "..") })
}

// LeavesContentProgramsOn reports whether a git command with args, run in
// the isolated environment (ExecPins, no global or system settings), can
// start a program the repository's settings name that ExecPins leaves on: a
// content filter (filter.<name>.clean, .smudge, .process) over working-tree
// content it reads or writes, a textconv or external diff program over
// content it shows, or a merge driver. It is read from the command line
// alone: a subcommand contentJudges holds is judged by its options as below;
// one startsNothing holds does not ask; any other asks. The four options that
// switch a program on (--textconv, --ext-diff, --filters, --modified) are
// read in any prefix git takes for them, and -m and -u inside a bundle of
// short options.
//
//   - hash-object of a blob from a path: unless --no-filters or
//     FilterOverrides (standard input without --path, and a tree, commit or
//     tag, are hashed as given).
//   - cat-file: only with --filters (unless FilterOverrides) or --textconv.
//   - status, archive, ls-files -m, update-index (but --cacheinfo and
//     --index-info): unless FilterOverrides. A status without the blanks
//     asks in any repository.
//   - diff: when a side is the working tree (no --cached or --staged, fewer
//     than two revisions) unless FilterOverrides, and when it shows content
//     unless --no-textconv and --no-ext-diff.
//   - diff-files, and diff-index without --cached: a patch reads the working
//     tree through its filters unless FilterOverrides; diff-tree, diff-index
//     and diff-files run textconv or an external diff only with --textconv or
//     --ext-diff.
//   - show: with --textconv or --ext-diff, or showing a commit's patch
//     unless --no-textconv and --no-ext-diff (a blob, rev:path, is shown as
//     stored unless --textconv is given; :/text and rev^{/text} name
//     commits, not blobs).
//   - log, whatchanged and reflog: with --textconv or --ext-diff, or with a
//     patch, a stat or a content search unless --no-textconv and
//     --no-ext-diff.
//   - blame and annotate: unless --no-textconv and, for the working tree's
//     file, FilterOverrides.
//   - grep: only with --textconv.
//   - apply: with --3way, which merges through the merge drivers, whatever
//     else it carries; otherwise unless --cached, a stat alone, or
//     FilterOverrides.
//   - merge-tree: unless MergeDriverOverrides.
//   - add, stash, checkout, switch, restore, checkout-index, commit, rm
//     (but --cached), reset (but --soft), read-tree -u, worktree add, am,
//     clone: unless FilterOverrides; merge, cherry-pick, revert, rebase and
//     pull also unless MergeDriverOverrides.
func LeavesContentProgramsOn(args []string) bool {
	g := readGitArgs(args)
	if judge, ok := contentJudges[g.verb]; ok {
		return judge(g, g.filtersBlanked())
	}
	return !StartsNothing(g.verb)
}

// JudgesContentPrograms reports whether verb is a subcommand
// LeavesContentProgramsOn reads the options of: one that can start a
// content filter, a textconv or external diff program, or a merge driver.
func JudgesContentPrograms(verb string) bool {
	_, ok := contentJudges[verb]
	return ok
}

// StartsNothing reports whether verb is a subcommand startsNothing holds,
// which LeavesContentProgramsOn lets run without asking.
func StartsNothing(verb string) bool {
	_, ok := startsNothing[verb]
	return ok
}

// startsNothing are the subcommands, beyond those contentJudges reads, that
// abcd runs through an isolated runner, each with the form run and why it
// starts no program a repository's settings name. It holds only those: a
// subcommand abcd does not run is in neither list, so it asks. config, var
// and rev-parse are the drive's comparison's own reads, which must not ask.
var startsNothing = map[string]string{
	"rev-parse":        "names objects and paths",
	"rev-list":         "lists commits",
	"merge-base":       "reads commits",
	"config":           "reads settings (--get, --get-regexp, --list); never --edit",
	"var":              "reads settings and identities",
	"for-each-ref":     "lists refs",
	"symbolic-ref":     "reads a ref",
	"update-ref":       "writes a ref",
	"ls-tree":          "lists a tree",
	"ls-remote":        "lists a remote's refs; the network is its runner's (RunLimitedNetContext)",
	"check-attr":       "reads attributes",
	"check-ignore":     "reads ignore rules",
	"check-ref-format": "checks a name",
	"commit-tree":      "writes a commit from a tree; ExecPins keeps commit signing off",
	"write-tree":       "writes the index as a tree",
	"init":             "makes an empty repository (init -q --template=)",
	"remote":           "lists remotes, reads one's URL or removes one (remote, remote get-url, remote remove)",
	"shortlog":         "counts commits by author (shortlog -sne HEAD), with no diff",
	"cherry":           "compares commits' patch ids (cherry <upstream> <head>), hashed from stored content with no textconv or external diff",
	"branch":           "lists branches (branch --format=... --no-merged <rev>); never --edit-description, which opens the editor",
	"tag": "lists tags (tag --list, with --sort, --contains or --merged); never -v, which starts the signing program, " +
		"or an annotation, which opens the editor",
	"describe": "names a commit by a tag (describe --tags --match <pattern> --abbrev=0 <rev>); never --dirty or --broken, " +
		"which refresh the index through the content filters",
}

// contentJudges is LeavesContentProgramsOn by subcommand: each says whether
// a command line g, its filters blanked or not, can start a program.
var contentJudges = map[string]func(g gitArgs, blanked bool) bool{
	"hash-object": func(g gitArgs, blanked bool) bool {
		// git filters a blob read from a path alone: not standard input
		// without --path, and not an object of another type.
		other := g.has("--type=tree", "--type=commit", "--type=tag") ||
			g.has("-t") && slices.ContainsFunc(g.revs, func(r string) bool { return r == "tree" || r == "commit" || r == "tag" })
		return !blanked && !g.has("--no-filters") && !other && !(g.has("--stdin") && !g.has("--path"))
	},
	"cat-file": func(g gitArgs, blanked bool) bool {
		return (g.hasAbbrev("--filters") && !blanked) || g.hasAbbrev("--textconv")
	},
	"status":   func(_ gitArgs, blanked bool) bool { return !blanked },
	"archive":  func(_ gitArgs, blanked bool) bool { return !blanked },
	"ls-files": func(g gitArgs, blanked bool) bool { return (g.hasShort('m') || g.hasAbbrev("--modified")) && !blanked },
	"update-index": func(g gitArgs, blanked bool) bool {
		return !blanked && !g.has("--cacheinfo", "--index-info")
	},
	"diff": func(g gitArgs, blanked bool) bool {
		worktree := !g.has("--cached", "--staged") && !g.twoTrees()
		return (worktree && !blanked) || (!g.noPatch() && !g.diffProgramsOff()) || g.diffProgramsAsked()
	},
	"diff-files": func(g gitArgs, blanked bool) bool { return (g.patchy() && !blanked) || g.diffProgramsAsked() },
	"diff-index": func(g gitArgs, blanked bool) bool {
		return (!g.has("--cached") && g.patchy() && !blanked) || g.diffProgramsAsked()
	},
	"diff-tree": func(g gitArgs, _ bool) bool { return g.diffProgramsAsked() },
	"show": func(g gitArgs, _ bool) bool {
		blob := func(r string) bool {
			return strings.Contains(r, ":") && !strings.HasPrefix(r, ":/") && !strings.Contains(r, "^{/")
		}
		blobs := len(g.revs) > 0 && !slices.ContainsFunc(g.revs, func(r string) bool { return !blob(r) })
		return g.diffProgramsAsked() || (!blobs && !g.noPatch() && !g.diffProgramsOff())
	},
	"log":         logJudge,
	"whatchanged": logJudge,
	"reflog":      logJudge,
	"blame":       blameJudge,
	"annotate":    blameJudge,
	"grep":        func(g gitArgs, _ bool) bool { return g.hasAbbrev("--textconv", "--text") },
	"apply": func(g gitArgs, blanked bool) bool {
		if g.hasAbbrev("--3way") || g.hasShort('3') {
			return true
		}
		stat := g.has("--stat", "--numstat", "--summary") && !g.has("--apply", "--check")
		return !g.has("--cached") && !stat && !blanked
	},
	"merge-tree":     func(g gitArgs, _ bool) bool { return !g.driversBuiltin() },
	"add":            writerJudge,
	"stash":          writerJudge,
	"checkout":       writerJudge,
	"switch":         writerJudge,
	"restore":        writerJudge,
	"checkout-index": writerJudge,
	"commit":         writerJudge,
	"am":             writerJudge,
	"clone":          writerJudge,
	"rm":             func(g gitArgs, blanked bool) bool { return !g.has("--cached") && !blanked },
	"reset":          func(g gitArgs, blanked bool) bool { return !g.has("--soft") && !blanked },
	"read-tree":      func(g gitArgs, blanked bool) bool { return g.hasShort('u') && !blanked },
	"worktree": func(g gitArgs, blanked bool) bool {
		return len(g.revs) > 0 && g.revs[0] == "add" && !blanked
	},
	"merge":       mergerJudge,
	"cherry-pick": mergerJudge,
	"revert":      mergerJudge,
	"rebase":      mergerJudge,
	"pull":        mergerJudge,
}

func logJudge(g gitArgs, _ bool) bool {
	return g.diffProgramsAsked() || (g.patchy() && !g.diffProgramsOff())
}

func blameJudge(g gitArgs, blanked bool) bool {
	rev := g.paths && len(g.revs) > 0 || !g.paths && len(g.revs) > 1
	return !g.has("--no-textconv") || (!rev && !blanked) || g.has("--contents")
}

func writerJudge(_ gitArgs, blanked bool) bool { return !blanked }

func mergerJudge(g gitArgs, blanked bool) bool { return !blanked || !g.driversBuiltin() }

// askIfContentRuns asks the check SetBeforeHookedGit set before an isolated
// command that LeavesContentProgramsOn says can start a program the
// repository names, setting its error as cmd.Err so Run, Output and Start
// return it and git never starts. With no check set it does nothing.
func askIfContentRuns(cmd *exec.Cmd, args []string) {
	if cmd.Err != nil || !LeavesContentProgramsOn(args) {
		return
	}
	if err := askBeforeHooked(args); err != nil {
		cmd.Err = err
	}
}
