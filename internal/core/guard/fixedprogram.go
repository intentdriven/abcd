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

// assignsOperands are the builtins that set a variable an operand names.
var assignsOperands = []string{
	"declare", "export", "getopts", "let", "local", "mapfile", "printf", "read",
	"readarray", "readonly", "typeset", "unset", "wait",
}

// programFixed reports whether the line text, read as segs, fixes the program of
// segs[i], a command of the line itself: its first word of command position is a
// variable's value and text, every variable in it is one the line fixes, and every
// reading of the command with the values in place names its program.
func programFixed(text string, segs []segment, i int) bool {
	s := segs[i]
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
		if !variableFixed(text, segs, i, name) {
			return false
		}
	}
	placed := false
	for _, r := range segs {
		if r.valuesPlaced && r.carrier == i+1 {
			if placed = idx < len(r.tokens) && !unfixed(r.tokens[idx]); !placed {
				return false
			}
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
func variableFixed(text string, segs []segment, i int, name string) bool {
	set := map[int]bool{}
	// A command the line opens with always runs in this shell: none before it and
	// none of its own words runs a subshell, redirects, or stands behind a `||`.
	opening := func(j int) bool {
		if j >= i || segs[j].end > len(text) {
			return false
		}
		upTo := text[:segs[j].end]
		return segs[j].carrier == 0 && segs[j].arrayOf == "" && !segs[j].background &&
			len(segs[j].tokens) > 0 && segs[j+1].piped.list == nil &&
			!strings.ContainsAny(upTo, "(`<>") && !strings.Contains(upTo, "||")
	}
	j := 0
	for ; opening(j); j++ {
		assigns := false
		for k, tok := range segs[j].tokens {
			n, eq, appends, subscript, ok := splitAssignment(tok)
			if !ok {
				assigns = false
				break
			}
			assigns = true
			if n != name {
				continue
			}
			if appends || subscript || segs[j].globAt(k) || !fixedValue.MatchString(tok[eq+1:]) {
				return false
			}
			set[j] = true
		}
		if !assigns {
			break
		}
	}
	if t := segs[j].tokens; opening(j) && len(t) > 3 && t[0] == "for" && t[1] == name && t[2] == "in" {
		for k := 3; k < len(t); k++ {
			if segs[j].globAt(k) || !fixedValue.MatchString(t[k]) {
				return false
			}
		}
		for _, d := range segs[j:i] {
			if len(d.tokens) > 0 && d.tokens[0] == "done" {
				return false
			}
		}
		set[j] = true
	}
	if len(set) == 0 {
		return false
	}
	mention := regexp.MustCompile(`(^|[^A-Za-z0-9_])` + name + `([^A-Za-z0-9_]|$)`)
	for k, s := range segs {
		if containsString(s.assignsInText, name) || containsString(s.numbersInText, name) ||
			!set[k] && !s.valuesPlaced && segmentMaySet(s, k == i, name, mention) {
			return false
		}
	}
	return true
}

// segmentMaySet reports whether a command may give the variable name a value: one
// that spells the name, runs text as commands of this shell, hands a builtin that
// sets variables a word the line does not fix, or runs a program the line does not
// fix by another variable. use says the command is the one that uses the variable,
// whose own program word is not one of these.
func segmentMaySet(s segment, use bool, name string, mention *regexp.Regexp) bool {
	for _, tok := range s.tokens {
		if mention.MatchString(tok) {
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
		for _, tok := range s.tokens[a.idx+1:] {
			if unfixed(tok) && containsString(assignsOperands, base) {
				return true
			}
		}
	}
	return false
}
