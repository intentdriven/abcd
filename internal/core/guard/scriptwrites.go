package guard

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
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
		tok := s.tokens[a.idx]
		if isUnknown(tok) {
			continue
		}
		w := writerArgs{rc: rc, s: s, st: st, from: a.idx + 1}
		switch strings.ToLower(path.Base(tok)) {
		case "tee":
			out = append(out, w.operands(nil, nil)...)
		case "cp", "mv", "install":
			out = append(out, w.copyTarget(strings.ToLower(path.Base(tok)) == "install")...)
		case "ln":
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
			if w.tarExtracts() {
				d := w.dirTarget([]string{"-C", "--directory"})
				if !w.tarToProgram() {
					d = asTreeIn(d)
				}
				out = append(out, d)
			}
		case "unzip":
			out = append(out, asTreeIn(w.dirTarget([]string{"-d"})))
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
	}
	return out
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
func (w writerArgs) copyTarget(install bool) []writeTarget {
	letters, longs := "St", []string{"--suffix", "--sparse", "--no-preserve"}
	if install {
		letters, longs = "Stmog", []string{"--suffix", "--mode", "--owner", "--group", "--strip-program"}
	}
	args := w.readCopyArgs(letters, longs)
	if install && (args.short['d'] || args.long("--directory")) {
		var out []writeTarget
		for _, i := range args.operands {
			out = append(out, w.at(i))
		}
		return out
	}
	return w.sourceLandings(args, 1)
}

// lnTarget is the link ln makes: the -t value, else the last of two or more
// operands, and with a single operand a link of the target's own name in the
// directory ln runs in (`ln -s /path/s.sh` writes ./s.sh). Each is read as
// sourceLandings reads a copy, since a link to a directory places its tree.
func (w writerArgs) lnTarget() []writeTarget {
	args := w.readCopyArgs("St", []string{"--suffix"})
	if len(args.operands) == 1 && args.target == nil {
		i := args.operands[0]
		tok := w.s.tokens[i]
		if isUnknown(tok) || w.s.globAt(i) || strings.ContainsRune(tok, varMark) {
			return []writeTarget{unplacedWrite("")}
		}
		t := writeAt(w.rc, w.st, path.Base(tok), false)
		t.tree = t.ok
		return []writeTarget{t}
	}
	return w.sourceLandings(args, 2)
}

// copyArgs is the words of cp, mv, install or ln read as GNU getopt reads
// them: the operands, the -t value, -T, and each short option given, a short
// option taking a value ending the cluster it stands in. unsure marks an
// option word the line does not fix, which may be -T.
type copyArgs struct {
	operands []int
	target   *writeTarget
	noTarget bool
	unsure   bool
	short    map[byte]bool
	longs    []string
}

// long reports whether a long option given is name, or a prefix of it at
// least two letters long, as getopt takes an abbreviation.
func (a copyArgs) long(name string) bool {
	for _, l := range a.longs {
		if strings.HasPrefix(name, l) && len(l) > len("--")+1 {
			return true
		}
	}
	return false
}

// readCopyArgs reads the words after w.from: valueLetters are the short
// options that take a value, glued on or the next word, and valueLongs the
// long ones, which take the next word unless an `=` carries it. A long option
// is read whole or by a prefix of it, as getopt reads it.
func (w writerArgs) readCopyArgs(valueLetters string, valueLongs []string) copyArgs {
	a := copyArgs{short: map[byte]bool{}}
	toks := w.s.tokens
	opts := true
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		switch {
		case opts && t == "--":
			opts = false
		case opts && strings.HasPrefix(t, "--"):
			name, _, glued := strings.Cut(t, "=")
			a.longs = append(a.longs, name)
			switch {
			case len(name) > len("--t") && strings.HasPrefix("--target-directory", name):
				target := w.at(i + 1)
				if glued {
					target = w.valueAt(i, len(name)+1)
				} else {
					i++
				}
				a.target = &target
			case len(name) > len("--no-t") && strings.HasPrefix("--no-target-directory", name):
				a.noTarget = true
			case !glued && len(name) > len("--s") && slices.ContainsFunc(valueLongs, func(l string) bool { return strings.HasPrefix(l, name) }):
				i++
			}
		case opts && strings.HasPrefix(t, "-") && t != "-":
			if isUnknown(t) || strings.ContainsRune(t, varMark) {
				a.unsure = true
			}
			for j := 1; j < len(t); j++ {
				c := t[j]
				a.short[c] = true
				if c == 'T' {
					a.noTarget = true
				}
				if strings.IndexByte(valueLetters, c) < 0 {
					continue
				}
				var value writeTarget
				if j+1 < len(t) {
					value = w.valueAt(i, j+1)
				} else {
					value = w.at(i + 1)
					i++
				}
				if c == 't' {
					a.target = &value
				}
				break
			}
		default:
			a.operands = append(a.operands, i)
		}
	}
	return a
}

