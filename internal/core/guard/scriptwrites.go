package guard

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The files a command writes (adr-2610091150447054 decision 6). A script run
// after an earlier segment on the same line writes it, or a directory above
// it, is not the file the guard reads at check time, so the run blocks; a
// write the guard cannot place warns. The writers are an enumeration in the
// sense of adr-42 decision 5 — incomplete by design, extended over time, and
// named in commands/guard.md — beside every redirection, which is read
// whatever the command.

// writesOf is every file s writes: its redirections that open a file for
// writing, and the targets of a listed writer.
func writesOf(rc *readCtx, s segment, st *shellState) []writeTarget {
	var out []writeTarget
	for _, r := range s.redirects {
		switch r.op {
		case "<", "<>":
			continue // read, or opened read-write without a byte changed
		case "<&":
			// A duplicate of an input descriptor, or a redirection error
			// where the word names no descriptor: no file is written.
			continue
		case ">&":
			if t := r.target.text; r.target.ok && (t == "-" || isAllDigits([]byte(t))) {
				continue // a descriptor, not a file
			}
		}
		if !r.target.ok {
			out = append(out, unplacedWrite(r.target.base))
			continue
		}
		out = append(out, writeAt(rc, st, r.target.text, r.target.tilde))
	}
	return append(out, writerTargets(rc, s, st)...)
}

// unplacedWrite is a write the guard does not place: base is its path's last
// component where the line fixes it, else "", a write that can land on any
// name.
func unplacedWrite(base string) writeTarget {
	return writeTarget{base: base, wrote: true}
}

// writeAt is a write to word, resolved as the shell opens it. Where the line
// leaves the working directory unknown, a relative word is not placed, and
// is marked as written there (rel), with the name it writes there and the
// directory's identity (cwdName).
func writeAt(rc *readCtx, st *shellState, word string, tilde bool) writeTarget {
	p, ok := st.resolve(rc, word, tilde)
	w := writeTarget{path: p, ok: ok, wrote: true}
	if !ok && !tilde && st.relativeUnknown(word) {
		w.rel, w.base = true, filepath.Base(word)
		w.name, w.gen = cwdName(word), st.dirGen
	}
	return w
}

// writerTargets is what the listed writers among s's commands write.
func writerTargets(rc *readCtx, s segment, st *shellState) []writeTarget {
	var out []writeTarget
	for _, a := range commandSites(s) {
		out = append(out, siteTargets(rc, s, st, a)...)
	}
	return out
}

// siteTargets is what the command at a writes, when it is a listed writer.
func siteTargets(rc *readCtx, s segment, st *shellState, a arrival) []writeTarget {
	var out []writeTarget
	tok := s.tokens[a.idx]
	if isUnknown(tok) {
		return nil
	}
	w := writerArgs{rc: rc, s: s, st: st, from: a.idx + 1}
	switch strings.ToLower(path.Base(tok)) {
	case "tee":
		out = append(out, w.operands(nil, nil)...)
	case "cp", "mv", "install":
		w.s = xargsView(s, a.idx)
		out = append(out, w.copyTarget(strings.ToLower(path.Base(tok)))...)
	case "ln":
		w.s = xargsView(s, a.idx)
		out = append(out, w.lnTarget()...)
	case "dd":
		for i := w.from; i < len(s.tokens); i++ {
			if strings.HasPrefix(s.tokens[i], "of=") {
				out = append(out, w.valueAt(i, len("of=")))
			}
		}
	case "curl":
		out = append(out, w.curlTargets()...)
	case "wget":
		out = append(out, w.flagValues([]string{"-O", "--output-document"})...)
	case "sed":
		out = append(out, w.sedTargets()...)
	case "patch":
		out = append(out, w.patchTargets()...)
	case "git":
		out = append(out, w.gitTargets()...)
	case "tar":
		out = append(out, w.tarTargets()...)
	case "unzip":
		out = append(out, w.unzipTargets()...)
	case "rsync":
		out = append(out, w.rsyncTarget()...)
	case "find":
		out = append(out, findRuns(rc, s, st, a.idx+1)...)
	case "go":
		out = append(out, w.goBuildTarget()...)
	case "sort":
		out = append(out, w.flagValues([]string{"-o", "--output"})...)
	case "ditto":
		for _, d := range w.lastOf(w.operandIdx([]string{"--arch", "--bom", "--zlibCompressionLevel"}), 2) {
			out = append(out, asTreeIn(d))
		}
	case "split":
		out = append(out, w.splitTarget()...)
	case "uniq":
		out = append(out, w.secondOf(w.operandIdx([]string{"-f", "-s", "-w", "--skip-fields", "--skip-chars", "--check-chars"}))...)
	}
	return out
}

