package guard

import (
	"path"
	"regexp"
	"strings"
)

// fixedprogram.go reports where the line itself fixes a program a variable names.
// Such a command is read as that program already (carriedReadings); the update with
// a tag, whose words are common to other programs, is then not read as one abcd may
// run as well. The test is narrow: the line opens with the assignment, or with a loop
// over fixed words, and no other command of it can give the variable a value.

// fixedValue is a value that names a program the shell neither splits nor expands.
var fixedValue = regexp.MustCompile(`^[A-Za-z0-9_./@+-]+$`)

// How a builtin of bash or zsh names the variables it sets: every operand
// (setsOperands), the name before an operand's `=` (setsDeclared), the value of
// one of its options alone (setsOptioned), its second operand (setsSecond), or
// any word it is handed, for one that runs text as commands too (setsAny).
const (
	setsOperands = iota
	setsDeclared
	setsOptioned
	setsSecond
	setsAny
)

// setter is how one builtin names what it sets: how, the options whose value is
// a name, or text it runs (names), and the options whose value is neither
// (values).
type setter struct {
	how           int
	names, values []string
}

// setters are the builtins that set a variable a word they are handed names.
var setters = map[string]setter{
	"declare": {how: setsDeclared}, "export": {how: setsDeclared}, "float": {how: setsDeclared},
	"integer": {how: setsDeclared}, "local": {how: setsDeclared}, "private": {how: setsDeclared},
	"readonly": {how: setsDeclared}, "typeset": {how: setsDeclared},
	"printf": {how: setsOptioned, names: []string{"-v"}}, "print": {how: setsOptioned, names: []string{"-v"}},
	"wait": {how: setsOptioned, names: []string{"-p"}}, "set": {how: setsOptioned, names: []string{"-A", "+A"}},
	"read":      {how: setsOperands, names: []string{"-a", "-A"}, values: []string{"-d", "-i", "-k", "-n", "-N", "-p", "-t", "-u"}},
	"mapfile":   {how: setsOperands, names: []string{"-C"}, values: []string{"-c", "-d", "-n", "-O", "-s", "-u"}},
	"readarray": {how: setsOperands, names: []string{"-C"}, values: []string{"-c", "-d", "-n", "-O", "-s", "-u"}},
	"getln":     {how: setsOperands}, "let": {how: setsOperands}, "unset": {how: setsOperands}, "vared": {how: setsOperands},
	"getopts": {how: setsSecond},
	"emulate": {how: setsAny}, "strftime": {how: setsAny}, "sysread": {how: setsAny}, "trap": {how: setsAny},
	"zformat": {how: setsAny}, "zparseopts": {how: setsAny}, "zstyle": {how: setsAny},
}

// implicitNames are the variables builtins set with no word naming them.
var implicitNames = map[string][]string{
	"read": {"REPLY"}, "select": {"REPLY"}, "sysread": {"REPLY"},
	"getopts": {"OPTARG", "OPTIND"}, "mapfile": {"MAPFILE"}, "readarray": {"MAPFILE"},
}

// shellNames are the variables the shell itself gives values as the line runs,
// so the line never fixes them.
var shellNames = []string{
	"_", "BASH_COMMAND", "BASH_REMATCH", "BASH_SOURCE", "COPROC", "DIRSTACK", "FUNCNAME",
	"MATCH", "OLDPWD", "PWD", "READLINE_LINE", "match",
}

// tiedNames are the names zsh ties to another, so an assignment to one sets the
// other too (psvar and PSVAR, PS1 and PROMPT, path and PATH), and argv, which is
// the positional parameters. They are compared in lower case, so either half of a
// pair, in any case, is one of them.
var tiedNames = map[string]bool{
	"psvar": true, "ps1": true, "prompt": true, "ps2": true, "prompt2": true, "ps3": true,
	"prompt3": true, "ps4": true, "prompt4": true, "rps1": true, "rprompt": true, "rps2": true,
	"rprompt2": true, "cdpath": true, "watch": true, "fignore": true, "path": true, "fpath": true,
	"manpath": true, "mailpath": true, "module_path": true, "histchars": true, "argv": true,
}

// arithTests are the [[ operators that read their operands as arithmetic.
var arithTests = map[string]bool{"-eq": true, "-ne": true, "-lt": true, "-le": true, "-gt": true, "-ge": true}

// tiedName reports whether an assignment to name sets another name too.
func tiedName(name string) bool { return tiedNames[strings.ToLower(name)] }

