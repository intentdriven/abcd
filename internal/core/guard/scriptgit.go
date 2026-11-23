package guard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// What a git command writes, read from its words as git reads them. git's
// parse-options reads a subcommand's options anywhere before `--`: a long
// option by its whole name or by a prefix that names no other, a negation as
// `--no-<name>`, short options in clusters, and a value glued on or as the
// next word (readOpts, with the subcommand's gitOpts). What a subcommand that
// otherwise writes no file is exempted for is an allowlist (optRead.only):
// any option it does not name voids the exemption.

// gitOpts are git 2.52's options for the subcommands whose options decide
// what they write, as `git <subcommand> --help-all` lists them.
var gitOpts = map[string]optSpec{
	"apply": {values: "pC", negate: true, long: map[string]int{
		"exclude": optValue, "include": optValue, "no-add": optFlag, "add": optFlag, "stat": optFlag,
		"allow-binary-replacement": optFlag, "binary": optFlag, "numstat": optFlag, "summary": optFlag,
		"check": optFlag, "index": optFlag, "intent-to-add": optFlag, "cached": optFlag,
		"unsafe-paths": optFlag, "apply": optFlag, "3way": optFlag, "ours": optFlag, "theirs": optFlag,
		"union": optFlag, "build-fake-ancestor": optValue, "whitespace": optValue,
		"ignore-space-change": optFlag, "ignore-whitespace": optFlag, "reverse": optFlag,
		"unidiff-zero": optFlag, "reject": optFlag, "allow-overlap": optFlag, "verbose": optFlag,
		"quiet": optFlag, "inaccurate-eof": optFlag, "recount": optFlag, "directory": optValue,
		"allow-empty": optFlag,
	}},
	"am": {values: "Cp", optional: "S", negate: true, long: map[string]int{
		"interactive": optFlag, "no-verify": optFlag, "verify": optFlag, "binary": optFlag, "3way": optFlag,
		"quiet": optFlag, "signoff": optFlag, "utf8": optFlag, "keep": optFlag, "keep-non-patch": optFlag,
		"message-id": optFlag, "keep-cr": optFlag, "scissors": optFlag, "quoted-cr": optValue,
		"whitespace": optValue, "ignore-space-change": optFlag, "ignore-whitespace": optFlag,
		"directory": optValue, "exclude": optValue, "include": optValue, "patch-format": optValue,
		"reject": optFlag, "resolvemsg": optValue, "continue": optFlag, "resolved": optFlag, "skip": optFlag,
		"abort": optFlag, "quit": optFlag, "show-current-patch": optOptional, "retry": optFlag,
		"allow-empty": optFlag, "committer-date-is-author-date": optFlag, "ignore-date": optFlag,
		"rerere-autoupdate": optFlag, "gpg-sign": optOptional, "empty": optValue, "rebasing": optFlag,
	}},
	"reset": {values: "U", negate: true, long: map[string]int{
		"quiet": optFlag, "no-refresh": optFlag, "refresh": optFlag, "mixed": optFlag, "soft": optFlag,
		"hard": optFlag, "merge": optFlag, "keep": optFlag, "recurse-submodules": optOptional, "patch": optFlag,
		"unified": optValue, "inter-hunk-context": optValue, "intent-to-add": optFlag,
		"pathspec-from-file": optValue, "pathspec-file-nul": optFlag,
	}},
	"read-tree": {negate: true, long: map[string]int{
		"super-prefix": optValue, "index-output": optValue, "empty": optFlag, "verbose": optFlag,
		"trivial": optFlag, "aggressive": optFlag, "reset": optFlag, "prefix": optValue,
		"exclude-per-directory": optValue, "dry-run": optFlag, "no-sparse-checkout": optFlag,
		"sparse-checkout": optFlag, "debug-unpack": optFlag, "recurse-submodules": optOptional, "quiet": optFlag,
	}},
	"checkout": {values: "bBU", negate: true, long: map[string]int{
		"guess": optFlag, "overlay": optFlag, "quiet": optFlag, "recurse-submodules": optOptional,
		"progress": optFlag, "merge": optFlag, "conflict": optValue, "detach": optFlag, "track": optOptional,
		"force": optFlag, "orphan": optValue, "overwrite-ignore": optFlag, "ignore-other-worktrees": optFlag,
		"ours": optFlag, "theirs": optFlag, "patch": optFlag, "unified": optValue, "inter-hunk-context": optValue,
		"ignore-skip-worktree-bits": optFlag, "pathspec-from-file": optValue, "pathspec-file-nul": optFlag,
	}},
	"switch": {values: "cC", negate: true, long: map[string]int{
		"create": optValue, "force-create": optValue, "guess": optFlag, "discard-changes": optFlag,
		"quiet": optFlag, "recurse-submodules": optOptional, "progress": optFlag, "merge": optFlag,
		"conflict": optValue, "detach": optFlag, "track": optOptional, "force": optFlag, "orphan": optValue,
		"overwrite-ignore": optFlag, "ignore-other-worktrees": optFlag,
	}},
	"restore": {values: "sU", negate: true, long: map[string]int{
		"source": optValue, "staged": optFlag, "worktree": optFlag, "ignore-unmerged": optFlag,
		"overlay": optFlag, "quiet": optFlag, "recurse-submodules": optOptional, "progress": optFlag,
		"merge": optFlag, "conflict": optValue, "ours": optFlag, "theirs": optFlag, "patch": optFlag,
		"unified": optValue, "inter-hunk-context": optValue, "ignore-skip-worktree-bits": optFlag,
		"pathspec-from-file": optValue, "pathspec-file-nul": optFlag,
	}},
	"mv": {negate: true, long: map[string]int{
		"verbose": optFlag, "dry-run": optFlag, "force": optFlag, "sparse": optFlag,
	}},
	"rm": {negate: true, long: map[string]int{
		"dry-run": optFlag, "quiet": optFlag, "cached": optFlag, "force": optFlag, "ignore-unmatch": optFlag,
		"sparse": optFlag, "pathspec-from-file": optValue, "pathspec-file-nul": optFlag,
	}},
	"format-patch": {values: "ov", negate: true, long: map[string]int{
		"numbered": optFlag, "no-numbered": optFlag, "signoff": optFlag, "stdout": optFlag,
		"cover-letter": optFlag, "numbered-files": optFlag, "suffix": optValue, "start-number": optValue,
		"reroll-count": optValue, "filename-max-length": optValue, "rfc": optOptional,
		"cover-from-description": optValue, "description-file": optValue, "subject-prefix": optValue,
		"output-directory": optValue, "keep-subject": optFlag, "no-binary": optFlag, "binary": optFlag,
		"zero-commit": optFlag, "ignore-if-in-upstream": optFlag, "no-stat": optFlag, "add-header": optValue,
		"to": optValue, "cc": optValue, "from": optOptional, "in-reply-to": optValue, "attach": optOptional,
		"inline": optOptional, "thread": optOptional, "signature": optValue, "base": optValue,
		"signature-file": optValue, "quiet": optFlag, "progress": optFlag, "interdiff": optValue,
		"range-diff": optValue, "creation-factor": optValue, "force-in-body-from": optFlag,
	}},
	"archive": {values: "o", negate: true, long: map[string]int{
		"format": optValue, "prefix": optValue, "add-file": optValue, "add-virtual-file": optValue,
		"output": optValue, "worktree-attributes": optFlag, "verbose": optFlag, "mtime": optValue,
		"list": optFlag, "remote": optValue, "exec": optValue,
	}},
	"bugreport": {values: "os", negate: true, long: map[string]int{
		"diagnose": optOptional, "output-directory": optValue, "suffix": optValue,
	}},
	"diagnose": {values: "os", negate: true, long: map[string]int{
		"output-directory": optValue, "suffix": optValue, "mode": optValue,
	}},
	"interpret-trailers": {negate: true, long: map[string]int{
		"in-place": optFlag, "trim-empty": optFlag, "where": optValue, "if-exists": optValue,
		"if-missing": optValue, "only-trailers": optFlag, "only-input": optFlag, "unfold": optFlag,
		"parse": optFlag, "no-divider": optFlag, "divider": optFlag, "trailer": optValue,
	}},
	"merge-file": {values: "L", negate: true, long: map[string]int{
		"stdout": optFlag, "object-id": optFlag, "diff3": optFlag, "zdiff3": optFlag, "ours": optFlag,
		"theirs": optFlag, "union": optFlag, "diff-algorithm": optValue, "marker-size": optValue, "quiet": optFlag,
	}},
	"init": {values: "b", negate: true, long: map[string]int{
		"template": optValue, "bare": optFlag, "shared": optOptional, "quiet": optFlag,
		"separate-git-dir": optValue, "initial-branch": optValue, "object-format": optValue, "ref-format": optValue,
	}},
}

