package guard

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

// The carried-value reading (iss-2609281134544802, ruling AT).
//
// A parameter expansion is an unknown word (unknown.go): the guard reads what
// its value can spell as a flag or a program name, but not what an earlier
// command put in it, because a static reading cannot know a variable's value
// at run time. One place it can: a value the same text assigns before it uses
// it. There the use is read once more with the value in place, as another
// reading of the command (segment.fromFixedOutput), so every verdict the
// unknown reading reaches still stands and the value adds what it holds:
//
//   - a pid list a search printed (`p=$(pgrep make); kill $p`, or
//     `pgrep make | while read p; do kill $p; done`) reaches kill as the
//     substitution's output does, which the kill entries read by its source;
//   - a stream path (`f=<(curl …); bash "$f"`) is the stream it names;
//   - text a string runs (`X='…'; eval "$X"`) is judged as the command it
//     holds, and output the guard did not read, run as a string, is refused
//     (carriedOutputSignal);
//   - a program name (`P=pkill; $P make`) is that program.
//
// A variable the text does not assign is read as it always was, so the pid a
// caller recorded earlier (`kill "$pid"`) and a program named by a variable
// (`"$GO" build`) stay allowed. Which assignment reaches which use is not
// modelled past order — a subshell's, a branch's or a loop's assignment is
// read as reaching every later use — so a name assigned more than once is
// read once with each of its values (maxCarriedValues).

// carriedValue is one value an assignment gave a variable, as the word it
// left: tok holds unknownMark where a substitution's output goes; tmpl is the
// value spelled with varMark at each variable it holds, set only when every
// unknown part of it is a variable's (segment.variable); spell is the value
// as the line wrote its variables (segment.spelled), "" where it cannot say;
// feeds names the commands whose output it holds, as segment.feeds does; and
// output records that it holds output the guard did not read.
type carriedValue struct {
	tok    string
	tmpl   string
	spell  string
	feeds  []feed
	output bool
}

// maxCarriedValues bounds the values one name is read with. A name assigned
// more often holds the first maxCarriedValues-1 values and then one that
// stands for the rest: output the guard did not read, fed by every command
// any of them held, so a value past the bound is never dropped unread.
const maxCarriedValues = 8

// declarers assign each NAME=VALUE operand as a plain assignment does.
var declarers = map[string]bool{"export": true, "declare": true, "typeset": true, "local": true, "readonly": true}

// readValueFlags are read's options that take the next word: `-a` names the
// array it fills, so its value is a name read assigns.
var readValueFlags = map[string]bool{"-a": true, "-d": true, "-i": true, "-n": true, "-N": true, "-p": true, "-t": true, "-u": true}

// carriedReadings returns, per index of segs, the readings of that segment
// with the values the commands before it assigned put in place, and the
// signals those readings raise. segs is one text's commands in order, and
// each command whose words hold a value records the commands that value's
// output came from (segment.carries) in place, where the runs that name it
// read it.
func carriedReadings(segs []segment) (map[int][]segment, []payloadSignal) {
	fileCarries(segs)
	var vals map[string][]carriedValue
	var out map[int][]segment
	var signals []payloadSignal
	ends := loopEnds(segs)
	for k, s := range segs {
		readings := []segment{s}
		if len(vals) > 0 && len(s.variable) > 0 {
			ds, held, runsOutput := carriedSegments(s, vals)
			// The output a value holds is what this command's words hold,
			// so a command it prints into reads it too: `echo "$p" | xargs
			// kill` hands kill what pgrep printed.
			segs[k].carries = mergeRuns(append(segs[k].carries, held...))
			if len(ds) > 0 {
				if out == nil {
					out = map[int][]segment{}
				}
				out[k] = ds
				readings = ds
			}
			if runsOutput {
				signals = append(signals, carriedOutputSignal())
			}
		}
		var loopIn []feed
		if e, ok := ends[k]; ok {
			loopIn = append(wordFeeds(segs[e], allWords), segs[e].carries...)
		}
		for _, r := range readings {
			vals = recordAssignments(r, vals, loopIn)
		}
	}
	return out, signals
}

