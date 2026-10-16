package guard

import (
	"net/url"
	"path"
	"path/filepath"
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
		case "<&", ">&":
			if t := r.target.text; r.target.ok && (t == "-" || isAllDigits([]byte(t))) {
				continue // a descriptor, not a file
			}
		}
		if !r.target.ok {
			out = append(out, writeTarget{base: r.target.base})
			continue
		}
		out = append(out, writeAt(rc, st, r.target.text, r.target.tilde))
	}
	return append(out, writerTargets(rc, s, st)...)
}

// writeAt is a write to word, resolved as the shell opens it. Where the line
// leaves the working directory unknown, a relative word is not placed, and
// is marked as written there (rel), with the name it writes there and the
// directory's identity (cwdName).
func writeAt(rc *readCtx, st *shellState, word string, tilde bool) writeTarget {
	p, ok := st.resolve(rc, word, tilde)
	w := writeTarget{path: p, ok: ok}
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
			out = append(out, w.copyTarget()...)
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
			out = append(out, w.dirTarget([]string{"-d", "--directory"}))
			out = append(out, w.flagValues([]string{"-o", "--output"})...)
		case "git":
			out = append(out, w.gitTargets()...)
		case "tar":
			if w.tarExtracts() {
				out = append(out, w.dirTarget([]string{"-C", "--directory"}))
			}
		case "unzip":
			out = append(out, w.dirTarget([]string{"-d"}))
		case "rsync":
			out = append(out, w.rsyncTarget()...)
		case "find":
			out = append(out, findRuns(rc, s, st, a.idx+1)...)
		case "go":
			out = append(out, w.goBuildTarget()...)
		case "sort":
			out = append(out, w.flagValues([]string{"-o", "--output"})...)
		case "ditto":
			out = append(out, w.lastOf(w.operandIdx([]string{"--arch", "--bom", "--zlibCompressionLevel"}), 2)...)
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
	return []writeTarget{w.at(last)}
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
			sub := segment{tokens: toks, chain: s.chain, globbed: s.globSlice(i+1, end)}
			in := st
			if action == "-execdir" || action == "-okdir" || isUnknown(action) {
				in = st.unresolved()
			}
			out = append(out, writerTargets(rc, sub, in)...)
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
			return writeTarget{base: tokenBase(w.s.tokens[i])}
		}
		return writeTarget{}
	}
	return writeAt(w.rc, w.st, tok, strings.HasPrefix(tok, "~"))
}

// valueAt is the path that token i carries after its first skip bytes.
func (w writerArgs) valueAt(i, skip int) writeTarget {
	tok := w.s.tokens[i]
	if isUnknown(tok) || w.s.globAt(i) || strings.ContainsRune(tok, varMark) {
		return writeTarget{}
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
	return writeTarget{path: w.st.dir, ok: ok, rel: !ok}
}

// copyTarget is the destination of cp, mv or install: the -t value, else the
// last operand.
func (w writerArgs) copyTarget() []writeTarget {
	if vs := w.flagValues([]string{"-t", "--target-directory"}); len(vs) > 0 {
		return vs
	}
	idx := w.operandIdx([]string{"-S", "--suffix", "-m", "--mode", "-o", "--owner", "-g", "--group"})
	if len(idx) == 0 {
		return nil
	}
	return []writeTarget{w.at(idx[len(idx)-1])}
}

// lnTarget is the link ln makes: the -t value, else the last of two or more
// operands, and with a single operand a link of the target's own name in the
// directory ln runs in (`ln -s /path/s.sh` writes ./s.sh).
func (w writerArgs) lnTarget() []writeTarget {
	if vs := w.flagValues([]string{"-t", "--target-directory"}); len(vs) > 0 {
		return vs
	}
	idx := w.operandIdx([]string{"-S", "--suffix"})
	switch len(idx) {
	case 0:
		return nil
	case 1:
		tok := w.s.tokens[idx[0]]
		if isUnknown(tok) || w.s.globAt(idx[0]) || strings.ContainsRune(tok, varMark) {
			return []writeTarget{{}}
		}
		return []writeTarget{writeAt(w.rc, w.st, path.Base(tok), false)}
	}
	return []writeTarget{w.at(idx[len(idx)-1])}
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
			out = append(out, writeTarget{})
			continue
		}
		u, err := url.Parse(t)
		if err != nil || path.Base(u.Path) == "/" || path.Base(u.Path) == "." {
			out = append(out, writeTarget{})
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
// directory `git clone` makes.
func (w writerArgs) gitTargets() []writeTarget {
	toks := w.s.tokens
	i := w.from
	for i < len(toks) {
		t := toks[i]
		if t == "-C" || t == "-c" || t == "--git-dir" || t == "--work-tree" || t == "--namespace" {
			if t == "-C" {
				return []writeTarget{{}} // another directory: not placed here
			}
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
			return []writeTarget{w.at(idx[1])}
		case len(idx) == 1:
			repo, ok := fixedToken(w.s, idx[0])
			if !ok {
				return []writeTarget{{}}
			}
			name := strings.TrimSuffix(path.Base(strings.TrimRight(repo, "/")), ".git")
			if i := strings.LastIndexByte(name, ':'); i >= 0 {
				name = name[i+1:]
			}
			return []writeTarget{writeAt(w.rc, w.st, name, false)}
		}
	}
	return nil
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