// gitBundleCreateOpts are `git bundle create`'s own options, before its file.
var gitBundleCreateOpts = optSpec{negate: true, long: map[string]int{
	"quiet": optFlag, "progress": optFlag, "version": optValue, "all-progress": optFlag,
	"all-progress-implied": optFlag,
}}

// gitRead is the words of the git subcommand at index i of w's segment read
// by its gitOpts as git's parse-options reads them. loose is whether a word
// the line does not fix, or a glob, stands before `--` where it is not an
// option's value: it may expand to an option word.
func (w writerArgs) gitRead(i int) (r optRead, loose bool) {
	toks := w.s.tokens
	r = readOpts(w.s, i+1, gitOpts[toks[i]])
	for _, j := range r.operands {
		if r.end > 0 && j > r.end {
			break
		}
		if t := toks[j]; t != "" && (t[0] == unknownMark || t[0] == varMark) || w.s.globAt(j) {
			loose = true
		}
	}
	return r, loose
}

// gitSub is where the subcommand of the git command w reads stands, after
// git's own options, and whether those options move the working tree
// somewhere the line does not fix (-C, --work-tree, --git-dir, or an option
// word the line does not fix).
func (w writerArgs) gitSub() (int, bool) {
	toks := w.s.tokens
	i := w.from
	elsewhere := false
	for ; i < len(toks) && strings.HasPrefix(toks[i], "-"); i++ {
		switch t := toks[i]; {
		case isUnknown(t) || strings.ContainsRune(t, varMark):
			elsewhere = true
		case t == "-C" || t == "--work-tree" || t == "--git-dir":
			elsewhere = true
			i++
		case t == "-c" || t == "--namespace":
			i++
		case strings.HasPrefix(t, "--work-tree=") || strings.HasPrefix(t, "--git-dir="):
			elsewhere = true
		}
	}
	return i, elsewhere
}

