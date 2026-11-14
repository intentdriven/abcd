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
// the table does not know asks (fix round 6), and so does one on the
// starts-nothing list in a form that can start a program or reach the network
// (iss-2611140432160343).

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
	// dashdash is whether the command line holds a "--", so that what
	// stands before it is read as revisions and options alone.
	dashdash bool
	// words are the options and revisions in the order given, for an
	// option whose value can be the next word (--format, --pretty).
	words []string
	// value says, for each of words, whether it is the value of the option
	// before it (valuedNext).
	value []bool
	// counted says, for each of words, whether it can be read as a word that
	// makes the command not ask (given, givenOpts, givenRevs): it is not the
	// value of the option before it, and the word before it is known not to
	// take it as a value (counts).
	counted []bool
}

// valued are the options of one subcommand that git takes a value for as
// the next word when no "=" gives one: long names, and the letters of short
// options, which take the next word when the option ends a word of
// single-dash letters. exact is for the diff and revision options, which git
// matches whole; the others are matched in any prefix git takes for them
// (one git would refuse as ambiguous counts too), so more words are taken as
// values, and more commands ask.
//
// noValue are the options known to take no value, matched whole: the ones
// abcd's own calls give before a word that makes the command not ask, and no
// others. quiet are the words the subcommand's judge reads as making it not
// ask, alone or as name=value. Neither list can name every option git takes,
// so the word after an option on neither list is not read as one that makes
// the command not ask (counts), and the command asks.
type valued struct {
	long    []string
	short   string
	exact   bool
	noValue []string
	quiet   []string
}

// counts reports whether the word after prev, a word that is not the first
// after the subcommand, is known not to be prev's value: prev is not an
// option, holds "=", is on noValue, or is a word of quiet that was counted
// itself (prevCounted). The first word after the subcommand always is.
func (v valued) counts(prev string, prevCounted bool) bool {
	return !strings.HasPrefix(prev, "-") || strings.Contains(prev, "=") ||
		slices.Contains(v.noValue, prev) || prevCounted && named(prev, v.quiet)
}

// takesNext reports whether option word a takes the word after it as its
// value.
func (v valued) takesNext(a string) bool {
	if strings.Contains(a, "=") || len(a) < 2 || a[0] != '-' {
		return false
	}
	if strings.HasPrefix(a, "--") {
		return len(a) > 2 && slices.ContainsFunc(v.long, func(n string) bool {
			return n == a || !v.exact && strings.HasPrefix(n, a)
		})
	}
	return strings.IndexByte(v.short, a[len(a)-1]) >= 0
}

// diffValued are the diff options that take their value as the next word,
// and revValued the revision and log options that do (-n 1, --author x).
var (
	diffValued = []string{"--output", "--output-indicator-new", "--output-indicator-old", "--output-indicator-context",
		"--stat-width", "--stat-name-width", "--stat-graph-width", "--stat-count", "--unified", "--skip-to", "--rotate-to",
		"--find-object", "--diff-filter", "--word-diff-regex", "--color-moved-ws", "--ignore-matching-lines",
		"--inter-hunk-context", "--src-prefix", "--dst-prefix", "--line-prefix", "--ws-error-highlight", "--anchored",
		"--diff-algorithm", "--max-depth", "--diff-merges"}
	revValued = []string{"--max-count", "--skip", "--since", "--after", "--until", "--before", "--min-age", "--max-age",
		"--since-as-filter", "--author", "--committer", "--grep", "--grep-reflog", "--exclude", "--date",
		"--decorate-refs", "--decorate-refs-exclude"}
	diffLog = valued{long: append(slices.Clip(diffValued), revValued...), short: "UlOSGInL", exact: true,
		noValue: []string{"--no-renames", "-z", "--no-color", "-p", "--shortstat"},
		quiet: []string{"--cached", "--staged", "-s", "--no-patch", "--name-only", "--name-status", "--raw", "--quiet",
			"--no-textconv", "--no-ext-diff"}}
)

