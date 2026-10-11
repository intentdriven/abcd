package guard

import (
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
			segs[k].carries = append(segs[k].carries, held...)
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
			loopIn = wordFeeds(segs[e], allWords)
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
		feeds: append(append([]feed(nil), last.feeds...), v.feeds...)}
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