// gitRoot is the working tree a git command runs on: the session's own when
// the command runs inside it and nothing moves it elsewhere, else the root of
// the filesystem.
func (w writerArgs) gitRoot(elsewhere bool) string {
	if d := w.rc.files.dir; d != "" && !elsewhere && w.st.dirOK {
		if _, in := fsutil.RelWithin(d, w.st.dir, approveFoldsCase()); in {
			return d
		}
	}
	return "/"
}

// applyWritesNoFile reports whether git apply, read as r, writes no
// working-tree file: with --cached (the index alone), or with --check,
// --stat, --numstat or --summary, which turn the apply off, and no option
// beyond those and the ones that change only how the patch is read.
func applyWritesNoFile(r optRead) bool {
	return r.only("pCRvqz", "cached", "check", "stat", "numstat", "summary", "exclude", "include",
		"reverse", "unidiff-zero", "verbose", "quiet", "whitespace", "ignore-space-change",
		"ignore-whitespace", "recount", "inaccurate-eof", "allow-empty", "directory", "binary",
		"allow-binary-replacement", "build-fake-ancestor") &&
		(r.has(0, "cached") || r.has(0, "check") || r.has(0, "stat") || r.has(0, "numstat") || r.has(0, "summary"))
}

// amWritesNoFile reports whether git am, read as r, writes no working-tree
// file: it only shows the patch it stopped at, or forgets the session.
func amWritesNoFile(r optRead) bool {
	return r.only("", "show-current-patch", "quit") && (r.has(0, "show-current-patch") || r.has(0, "quit"))
}