// gitTrees is where the git command w reads may write working-tree files
// from a patch or from the object store, any of which may be a link: the
// working tree it runs in (whole) for apply, am, a stash applied or pushed,
// a hard, merge or keep reset, merge, cherry-pick, revert, rebase, pull, a
// branch switched to, checkout-index, read-tree -u, bisect, sparse-checkout
// and submodule; for checkout and restore, each path they name after their
// options (checkout's first word, before any --, may be a branch, and so
// counts as the whole tree too). The working tree is the session's own when
// the command runs inside it, else the root of the filesystem; with --work-tree
// or -C, or an option word the line does not fix, it is not fixed, read as
// the root too; a subcommand or a path the line does not fix counts as the
// whole tree. A tree is returned with ok set; nothing for a command that
// writes no working-tree file.
func (w writerArgs) gitTrees() []writeTarget {
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
	if i >= len(toks) {
		return nil
	}
	root := "/"
	if d := w.rc.files.dir; d != "" && !elsewhere && w.st.dirOK {
		if _, in := fsutil.RelWithin(d, w.st.dir, approveFoldsCase()); in {
			root = d
		}
	}
	whole := []writeTarget{{path: root, ok: true, wrote: true, tree: true}}
	rest := toks[i+1:]
	has := func(words ...string) bool {
		for _, t := range rest {
			for _, x := range words {
				if t == x || strings.HasPrefix(t, x+"=") {
					return true
				}
			}
		}
		return false
	}
	if isUnknown(toks[i]) || strings.ContainsRune(toks[i], varMark) {
		return whole
	}
	switch toks[i] {
	case "apply":
		if has("--cached") || (has("--check", "--stat", "--numstat", "--summary") && !has("--apply")) {
			return nil
		}
		return whole
	case "stash":
		if len(rest) > 0 && (rest[0] == "list" || rest[0] == "show" || rest[0] == "create" || rest[0] == "store" || rest[0] == "drop" || rest[0] == "clear") {
			return nil
		}
		return whole
	case "reset":
		if has("--hard", "--merge", "--keep") {
			return whole
		}
		return nil
	case "read-tree":
		if has("-u") {
			return whole
		}
		return nil
	case "am", "merge", "cherry-pick", "revert", "rebase", "pull", "switch", "checkout-index", "bisect", "sparse-checkout", "submodule":
		return whole
	case "checkout", "restore":
		sub := w
		sub.from = i + 1
		idx := sub.operandIdx([]string{"-b", "-B", "--orphan", "-s", "--source", "--conflict", "--pathspec-from-file"})
		dashes := false
		for _, t := range rest {
			if t == "--" {
				dashes = true
			}
		}
		if len(idx) == 0 || has("--pathspec-from-file") || (toks[i] == "checkout" && !dashes) {
			return whole
		}
		var out []writeTarget
		for _, j := range idx {
			if t := w.at(j); t.ok {
				t.tree = true
				out = append(out, t)
			} else {
				return whole
			}
		}
		return out
	}
	return nil
}

// rsyncTarget is the destination rsync writes into: the last of two or more
// operands, unless it names another host, which no local check judges.
func (w writerArgs) rsyncTarget() []writeTarget {
	idx := w.operandIdx(rsyncValueFlags)
	if len(idx) < 2 {
		return nil
	}
	last := idx[len(idx)-1]
	if tok, ok := fixedToken(w.s, last); ok && remoteSpec(tok) {
		return nil
	}
	return []writeTarget{asTreeIn(w.at(last))}
}