// valuedNext is valued by subcommand, for each subcommand whose judge reads
// a word as one that makes it not ask; a subcommand it does not hold has
// empty lists. Where it is unclear whether an option takes its value as the
// next word, it is listed, so the word after it is read as its value and the
// command asks. Options whose value can only be given after "=" (tag's
// --column and --color, init's --shared) are not.
var valuedNext = map[string]valued{
	"tag": {long: []string{"--format", "--sort", "--cleanup", "--contains", "--no-contains", "--merged", "--no-merged",
		"--points-at", "--message", "--file", "--local-user", "--trailer"}, short: "mFu",
		quiet: []string{"-l", "--list", "-n", "--contains", "--no-contains", "--merged", "--no-merged", "--points-at"}},
	"init": {long: []string{"--template", "--separate-git-dir", "--object-format", "--ref-format", "--initial-branch"}, short: "b",
		noValue: []string{"-q"}},
	"diff":        diffLog,
	"diff-index":  diffLog,
	"show":        diffLog,
	"log":         diffLog,
	"whatchanged": diffLog,
	"reflog":      diffLog,
	"blame": {long: append([]string{"--contents", "--ignore-rev", "--ignore-revs-file"}, revValued...), short: "LS",
		quiet: []string{"--no-textconv"}},
	"hash-object":  {long: []string{"--path"}, short: "t", noValue: []string{"-w"}, quiet: []string{"--no-filters", "--stdin"}},
	"update-index": {long: []string{"--cacheinfo", "--chmod", "--index-version"}, noValue: []string{"--add"}, quiet: []string{"--cacheinfo", "--index-info"}},
	"apply": {long: []string{"--exclude", "--include", "--directory", "--whitespace", "--build-fake-ancestor"}, short: "pC",
		noValue: []string{"--check"}, quiet: []string{"--cached", "--stat", "--numstat", "--summary"}},
	"rm":    {long: []string{"--pathspec-from-file"}, noValue: []string{"-q"}, quiet: []string{"--cached"}},
	"reset": {long: []string{"--pathspec-from-file"}, quiet: []string{"--soft"}},
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
	v := valuedNext[g.verb]
	takes := false
	for j, a := range args[i+1:] {
		if a == "--" {
			g.paths = j+i+2 < len(args)
			g.dashdash = true
			break
		}
		counted := !takes
		if j > 0 {
			counted = counted && v.counts(g.words[j-1], g.counted[j-1])
		}
		g.words = append(g.words, a)
		g.value = append(g.value, takes)
		g.counted = append(g.counted, counted)
		takes = !takes && v.takesNext(a)
		if strings.HasPrefix(a, "-") {
			g.opts = append(g.opts, a)
		} else {
			g.revs = append(g.revs, a)
		}
	}
	return g
}

// named reports whether option o is one of names, alone or as name=value.
func named(o string, names []string) bool {
	for _, n := range names {
		if o == n || strings.HasPrefix(o, n+"=") {
			return true
		}
	}
	return false
}

// has reports whether any option is one of names, alone or as name=value,
// every word read, another option's value included, so it is read only as a
// reason to ask; a reason not to is read by given.
func (g gitArgs) has(names ...string) bool {
	return slices.ContainsFunc(g.opts, func(o string) bool { return named(o, names) })
}

// given is has for a word that makes the command not ask: a counted option.
func (g gitArgs) given(names ...string) bool {
	for i, w := range g.words {
		if g.counted[i] && strings.HasPrefix(w, "-") && named(w, names) {
			return true
		}
	}
	return false
}

// givenOpts are the counted options.
func (g gitArgs) givenOpts() []string {
	var out []string
	for i, w := range g.words {
		if g.counted[i] && strings.HasPrefix(w, "-") {
			out = append(out, w)
		}
	}
	return out
}

// givenRevs are the counted revisions.
func (g gitArgs) givenRevs() []string {
	var out []string
	for i, w := range g.words {
		if g.counted[i] && !strings.HasPrefix(w, "-") {
			out = append(out, w)
		}
	}
	return out
}