// applyTargets is what git apply or git am (name), whose words r reads,
// writes, in the working tree at root, which the guard does not read from the
// patch: in a stage, a tree of names the line does not fix at root, or below
// the directory --directory names there; with --unsafe-paths, any path at all;
// with --build-fake-ancestor, the index file it names. An option word the line
// does not fix may be either, and so, in a stage, may a word that names no
// option, so neither is placed. Outside a stage, git itself keeps a plain
// apply inside the working tree and out of its git directory.
func (w writerArgs) applyTargets(name string, r optRead, loose bool, root string, stage bool) []writeTarget {
	if r.unsure || stage && (loose || r.unknown) {
		return []writeTarget{whyUnplaced("this runs git "+name+" with an option word the line does not fix, or one git "+name+" does not have, which may be --unsafe-paths or --directory, so the guard cannot say where the patch's files land.",
			"spell out each option of git "+name+" in full.")}
	}
	if r.has(0, "unsafe-paths") {
		return []writeTarget{whyUnplaced("this runs git "+name+" with --unsafe-paths, which writes each path its patch names wherever it leads, outside the working tree as well, and the guard does not read the patch, so it cannot say which files it writes.",
			"drop --unsafe-paths, so git "+name+" writes only inside the working tree.")}
	}
	var out []writeTarget
	if v, ok := r.last(0, "build-fake-ancestor"); ok {
		out = append(out, w.valueOf(v))
	}
	if v, ok := r.last(0, "directory"); ok {
		word, ok := w.wordOf(v)
		if !ok {
			return append(out, unplacedWrite(""))
		}
		return append(out, asTreeIn(writeTarget{path: filepath.Join(root, word), ok: true, wrote: true}))
	}
	if stage && !(name == "apply" && applyWritesNoFile(r)) && !(name == "am" && amWritesNoFile(r)) {
		out = append(out, whyUnplaced("this runs git "+name+" in a drive's stage, which writes each path its patch names anywhere in the working tree, the files that decide how the lane is judged among them, and the guard does not read the patch, so it cannot show the patch leaves those files alone.",
			"apply the patch with --directory naming the directory its paths go in, check it with --check or --stat, or write the files it changes directly."))
	}
	return out
}