// maySetBy reports whether the builtin base, handed the words args, may set a
// variable no word of the line names: one a word it is handed and the line does not
// fix can name.
func maySetBy(base string, args []string) bool {
	st, ok := setters[base]
	if !ok {
		return false
	}
	operand := 0
	for i := 0; i < len(args); i++ {
		t := args[i]
		switch {
		case st.how == setsAny:
			if unfixed(t) {
				return true
			}
			continue
		case containsString(st.names, t):
			if i+1 < len(args) && unfixed(args[i+1]) {
				return true
			}
			i++
			continue
		case containsString(st.values, t):
			i++
			continue
		case st.how == setsOptioned:
			// Options stand before the first word that is not one; one the line
			// does not fix may be a name's option with the word after it.
			if unfixed(t) {
				if i+1 < len(args) {
					return true
				}
				continue
			}
			if !strings.HasPrefix(t, "-") && !strings.HasPrefix(t, "+") {
				return false
			}
			continue
		case !unfixed(t) && strings.HasPrefix(t, "-"):
			continue
		}
		operand++
		switch st.how {
		case setsDeclared:
			name, _, _ := strings.Cut(t, "=")
			if unfixed(name) {
				return true
			}
		case setsSecond:
			if operand == 2 && unfixed(t) {
				return true
			}
		default:
			if unfixed(t) {
				return true
			}
		}
	}
	return false
}

// quietPrograms are the commands a line may hold beside the one a variable
// names and still fix that program (quietCommand). Each can set a variable in
// some form, and quietCommand admits only the forms outside these: printf with
// -v, a %n in its words or a word it does not read as plain; a test with -v,
// a word holding `[`, `$`, a backtick or an assignment, or, in [[, a variable's
// value read as arithmetic; and an assignment to a name zsh ties to another
// (tiedNames) or to argv.
var quietPrograms = map[string]bool{
	"echo": true, "printf": true, "test": true, "[": true, "[[": true,
	"true": true, "false": true, ":": true, "cd": true,
}

// quietKeywords open a command without running one; quietClosers close a
// compound command and are all the command holds.
var (
	quietKeywords = map[string]bool{"do": true, "then": true, "else": true, "elif": true, "if": true, "!": true, "{": true}
	quietClosers  = map[string]bool{"done": true, "fi": true, "}": true, "esac": true}
)

// quietCommand reports whether s is a command that cannot give any variable of
// the line a value: one of quietPrograms, a plain assignment of fixed text, a
// for-loop's opening over words, a case's opening or one of its patterns, or a
// keyword that runs nothing, every word of it free of any expansion other than
// a plain variable's, and of a pattern outside a case's patterns. Every
// other command may set a variable, and so may any word the guard does not read
// as plain.
func quietCommand(s segment) bool {
	if s.arrayOf != "" || len(s.assignsInText) > 0 || len(s.numbersInText) > 0 {
		return false
	}
	toks := s.tokens
	from := 0
	for from < len(toks) && quietKeywords[toks[from]] {
		from++
	}
	if from == len(toks) || len(toks)-from == 1 && quietClosers[toks[from]] {
		return true
	}
	// A case matches its word against patterns, which bash neither runs nor
	// expands as file names.
	matches := s.casePattern || toks[from] == "case"
	for i := from; i < len(toks); i++ {
		t := toks[i]
		if s.globAt(i) && !matches && t != "[" && t != "]" && t != "[[" && t != "]]" {
			return false
		}
		if isUnknown(t) && !plainVariableWord(s, i) {
			return false
		}
	}
	prog, args := toks[from], toks[from+1:]
	switch {
	case s.casePattern:
		return true
	case prog == "case":
		return len(args) >= 2 && args[1] == "in"
	case allAssignments(toks[from:]):
		for _, t := range toks[from:] {
			name, eq, appends, subscript, ok := splitAssignment(t)
			if !ok || appends || subscript || unfixed(t[eq+1:]) || tiedName(name) {
				return false
			}
		}
		return true
	case prog == "for":
		return len(args) >= 2 && isName(args[0]) && args[1] == "in"
	case prog == "printf":
		for _, a := range args {
			if strings.Contains(a, "%n") {
				return false
			}
		}
		return len(args) == 0 || !unfixed(args[0]) && (args[0] == "--" || !strings.HasPrefix(args[0], "-"))
	case prog == "test" || prog == "[" || prog == "[[":
		// A word a test reads as a subscript or as arithmetic runs what it
		// holds (`-v 'a[$(…)]'`, `[[ 'a[$(…)]' -eq 0 ]]`).
		arith := false
		for _, a := range args {
			arith = arith || prog == "[[" && arithTests[a]
		}
		for i, a := range args {
			if arith && isUnknown(a) && plainVariableWord(s, from+1+i) {
				// [[ reads an operand of -eq and its kin as arithmetic, which
				// runs a subscript the value holds.
				return false
			}
			if a == "-v" || strings.ContainsAny(a, "[$`") && a != "[" && a != "[[" ||
				strings.Contains(a, "=") && a != "=" && a != "==" && a != "!=" && a != "=~" {
				return false
			}
		}
	}
	return quietPrograms[prog]
}