// asTreeIn is d as the directory a tree of names no line fixes lands in: an
// archive unpacked there, or a directory's contents copied into it.
func asTreeIn(d writeTarget) writeTarget {
	d.tree, d.in = true, true
	return d
}

// lastOf is the last operand of idx when there are at least min of them: the
// destination of a copy.
func (w writerArgs) lastOf(idx []int, min int) []writeTarget {
	if len(idx) < min {
		return nil
	}
	return []writeTarget{w.at(idx[len(idx)-1])}
}

// secondOf is the second operand of idx, the output file of a filter that
// takes `input output` (uniq), where there is one; `-` is standard output.
func (w writerArgs) secondOf(idx []int) []writeTarget {
	if len(idx) < 2 || w.s.tokens[idx[1]] == "-" {
		return nil
	}
	return []writeTarget{w.at(idx[1])}
}

// splitTarget is the prefix split writes its pieces under: its second
// operand, else `x` in the directory it runs in. A piece is the prefix with
// a suffix after it, so the prefix is judged as the path written.
func (w writerArgs) splitTarget() []writeTarget {
	idx := w.operandIdx([]string{"-a", "-b", "-C", "-l", "-n", "-p", "-t", "--suffix-length", "--bytes",
		"--line-bytes", "--lines", "--number", "--separator", "--additional-suffix"})
	if len(idx) >= 2 {
		return []writeTarget{w.at(idx[1])}
	}
	return []writeTarget{writeAt(w.rc, w.st, "x", false)}
}

// goBuildTarget is the file or directory `go build -o` writes, in Go's own
// flag spellings: -o or --o, with the value as the next word or after `=`.
func (w writerArgs) goBuildTarget() []writeTarget {
	toks := w.s.tokens
	i := w.from
	for i+1 < len(toks) && toks[i] == "-C" {
		i += 2 // go -C dir: another directory, which the reading places the same way
	}
	if i >= len(toks) || toks[i] != "build" {
		return nil
	}
	var out []writeTarget
	for j := i + 1; j < len(toks); j++ {
		t := toks[j]
		switch {
		case t == "--":
			return out
		case t == "-o" || t == "--o":
			if j+1 < len(toks) {
				out = append(out, w.at(j+1))
			}
			j++
		case strings.HasPrefix(t, "-o="):
			out = append(out, w.valueAt(j, len("-o=")))
		case strings.HasPrefix(t, "--o="):
			out = append(out, w.valueAt(j, len("--o=")))
		}
	}
	return out
}

// findRuns is what the commands a find runs write: each -exec, -execdir,
// -ok or -okdir command, up to its `;` (or a `+` after `{}`), read as a
// command of its own with every `{}` a name the line does not fix, since find
// puts each file it finds there. A command -execdir or -okdir runs runs in
// the found file's directory, which the line does not fix.
func findRuns(rc *readCtx, s segment, st *shellState, from int) []writeTarget {
	var out []writeTarget
	for _, run := range findActions(s, from) {
		in := st
		if run.elsewhere {
			in = st.unresolved()
		}
		out = append(out, writerTargets(rc, run.seg, in)...)
	}
	return out
}

// findAction is one command a find runs: its words, with each found path a word no line
// fixes, and whether it runs somewhere the reading does not place.
type findAction struct {
	seg       segment
	elsewhere bool
}

// findActions is what a find whose arguments start at from runs.
func findActions(s segment, from int) []findAction {
	var out []findAction
	for i := from; i < len(s.tokens); i++ {
		action := s.tokens[i]
		// An unknown word may be -exec (unknown.go), so what follows it is
		// read as the command find runs.
		if action != "-exec" && action != "-execdir" && action != "-ok" && action != "-okdir" && !isUnknown(action) {
			continue
		}
		end := i + 1
		for end < len(s.tokens) && s.tokens[end] != ";" && (s.tokens[end] != "+" || s.tokens[end-1] != "{}") {
			end++
		}
		if end > i+1 {
			toks := make([]string, 0, end-i-1)
			for _, t := range s.tokens[i+1 : end] {
				toks = append(toks, strings.ReplaceAll(t, "{}", unknownText))
			}
			out = append(out, findAction{
				seg:       segment{tokens: toks, chain: s.chain, globbed: s.globSlice(i+1, end)},
				elsewhere: action == "-execdir" || action == "-okdir" || isUnknown(action),
			})
		}
		i = end
	}
	return out
}