// loopEnds maps each `while` or `until` command of segs to the `done` that
// closes its loop, whose redirections are the loop's: `while read p; do …;
// done < <(pgrep make)` reads pgrep's output in its condition.
func loopEnds(segs []segment) map[int]int {
	var ends map[int]int
	var open []int
	for k, s := range segs {
		if len(s.tokens) == 0 {
			continue
		}
		switch s.tokens[0] {
		case "while", "until", "for", "select":
			open = append(open, k)
		case "done":
			if len(open) == 0 {
				continue
			}
			o := open[len(open)-1]
			open = open[:len(open)-1]
			if t := segs[o].tokens[0]; t == "while" || t == "until" {
				if ends == nil {
					ends = map[int]int{}
				}
				ends[o] = k
			}
		}
	}
	return ends
}

// allWords keeps every word of a segment (wordFeeds).
func allWords(int) bool { return true }

// recordAssignments adds the values s assigns to vals, and drops the names it
// unsets. loopIn is what reaches s's standard input from the loop it is the
// condition of, nil when it is none.
func recordAssignments(s segment, vals map[string][]carriedValue, loopIn []feed) map[string][]carriedValue {
	add := func(name string, v carriedValue) {
		if vals == nil {
			vals = map[string][]carriedValue{}
		}
		vals[name] = addValue(vals[name], v)
	}
	word := func(i, from int) carriedValue {
		v := carriedValue{tok: s.tokens[i][from:], feeds: s.feeds[i]}
		if tmpl, ok := s.variable[i]; ok && len(tmpl) >= from {
			v.tmpl = tmpl[from:]
		}
		if sp := s.spelled[i]; len(sp) == 1 && len(sp[0]) >= from {
			v.spell = sp[0][from:]
		}
		v.output = isUnknown(v.tok) && v.tmpl == ""
		return v
	}
	assign := func(i int) {
		tok := s.tokens[i]
		if !isAssignment(tok) {
			return
		}
		eq := strings.IndexByte(tok, '=')
		add(tok[:eq], word(i, eq+1))
	}
	if s.arrayOf != "" {
		return vals
	}
	all := len(s.tokens) > 0
	for _, t := range s.tokens {
		all = all && isAssignment(t)
	}
	if all {
		for i := range s.tokens {
			assign(i)
		}
		return vals
	}
	// `for NAME in WORDS`: the name takes each word in turn.
	if len(s.tokens) >= 3 && s.tokens[0] == "for" && s.tokens[2] == "in" && isName(s.tokens[1]) {
		for i := 3; i < len(s.tokens); i++ {
			add(s.tokens[1], word(i, 0))
		}
		return vals
	}
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		if isUnknown(tok) {
			continue
		}
		switch name := path.Base(tok); {
		case declarers[name]:
			for i := a.idx + 1; i < len(s.tokens); i++ {
				assign(i)
			}
		case name == "unset":
			for _, w := range s.tokens[a.idx+1:] {
				delete(vals, w)
			}
		case name == "read":
			// What read assigns is a line of its standard input: output the
			// guard did not read, fed by whatever reaches that input — a
			// pipe, a group's input, a process substitution or here-string
			// in its own words or its loop's.
			v := carriedValue{tok: unknownText, output: true}
			if s.piped.list != nil {
				v.feeds = append(v.feeds, s.piped)
			}
			v.feeds = append(v.feeds, s.stdinIn...)
			v.feeds = append(v.feeds, wordFeeds(s, allWords)...)
			v.feeds = append(v.feeds, s.carries...)
			v.feeds = append(v.feeds, loopIn...)
			args := s.tokens[a.idx+1:]
			for i := 0; i < len(args); i++ {
				w := args[i]
				switch {
				case w == "-a" && i+1 < len(args):
					i++
					if isName(args[i]) {
						add(args[i], v)
					}
				case readValueFlags[w]:
					i++
				case strings.HasPrefix(w, "-"):
				case isName(w):
					add(w, v)
				}
			}
		}
	}
	return vals
}

// addValue appends v to a name's values, folding every value past the bound
// into one that stands for output the guard did not read.
func addValue(vs []carriedValue, v carriedValue) []carriedValue {
	for _, have := range vs {
		if have.tok == v.tok && have.tmpl == v.tmpl && len(have.feeds) == 0 && len(v.feeds) == 0 {
			return vs
		}
	}
	if len(vs) < maxCarriedValues {
		return append(vs, v)
	}
	last := vs[len(vs)-1]
	out := append([]carriedValue(nil), vs[:len(vs)-1]...)
	merged := carriedValue{tok: unknownText, output: true,
		feeds: mergeRuns(append(append([]feed(nil), last.feeds...), v.feeds...))}
	return append(out, merged)
}