// plainVariableWord reports whether word i of s holds no expansion but a plain
// variable's (`$NAME`, `${NAME}`, a positional), with no transform, flag, default
// or subscript.
func plainVariableWord(s segment, i int) bool {
	tmpl, ok := s.variable[i]
	if !ok || tmpl == "" || s.transformed[i] {
		return false
	}
	for _, sp := range s.spelled[i] {
		if !plainSpelling(sp) {
			return false
		}
	}
	return len(s.spelled[i]) > 0
}

// plainSpelling reports whether every `$` of a word's spelling opens a plain
// name or positional, bare or in braces.
func plainSpelling(sp string) bool {
	for i := 0; i < len(sp); i++ {
		if sp[i] != '$' {
			continue
		}
		j := i + 1
		braced := j < len(sp) && sp[j] == '{'
		if braced {
			j++
		}
		k := j
		for k < len(sp) && (nameByte(sp[k]) || k == j && (sp[k] == '@' || sp[k] == '*' || sp[k] == '#' || sp[k] == '?')) {
			k++
		}
		if k == j || braced && (k >= len(sp) || sp[k] != '}') {
			return false
		}
		i = k - 1
	}
	return true
}

// fixedReader answers, for the commands of one line, whether the line fixes the
// program a command's variables name. It reads the line once for what every
// command shares, and each variable's mentions once, so its cost is linear in
// the line.
type fixedReader struct {
	text string
	segs []segment
	// opening is how many commands the line opens with that always run in this
	// shell, assigns is how many of those assign and nothing else, and loop is
	// the for-loop that follows them, -1 where none does, closed at loopEnd.
	opening, assigns, loop, loopEnd int
	// placed is the readings with values in place, by the command they read.
	placed map[int][]int
	// names is what the line gives each variable it fixes, read on first use.
	names map[string]*fixedName
	// read counts the words and bytes read, so a test can hold the cost.
	read int
	// loud is the commands of the line other than readings that are not
	// quietCommand, up to two of them.
	loud []int
}

// fixedName is what the line does with one variable: whether the commands it
// opens with give it fixed values only, and which commands may set it otherwise.
type fixedName struct {
	set, broken bool
	inLoop      bool
	maySet      []int
}

func newFixedReader(text string, segs []segment) *fixedReader {
	f := &fixedReader{text: text, segs: segs, loop: -1, loopEnd: len(segs)}
	scanned, open := 0, true
	for j := range segs {
		s := segs[j]
		if s.carrier != 0 {
			if s.valuesPlaced {
				if f.placed == nil {
					f.placed = map[int][]int{}
				}
				f.placed[s.carrier-1] = append(f.placed[s.carrier-1], j)
			}
			open = false
			continue
		}
		if !open {
			continue
		}
		// A command the line opens with always runs in this shell: none before
		// it and none of its own words runs a subshell, redirects, or stands
		// behind a `||`.
		if s.end > len(text) || s.end < scanned || s.arrayOf != "" || s.background || len(s.tokens) == 0 ||
			j+1 < len(segs) && segs[j+1].piped.list != nil {
			open = false
			continue
		}
		upTo := text[scanned:s.end]
		f.read += len(upTo)
		if strings.ContainsAny(upTo, "(`<>") || strings.Contains(text[max(scanned-1, 0):s.end], "||") {
			open = false
			continue
		}
		scanned = s.end
		f.opening = j + 1
	}
	for j, s := range segs {
		if len(f.loud) == 2 {
			break
		}
		f.read += len(s.tokens)
		if s.carrier == 0 && !s.valuesPlaced && !quietCommand(s) {
			f.loud = append(f.loud, j)
		}
	}
	for f.assigns < f.opening && len(segs[f.assigns].tokens) > 0 && allAssignments(segs[f.assigns].tokens) {
		f.assigns++
	}
	if j := f.assigns; j < f.opening {
		if t := segs[j].tokens; len(t) > 3 && t[0] == "for" && t[2] == "in" {
			f.loop = j
			for k := j + 1; k < len(segs); k++ {
				f.read++
				if segs[k].carrier == 0 && len(segs[k].tokens) > 0 && segs[k].tokens[0] == "done" {
					f.loopEnd = k
					break
				}
			}
		}
	}
	return f
}

// program reports whether the line fixes the program of segs[i], a command of the
// line itself: its first word of command position is a variable's value and text,
// every variable in it is one the line fixes, and every reading of the command with
// the values in place names its program. Those readings hold every combination of
// the values (carriedSegments).
func (f *fixedReader) program(i int) bool {
	s := f.segs[i]
	for _, k := range f.loud {
		if k != i {
			return false
		}
	}
	sites := commandSites(s)
	if s.carrier != 0 || len(sites) == 0 {
		return false
	}
	idx := sites[0].idx
	names, ok := variableNames(s, idx)
	if !ok || !isUnknown(path.Base(s.tokens[idx])) {
		return false
	}
	for _, name := range names {
		if !f.variableFixed(i, name) {
			return false
		}
	}
	placed := false
	for _, k := range f.placed[i] {
		r := f.segs[k]
		if placed = idx < len(r.tokens) && !unfixed(r.tokens[idx]); !placed {
			return false
		}
	}
	return placed
}