// writerArgs reads one writer's arguments, from index from of s.
type writerArgs struct {
	rc   *readCtx
	s    segment
	st   *shellState
	from int
}

func (w writerArgs) at(i int) writeTarget {
	tok, ok := fixedToken(w.s, i)
	if !ok {
		if i >= 0 && i < len(w.s.tokens) && !w.s.globAt(i) {
			return unplacedWrite(tokenBase(w.s.tokens[i]))
		}
		return unplacedWrite("")
	}
	return writeAt(w.rc, w.st, tok, strings.HasPrefix(tok, "~"))
}

// valueAt is the path that token i carries after its first skip bytes.
func (w writerArgs) valueAt(i, skip int) writeTarget {
	tok := w.s.tokens[i]
	if isUnknown(tok) || w.s.globAt(i) || strings.ContainsRune(tok, varMark) {
		return unplacedWrite("")
	}
	v := tok[skip:]
	return writeAt(w.rc, w.st, v, strings.HasPrefix(v, "~"))
}

// tokenBase is the last path component of a token the line does not fix,
// where that component is literal: the tokenizer has already removed the
// quotes, and marks a substitution's output or a variable's value in place.
func tokenBase(tok string) string {
	i := strings.LastIndexByte(tok, '/')
	if i < 0 {
		return ""
	}
	rest := tok[i+1:]
	if rest == "" || rest == "." || rest == ".." || isUnknown(rest) || strings.ContainsRune(rest, varMark) {
		return ""
	}
	return rest
}

// operands is the operand indexes after w.from, stepping options; an option
// in valueFlags steps its value too. A `--` ends the options.
func (w writerArgs) operandIdx(valueFlags []string) []int {
	var idx []int
	opts := true
	for i := w.from; i < len(w.s.tokens); i++ {
		t := w.s.tokens[i]
		switch {
		case opts && t == "--":
			opts = false
		case opts && strings.HasPrefix(t, "-") && t != "-":
			if containsString(valueFlags, t) {
				i++
			}
		default:
			idx = append(idx, i)
		}
	}
	return idx
}

func (w writerArgs) operands(valueFlags []string, skip func(int) bool) []writeTarget {
	var out []writeTarget
	for _, i := range w.operandIdx(valueFlags) {
		if skip != nil && skip(i) {
			continue
		}
		out = append(out, w.at(i))
	}
	return out
}

// flagValues is the value of each named option: `-o f`, `-of`, `--output f`,
// `--output=f`. A value of `-` is standard output.
func (w writerArgs) flagValues(flags []string) []writeTarget {
	var out []writeTarget
	toks := w.s.tokens
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		if t == "--" {
			break
		}
		for _, f := range flags {
			switch {
			case t == f && i+1 < len(toks):
				if toks[i+1] != "-" {
					out = append(out, w.at(i+1))
				}
				i++
			case strings.HasPrefix(f, "--") && strings.HasPrefix(t, f+"="):
				out = append(out, w.valueAt(i, len(f)+1))
			case !strings.HasPrefix(f, "--") && strings.HasPrefix(t, f) && len(t) > len(f) && t[len(f):] != "-":
				out = append(out, w.valueAt(i, len(f)))
			}
		}
	}
	return out
}

// dirTarget is the directory a writer writes into: the named option's value,
// else the directory it runs in.
func (w writerArgs) dirTarget(flags []string) writeTarget {
	if vs := w.flagValues(flags); len(vs) > 0 {
		return vs[len(vs)-1]
	}
	ok := w.st.dirOK && w.st.dir != ""
	return writeTarget{path: w.st.dir, ok: ok, rel: !ok, wrote: true}
}