// plain reports whether any word that is not an option is not the value of
// the option before it: a path, or a revision, as a reason to ask.
func (g gitArgs) plain() bool {
	for i, w := range g.words {
		if !g.value[i] && !strings.HasPrefix(w, "-") {
			return true
		}
	}
	return false
}

// hasAbbrev reports whether any option is name, name=value, or a shorter
// prefix of name past "--", which git's option parser takes as name when no
// other option of the subcommand shares it. others are options of their own
// that are prefixes of name (--text of --textconv), which are not it. A
// prefix git would refuse as ambiguous is counted too. Every judge reads
// what hasAbbrev reports as a reason to ask, never as a reason not to (a
// listing or a read is recognised by exact words: has), so counting more
// asks more.
func (g gitArgs) hasAbbrev(name string, others ...string) bool {
	return slices.ContainsFunc(g.opts, func(o string) bool {
		o, _, _ = strings.Cut(o, "=")
		return len(o) > 2 && strings.HasPrefix(o, "--") && strings.HasPrefix(name, o) && !slices.Contains(others, o)
	})
}

// hasShort reports whether any option gives the one-letter option c, alone
// or in a bundle of short options (-tm). A bundle whose letter c is another
// option's value (-mfinal) is counted too. Every judge reads what hasShort
// reports as a reason to ask, never as a reason not to, so counting more
// asks more.
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
func (g gitArgs) diffProgramsOff() bool { return g.given("--no-textconv") && g.given("--no-ext-diff") }

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
	return g.given("-s", "--no-patch", "--name-only", "--name-status", "--raw", "--quiet") && !g.patchy()
}

// twoTrees reports whether a diff compares two revisions, so that no side
// is the working tree: a range, or two revisions before a "--". Without a
// "--" git may read the second word as a path, which leaves the working tree
// a side, so two words alone are not taken as two revisions. An option's
// value is not a revision.
func (g gitArgs) twoTrees() bool {
	revs := g.givenRevs()
	return len(revs) >= 2 && g.dashdash || slices.ContainsFunc(revs, func(r string) bool { return strings.Contains(r, "..") })
}

// LeavesContentProgramsOn reports whether a git command with args, run in
// the isolated environment (ExecPins, no global or system settings), can
// start a program the repository's settings name that ExecPins leaves on: a
// content filter (filter.<name>.clean, .smudge, .process) over working-tree
// content it reads or writes, a textconv or external diff program over
// content it shows, or a merge driver. It is read from the command line
// alone: a subcommand contentJudges holds is judged by its options as below;
// one startsNothing holds does not ask, unless startsInForm says the form it
// is given can start a program or reach the network; any other asks. The four options that
// switch a program on (--textconv, --ext-diff, --filters, --modified) are
// read in any prefix git takes for them, and -m and -u inside a bundle of
// short options. A word that makes a command not ask counts only where the
// word before it is known not to take it as a value (valued.counts).
//
//   - hash-object of a blob from a path: unless --no-filters or
//     FilterOverrides (standard input without --path, and a tree, commit or
//     tag, are hashed as given).
//   - cat-file: only with --filters (unless FilterOverrides) or --textconv.
//   - status, archive, ls-files -m, update-index (but --cacheinfo or
//     --index-info with no path besides --cacheinfo's value): unless
//     FilterOverrides. A status without the blanks asks in any repository.
//   - diff: when a side is the working tree (no --cached or --staged, and
//     neither a range nor two revisions before a "--") unless
//     FilterOverrides, and when it shows content
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
	if !StartsNothing(g.verb) {
		return true
	}
	if judge, ok := startsInForm[g.verb]; ok {
		return judge(g)
	}
	return false
}

// JudgesContentPrograms reports whether verb is a subcommand
// LeavesContentProgramsOn reads the options of: one that can start a
// content filter, a textconv or external diff program, or a merge driver.
func JudgesContentPrograms(verb string) bool {
	_, ok := contentJudges[verb]
	return ok
}