// sourceLandings is what a command that moves, copies or links its sources
// to a destination writes (cp, mv, install, ln): the destination, the -t value
// or else the last of at least min operands, and the tree each source places.
// A source lands in the destination under its own last name when the
// destination is a directory: named by -t, spelled with a trailing slash, one
// of several sources, or a directory standing there now (followed through a
// link, as the commands follow it); unless -T says the destination is never
// one. Otherwise the source takes the destination's name. A source whose last
// name is a glob lands under any name it matches; one ending in a slash, `.`
// or a name no line fixes, under any name at all, since the copy may be of a
// directory's contents.
func (w writerArgs) sourceLandings(args copyArgs, min int) []writeTarget {
	srcs := args.operands
	var dest writeTarget
	into := false
	switch {
	case args.target != nil:
		dest, into = *args.target, true
	case len(srcs) < min:
		return nil
	default:
		last := srcs[len(srcs)-1]
		dest, srcs = w.at(last), srcs[:len(srcs)-1]
		into = len(srcs) > 1 || strings.HasSuffix(w.s.tokens[last], "/") || (dest.ok && isDir(dest.path))
	}
	if !dest.ok {
		return []writeTarget{dest}
	}
	if args.noTarget || !into {
		dest.tree = true
		return []writeTarget{dest}
	}
	out := []writeTarget{dest}
	if args.unsure {
		out = append(out, asTreeIn(dest))
	}
	for _, i := range srcs {
		out = append(out, w.landingIn(dest, i))
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

// patchTargets is what patch writes: the -o file where one is named; else the
// file its first operand names, which it patches alone; else every file the
// patch names, a tree of names no line fixes in the directory -d names or the
// one it runs in.
func (w writerArgs) patchTargets() []writeTarget {
	if out := w.flagValues([]string{"-o", "--output"}); len(out) > 0 {
		return out
	}
	idx := w.operandIdx([]string{"-d", "--directory", "-i", "--input", "-p", "--strip", "-o", "--output",
		"-r", "--reject-file", "-D", "--ifdef", "-F", "--fuzz", "-z", "--suffix", "-B", "--prefix",
		"-Y", "--basename-prefix", "-V", "--version-control", "-g", "--get", "--quoting-style"})
	if len(idx) > 0 {
		return []writeTarget{w.at(idx[0])}
	}
	return []writeTarget{asTreeIn(w.dirTarget([]string{"-d", "--directory"}))}
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

// tarToProgram reports whether an extracting tar hands each file to standard
// output or to a program (-O, --to-stdout, --to-command) rather than writing it.
func (w writerArgs) tarToProgram() bool {
	toks := w.s.tokens
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "--to-stdout" || strings.HasPrefix(t, "--to-command"):
			return true
		case i == w.from && !strings.HasPrefix(t, "-") && strings.ContainsRune(t, 'O'):
			return true
		case isShortCluster(t) && strings.ContainsRune(t[1:], 'O'):
			return true
		}
	}
	return false
}

// tarExtracts reports whether a tar command extracts: a mode letter x in its
// first word (`tar xzf`, `tar -xzf`), or --extract / --get.
func (w writerArgs) tarExtracts() bool {
	toks := w.s.tokens
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "--extract" || t == "--get":
			return true
		case i == w.from && !strings.HasPrefix(t, "-") && strings.ContainsRune(t, 'x'):
			return true
		case isShortCluster(t) && strings.ContainsRune(t[1:], 'x'):
			return true
		}
	}
	return false
}