// copyTarget is what cp, mv or install writes: its destination, and the tree
// each source places (sourceLandings). install -d makes every operand a
// directory.
func (w writerArgs) copyTarget(name string) []writeTarget {
	var out []writeTarget
	for _, r := range readOptsEach(w.s, w.from, copySpec(name)) {
		if v, ok := r.last('M', ""); ok && v.ok && name == "install" {
			out = append(out, w.valueOf(v)) // macOS install's metalog
		}
		if name == "install" && r.has('d', "directory") {
			for _, i := range r.operands {
				out = append(out, w.at(i))
			}
			continue
		}
		out = append(out, w.sourceLandings(r, 1)...)
	}
	return out
}

// lnTarget is the link ln makes: the -t value, else the last of two or more
// operands, and with a single operand a link of the target's own name in the
// directory ln runs in (`ln -s /path/s.sh` writes ./s.sh). Each is read as
// sourceLandings reads a copy, since a link to a directory places its tree.
func (w writerArgs) lnTarget() []writeTarget {
	var out []writeTarget
	for _, r := range readOptsEach(w.s, w.from, lnOpts) {
		if _, ok := r.last('t', "target-directory"); len(r.operands) == 1 && !ok {
			i := r.operands[0]
			tok := w.s.tokens[i]
			if isUnknown(tok) || w.s.globAt(i) || strings.ContainsRune(tok, varMark) {
				out = append(out, unplacedWrite(""))
				continue
			}
			t := writeAt(w.rc, w.st, path.Base(tok), false)
			t.tree = t.ok
			out = append(out, t)
			continue
		}
		out = append(out, w.sourceLandings(r, 2)...)
	}
	return out
}

// copyPlan is where a move, copy or link of operands lands: its destination,
// whether each source lands in it under its own name (into), the sources, and
// whether the destination may also take the source's tree as itself (asWell).
// A source lands in the destination when it is a directory: named by -t, one
// of several sources, a directory standing there now (followed through a
// link, as the commands follow it), or spelled with a trailing slash, which on
// a destination that does not exist yet makes it the source's tree instead;
// unless -T says the destination is never one. Otherwise the source takes the
// destination's name. ok is false where there is no destination.
type copyPlan struct {
	dest         writeTarget
	srcs         []int
	into, asWell bool
	ok           bool
}

func (w writerArgs) planCopy(r optRead, operands []int, min int) copyPlan {
	var p copyPlan
	if v, ok := r.last('t', "target-directory"); ok {
		p.dest, p.into, p.srcs = w.valueOf(v), true, operands
	} else {
		if len(operands) < min {
			return p
		}
		last := operands[len(operands)-1]
		p.dest, p.srcs = w.at(last), operands[:len(operands)-1]
		dir := p.dest.ok && isDir(p.dest.path)
		slash := strings.HasSuffix(w.s.tokens[last], "/")
		p.into = len(p.srcs) > 1 || slash || dir
		p.asWell = slash && !dir && len(p.srcs) <= 1
	}
	if r.has('T', "no-target-directory") {
		p.into = false
	}
	p.ok = true
	return p
}

// sourceLandings is what a command that moves, copies or links its sources
// to a destination writes (cp, mv, install, ln), the -t value or else the
// last of at least min operands, under each reading of its operands
// (optRead.readings): the destination, and the tree each source places there
// (copyPlan). A source whose last name is a glob lands under any name it
// matches; one ending in a slash, `.` or a name no line fixes, under any name
// at all, since the copy may be of a directory's contents. An option word no
// line fixes may be -t with its destination glued on, so the destination is
// not placed; an operand no line fixes may be -t, naming the word after it as
// the destination, which is read as a tree of any names landing there.
func (w writerArgs) sourceLandings(r optRead, min int) []writeTarget {
	var out []writeTarget
	for _, operands := range r.readings() {
		p := w.planCopy(r, operands, min)
		if !p.ok {
			continue
		}
		if !p.dest.ok {
			out = append(out, p.dest)
			continue
		}
		dest := p.dest
		if !p.into || p.asWell {
			dest.tree = true
		}
		out = append(out, dest)
		if r.unsure {
			out = append(out, unplacedWrite(""))
		}
		if p.into {
			for _, i := range p.srcs {
				out = append(out, w.landingIn(p.dest, i))
			}
		}
	}
	for _, i := range r.operands {
		if tok := w.s.tokens[i]; (isUnknown(tok) || strings.ContainsRune(tok, varMark)) && i+1 < len(w.s.tokens) {
			if next := w.at(i + 1); next.ok {
				out = append(out, asTreeIn(next))
			}
		}
	}
	return out
}

