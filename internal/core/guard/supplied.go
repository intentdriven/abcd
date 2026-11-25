package guard

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
// words: xargs's view of it (xargsView), and each command a find runs
// (findActions), with the commands of a string either hands a shell. Where xargs
// adds words it adds two, since its input may hold the verb and its operand. A
// program of a wholly unknown name is read as neither xargs nor find.
func suppliedReadings(s segment) []segment {
	var runs []segment
	xargs := -1
	for _, a := range arrivalsOf(s) {
		if tok := s.tokens[a.idx]; a.wrapper && xargs < 0 && !anyProgram(tok) && commandNamed(s, a, "xargs") {
			xargs = a.idx
		}
	}
	viewed := false
	for _, a := range commandSites(s) {
		if !anyProgram(s.tokens[a.idx]) && commandNamed(s, a, "find") {
			for _, run := range findActions(s, a.idx+1) {
				runs = append(runs, run.seg)
			}
		}
		if xargs >= 0 && a.idx > xargs && !viewed {
			viewed = true
			v := carriedCopy(xargsView(s, a.idx))
			if len(v.tokens) > len(s.tokens) {
				v.tokens = append(v.tokens, unknownText)
			}
			runs = append(runs, v)
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
// the line spells the entry's verb.
func suppliedMatch(p Pattern, readings []segment) (hit, named bool) {
	for _, r := range readings {
		h, n := matchSegmentNamed(p, r)
		if n {
			return true, true
		}
		hit = hit || h && containsString(r.tokens, p.Subcommand)
	}
	return hit, false
}
