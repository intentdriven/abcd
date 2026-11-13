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
// and no blob through a program does not ask. Status, which knows whether it
// blanked the filters even when the repository names none, decides for
// itself (isolatedGitBlanked).

// globalValued are git's global options that take their value as the next
// argument, before the subcommand.
var globalValued = []string{"-c", "-C", "--git-dir", "--work-tree", "--namespace", "--config-env", "--super-prefix"}

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
func (g gitArgs) diffProgramsAsked() bool { return g.has("--textconv") || g.has("--ext-diff") }

// patchy reports whether the command writes a patch or counts lines of
// content (a stat), or searches content (pickaxe, line log): what a textconv
// program is run for.
func (g gitArgs) patchy() bool {
	return g.has("-p", "-u", "--patch", "--patch-with-stat", "--patch-with-raw", "--cc", "--check", "--compact-summary") ||
		g.hasPrefix("--stat", "--numstat", "--shortstat", "--dirstat", "-U", "--unified", "--word-diff", "--color-words", "--function-context") ||
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
// alone, conservatively: a command it is unsure of asks.
//
//   - hash-object of a blob from a path: unless --no-filters or
//     FilterOverrides (standard input without --path, and a tree, commit or
//     tag, are hashed as given).
//   - cat-file: only with --filters (unless FilterOverrides) or --textconv.
//   - status, archive, ls-files -m, update-index (but --cacheinfo and
//     --index-info): unless FilterOverrides.
//   - diff: when a side is the working tree (no --cached or --staged, fewer
//     than two revisions) unless FilterOverrides, and when it shows content
//     unless --no-textconv and --no-ext-diff.
//   - diff-files, and diff-index without --cached: a patch reads the working
//     tree through its filters unless FilterOverrides; diff-tree, diff-index
//     and diff-files run textconv or an external diff only with --textconv or
//     --ext-diff.
//   - show: with --textconv or --ext-diff, or showing a commit's patch
//     unless --no-textconv and --no-ext-diff (a blob, rev:path, is shown as
//     stored unless --textconv is given).
//   - log and whatchanged: with --textconv or --ext-diff, or with a patch, a
//     stat or a content search unless --no-textconv and --no-ext-diff.
//   - blame and annotate: unless --no-textconv and, for the working tree's
//     file, FilterOverrides.
//   - grep: only with --textconv.
//   - apply: unless --cached, a stat alone, or FilterOverrides.
//   - merge-tree: unless MergeDriverOverrides.
//   - add, stash, checkout, switch, restore, checkout-index, commit, rm
//     (but --cached), reset (but --soft), read-tree -u, worktree add, am,
//     clone: unless FilterOverrides; merge, cherry-pick, revert, rebase and
//     pull also unless MergeDriverOverrides.
//
// merge-file starts nothing: it merges the files it is given with git's own
// merge, reading no attributes. Every other subcommand reads no working-tree
// content and no blob through a program.
func LeavesContentProgramsOn(args []string) bool {
	g := readGitArgs(args)
	judge, ok := contentJudges[g.verb]
	return ok && judge(g, g.filtersBlanked())
}

// JudgesContentPrograms reports whether verb is a subcommand
// LeavesContentProgramsOn reads the options of: one that can start a
// content filter, a textconv or external diff program, or a merge driver.
// Every other subcommand starts none of them.
func JudgesContentPrograms(verb string) bool {
	_, ok := contentJudges[verb]
	return ok
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
	"cat-file": func(g gitArgs, blanked bool) bool { return (g.has("--filters") && !blanked) || g.has("--textconv") },
	"status":   func(_ gitArgs, blanked bool) bool { return !blanked },
	"archive":  func(_ gitArgs, blanked bool) bool { return !blanked },
	"ls-files": func(g gitArgs, blanked bool) bool { return g.has("-m", "--modified") && !blanked },
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
		blobs := len(g.revs) > 0 && !slices.ContainsFunc(g.revs, func(r string) bool { return !strings.Contains(r, ":") })
		return g.diffProgramsAsked() || (!blobs && !g.noPatch() && !g.diffProgramsOff())
	},
	"log":         logJudge,
	"whatchanged": logJudge,
	"blame":       blameJudge,
	"annotate":    blameJudge,
	"grep":        func(g gitArgs, _ bool) bool { return g.has("--textconv") },
	"apply": func(g gitArgs, blanked bool) bool {
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
	"read-tree":      func(g gitArgs, blanked bool) bool { return g.has("-u") && !blanked },
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