// landingIn is the tree source i places in the directory d.
func (w writerArgs) landingIn(d writeTarget, i int) writeTarget {
	tok := w.s.tokens[i]
	land := writeTarget{path: d.path, ok: true, wrote: true, tree: true, in: true}
	if isUnknown(tok) || strings.ContainsRune(tok, varMark) || strings.HasSuffix(tok, "/") {
		return land
	}
	base := path.Base(tok)
	if base == "." || base == ".." || base == "/" {
		return land
	}
	if w.s.globAt(i) {
		land.top = base
		return land
	}
	land.path, land.in = filepath.Join(d.path, base), false
	return land
}

// isDir reports whether p is a directory now, through any link.
func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// patchTargets is what patch writes: nothing on a dry run (--dry-run, or
// Apple patch's -C and --check); the -o file where one is named; else the
// file its first operand names, which it patches alone; else every file the
// patch names, a tree of names no line fixes in the directory it works in;
// and the reject file (-r) and the backups a prefix (-B, -Y) or a suffix
// holding a slash (-z) names. patch changes to the -d directory before it
// opens any file, so a relative name is read inside it. Its options are read
// as getopt reads them (patchOpts).
func (w writerArgs) patchTargets() []writeTarget {
	r := readOpts(w.s, w.from, patchOpts)
	if r.has('C', "dry-run") || r.has(0, "check") {
		return nil
	}
	dir, under := w.dirTarget(nil), false
	if v, ok := r.last('d', "directory"); ok {
		dir, under = w.valueOf(v), true
	}
	at := func(word string, ok bool) writeTarget {
		tilde := strings.HasPrefix(word, "~")
		switch {
		case !ok:
			return unplacedWrite("")
		case !under || filepath.IsAbs(word) || tilde:
			return writeAt(w.rc, w.st, word, tilde)
		case !dir.ok:
			return unplacedWrite("")
		}
		return writeTarget{path: filepath.Join(dir.path, word), ok: true, wrote: true}
	}
	var out []writeTarget
	if v, ok := r.last('r', "reject-file"); ok {
		out = append(out, at(w.wordOf(v)))
	}
	if v, ok := r.last('B', "prefix"); ok {
		word, fixed := w.wordOf(v)
		t := at(word, fixed)
		if t.ok {
			// each backup is the prefix and the patched file's name
			top := filepath.Base(word) + "*"
			if strings.HasSuffix(word, "/") {
				top = ""
			} else {
				t.path = filepath.Dir(t.path)
			}
			t.tree, t.in, t.top = true, true, top
		}
		out = append(out, t)
	}
	if _, ok := r.last('Y', "basename-prefix"); ok {
		out = append(out, asTreeIn(dir))
	}
	if v, ok := r.last('z', "suffix"); ok {
		if word, fixed := w.wordOf(v); !fixed || strings.Contains(word, "/") {
			out = append(out, asTreeIn(dir))
		}
	}
	switch v, ok := r.last('o', "output"); {
	case ok:
		out = append(out, at(w.wordOf(v)))
	case len(r.operands) > 0:
		out = append(out, at(w.wordOf(optVal{idx: r.operands[0], ok: true})))
	default:
		out = append(out, asTreeIn(dir))
	}
	return out
}

// wordOf is the text of the value v carries, and whether the line fixes it.
func (w writerArgs) wordOf(v optVal) (string, bool) {
	if !v.ok || v.idx >= len(w.s.tokens) {
		return "", false
	}
	tok := w.s.tokens[v.idx]
	if isUnknown(tok) || w.s.globAt(v.idx) || strings.ContainsRune(tok, varMark) {
		return "", false
	}
	return tok[v.skip:], true
}