// StartsNothing reports whether verb is a subcommand startsNothing holds,
// which LeavesContentProgramsOn lets run without asking in the form abcd runs
// it (startsInForm judges the others).
func StartsNothing(verb string) bool {
	_, ok := startsNothing[verb]
	return ok
}

// startsNothing are the subcommands, beyond those contentJudges reads, that
// abcd runs through an isolated runner, each with the form run and why it
// starts no program a repository's settings name. It holds only those: a
// subcommand abcd does not run is in neither list, so it asks. A subcommand
// with a form that can start a program or reach the network is judged in that
// form by startsInForm. config, var and rev-parse are the drive's comparison's
// own reads, which must not ask. ls-remote is on neither list: it reaches the
// network, and the networked runners ask before it runs.
var startsNothing = map[string]string{
	"rev-parse":        "names objects and paths",
	"rev-list":         "lists commits; a format that shows a signature check (%G) asks",
	"merge-base":       "reads commits",
	"config":           "reads settings (--get, --get-regexp, --list); --edit, -e and edit, which open the editor, ask",
	"var":              "reads settings and identities",
	"for-each-ref":     "lists refs; a format that shows a signature check (%(signature)) asks",
	"symbolic-ref":     "reads a ref",
	"update-ref":       "writes a ref",
	"ls-tree":          "lists a tree",
	"check-attr":       "reads attributes",
	"check-ignore":     "reads ignore rules",
	"check-ref-format": "checks a name",
	"commit-tree":      "writes a commit from a tree; ExecPins keeps commit signing off, and -S or --gpg-sign, which start the signing program, ask",
	"write-tree":       "writes the index as a tree",
	"init": "makes an empty repository (init -q --template=); any other form, which copies a template's hooks in " +
		"(git's own when none is named), asks",
	"remote": "lists remotes, reads one's URL or removes one (remote, remote get-url, remote remove); " +
		"a form that reaches the network (update, show, prune, add -f, set-head -a) or one not known asks",
	"shortlog": "counts commits by author (shortlog -sne HEAD), with no diff; a format that shows a signature check asks",
	"cherry":   "compares commits' patch ids (cherry <upstream> <head>), hashed from stored content with no textconv or external diff",
	"branch": "lists branches (branch --format=... --no-merged <rev>); --edit-description, which opens the editor, " +
		"and a format that shows a signature check, ask",
	"tag": "lists tags (tag --list, with --sort, --contains or --merged); only a listing given in exact words, with no word " +
		"that makes, signs, verifies or deletes a tag and no format that shows a signature check, does not ask",
	"describe": "names a commit by a tag (describe --tags --match <pattern> --abbrev=0 <rev>); --dirty and --broken, " +
		"which refresh the index through the content filters, ask",
}