// isName reports whether w is a shell variable name.
func isName(w string) bool {
	return w != "" && isAssignment(w+"=")
}

// carriedUse is one word of a segment whose every variable the text assigned:
// its index, and the name at each of its sites in order.
type carriedUse struct {
	i     int
	names []string
}

// carriedSegments returns the readings of s with each word whose every
// variable the text assigned spelled with those values, one reading per value
// a name holds, the commands whose output those values hold, and whether a
// value holding output the guard did not read reaches a string s runs.
func carriedSegments(s segment, vals map[string][]carriedValue) ([]segment, []feed, bool) {
	idx := make([]int, 0, len(s.variable))
	for i := range s.variable {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var uses []carriedUse
	n := 0
	for _, i := range idx {
		tmpl := s.variable[i]
		if tmpl == "" || i >= len(s.tokens) || len(s.spelled[i]) != 1 {
			continue
		}
		names, ok := siteNames(tmpl, s.spelled[i][0])
		if !ok {
			continue
		}
		for _, nm := range names {
			if len(vals[nm]) == 0 {
				ok = false
				break
			}
			n = max(n, len(vals[nm]))
		}
		if ok {
			uses = append(uses, carriedUse{i: i, names: names})
		}
	}
	if len(uses) == 0 {
		return nil, nil, false
	}
	var held []feed
	for _, u := range uses {
		for _, nm := range u.names {
			for _, v := range vals[nm] {
				held = append(held, v.feeds...)
			}
		}
	}
	var out []segment
	runsOutput := false
	for k := 0; k < n; k++ {
		d := carriedCopy(s)
		var outputAt []int
		for _, u := range uses {
			picked := make([]carriedValue, len(u.names))
			output := false
			for j, nm := range u.names {
				vs := vals[nm]
				picked[j] = vs[min(k, len(vs)-1)]
				output = output || picked[j].output
			}
			placeValues(&d, u.i, s.variable[u.i], u.names, picked)
			if output {
				outputAt = append(outputAt, u.i)
			}
		}
		out = append(out, d)
		if len(outputAt) > 0 && !runsOutput {
			runsOutput = outputRunAsString(s, outputAt)
		}
	}
	return out, held, runsOutput
}

// siteNames pairs a word's variable template with its spelling and returns
// the name each site spells, in order. ok is false unless every site is a
// plain `$NAME` or `${NAME}`: a default, a transform or a special parameter
// is not a value the text assigned.
func siteNames(tmpl, spell string) (names []string, ok bool) {
	j := 0
	for i := 0; i < len(tmpl); i++ {
		if tmpl[i] != varMark {
			if j >= len(spell) || spell[j] != tmpl[i] {
				return nil, false
			}
			j++
			continue
		}
		if j >= len(spell) || spell[j] != '$' {
			return nil, false
		}
		j++
		braced := j < len(spell) && spell[j] == '{'
		if braced {
			j++
		}
		start := j
		for j < len(spell) && (spell[j] == '_' || spell[j] >= 'a' && spell[j] <= 'z' || spell[j] >= 'A' && spell[j] <= 'Z' || j > start && spell[j] >= '0' && spell[j] <= '9') {
			j++
		}
		if j == start {
			return nil, false
		}
		names = append(names, spell[start:j])
		if braced {
			if j >= len(spell) || spell[j] != '}' {
				return nil, false
			}
			j++
		}
	}
	return names, j == len(spell)
}

// carriedCopy is s as a reading of its own: its per-word records copied so a
// word can be rewritten, and its walk and its place in a tokenize call's
// output dropped, since its words are not the ones that call emitted.
func carriedCopy(s segment) segment {
	d := s
	d.tokens = append([]string(nil), s.tokens...)
	d.variable = copyMap(s.variable)
	d.spelled = copyMap(s.spelled)
	d.feeds = copyMap(s.feeds)
	d.ifsSplit = copyMap(s.ifsSplit)
	d.varLead = copyMap(s.varLead)
	d.leadFrom = copyMap(s.leadFrom)
	d.arrivals, d.walked, d.walkCapped = nil, false, false
	d.home, d.at = nil, 0
	d.fromFixedOutput, d.valuesPlaced = true, true
	return d
}

func copyMap[V any](m map[int]V) map[int]V {
	if m == nil {
		return nil
	}
	out := make(map[int]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// placeValues rewrites word i of d, whose template is tmpl, with vs in place
// of its sites in order, the site at each standing for names[j]: the word the
// shell builds. A value's own text is read as text, and a stream path as the
// stream it names; a value holding output, a variable's or a substitution's,
// is read as a variable's value is, by its known text and as data, with the
// commands its output came from as the word's feeds. So output standing as
// a program name or a script reads as it did before the value was followed
// (variableCarried), and what it adds is what those commands printed: a kill
// handed pgrep's output by way of a variable is a kill by name.
func placeValues(d *segment, i int, tmpl string, names []string, vs []carriedValue) {
	var tok, vt, sp strings.Builder
	j := 0
	for k := 0; k < len(tmpl); k++ {
		c := tmpl[k]
		if c != varMark {
			tok.WriteByte(c)
			vt.WriteByte(c)
			sp.WriteByte(c)
			continue
		}
		v := vs[j]
		tok.WriteString(v.tok)
		switch {
		case v.tmpl != "":
			vt.WriteString(v.tmpl)
		default:
			vt.WriteString(strings.ReplaceAll(v.tok, unknownText, varText))
		}
		switch {
		case v.spell != "":
			sp.WriteString(v.spell)
		case !isUnknown(v.tok):
			sp.WriteString(v.tok)
		default:
			sp.WriteString("${" + names[j] + "}")
		}
		j++
		if len(v.feeds) > 0 {
			if d.feeds == nil {
				d.feeds = map[int][]feed{}
			}
			d.feeds[i] = append(append([]feed(nil), d.feeds[i]...), v.feeds...)
		}
	}
	d.tokens[i] = tok.String()
	delete(d.ifsSplit, i)
	delete(d.varLead, i)
	delete(d.leadFrom, i)
	if isUnknown(d.tokens[i]) {
		d.variable[i] = vt.String()
		d.spelled[i] = []string{sp.String()}
	} else {
		delete(d.variable, i)
		delete(d.spelled, i)
	}
}

// outputRunAsString reports whether a word at one of the indices at reaches a
// command string s hands a shell, eval or another runner: read with only those
// words spelled as their variables, a string some runner takes holds a
// variable's mark.
func outputRunAsString(s segment, at []int) bool {
	v := s
	v.tokens = append([]string(nil), s.tokens...)
	for _, i := range at {
		v.tokens[i] = s.variable[i]
	}
	for _, ref := range payloadsOf(v) {
		if strings.IndexByte(ref.payload, varMark) >= 0 {
			return true
		}
		for _, t := range ref.trailing {
			if strings.IndexByte(t, varMark) >= 0 {
				return true
			}
		}
	}
	return false
}

// Output carried through a file (iss-2610112214257996, in the spirit of ruling
// AS: a pid list that came from a search blocks, a recorded pid does not).
//
// A command that reads a file an earlier command of the same text wrote — as
// its standard input (`< p`, `$(< p)`) or as an operand naming it (`cat p`,
// `xargs -a p`) — reads what that command printed, so it carries it
// (segment.carries) as a pipe from it would. The file is matched by the name
// the line writes it under, so `pgrep make > p; kill $(cat p)` is a kill fed
// by the search and `kill $(cat pidfile)`, a pid file the line did not write,
// is not. A file written under another spelling of the same path, through a
// variable, or by a command that names it as an operand (`tee p`) is not
// followed.
//
// A word whose unknown parts are only variables the line does not set reads
// as such a word always has (ruling AT): a value from before the line, a
// recorded pid, not any file the line wrote (iss-2610120916220240). It reads
// a file written under a name the line does not fix, which can be spelled by
// the same variable, and one whose name its fixed text can still spell
// (`$HOME/p` for `~/p`); a bare `$pid` names none.
//
// A ps listing is not a search, since `ps -o pid= -p N` reports the pid the
// caller names; a grep, or an awk program that matches, filtering a ps
// listing by a name is (segment.processSearch).

// fileCarries records, in place, what each command of segs reads from a file
// an earlier one wrote, and which commands filter a ps listing by a name. A
// file name the line does not fix, written or read, can be any file, so a read
// of one reads every file written before it, and a write to one reaches every
// read after it; those are held as one covering run per call (mergeRuns), so
// a line of many writes and reads costs what its commands do.
func fileCarries(segs []segment) {
	var written map[string][]feed
	var wild, all []feed
	var assigned map[string]bool
	// fetched starts from what the string's carrier was handed, and is
	// copied before it grows, so each command keeps the set as of itself.
	var fetched *fetchedSet
	if len(segs) > 0 {
		fetched = segs[0].fetchedIn
	}
	fetch := func(name string, wildName bool) {
		n := &fetchedSet{names: map[string]bool{}}
		if fetched != nil {
			for k := range fetched.names {
				n.names[k] = true
			}
			n.wild = fetched.wild
		}
		if wildName {
			n.wild = true
		} else {
			n.names[path.Clean(name)] = true
		}
		fetched = n
	}
	isFetched := func(w string) bool {
		if isUnknown(w) || strings.ContainsRune(w, varMark) {
			return fetched.wild
		}
		return fetched.names[path.Clean(w)]
	}
	for k := range segs {
		s := &segs[k]
		if fetched != nil {
			s.runsFetched = s.runsFetched || runsFetchedFile(*s, isFetched)
		}
		names, wildName := fetchWrites(*s)
		for _, n := range names {
			fetch(n, false)
		}
		if wildName && (fetched == nil || !fetched.wild) {
			fetch("", true)
		}
		if len(all) > 0 {
			if assigned == nil {
				assigned = namesAssigned(segs)
			}
			var got []feed
			read := func(name string, fixed bool) {
				if !fixed {
					got = append(got, all...)
					return
				}
				got = append(append(got, written[name]...), wild...)
			}
			for _, r := range s.redirects {
				if (r.op == "<" || r.op == "<>") && (r.fd == -1 || r.fd == 0) {
					read(r.target.text, r.target.ok)
				}
			}
			for i, tok := range s.tokens {
				tally(1)
				if isUnknown(tok) && onlyOutsideVariables(*s, i, assigned) {
					got = append(got, wild...)
					if knownText(tok) == "" {
						continue
					}
					for name, fs := range written {
						tally(1)
						// A relative name is also the file a word whose
						// fixed tail is `/NAME` names (`$PWD/p` for p).
						if unknownCouldBe(tok, name) || !strings.HasPrefix(name, "/") && unknownCouldBe(tok, "/"+name) {
							got = append(got, fs...)
						}
					}
					continue
				}
				read(tok, !isUnknown(tok))
			}
			s.carries = append(s.carries, mergeRuns(got)...)
		}
		s.processSearch = filtersProcessList(*s)
		for _, r := range s.redirects {
			if !writesOutput(r) {
				continue
			}
			fs := outputOf(*s)
			if fetches(*s) || runsFetch(fs) {
				switch {
				case !r.target.ok:
					fetch("", true)
				case r.target.text != "/dev/null":
					fetch(r.target.text, false)
				}
			}
			all = mergeRuns(append(all, fs...))
			if !r.target.ok {
				wild = mergeRuns(append(wild, fs...))
				continue
			}
			if written == nil {
				written = map[string][]feed{}
			}
			written[r.target.text] = mergeRuns(append(written[r.target.text], fs...))
		}
		s.fetched = fetched
	}
}

// onlyOutsideVariables reports whether every unknown part of word i of s is
// a plain variable (siteNames) that assigned does not hold: a value from
// before the line.
func onlyOutsideVariables(s segment, i int, assigned map[string]bool) bool {
	tmpl, ok := s.variable[i]
	if !ok || tmpl == "" || len(s.spelled[i]) != 1 {
		return false
	}
	names, ok := siteNames(tmpl, s.spelled[i][0])
	if !ok {
		return false
	}
	for _, n := range names {
		if assigned[n] {
			return false
		}
	}
	return true
}

// namesAssigned is every name a command of segs can assign: by an assignment
// word anywhere in it (a prefix, a declarer's operand, `+=`), an array
// assignment, a `for` or `select` loop, a `${NAME:=w}` or `${NAME=w}`
// anywhere in the text, REPLY, which a bare read and a select set, and any
// name-shaped word of a
// command that assigns its operands (read, mapfile, getopts, printf -v and
// the rest), or of one whose name the line does not fix. It reaches past
// what the shell assigns, which only keeps a word read as any file.
func namesAssigned(segs []segment) map[string]bool {
	out := map[string]bool{}
	for _, s := range segs {
		if s.arrayOf != "" {
			out[s.arrayOf] = true
		}
		if len(s.tokens) >= 2 && (s.tokens[0] == "for" || s.tokens[0] == "select") {
			out[s.tokens[1]] = true
		}
		// select sets REPLY to the line it reads, as a bare read does.
		if len(s.tokens) > 0 && s.tokens[0] == "select" {
			out["REPLY"] = true
		}
		for _, n := range s.assignsInText {
			out[n] = true
		}
		for _, t := range s.tokens {
			tally(1)
			if eq := strings.IndexByte(t, '='); eq > 0 && isName(strings.TrimSuffix(t[:eq], "+")) {
				out[strings.TrimSuffix(t[:eq], "+")] = true
			}
		}
		for _, a := range commandSites(s) {
			tok := s.tokens[a.idx]
			if !isUnknown(tok) && !nameCouldBeAny(tok, operandAssigners) {
				continue
			}
			if nameCouldBe(tok, "read") {
				out["REPLY"] = true
			}
			for _, w := range s.tokens[a.idx+1:] {
				if isName(w) {
					out[w] = true
				}
			}
		}
	}
	return out
}

// operandAssigners are the builtins that assign the names they are handed.
var operandAssigners = []string{"read", "mapfile", "readarray", "getopts", "printf", "let", "local",
	"declare", "typeset", "export", "readonly", "vared", "zparseopts", "print"}

// A script downloaded to a file and run on the same line
// (iss-2610120916248850) is the stream `curl … | sh` is, behind a file name:
// what runs is text the guard never read, and the file it could read at
// check time is not the one the download leaves. fileCarries records the
// names the line's fetches write — a redirection of output a fetch printed,
// curl's -o and -O, wget's -O or the name it saves under, a tee fed by a
// fetch — and marks a later command that runs one (segment.runsFetched): a
// shell's script or standard input, a `source` or `.` operand, or a path
// run directly. Names are matched as the line writes them, as the pid-list
// carry matches them; a name the line does not fix can be any file.

// fetchedSet is the names the line's fetches wrote, and whether one wrote a
// name the line does not fix, which can be any file. It is never changed
// once a command holds it.
type fetchedSet struct {
	names map[string]bool
	wild  bool
}

// fetchers are the programs that write what a network fetch returns.
var fetchers = []string{"curl", "wget"}

// fetches reports whether s itself runs a fetch.
func fetches(s segment) bool {
	for _, a := range commandSites(s) {
		if nameCouldBeAny(s.tokens[a.idx], fetchers) {
			return true
		}
	}
	return false
}

// runsFetch reports whether any command of runs is a fetch.
func runsFetch(runs []feed) bool {
	for _, f := range runs {
		for i := f.lo; i < f.hi && i < len(f.list.segs); i++ {
			tally(1)
			if fetches(f.list.segs[i]) {
				return true
			}
		}
	}
	return false
}

// fetchWrites is the files s's commands write from a fetch by naming them
// (redirections are fileCarries'): curl's -o value or, with -O, each URL's
// last path segment; wget's -O value, else each URL's last path segment
// under its -P directory; and every operand of a tee a fetch feeds. wild
// reports a name the line does not fix. A value of `-` is standard output.
func fetchWrites(s segment) (names []string, wild bool) {
	add := func(w string) {
		switch {
		case isUnknown(w) || strings.ContainsRune(w, varMark):
			wild = true
		case w != "-" && w != "" && w != "/dev/null":
			names = append(names, w)
		}
	}
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		args := s.tokens[a.idx+1:]
		switch {
		case isUnknown(tok):
		case nameCouldBe(tok, "curl"):
			out, remote, urls := fetchArgs(args, 'o', "--output", 'O', "--remote-name", curlValueShorts)
			for _, o := range out {
				add(o)
			}
			if remote || containsString(args, "--remote-name-all") {
				for _, u := range urls {
					add(urlBase(u))
				}
			}
		case nameCouldBe(tok, "wget"):
			out, _, urls := fetchArgs(args, 'O', "--output-document", 0, "", wgetValueShorts)
			for _, o := range out {
				add(o)
			}
			if len(out) == 0 {
				dir, _, _ := fetchArgs(args, 'P', "--directory-prefix", 0, "", wgetValueShorts)
				for _, u := range urls {
					b := urlBase(u)
					add(b)
					if len(dir) > 0 && !isUnknown(b) {
						add(path.Join(dir[len(dir)-1], b))
					}
				}
			}
		case nameCouldBe(tok, "tee"):
			runs := append([]feed(nil), s.stdinIn...)
			if s.piped.list != nil {
				runs = append(runs, s.piped)
			}
			if !runsFetch(runs) {
				continue
			}
			for _, w := range args {
				if !strings.HasPrefix(w, "-") {
					add(w)
				}
			}
		}
	}
	return names, wild
}

// curlValueShorts and wgetValueShorts are the short options of each that
// take a value, so the rest of a cluster after one is its value.
const (
	curlValueShorts = "AbcCdDeEFHKmoPQrtTuUwxXyYz"
	wgetValueShorts = "aABdDeiIlOoPQRtTUwX"
)

// fetchArgs walks a fetch's arguments: the values of the short option short
// or the long option long (glued, `=`-joined or the next word), whether the
// short option flag or the long option flagLong is given, and the URLs.
func fetchArgs(args []string, short byte, long string, flag byte, flagLong string, valueShorts string) (vals []string, flagged bool, urls []string) {
	for i := 0; i < len(args); i++ {
		w := args[i]
		tally(1)
		switch {
		case isUnknown(w) && (knownLead(w) == "" || strings.HasPrefix(w, "-")):
			// A word the line does not fix may be the option itself, its
			// value unknown.
			vals = append(vals, unknownText)
		case w == "--":
			for _, u := range args[i+1:] {
				if strings.Contains(u, "://") {
					urls = append(urls, u)
				}
			}
			return vals, flagged, urls
		case long != "" && w == long:
			if i+1 < len(args) {
				vals = append(vals, args[i+1])
				i++
			}
		case long != "" && strings.HasPrefix(w, long+"="):
			vals = append(vals, w[len(long)+1:])
		case flagLong != "" && w == flagLong:
			flagged = true
		case len(w) > 1 && w[0] == '-' && w[1] != '-':
			for j := 1; j < len(w); j++ {
				c := w[j]
				if c == flag {
					flagged = true
					continue
				}
				if strings.IndexByte(valueShorts, c) < 0 {
					continue
				}
				v := w[j+1:]
				if v == "" && i+1 < len(args) {
					v = args[i+1]
					i++
				}
				if c == short {
					vals = append(vals, v)
				}
				break
			}
		case strings.Contains(w, "://"):
			urls = append(urls, w)
		}
	}
	return vals, flagged, urls
}

// urlBase is the last path segment of a URL, the name a fetch saves it under;
// unknownText where the line does not fix it.
func urlBase(u string) string {
	if isUnknown(u) || strings.ContainsRune(u, varMark) {
		return unknownText
	}
	p, err := url.Parse(u)
	if err != nil {
		return unknownText
	}
	b := path.Base(p.Path)
	if b == "/" || b == "." {
		return "index.html"
	}
	return b
}

// runsFetchedFile reports whether s runs a file isFetched names as a script:
// a shell's script operand, `--rcfile`/`--init-file` value or standard
// input, a `source` or `.` operand, or a path run directly.
func runsFetchedFile(s segment, isFetched func(string) bool) bool {
	stdin := false
	for _, r := range s.redirects {
		if (r.op == "<" || r.op == "<>") && (r.fd == -1 || r.fd == 0) && isFetched(r.target.text) {
			stdin = true
		}
	}
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		args := s.tokens[a.idx+1:]
		carried := func(i int) bool { return variableCarried(s, a.idx+1+i) }
		switch {
		case nameCouldBeAny(tok, shellFamily) && shellReadsStream(args, stdin, carried, isFetched):
			return true
		case nameCouldBeAny(tok, sourceBuiltins) && sourceReadsStream(args, stdin, carried, isFetched):
			return true
		case !isUnknown(tok) && strings.Contains(tok, "/") && isFetched(tok):
			return true
		}
	}
	return false
}

// mergeRuns holds runs as one covering run per tokenize call they name: the
// commands between two runs of one call are read with them, which can only
// add a source, never drop one.
func mergeRuns(fs []feed) []feed {
	if len(fs) < 2 {
		return fs
	}
	var out []feed
	for _, f := range fs {
		tally(1)
		merged := false
		for i := range out {
			if out[i].list == f.list {
				out[i].lo, out[i].hi = min(out[i].lo, f.lo), max(out[i].hi, f.hi)
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, f)
		}
	}
	return out
}

// writesOutput reports whether a redirection sends a command's standard
// output into the file it names.
func writesOutput(r redirect) bool {
	switch r.op {
	case ">", ">>", ">|":
		return r.fd == -1 || r.fd == 1
	case "&>", "&>>":
		return true
	case ">&":
		return (r.fd == -1 || r.fd == 1) && strings.Trim(r.target.text, "0123456789-") != ""
	}
	return false
}

// outputOf is every run whose output can reach what s prints: s itself, what
// it reads on its standard input, and what its words hold. What s carries is
// read through s itself (segmentHits reads its carries), so it is copied only
// for a command no tokenize call holds.
func outputOf(s segment) []feed {
	var out []feed
	if s.home != nil {
		out = append(out, feed{list: s.home, lo: s.at, hi: s.at + 1})
	} else {
		out = append(out, s.carries...)
	}
	if s.piped.list != nil {
		out = append(out, s.piped)
	}
	out = append(out, s.stdinIn...)
	return append(out, wordFeeds(s, allWords)...)
}

// grepNames and awkNames are the filters that pick lines of a listing by
// what they hold.
var (
	grepNames = []string{"grep", "egrep", "fgrep", "rg"}
	awkNames  = []string{"awk", "gawk", "mawk", "nawk"}
)

// filtersProcessList reports whether s is a grep, or an awk whose program
// matches (awkMatches), reading a ps listing: a search by name, as pgrep is.
func filtersProcessList(s segment) bool {
	filter := false
	for _, a := range commandSites(s) {
		tok := s.tokens[a.idx]
		if nameCouldBeAny(tok, grepNames) || nameCouldBeAny(tok, awkNames) && awkMatches(s.tokens[a.idx+1:]) {
			filter = true
			break
		}
	}
	if !filter {
		return false
	}
	runs := append([]feed(nil), s.stdinIn...)
	if s.piped.list != nil {
		runs = append(runs, s.piped)
	}
	runs = append(runs, s.carries...)
	for _, f := range runs {
		for i := f.lo; i < f.hi && i < len(f.list.segs); i++ {
			tally(1)
			for _, a := range commandSites(f.list.segs[i]) {
				if nameCouldBe(f.list.segs[i].tokens[a.idx], "ps") {
					return true
				}
			}
		}
	}
	return false
}

// awkMatches reports whether an awk command's program selects lines by what
// they hold: a regular expression, a match operator or an equality test. A
// program read from a file (`-f`), or a word the line does not fix before or
// as the program, is taken as one that does.
func awkMatches(args []string) bool {
	for i := 0; i < len(args); i++ {
		w := args[i]
		switch {
		case w == "-f" || strings.HasPrefix(w, "--file") || isUnknown(w):
			return true
		case w == "-F" || w == "-v":
			i++
		case w == "--":
		case strings.HasPrefix(w, "-") && len(w) > 1:
		default:
			for _, m := range []string{"/", "~", "==", "index(", "match("} {
				if strings.Contains(w, m) {
					return true
				}
			}
			return false
		}
	}
	return false
}

// processSearchReading is how a command that filters a ps listing by a name
// is read by an entry that names its sources (segmentHits): as the pgrep
// whose search it makes.
var processSearchReading = segment{tokens: []string{"pgrep", "name"}}

// carriedOutputSignal is the verdict on a line that runs, as a command string,
// a variable it set from output the guard did not read (ruling AT).
func carriedOutputSignal() payloadSignal {
	return payloadSignal{
		verdict: VerdictBlock,
		family:  familyShell,
		reason: "This line runs, as a command string, a variable it set from output the guard did not read — a command substitution or a `read` — " +
			"so the command that string runs is not something the guard has checked.",
		successor: "Write the command out where it runs (`sh -c '<commands>'`, or the command itself) so the guard reads it; " +
			"if it comes from a file or a download, read it yourself first.",
	}
}