// curlTargets is curl's -o value, and with -O the last path segment of each
// URL it fetches, in the directory it runs in.
func (w writerArgs) curlTargets() []writeTarget {
	out := w.flagValues([]string{"-o", "--output"})
	toks := w.s.tokens
	remote := false
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		if t == "--remote-name" || t == "--remote-name-all" ||
			(isShortCluster(t) && strings.ContainsRune(t[1:], 'O')) {
			remote = true
		}
	}
	if !remote {
		return out
	}
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		if !strings.Contains(t, "://") {
			continue
		}
		if isUnknown(t) || strings.ContainsRune(t, varMark) {
			out = append(out, unplacedWrite(""))
			continue
		}
		u, err := url.Parse(t)
		if err != nil || path.Base(u.Path) == "/" || path.Base(u.Path) == "." {
			out = append(out, unplacedWrite(""))
			continue
		}
		out = append(out, writeAt(w.rc, w.st, path.Base(u.Path), false))
	}
	return out
}

// sedTargets is the files `sed -i` edits in place: every operand after the
// script, or every operand when -e or -f carries the script.
func (w writerArgs) sedTargets() []writeTarget {
	toks := w.s.tokens
	inPlace, scripted := false, false
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "--":
			i = len(toks)
		case strings.HasPrefix(t, "-i") || strings.HasPrefix(t, "-I") || strings.HasPrefix(t, "--in-place") ||
			(isShortCluster(t) && (strings.ContainsAny(t[1:], "iI"))):
			// -i with its backup suffix glued on (-i.bak) edits in place too.
			inPlace = true
		case t == "-e" || t == "-f" || strings.HasPrefix(t, "--expression") || strings.HasPrefix(t, "--file"):
			scripted = true
		}
	}
	if !inPlace {
		return nil
	}
	idx := w.operandIdx([]string{"-e", "-f", "--expression", "--file", "-l"})
	if !scripted && len(idx) > 0 {
		idx = idx[1:]
	}
	var out []writeTarget
	for _, i := range idx {
		out = append(out, w.at(i))
	}
	return out
}

// gitTargets is the paths `git checkout` and `git restore` write, and the
// directory `git clone` makes. Every `git -C`, whatever its subcommand, is a
// write the guard does not place: -C moves git to a directory the guard does
// not follow, where a subcommand, its options, a configuration value on the
// line or an alias can change any file.
func (w writerArgs) gitTargets() []writeTarget {
	toks := w.s.tokens
	i := w.from
	for i < len(toks) {
		t := toks[i]
		if t == "-C" {
			return []writeTarget{unplacedWrite("")}
		}
		if t == "-c" || t == "--git-dir" || t == "--work-tree" || t == "--namespace" {
			i += 2
			continue
		}
		if strings.HasPrefix(t, "-") {
			i++
			continue
		}
		break
	}
	if i >= len(toks) {
		return nil
	}
	sub := w
	sub.from = i + 1
	switch toks[i] {
	case "checkout", "restore":
		return sub.operands([]string{"-b", "-B", "--orphan", "-s", "--source", "--conflict", "--pathspec-from-file"}, nil)
	case "clone":
		idx := sub.operandIdx([]string{"-b", "--branch", "-o", "--origin", "--depth", "-c", "--config",
			"--reference", "--separate-git-dir", "-u", "--upload-pack", "--template", "--filter", "-j", "--jobs"})
		switch {
		case len(idx) >= 2:
			t := w.at(idx[1])
			t.tree = t.ok
			return []writeTarget{t}
		case len(idx) == 1:
			repo, ok := fixedToken(w.s, idx[0])
			if !ok {
				return []writeTarget{unplacedWrite("")}
			}
			name := strings.TrimSuffix(path.Base(strings.TrimRight(repo, "/")), ".git")
			if i := strings.LastIndexByte(name, ':'); i >= 0 {
				name = name[i+1:]
			}
			t := writeAt(w.rc, w.st, name, false)
			t.tree = t.ok
			return []writeTarget{t}
		}
	}
	return nil
}