// startsInForm judges, for a subcommand on the starts-nothing list that has
// a form able to start a program (a signing program, the editor, a hook a
// template copies in, the content filters) or reach the network, whether a
// command line g is in such a form. An option that makes it ask is read in
// any prefix git takes for it and a short one inside a bundle or glued to a
// value; a prefix git would refuse as ambiguous counts too, so each reading
// asks more, never less. A form that does not ask is recognised by counted
// words (valued.counts): a listing by exact words, the empty template in any
// prefix git takes for --template=.
var startsInForm = map[string]func(g gitArgs) bool{
	"tag":          tagAsks,
	"describe":     func(g gitArgs) bool { return g.hasAbbrev("--dirty") || g.hasAbbrev("--broken") },
	"branch":       func(g gitArgs) bool { return g.hasAbbrev("--edit-description") || g.signatureShown() },
	"rev-list":     func(g gitArgs) bool { return g.signatureShown() },
	"shortlog":     func(g gitArgs) bool { return g.signatureShown() },
	"for-each-ref": func(g gitArgs) bool { return g.signatureShown() },
	"config": func(g gitArgs) bool {
		return g.hasAbbrev("--edit") || g.hasShort('e') || len(g.revs) > 0 && g.revs[0] == "edit"
	},
	"commit-tree": func(g gitArgs) bool {
		return g.hasShort('S') || g.hasAbbrev("--gpg-sign")
	},
	"init": func(g gitArgs) bool {
		// Only --template= with no value copies no template: with none
		// named git copies its own, hooks included, and --template given
		// its value as the next word, or any directory, copies that one's.
		// An empty template that is not counted (--separate-git-dir
		// --template=, where it is that option's value) is not one.
		// --no-template takes back any template named before it, so git
		// copies its own.
		if g.hasAbbrev("--no-template") {
			return true
		}
		empty := false
		for _, o := range g.opts {
			name, value, valued := strings.Cut(o, "=")
			if len(name) > 2 && strings.HasPrefix("--template", name) && (!valued || value != "") {
				return true
			}
		}
		for _, o := range g.givenOpts() {
			name, value, valued := strings.Cut(o, "=")
			if len(name) > 2 && strings.HasPrefix("--template", name) && valued && value == "" {
				empty = true
			}
		}
		return !empty
	},
	"remote": func(g gitArgs) bool {
		if len(g.revs) == 0 {
			return false
		}
		switch g.revs[0] {
		case "get-url", "remove", "rm", "rename", "set-url", "set-branches":
			return false
		case "add":
			return g.hasShort('f') || g.hasAbbrev("--fetch")
		case "set-head":
			return g.hasShort('a') || g.hasAbbrev("--auto")
		}
		return true
	},
}

// tagMakes are the short options of tag that make, sign, verify or delete a
// tag, or give its message (-a, -s, -u, -m, -F, -f, -d, -e, -v), and
// tagMakesLong their long forms, with --trailer, which makes an annotated
// tag.
const tagMakes = "asumFfdev"

var tagMakesLong = []string{"--annotate", "--sign", "--local-user", "--message", "--file", "--force", "--delete", "--edit", "--verify", "--trailer"}

// tagAsks judges tag: it does not ask only for a listing, a command line
// that holds a listing word of its own (-l, --list, -n, -n<digits>,
// --contains, --no-contains, --merged, --no-merged, --points-at, each
// exactly, the last five alone or with =value, and each counted: tag
// --format -l v1 makes v1) or no argument at all, and
// holds no option of tagMakes, alone, in a bundle or glued to a value
// (-mfinal), none of tagMakesLong in any prefix, and no format that shows a
// signature check. Anything else asks: a tag made can be signed through the
// signing program the repository names (tag.gpgSign,
// tag.forceSignAnnotated) or open the editor.
func tagAsks(g gitArgs) bool {
	for _, o := range g.opts {
		if len(o) > 1 && o[0] == '-' && o[1] != '-' && strings.ContainsAny(o[1:], tagMakes) {
			return true
		}
	}
	if slices.ContainsFunc(tagMakesLong, func(n string) bool { return g.hasAbbrev(n) }) || g.signatureShown() {
		return true
	}
	if len(g.words) == 0 && !g.dashdash {
		return false
	}
	listing := slices.ContainsFunc(g.givenOpts(), func(o string) bool {
		if o == "-l" || o == "--list" || o == "-n" {
			return true
		}
		if digits, ok := strings.CutPrefix(o, "-n"); ok && strings.Trim(digits, "0123456789") == "" {
			return true
		}
		return false
	}) || g.given("--contains", "--no-contains", "--merged", "--no-merged", "--points-at")
	return !listing
}

// signatureShown reports whether the command shows a check of a signature,
// which runs the verifying program the repository names (gpg.program and
// its kin): --show-signature in any prefix git takes for it, or a --format
// or --pretty value, given after "=" or as the next word, the option in
// any prefix, that holds a %G placeholder or a %(signature atom. One reading
// for log, show, rev-list, shortlog, for-each-ref, branch and tag.
func (g gitArgs) signatureShown() bool {
	if g.hasAbbrev("--show-signature") {
		return true
	}
	for i, w := range g.words {
		name, value, valued := strings.Cut(w, "=")
		if len(name) <= 2 || !strings.HasPrefix(name, "--") ||
			!strings.HasPrefix("--format", name) && !strings.HasPrefix("--pretty", name) {
			continue
		}
		if !valued && i+1 < len(g.words) {
			value = g.words[i+1]
		}
		if strings.Contains(value, "%G") || strings.Contains(value, "%(signature") {
			return true
		}
	}
	return false
}

