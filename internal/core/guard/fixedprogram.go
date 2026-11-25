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

// assignsOperands are the builtins of bash and zsh that set a variable an operand
// names.
var assignsOperands = []string{
	"declare", "export", "float", "getln", "getopts", "integer", "let", "local",
	"mapfile", "print", "printf", "private", "read", "readarray", "readonly", "set",
	"strftime", "sysread", "typeset", "unset", "vared", "wait", "zformat",
	"zparseopts", "zstyle",
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
	if !ok || tmpl == "" || len(s.spelled[idx]) != 1 {
		return nil, false
	}
	return siteNames(tmpl, s.spelled[idx][0])
}

// variableFixed reports whether the line fixes name where segs[i] uses it: the
// assignments the line opens with give it fixed values, or a loop over fixed words
// that stands directly after them and is still open at i, and no other command of
// the line can set it.
func (f *fixedReader) variableFixed(i int, name string) bool {
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
// that spells the name, runs text as commands of this shell, hands a builtin that
// sets variables a word the line does not fix, or runs a program the line does not
// fix by another variable. use says the command is the one that uses the variable,
// whose own program word is not one of these. read counts the words it reads.
func segmentMaySet(s segment, use bool, name string, read *int) bool {
	for _, tok := range s.tokens {
		*read += len(tok) + 1
		if mentions(tok, name) {
			return true
		}
	}
	for _, a := range commandSites(s) {
		if use {
			break
		}
		base := path.Base(s.tokens[a.idx])
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
		if base == "eval" || base == "." || base == "source" || s.globAt(a.idx) {
			return true
		}
		if containsString(assignsOperands, base) {
			for _, tok := range s.tokens[a.idx+1:] {
				if unfixed(tok) {
					return true
				}
			}
		}
	}
	return false
}