// gitTrees is where the git command w reads may write working-tree files
// from a patch or from the object store, any of which may be a link: the
// working tree it runs in (whole) for apply, am, a stash applied or pushed,
// a reset that is not one of soft, mixed or patch alone, merge, cherry-pick,
// revert, rebase, pull, a branch switched to, checkout-index, read-tree with
// any option but those that read into the index alone, bisect,
// sparse-checkout and submodule; for checkout and restore, each path they name
// after their options (checkout's first word, with no --, may be a branch, and
// so counts as the whole tree too; with a --, only the paths after it). A
// subcommand git does not document counts as the whole tree where it may be an
// alias (gitExtension). A branch made at HEAD with no start point (checkout -b,
// -B or --orphan, switch -c, -C or --orphan), restore --staged without
// --worktree, the stash, submodule and bisect commands that only list or show,
// and apply and am as applyWritesNoFile and amWritesNoFile read them write no
// file, each only with the options its allowlist names. The working tree is
// the session's own when the command runs inside it, else the root of the
// filesystem; with --work-tree or -C, or an option word the line does not fix,
// it is not fixed, read as the root too; a subcommand, a path or an option
// word the line does not fix counts as the whole tree. A tree is returned
// with ok set; nothing for a command that writes no working-tree file.
func (w writerArgs) gitTrees() []writeTarget {
	toks := w.s.tokens
	i, elsewhere := w.gitSub()
	if i >= len(toks) {
		return nil
	}
	whole := []writeTarget{{path: w.gitRoot(elsewhere), ok: true, wrote: true, tree: true}}
	name := toks[i]
	if isUnknown(name) || strings.ContainsRune(name, varMark) {
		return whole
	}
	if !gitSubcommands[name] {
		if w.rc.files.stageHome != "" {
			if _, ok := w.gitExtension(i); ok {
				return nil // a program git runs, judged as any program is
			}
		}
		return whole
	}
	r, loose := w.gitRead(i)
	first := ""
	if i+1 < len(toks) {
		first = toks[i+1]
	}
	switch name {
	case "apply":
		if applyWritesNoFile(r) && !loose {
			return nil
		}
		return whole
	case "am":
		if amWritesNoFile(r) && !loose {
			return nil
		}
		return whole
	case "stash":
		switch first {
		case "list":
			if w.plainFrom(i+2, "--oneline", "--stat", "-p", "--patch") {
				return nil
			}
		case "show":
			if w.plainFrom(i+2, "-p", "--patch", "--stat", "-u", "--include-untracked", "--only-untracked",
				"--name-only", "--name-status", "--numstat") {
				return nil
			}
		case "create":
			return nil // its words are the message
		case "store":
			if w.plainFrom(i+2, "-m", "--message", "-q", "--quiet") {
				return nil
			}
		case "drop", "clear":
			if w.plainFrom(i+2, "-q", "--quiet") {
				return nil
			}
		}
		return whole
	case "reset":
		if r.only("qpNU", "quiet", "no-quiet", "no-refresh", "refresh", "mixed", "soft", "patch", "no-patch",
			"unified", "inter-hunk-context", "intent-to-add", "no-intent-to-add", "pathspec-from-file",
			"pathspec-file-nul", "no-pathspec-file-nul") && !loose {
			return nil
		}
		return whole
	case "read-tree":
		if r.only("vmniq", "empty", "no-empty", "verbose", "no-verbose", "trivial", "no-trivial", "aggressive",
			"no-aggressive", "reset", "no-reset", "prefix", "index-output", "exclude-per-directory", "dry-run",
			"no-dry-run", "no-sparse-checkout", "sparse-checkout", "quiet", "no-quiet") && !loose {
			return nil
		}
		return whole
	case "submodule":
		switch first {
		case "status":
			if w.plainFrom(i+2, "--cached", "--recursive", "-q", "--quiet") {
				return nil
			}
		case "summary":
			if w.plainFrom(i+2, "--cached", "--files", "-n", "--summary-limit", "--for-status", "-q", "--quiet") {
				return nil
			}
		}
		return whole
	case "bisect":
		switch first {
		case "log":
			if w.plainFrom(i + 2) {
				return nil
			}
		case "visualize", "view":
			if w.plainFrom(i+2, "--stat", "-p", "--patch", "--oneline") {
				return nil
			}
		}
		return whole
	case "switch":
		// a branch made at HEAD, with no start point, changes no file
		create := r.has('c', "create") || r.has('C', "force-create") || r.has(0, "orphan")
		if create && len(r.operands) == 0 && !loose && r.only("cCqt", "create", "force-create", "orphan", "quiet",
			"no-quiet", "progress", "no-progress", "track", "no-track", "guess", "no-guess") {
			return nil
		}
		return whole
	case "merge", "cherry-pick", "revert", "rebase", "pull", "checkout-index", "sparse-checkout":
		return whole
	case "checkout", "restore":
		if loose || r.unsure || r.unknown {
			return whole
		}
		if name == "restore" && r.has('S', "staged") && r.only("sSqpU", "source", "staged", "quiet", "no-quiet",
			"progress", "no-progress", "patch", "unified", "inter-hunk-context", "pathspec-from-file",
			"pathspec-file-nul") {
			return nil // the index alone
		}
		if name == "checkout" && (r.has('b', "") || r.has('B', "") || r.has(0, "orphan")) && len(r.operands) == 0 &&
			r.only("bBqtl", "orphan", "quiet", "no-quiet", "progress", "no-progress", "track", "no-track",
				"guess", "no-guess") {
			return nil // a branch made at HEAD, with no start point
		}
		if r.has(0, "pathspec-from-file") || (name == "checkout" && r.end == 0) {
			return whole
		}
		var out []writeTarget
		for _, j := range r.operands {
			if name == "checkout" && j < r.end {
				continue // the tree-ish the paths come from
			}
			t := w.at(j)
			if !t.ok || pathspecMagic(toks[j]) {
				return whole
			}
			t.tree = true
			out = append(out, t)
		}
		if len(out) == 0 {
			return whole
		}
		return out
	}
	return nil
}