// contentJudges is LeavesContentProgramsOn by subcommand: each says whether
// a command line g, its filters blanked or not, can start a program.
var contentJudges = map[string]func(g gitArgs, blanked bool) bool{
	"hash-object": func(g gitArgs, blanked bool) bool {
		// git filters a blob read from a path alone: not standard input
		// without --path, and not an object of another type (a counted -t
		// given its type as the next word).
		other := g.given("--type=tree", "--type=commit", "--type=tag")
		for i, w := range g.words {
			if w == "-t" && g.counted[i] && i+1 < len(g.words) && slices.Contains([]string{"tree", "commit", "tag"}, g.words[i+1]) {
				other = true
			}
		}
		return !blanked && !g.given("--no-filters") && !other && !(g.given("--stdin") && !g.has("--path"))
	},
	"cat-file": func(g gitArgs, blanked bool) bool {
		return (g.hasAbbrev("--filters") && !blanked) || g.hasAbbrev("--textconv")
	},
	"status":   func(_ gitArgs, blanked bool) bool { return !blanked },
	"archive":  func(_ gitArgs, blanked bool) bool { return !blanked },
	"ls-files": func(g gitArgs, blanked bool) bool { return (g.hasShort('m') || g.hasAbbrev("--modified")) && !blanked },
	"update-index": func(g gitArgs, blanked bool) bool {
		// A path besides --cacheinfo's value, or after "--", is read from the
		// working tree.
		return !blanked && (!g.given("--cacheinfo", "--index-info") || g.plain() || g.paths)
	},
	"diff": func(g gitArgs, blanked bool) bool {
		worktree := !g.given("--cached", "--staged") && !g.twoTrees()
		return (worktree && !blanked) || (!g.noPatch() && !g.diffProgramsOff()) || g.diffProgramsAsked()
	},
	"diff-files": func(g gitArgs, blanked bool) bool { return (g.patchy() && !blanked) || g.diffProgramsAsked() },
	"diff-index": func(g gitArgs, blanked bool) bool {
		return (!g.given("--cached") && g.patchy() && !blanked) || g.diffProgramsAsked()
	},
	"diff-tree": func(g gitArgs, _ bool) bool { return g.diffProgramsAsked() },
	"show": func(g gitArgs, _ bool) bool {
		blob := func(r string) bool {
			return strings.Contains(r, ":") && !strings.HasPrefix(r, ":/") && !strings.Contains(r, "^{/")
		}
		// Every revision, an option's value included, must name a blob, and
		// at least one that is not a value must be given: show -O HEAD:a
		// shows HEAD's patch.
		blobs := len(g.givenRevs()) > 0 && !slices.ContainsFunc(g.revs, func(r string) bool { return !blob(r) })
		return g.diffProgramsAsked() || g.signatureShown() || (!blobs && !g.noPatch() && !g.diffProgramsOff())
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
		stat := g.given("--stat", "--numstat", "--summary") && !g.has("--apply", "--check")
		return !g.given("--cached") && !stat && !blanked
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
	"rm":             func(g gitArgs, blanked bool) bool { return !g.given("--cached") && !blanked },
	"reset":          func(g gitArgs, blanked bool) bool { return !g.given("--soft") && !blanked },
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
	return g.diffProgramsAsked() || g.signatureShown() || (g.patchy() && !g.diffProgramsOff())
}

func blameJudge(g gitArgs, blanked bool) bool {
	revs := g.givenRevs()
	rev := g.paths && len(revs) > 0 || !g.paths && len(revs) > 1
	return !g.given("--no-textconv") || (!rev && !blanked) || g.has("--contents")
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
