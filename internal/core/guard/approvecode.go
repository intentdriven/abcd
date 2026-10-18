package guard

// approvecode.go reads the ways code reaches an interpreter other than
// through its -c or -e flag (inlineCode, approve.go): on its standard input,
// in an awk program or a sed script that runs a command, in tar's options
// that run a program, and in a vim or ex command holding a `!`. The guard
// reads none of that code, so the approver refuses each.

import (
	"path"
	"regexp"
	"strings"
)

// stdinValueFlags are interpreter options whose value is the next word: a
// value is not the script.
var stdinValueFlags = []string{"-W", "-X", "-r", "--require", "--import", "--loader", "--experimental-loader",
	"-C", "--conditions", "-I", "-d", "-l"}

// stdinOperands are the operands that name the standard input as the script.
var stdinOperands = append([]string{"-"}, stdinDevices...)

// codeOnStdin reports whether the command at s.tokens[idx], an interpreter in
// the interpreters table or ex (vim in its ex mode), runs code from its
// standard input: a pipe, a here-document, a here-string or a redirected file
// feeds it, and it names no script file, or names the standard input as one.
func codeOnStdin(s segment, idx int) bool {
	name := path.Base(s.tokens[idx])
	exMode := false
	switch name {
	case "ex":
		exMode = true
	case "vim", "vi", "nvim", "view":
		for _, t := range s.tokens[idx+1:] {
			if isShortCluster(t) && strings.ContainsAny(t[1:], "eE") {
				exMode = true
			}
		}
	}
	interp := exMode
	for _, in := range interpreters {
		if in.name.MatchString(name) {
			interp = true
		}
	}
	if !interp || !stdinFed(s) {
		return false
	}
	if exMode {
		return true // ex reads its commands from its standard input
	}
	for i := idx + 1; i < len(s.tokens); i++ {
		t := s.tokens[i]
		switch {
		case t == "<<<":
			return true // the here-string, with no script before it
		case isUnknown(t) || strings.ContainsRune(t, varMark):
			return true // may be `-`
		case containsString(stdinOperands, t):
			return true
		case t == "-m":
			return false // a module, run from its file
		case containsString(stdinValueFlags, t):
			i++
		case strings.HasPrefix(t, "-"):
		default:
			return false // the script, a file
		}
	}
	return true
}

// stdinFed reports whether s's standard input is a stream (a pipe, a
// here-document, a here-string) or a file a redirection opens.
func stdinFed(s segment) bool {
	if s.stdinStream {
		return true
	}
	for _, r := range s.redirects {
		// A redirection with no descriptor named (-1) is the operator's
		// default, standard input for `<`.
		if (r.fd == 0 || r.fd < 0) && (r.op == "<" || r.op == "<>") {
			return true
		}
	}
	return false
}

// awkRuns is the awk program text that runs a command: system(), a pipe to
// a command from print or printf, a command piped into getline, or a
// coprocess.
var awkRuns = regexp.MustCompile(`\bsystem\s*\(|\|\s*getline|\|&|\bprintf?\b[^;{}\n]*\|`)

// awkCode reports whether the awk command at s.tokens[idx] has a program
// that runs a command, or one the line does not fix.
func awkCode(s segment, idx int) bool {
	switch path.Base(s.tokens[idx]) {
	case "awk", "gawk", "mawk", "nawk":
	default:
		return false
	}
	var progs []string
	fromFile := false
	toks := s.tokens
	i := idx + 1
	for ; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "-f" || t == "--file":
			fromFile = true
			i++
		case t == "-e" || t == "--source":
			if i+1 < len(toks) {
				progs = append(progs, toks[i+1])
			}
			i++
		case strings.HasPrefix(t, "--source="):
			progs = append(progs, strings.TrimPrefix(t, "--source="))
		case t == "-F" || t == "-v" || t == "--assign" || t == "--field-separator":
			i++
		case t == "--":
			i++
			goto operands
		case strings.HasPrefix(t, "-") && t != "-":
		default:
			goto operands
		}
	}