// variableNames is the variables the word at idx of s is spelled with, where every
// unfixed part of it is a plain `$NAME` or `${NAME}`.
func variableNames(s segment, idx int) ([]string, bool) {
	tmpl, ok := s.variable[idx]
	if !ok || tmpl == "" || s.transformed[idx] || len(s.spelled[idx]) != 1 {
		return nil, false
	}
	return siteNames(tmpl, s.spelled[idx][0])
}

// variableFixed reports whether the line fixes name where segs[i] uses it: the
// assignments the line opens with give it fixed values, or a loop over fixed words
// that stands directly after them and is still open at i, and no other command of
// the line can set it.
func (f *fixedReader) variableFixed(i int, name string) bool {
	if containsString(shellNames, name) || tiedName(name) {
		return false
	}
	n := f.name(name)
	if n.broken || !n.set || n.inLoop && (i <= f.loop || i >= f.loopEnd) {
		return false
	}
	for _, k := range n.maySet {
		if k != i {
			return false
		}
		if segmentMaySet(f.segs[k], true, name, &f.read) {
			return false
		}
	}
	return true
}

// name reads what the line does with the variable name, once.
func (f *fixedReader) name(name string) *fixedName {
	if n, ok := f.names[name]; ok {
		return n
	}
	if f.names == nil {
		f.names = map[string]*fixedName{}
	}
	n := &fixedName{}
	f.names[name] = n
	set := map[int]bool{}
	for j := 0; j < f.assigns; j++ {
		for k, tok := range f.segs[j].tokens {
			f.read++
			v, eq, appends, subscript, _ := splitAssignment(tok)
			if v != name {
				continue
			}
			if appends || subscript || f.segs[j].globAt(k) || !fixedValue.MatchString(tok[eq+1:]) {
				n.broken = true
			}
			set[j] = true
		}
	}
	if j := f.loop; j >= 0 && f.segs[j].tokens[1] == name {
		t := f.segs[j].tokens
		for k := 3; k < len(t); k++ {
			f.read++
			if f.segs[j].globAt(k) || !fixedValue.MatchString(t[k]) {
				n.broken = true
			}
		}
		set[j], n.inLoop = true, true
	}
	n.set = len(set) > 0
	if n.broken || !n.set {
		return n
	}
	for k, s := range f.segs {
		switch {
		case containsString(s.assignsInText, name) || containsString(s.numbersInText, name):
			n.broken = true
			return n
		case !set[k] && !s.valuesPlaced && segmentMaySet(s, false, name, &f.read):
			n.maySet = append(n.maySet, k)
		}
	}
	return n
}

// mentions reports whether tok spells name as a whole word: not part of a longer
// name.
func mentions(tok, name string) bool {
	for at := 0; ; {
		i := strings.Index(tok[at:], name)
		if i < 0 {
			return false
		}
		i += at
		end := i + len(name)
		if (i == 0 || !nameByte(tok[i-1])) && (end == len(tok) || !nameByte(tok[end])) {
			return true
		}
		at = i + 1
	}
}

func nameByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// segmentMaySet reports whether a command may give the variable name a value: one
// that spells the name, runs text as commands of this shell, is a builtin that sets
// the name with no word naming it, hands a builtin that sets variables a word the
// line does not fix where that word can name one (maySetBy), or runs a program the
// line does not fix by another variable. use says the command is the one that uses
// the variable, whose own program word is not one of these. read counts the words
// it reads.
func segmentMaySet(s segment, use bool, name string, read *int) bool {
	for _, tok := range s.tokens {
		*read += len(tok) + 1
		if mentions(tok, name) {
			return true
		}
	}
	if len(s.tokens) > 0 && containsString(implicitNames[s.tokens[0]], name) {
		return true
	}
	for _, a := range commandSites(s) {
		if use {
			break
		}
		base := path.Base(s.tokens[a.idx])
		if containsString(implicitNames[base], name) {
			return true
		}
		if unfixed(base) {
			names, ok := variableNames(s, a.idx)
			for _, n := range names {
				ok = ok && n == name
			}
			if !ok {
				return true
			}
			break
		}
		// A test's `[` and `[[` are no pattern: nothing closes them.
		glob := s.globAt(a.idx) && base != "[" && base != "[["
		if base == "eval" || base == "." || base == "source" || glob || maySetBy(base, s.tokens[a.idx+1:]) {
			return true
		}
	}
	return false
}