// plainFrom reports whether every word of w's segment from index from is one
// the line fixes, and is either no option (it opens with no dash) or one of
// allowed, spelled whole.
func (w writerArgs) plainFrom(from int, allowed ...string) bool {
	toks := w.s.tokens
	for j := from; j < len(toks); j++ {
		t := toks[j]
		if isUnknown(t) || strings.ContainsRune(t, varMark) || w.s.globAt(j) {
			return false
		}
		if strings.HasPrefix(t, "-") && !containsString(allowed, t) {
			return false
		}
	}
	return true
}

// gitTargets is what a git command writes at paths its words name: the paths
// `git checkout` and `git restore` write, the directory `git clone` and `git
// init` make, where `git mv` moves its sources (as mv does), what git apply
// and git am write (applyTargets), and the files git's own options name: the
// --output of any command that shows a diff or a log, format-patch's and
// bugreport's output directory, archive's output, the file `bundle create`
// writes, read-tree's --index-output (in the working tree's root), the files
// interpret-trailers --in-place and merge-file rewrite, the marks
// fast-export and fast-import write, index-pack's -o, mailsplit's directory,
// mailinfo's two files and credential-store's --file. Every `git -C`, whatever
// its subcommand, is a write the guard does not place: -C moves git to a
// directory the guard does not follow, where a subcommand, its options, a
// configuration value on the line or an alias can change any file. In a stage,
// a subcommand git does not document is one the guard cannot place where it
// may be an alias (gitExtension), which is asked only there; and a path
// checkout or restore takes as a pathspec git matches itself (pathspecMagic)
// is not placed.
func (w writerArgs) gitTargets() []writeTarget {
	toks := w.s.tokens
	i, elsewhere := w.gitSub()
	for j := w.from; j < i; j++ {
		if toks[j] == "-C" {
			return []writeTarget{unplacedWrite("")}
		}
	}
	if i >= len(toks) {
		return nil
	}
	name := toks[i]
	stage := w.rc.files.stageHome != ""
	if isUnknown(name) || strings.ContainsRune(name, varMark) {
		if stage {
			return []writeTarget{whyUnplaced("this runs a git subcommand the line does not spell out, which may be an alias for any command, writing where the guard cannot say.",
				"spell the git subcommand out in full.")}
		}
		return nil
	}
	if !gitSubcommands[name] {
		if !stage {
			return nil
		}
		if why, ok := w.gitExtension(i); !ok {
			return []writeTarget{why}
		}
		return nil
	}
	sub := w
	sub.from = i + 1
	r, loose := w.gitRead(i)
	out := w.gitOutputs(i, r)
	root := w.gitRoot(elsewhere)
	switch name {
	case "apply", "am":
		out = append(out, sub.applyTargets(name, r, loose, root, stage)...)
	case "mv":
		out = append(out, sub.sourceLandings(r, 1)...)
	case "checkout", "restore":
		out = append(out, sub.operands([]string{"-b", "-B", "--orphan", "-s", "--source", "--conflict", "--pathspec-from-file"}, nil)...)
		if stage {
			for _, j := range r.operands {
				if name == "checkout" && j < r.end {
					continue // the tree-ish the paths come from
				}
				if pathspecMagic(toks[j]) {
					out = append(out, pathspecUnplaced(name))
				}
			}
		}
	case "read-tree":
		if v, ok := r.last(0, "index-output"); ok {
			word, ok := w.wordOf(v)
			switch {
			case !ok || root == "/" && !filepath.IsAbs(word):
				out = append(out, unplacedWrite(""))
			case filepath.IsAbs(word):
				out = append(out, writeTarget{path: filepath.Clean(word), ok: true, wrote: true})
			default:
				out = append(out, writeTarget{path: filepath.Join(root, word), ok: true, wrote: true})
			}
		}
	case "format-patch":
		if v, ok := r.last('o', "output-directory"); ok {
			out = append(out, asTreeIn(w.valueOf(v)))
		}
	case "bugreport", "diagnose":
		if v, ok := r.last('o', "output-directory"); ok {
			out = append(out, asTreeIn(w.valueOf(v)))
		}
	case "archive":
		if v, ok := r.last('o', "output"); ok {
			out = append(out, w.valueOf(v))
		}
	case "bundle":
		if i+1 < len(toks) && toks[i+1] == "create" {
			if br := readOpts(w.s, i+2, gitBundleCreateOpts); len(br.operands) > 0 {
				out = append(out, w.at(br.operands[0]))
			}
		}
	case "interpret-trailers":
		if r.has(0, "in-place") {
			for _, j := range r.operands {
				out = append(out, w.at(j))
			}
		}
	case "merge-file":
		if !r.has('p', "stdout") && len(r.operands) > 0 {
			out = append(out, w.at(r.operands[0]))
		}
	case "fast-export", "fast-import":
		out = append(out, sub.flagValues([]string{"--export-marks", "--export-pack-edges"})...)
	case "index-pack":
		out = append(out, sub.flagValues([]string{"-o"})...)
	case "credential-store":
		out = append(out, sub.flagValues([]string{"--file"})...)
	case "mailsplit":
		for j := i + 1; j < len(toks); j++ {
			if strings.HasPrefix(toks[j], "-o") {
				out = append(out, asTreeIn(w.valueAt(j, len("-o"))))
			}
		}
	case "mailinfo":
		mr := readOpts(w.s, i+1, optSpec{})
		for k, j := range mr.operands {
			if k < 2 {
				out = append(out, w.at(j))
			}
		}
	case "init":
		for _, j := range r.operands {
			t := w.at(j)
			t.tree = t.ok
			out = append(out, t)
		}
		if v, ok := r.last(0, "separate-git-dir"); ok {
			t := w.valueOf(v)
			t.tree = t.ok
			out = append(out, t)
		}
	case "clone":
		idx := sub.operandIdx([]string{"-b", "--branch", "-o", "--origin", "--depth", "-c", "--config",
			"--reference", "--separate-git-dir", "-u", "--upload-pack", "--template", "--filter", "-j", "--jobs"})
		switch {
		case len(idx) >= 2:
			t := w.at(idx[1])
			t.tree = t.ok
			out = append(out, t)
		case len(idx) == 1:
			repo, ok := fixedToken(w.s, idx[0])
			if !ok {
				return append(out, unplacedWrite(""))
			}
			base := strings.TrimSuffix(path.Base(strings.TrimRight(repo, "/")), ".git")
			if k := strings.LastIndexByte(base, ':'); k >= 0 {
				base = base[k+1:]
			}
			t := writeAt(w.rc, w.st, base, false)
			t.tree = t.ok
			out = append(out, t)
		}
	}
	return out
}