operands:
	if len(progs) == 0 && !fromFile && i < len(toks) {
		progs = append(progs, toks[i])
	}
	for _, p := range progs {
		if isUnknown(p) || strings.ContainsRune(p, varMark) || awkRuns.MatchString(p) {
			return true
		}
	}
	return false
}

// sedScripts is the scripts the sed command at s.tokens[idx] runs that the
// line spells: each -e value, or the first operand (the second, after BSD's
// empty -i suffix). ok is false where a script is a word the line does not
// fix.
func sedScripts(s segment, idx int) (scripts []string, ok bool) {
	toks := s.tokens
	if path.Base(toks[idx]) != "sed" {
		return nil, true
	}
	fromFile := false
	var ops []int
	for i := idx + 1; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "-e" || t == "--expression":
			if i+1 < len(toks) {
				scripts = append(scripts, toks[i+1])
				if isUnknown(toks[i+1]) || strings.ContainsRune(toks[i+1], varMark) {
					return nil, false
				}
			}
			i++
		case strings.HasPrefix(t, "--expression="):
			scripts = append(scripts, strings.TrimPrefix(t, "--expression="))
		case t == "-f" || t == "--file" || strings.HasPrefix(t, "--file="):
			fromFile = true
			if t == "-f" || t == "--file" {
				i++
			}
		case t == "-l" || t == "--line-length":
			i++
		case strings.HasPrefix(t, "-") && t != "-":
		default:
			ops = append(ops, i)
		}
	}
	if len(scripts) == 0 && !fromFile && len(ops) > 0 {
		at := ops[0]
		if toks[at] == "" && len(ops) > 1 {
			at = ops[1] // sed -i '' script file: the empty word is the suffix
		}
		if isUnknown(toks[at]) || strings.ContainsRune(toks[at], varMark) {
			return nil, false
		}
		scripts = append(scripts, toks[at])
	}
	return scripts, true
}

// sedReads is one sed script as sed reads it: whether it runs a command (the
// e command, or the e flag of s), the files its w and W commands and the w
// flag of s write, and ok, false where the script does not read as sed.
func sedReads(script string) (runs bool, writes []string, ok bool) {
	i, n := 0, len(script)
	toEOL := func() string {
		start := i
		for i < n && script[i] != '\n' {
			i++
		}
		return strings.TrimSpace(script[start:i])
	}
	delimited := func(d byte) bool {
		for i < n {
			c := script[i]
			i++
			switch {
			case c == '\\' && i < n:
				i++
			case c == d:
				return true
			}
		}
		return false
	}
	address := func() bool {
		switch {
		case i < n && script[i] >= '0' && script[i] <= '9':
			for i < n && (script[i] >= '0' && script[i] <= '9' || script[i] == '~') {
				i++
			}
		case i < n && script[i] == '$':
			i++
		case i < n && script[i] == '/':
			i++
			if !delimited('/') {
				return false
			}
			for i < n && (script[i] == 'I' || script[i] == 'M') {
				i++
			}
		case i+1 < n && script[i] == '\\':
			d := script[i+1]
			i += 2
			if !delimited(d) {
				return false
			}
		}
		return true
	}
	for i < n {
		c := script[i]
		if c == ' ' || c == '\t' || c == '\n' || c == ';' || c == '{' || c == '}' {
			i++
			continue
		}
		if c == '#' {
			toEOL()
			continue
		}
		if !address() {
			return false, nil, false
		}
		for i < n && (script[i] == ' ' || script[i] == '\t') {
			i++
		}
		if i < n && script[i] == ',' {
			i++
			for i < n && (script[i] == ' ' || script[i] == '\t') {
				i++
			}
			if i < n && (script[i] == '+' || script[i] == '~') {
				i++
			}
			if !address() {
				return false, nil, false
			}
		}
		for i < n && (script[i] == ' ' || script[i] == '\t' || script[i] == '!') {
			i++
		}
		if i >= n {
			return runs, writes, true
		}
		cmd := script[i]
		i++
		switch cmd {
		case '{', '}':
		case 's', 'y':
			if i >= n {
				return false, nil, false
			}
			d := script[i]
			i++
			if !delimited(d) || !delimited(d) {
				return false, nil, false
			}
			if cmd == 'y' {
				continue
			}
			for i < n && strings.IndexByte("gpiImMe0123456789w", script[i]) >= 0 {
				f := script[i]
				i++
				if f == 'e' {
					runs = true
				}
				if f == 'w' {
					writes = append(writes, toEOL())
					break
				}
			}
		case 'e':
			runs = true
			toEOL()
		case 'w', 'W':
			writes = append(writes, toEOL())
		case 'a', 'i', 'c', 'r', 'R':
			for {
				line := toEOL()
				if !strings.HasSuffix(line, "\\") || i >= n {
					break
				}
				i++
			}
		case 'b', 't', 'T', ':', 'v':
			for i < n && script[i] != ';' && script[i] != '\n' && !(cmd != ':' && script[i] == '}') {
				i++
			}
		case 'q', 'Q', 'l', 'L':
			for i < n && (script[i] == ' ' || script[i] >= '0' && script[i] <= '9') {
				i++
			}
		case '=', 'd', 'D', 'g', 'G', 'h', 'H', 'n', 'N', 'p', 'P', 'x', 'z', 'F':
		default:
			return false, nil, false
		}
	}
	return runs, writes, true
}

