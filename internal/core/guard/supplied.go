package guard

import (
	"path"
	"strings"
)

// supplied.go reads the words a command receives from the program that runs it:
// what xargs reads from its input, and the paths a find's -exec hands over. The
// line fixes none of them, so each is read as a word that can be any. The acts
// only the person takes are read with them, through the registry entry and under
// any program (personacts.go); every other entry reads the words the line holds.

// readsSupplied reports whether the registry entry id is an act whose words are
// its verbs and operand, which a supplied word can be. An act a flag names is
// not: a boolean flag is set only by a word the line fixes.
func readsSupplied(id string) bool {
	for _, a := range personActs {
		if a.entry == id {
			return a.adjacent || a.tag
		}
	}
	return false
}

// suppliedReadings is s as the command an xargs or a find runs may receive its
// words (xargsSupplied), and each command a find runs (findActions), with the
// commands of a string either hands a shell. xargs's command is read from each
// word the walk can reach as its program, up to maxUnknownSites of them, since a
// cluster of flags may or may not take the word after it (`-tI %`). A program of
// a wholly unknown name is read as neither xargs nor find.
func suppliedReadings(s segment) []segment {
	var runs []segment
	xargs := -1
	for _, a := range arrivalsOf(s) {
		if tok := s.tokens[a.idx]; a.wrapper && xargs < 0 && !anyProgram(tok) && commandNamed(s, a, "xargs") {
			xargs = a.idx
		}
	}
	viewed := 0
	for _, a := range commandSites(s) {
		if !anyProgram(s.tokens[a.idx]) && commandNamed(s, a, "find") {
			for _, run := range findActions(s, a.idx+1, true) {
				runs = append(runs, run.seg)
			}
		}
		if xargs >= 0 && a.idx > xargs && viewed < maxUnknownSites {
			viewed++
			runs = append(runs, xargsSupplied(s, xargs, a.idx))
		}
	}
	if len(runs) == 0 {
		return nil
	}
	out, _ := expandPayloads(runs, s.depth)
	walkSegments(out)
	return out
}

// suppliedMatch reports whether entry p fires on a reading of s with the words
// xargs or a find supplies. A program of a wholly unknown name counts only where
// the line spells the entry's verb, or where xargs's input is the program and its
// words (segment.programSupplied).
func suppliedMatch(p Pattern, readings []segment) (hit, named bool) {
	for _, r := range readings {
		h, n := matchSegmentNamed(p, r)
		if n {
			return true, true
		}
		hit = hit || h && (r.programSupplied || containsString(r.tokens, p.Subcommand))
	}
	return hit, false
}

// xargsSupplied is the command at site as the xargs at x runs it, from its program
// on. A word holding xargs's replacement string, the program's own included, holds
// the words xargs reads there
// (placeholderWords); with no replacement string, xargs adds two words after the
// command's own, since its input may hold the verb and its operand; and where the
// line does not fix the options, every word of the command can be any, the
// program's own included, and any of them a verb.
func xargsSupplied(s segment, x, site int) segment {
	repl, ok, all, unsure := xargsReplacement(s, x+1, site)
	d := segment{chain: s.chain, fromFixedOutput: true, valuesPlaced: true}
	var globbed []bool
	add := func(t string, glob bool) {
		d.tokens = append(d.tokens, t)
		globbed = append(globbed, glob)
	}
	d.programSupplied = unsure || ok && all && s.tokens[site] == repl
	for i := site; i < len(s.tokens); i++ {
		t := s.tokens[i]
		switch {
		case !ok && !unsure:
			add(t, s.globAt(i))
		case unsure:
			add(unknownText, false)
		default:
			ws := placeholderWords(t, repl, all)
			for _, w := range ws {
				add(w, len(ws) == 1 && s.globAt(i))
			}
		}
	}
	if !ok {
		add(unknownText, false)
		add(unknownText, false)
	}
	if containsBool(globbed) {
		d.globbed = globbed
	}
	return d
}

// xargsRunsInput reports whether an xargs s runs runs, as commands, words it reads
// from its input: its program is a word its input fills (xargsSupplied), or a
// command string a shell or eval it runs takes holds one, which the guard has not
// read. It reads up to maxUnknownSites of the words the walk can reach as xargs's
// program, as suppliedReadings does.
func xargsRunsInput(s segment) bool {
	if s.speculative {
		return false
	}
	named := false
	for _, t := range s.tokens {
		named = named || strings.Contains(t, "xargs")
	}
	if !named {
		return false
	}
	xargs := -1
	for _, a := range arrivalsOf(s) {
		if tok := s.tokens[a.idx]; a.wrapper && xargs < 0 && !anyProgram(tok) && commandNamed(s, a, "xargs") {
			xargs = a.idx
		}
	}
	if xargs < 0 {
		return false
	}
	own := 0
	for _, ref := range payloadsOf(s) {
		own += strings.Count(ref.payload, unknownText)
	}
	viewed := 0
	for _, a := range commandSites(s) {
		if a.idx <= xargs || viewed >= maxUnknownSites {
			continue
		}
		viewed++
		d := xargsSupplied(s, xargs, a.idx)
		if len(d.tokens) > 0 && d.tokens[0] != s.tokens[a.idx] && isUnknown(path.Base(d.tokens[0])) {
			return true
		}
		n := 0
		for _, ref := range payloadsOf(d) {
			n += strings.Count(ref.payload, unknownText)
		}
		if n > own {
			return true
		}
	}
	return false
}

// suppliedStringSignal is the fail-closed verdict on an xargs that runs, as
// commands, words it reads from its input (xargsRunsInput).
func suppliedStringSignal() payloadSignal {
	return payloadSignal{
		verdict: VerdictBlock,
		family:  familyShell,
		last:    true,
		reason: "This line has xargs run, as a command or as a command string a shell runs, words it reads from its input, " +
			"so the command that runs is not something the guard has checked.",
		successor: "Write the command out where it runs, so the guard reads it, and hand xargs only its operands.",
	}
}

// placeholderWords is the words a word t stands for, where p is the placeholder a
// program puts what it hands its command in place of: one word holding text no line
// fixes, and two where p stands for several words (several) and is the whole word,
// or where t holds a blank, as text a shell splits does.
func placeholderWords(t, p string, several bool) []string {
	switch {
	case !strings.Contains(t, p):
		return []string{t}
	case t == p && several:
		return []string{unknownText, unknownText}
	case strings.ContainsAny(t, " \t\n"):
		return []string{strings.ReplaceAll(t, p, unknownText+" "+unknownText)}
	}
	return []string{strings.ReplaceAll(t, p, unknownText)}
}

func containsBool(bs []bool) bool {
	for _, b := range bs {
		if b {
			return true
		}
	}
	return false
}