// gitOutputs is the file each --output among the words of the git command
// whose subcommand stands at index i names, before the `--` r found: the
// diff option every command that shows a diff or a log reads, which opens
// the file as it reads its options, whatever the command goes on to do.
func (w writerArgs) gitOutputs(i int, r optRead) []writeTarget {
	toks := w.s.tokens
	end := len(toks)
	if r.end > 0 {
		end = r.end
	}
	var out []writeTarget
	for j := i + 1; j < end; j++ {
		switch t := toks[j]; {
		case t == "--output" && j+1 < len(toks):
			out = append(out, w.at(j+1))
			j++
		case strings.HasPrefix(t, "--output="):
			out = append(out, w.valueAt(j, len("--output=")))
		}
	}
	return out
}

// gitExtension reads the subcommand at index i of w's segment, one git does
// not document. git 2.52 runs a git-<name> program it finds on PATH before it
// looks for an alias of that name, so the subcommand is that program, judged
// as any program is (ok), where one is found outside the working tree and the
// session's own and temporary directories, where the session could have
// written it, and the configuration git reads does not define alias.<name>.
// Otherwise it may be an alias, which can run any command and write any file,
// and why is the write the guard cannot place, with its reason.
func (w writerArgs) gitExtension(i int) (why writeTarget, ok bool) {
	name := w.s.tokens[i]
	shown := termsafe.Sanitize(name)
	alias := func(reason, remedy string) (writeTarget, bool) {
		return whyUnplaced("this runs `git "+shown+"`, which git does not document: "+reason, remedy), false
	}
	spell := "spell out the git command the alias stands for, or run the program by its full path."
	env := w.st.env(w.rc, w.s, w.from-1)
	pv, set := env["PATH"]
	p, found, unfixed := searchPath(w.rc, w.st, pv, set, "git-"+name, false, set, nil, nil)
	switch {
	case unfixed:
		return alias("the PATH the line sets is not one the guard can follow to the git-"+shown+" program git would run, so it may run an alias of that name, which can run any command and write any file.",
			"set PATH to directories spelled out in full, or run the program by its full path.")
	case !found:
		return alias("no git-"+shown+" program is on PATH, so git runs it as an alias, which can run any command and write any file, and the guard does not read what an alias does.", spell)
	}
	resolved := p
	if r, err := filepath.EvalSymlinks(p); err == nil {
		resolved = r
	}
	f := w.rc.files
	writable := append(append([]string{f.dir, f.stageHome}, f.runDirs...), approveTempDirs()...)
	for _, d := range writable {
		if d != "" && filepath.IsAbs(d) && tools.WithinTree(p, resolved, filepath.Clean(d)) {
			return alias("the git-"+shown+" program PATH finds ("+termsafe.Sanitize(fsutil.DisplayPath(p))+") lies where this session can write it, so what it does is not a program the guard can read.",
				"run the git command it stands for, spelled out, or install git-"+shown+" outside the working tree and the run's own directories.")
		}
	}
	root := f.dir
	if w.st.dirOK && w.st.dir != "" {
		root = w.st.dir
	}
	if _, err := os.Stat(root); err != nil || !filepath.IsAbs(root) {
		return alias("the guard cannot say which directory git reads its configuration in, so it cannot say whether alias."+shown+" is defined.", spell)
	}
	ctx, cancel := context.WithTimeout(context.Background(), aliasReadBudget)
	defer cancel()
	defined, err := gitutil.AliasDefinedContext(ctx, root, f.stageHome, name)
	if errors.Is(err, context.DeadlineExceeded) {
		return alias(fmt.Sprintf("git did not answer within %s when asked whether its configuration defines alias.%s, so the guard cannot show it is the git-%s program and not an alias (a configuration that includes a file which never finishes reading, such as a named pipe, holds git this way).", aliasReadBudget, shown, shown),
			"check that git answers here (git config --show-origin --get alias."+shown+"), then run the line again.")
	}
	if err != nil {
		return alias("git could not say whether its configuration defines alias."+shown+" ("+termsafe.Sanitize(fsutil.RedactHome(err.Error()))+"), so the guard cannot show it is the git-"+shown+" program and not an alias.",
			"check that git can read this repository's settings and your own (git config --show-origin --get alias."+shown+"), then run the line again.")
	}
	if defined {
		return alias("the configuration git reads here defines it as an alias (alias."+shown+"), which can run any command and write any file, and the guard does not read what an alias does.", spell)
	}
	return writeTarget{}, true
}