// sedCode reports whether the sed command at s.tokens[idx] runs a command
// from its script, or has a script the guard cannot read.
func sedCode(s segment, idx int) bool {
	scripts, ok := sedScripts(s, idx)
	if !ok {
		return true
	}
	for _, sc := range scripts {
		if runs, _, ok := sedReads(sc); runs || !ok {
			return true
		}
	}
	return false
}

// tarRuns are tar's long options that run a program, each with the shortest
// prefix GNU tar takes for it alone.
var tarRuns = [][2]string{
	{"--checkpoint-action", "--checkpoint-"}, {"--to-command", "--to-c"}, {"--info-script", "--inf"},
	{"--new-volume-script", "--new-"}, {"--rsh-command", "--rs"}, {"--rmt-command", "--rm"},
	{"--use-compress-program", "--use"},
}

// tarCode reports whether the tar command at s.tokens[idx] carries an option
// that runs a program: one of tarRuns by any prefix tar takes, or -I or -F.
func tarCode(s segment, idx int) bool {
	if path.Base(s.tokens[idx]) != "tar" {
		return false
	}
	for _, t := range s.tokens[idx+1:] {
		if t == "--" {
			return false
		}
		if t == "-I" || t == "-F" || strings.HasPrefix(t, "-I") && !strings.HasPrefix(t, "--") ||
			strings.HasPrefix(t, "-F") && !strings.HasPrefix(t, "--") {
			return true
		}
		name, _, _ := strings.Cut(t, "=")
		for _, r := range tarRuns {
			if strings.HasPrefix(r[0], name) && strings.HasPrefix(name, r[1]) {
				return true
			}
		}
	}
	return false
}

// vimCode reports whether the vim or ex command at s.tokens[idx] carries a
// command (-c, --cmd, +) holding a `!`, which runs a shell command, or one
// the line does not fix.
func vimCode(s segment, idx int) bool {
	switch path.Base(s.tokens[idx]) {
	case "vim", "vi", "nvim", "ex", "view", "vimdiff", "gvim", "evim":
	default:
		return false
	}
	toks := s.tokens
	for i := idx + 1; i < len(toks); i++ {
		t := toks[i]
		var cmd string
		switch {
		case t == "-c" || t == "--cmd":
			if i+1 >= len(toks) {
				continue
			}
			cmd = toks[i+1]
			i++
		case strings.HasPrefix(t, "+") && len(t) > 1:
			cmd = t[1:]
		default:
			continue
		}
		if isUnknown(cmd) || strings.ContainsRune(cmd, varMark) || strings.ContainsRune(cmd, '!') {
			return true
		}
	}
	return false
}

// otherCode reports whether the command at s.tokens[idx] is handed code the
// guard does not read other than through an interpreter's inline flag, and
// the program's name.
func otherCode(s segment, idx int) (string, bool) {
	name := path.Base(s.tokens[idx])
	if codeOnStdin(s, idx) || awkCode(s, idx) || sedCode(s, idx) || tarCode(s, idx) || vimCode(s, idx) {
		return name, true
	}
	return "", false
}