// aliasReadBudget bounds gitExtension's read of the configuration, as the
// guard hook's own git budget bounds its first git calls: git answers in
// milliseconds, and one that has not answered by then is waiting on
// something that may never come. A variable, so a test can shorten it.
var aliasReadBudget = 5 * time.Second

// pathspecMagic reports whether word is a pathspec git reads other than as a
// path (gitglossary(7)): magic that opens with ':' (`:/name` from the
// working tree's top, `:(glob)…`, `:!…`), or a wildcard git matches against
// its own index, which names files the word does not.
func pathspecMagic(word string) bool {
	return strings.HasPrefix(word, ":") || strings.ContainsAny(word, "*?[")
}

// pathspecUnplaced is a path git command name takes as a pathspec that names
// files the word does not spell, which the guard cannot place.
func pathspecUnplaced(name string) writeTarget {
	return whyUnplaced("this runs git "+name+" with a pathspec git matches itself (a leading ':' or a wildcard), which can name any file in the working tree, so the guard cannot say which files it reaches.",
		"name each path in full, relative to the directory the command runs in.")
}

// writeWhy is the reason a write the guard cannot place is refused, and the
// remedy, where a writer knows them better than the general refusal does.
type writeWhy struct{ reason, remedy string }

// whyUnplaced is a write the guard cannot place, refused with reason and
// remedy.
func whyUnplaced(reason, remedy string) writeTarget {
	t := unplacedWrite("")
	t.why = &writeWhy{reason: reason, remedy: remedy}
	return t
}
