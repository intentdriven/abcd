package guard

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// The unknown word — one reading, in one place, of a word the guard cannot
// know.
//
// A command substitution (`$( … )`, or its backtick spelling) runs a command
// and hands its OUTPUT to the word it sits in, and the output is not in the
// command line. The tokenizer therefore writes unknownMark into the word where
// the output goes, and every reader of a token asks this file what the word
// can be. An arithmetic expansion is not unknown in that sense: its output is a
// number, which no flag, subcommand or path the registry names can be.
//
// The rule is that an unknown word fails closed in every role it could play,
// and a reader that can read a word more than one way reads it every way — the
// union of the readings, never the first one:
//
//   - A word whose own text begins with a dash (`--$(x)`, `-r$(x)`,
//     `--"$(x)"`) is a flag of unknown name, and stands for every flag
//     alternative its known text can still become (unknownFlagCouldBe): one
//     that stands alone, one that takes the word after it as its value
//     (readWord), a shell's `-c` (clusterCouldCarry), a verb's payload flag
//     (flagCouldBe). It is never the `--` terminator, which is spelled with no
//     substitution.
//   - A word whose output may be empty is also the word its known text spells
//     (`"$(x)"-C` is `-C`), and one that is nothing but a substitution may be
//     no word at all (vanishable): `git $(true) push --force` is a push.
//   - As the value of a value flag it fills the value slot, and the value flag
//     never consumes the word after it.
//   - As an operand it is one operand of unknown value: it counts toward
//     min_operands, it keeps the operands after it in their positions, and a
//     subcommand compare at its position matches whatever its known text still
//     allows (wordCouldBe).
//   - In command position it is a program of unknown name: every program whose
//     name its known tail allows (nameCouldBe) — each entry's command, a shell
//     whose `-c` the payload reading opens, an exec-string verb, `env`, a
//     directory change — and a wrapper of unknown grammar, whose own options may
//     each take a value and whose command may follow them (commandArrivals).
//
// The walks that apply the rule live here too — to command position
// (commandArrivals) and over a command's operands (operandAcceptance,
// operandReadings) — so a reader asks this file where a command is and which
// words are its operands rather than stepping words itself. A test holds the
// package to it (unknownreaders_test.go): every function that reads a word's
// dash or a command's name is listed there with the rule it goes through.
//
// What stays deliberately outside the rule is a word that is WHOLLY a
// substitution standing where a flag could be: it is read as an operand, not as
// every flag, because that is how an everyday command spells its commit message
// and its branch (`git commit -m "$(cat msg)"`, `git push origin
// "$(git branch --show-current)"`), and reading it as every flag would refuse
// both. The same reason keeps an operand's `+` refspec prefix read from its
// known text only. Both residuals are recorded in .abcd/work/DECISIONS.md.

// unknownMark stands, inside a token, for the output of a substitution the
// guard did not run. It is the NUL byte, and it is unforgeable by construction:
// Check drops every NUL the command line itself carries before it reads a
// word, and the one decoder that could make a NUL from other bytes — an ANSI-C
// `$'…'` escape — ends its string at the first decoded NUL, as bash does
// (readAnsiCQuote). No other byte reaches a word decoded.
const unknownMark = '\x00'

// unknownText is unknownMark as a string, for building tokens.
const unknownText = "\x00"

// isUnknown reports whether a word carries a substitution's output.
func isUnknown(tok string) bool { return strings.IndexByte(tok, unknownMark) >= 0 }

// knownText is the word with every substitution's output taken as empty — the
// vanish reading, under which `"$(true)"--force` is `--force`.
func knownText(tok string) string {
	if !isUnknown(tok) {
		return tok
	}
	return strings.ReplaceAll(tok, unknownText, "")
}

// knownLead is the word's text before its first substitution: the part of it
// that is fixed whatever the substitution prints.
func knownLead(tok string) string {
	if i := strings.IndexByte(tok, unknownMark); i >= 0 {
		return tok[:i]
	}
	return tok
}

// unknownFromOpenExpansion reads a word in which a substitution's output lands
// inside a parameter expansion — `${X:-$(x)}`, `--${X:-$(x)}` — as unknown
// from that expansion's `${` on (review4-guard finding 3). What such an
// expansion prints is the substitution's output or the variable's value, and
// the text written after the output up to the closing `}` is its own syntax,
// not text beside the output: read as fixed, the `}` made a command name that
// could only end in a brace and a flag that could only be one that did. The
// outermost `${` still open where a mark lands starts the unknown; the text
// before it is kept, so `--${X:-$(x)}` is a dash-word and `${X:-$(x)}` a
// word that is wholly unknown. A `${…}` that closes before any mark, and a
// `$X` with no substitution in it, are left as written: that is the half
// iss-2609251824244354 defers.
func unknownFromOpenExpansion(tok string) string {
	if !isUnknown(tok) {
		return tok
	}
	depth, outer := 0, -1
	for i := 0; i < len(tok); i++ {
		switch {
		case tok[i] == unknownMark && depth > 0:
			return tok[:outer] + unknownText
		case tok[i] == '$' && i+1 < len(tok) && tok[i+1] == '{':
			if depth == 0 {
				outer = i
			}
			depth++
			i++
		case tok[i] == '}' && depth > 0:
			depth--
		}
	}
	return tok
}

// vanishable reports whether a word is nothing but substitutions, so an
// unquoted one may leave no word at all.
func vanishable(tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] != unknownMark {
			return false
		}
	}
	return true
}

// unknownFlagCouldBe reports whether an unknown word written with a leading dash
// can become the flag alternative alt once its substitutions print. What the
// word already spells bounds it: `-$(x)` can be any flag, `--no-$(x)` any long
// flag that begins `--no-`, and `-r$(x)` a short cluster, so any short flag
// but never a long one. `--author=$(x)` names its option already and can
// become no blocked one.
func unknownFlagCouldBe(tok, alt string) bool {
	if tok == "" || tok[0] != '-' || !isUnknown(tok) || alt == "" {
		return false
	}
	lead := knownLead(tok)
	switch {
	case lead == "-":
		return true
	case strings.HasPrefix(lead, "--"):
		return strings.HasPrefix(alt, "--") && strings.HasPrefix(alt, lead)
	default:
		return isShortFlag(alt)
	}
}

// unknownCouldBe reports whether an unknown word can print as exactly literal:
// its known pieces appear in literal in order, the first at its start and the
// last at its end, and each substitution between them prints what is left.
func unknownCouldBe(tok, literal string) bool {
	parts := strings.Split(tok, unknownText)
	first, last := parts[0], parts[len(parts)-1]
	if !strings.HasPrefix(literal, first) {
		return false
	}
	rest := literal[len(first):]
	if !strings.HasSuffix(rest, last) {
		return false
	}
	rest = rest[:len(rest)-len(last)]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(rest, mid)
		if i < 0 {
			return false
		}
		rest = rest[i+len(mid):]
	}
	return true
}

// wordCouldBe reports whether a word is literal, or an unknown word that can
// print as it.
func wordCouldBe(tok, literal string) bool {
	return tok == literal || (isUnknown(tok) && unknownCouldBe(tok, literal))
}

// nameCouldBe reports whether a word in command position can run the program
// called name: its basename is name, ignoring case (a case-insensitive
// filesystem runs `GIT` as git, gh-315), or it is an unknown word whose
// basename's known tail name ends in. A substitution may print a slash, so the
// basename of `$(x)`, `/usr/bin/$(x)` and `abc$(x)` is anything its tail allows;
// only the text after the word's last substitution is fixed.
func nameCouldBe(tok, name string) bool {
	base := path.Base(tok)
	if !isUnknown(base) {
		return strings.EqualFold(base, name)
	}
	tail := base[strings.LastIndexByte(base, unknownMark)+1:]
	return len(tail) <= len(name) && strings.EqualFold(name[len(name)-len(tail):], tail)
}

// anyProgram reports whether a command word can run any program at all: its
// basename ends in a substitution, so nothing about its name is fixed.
func anyProgram(tok string) bool {
	base := path.Base(tok)
	return base != "" && base[len(base)-1] == unknownMark
}

// nameCouldBeAny reports whether a command word can run any program in names.
func nameCouldBeAny(tok string, names []string) bool {
	for _, n := range names {
		if nameCouldBe(tok, n) {
			return true
		}
	}
	return false
}

// flagCouldBe reports whether a word can be the option alt: literally, as the
// known text its substitutions leave when they print nothing, or as an
// unknown dash-word its lead can still become.
func flagCouldBe(tok, alt string) bool {
	if tok == alt {
		return true
	}
	if !isUnknown(tok) || vanishable(tok) {
		return false
	}
	return knownText(tok) == alt || unknownFlagCouldBe(tok, alt)
}

// clusterCouldCarry reports whether a word can be a short-option cluster
// carrying letter — `-lc` carries c — read as flagCouldBe reads one flag.
func clusterCouldCarry(tok string, letter byte) bool {
	carries := func(w string) bool { return isShortCluster(w) && strings.IndexByte(w[1:], letter) >= 0 }
	if carries(tok) {
		return true
	}
	if !isUnknown(tok) || vanishable(tok) {
		return false
	}
	return carries(knownText(tok)) || unknownFlagCouldBe(tok, "-"+string(letter))
}

// wordReadings is every way an option parser can read one argument word.
type wordReadings struct {
	operand bool // an operand (a word that does not start with a dash)
	flag    bool // an option that stands alone
	takes   bool // an option that takes the word after it as its value
	vanish  bool // no word at all: a substitution that printed nothing
}

// readWord reads one argument word for a parser whose value-taking options are
// valueFlags. A known word has exactly one reading. An unknown one has every
// reading it can: a dash-word is a flag, and a value flag whenever one of
// valueFlags is a flag it can become; a word led by a substitution is an
// operand, and also the word its known text spells; one that is nothing but a
// substitution is an operand or no word (never a flag: the recorded residual).
// A `--name=` word names its option already and takes nothing.
func readWord(tok string, valueFlags []string) wordReadings {
	var r wordReadings
	if !isUnknown(tok) {
		switch {
		case !strings.HasPrefix(tok, "-"):
			r.operand = true
		case !strings.Contains(tok, "=") && containsString(valueFlags, tok):
			r.takes = true
		default:
			r.flag = true
		}
		return r
	}
	if vanishable(tok) {
		return wordReadings{operand: true, vanish: true}
	}
	if strings.HasPrefix(tok, "-") {
		r.flag = true
		for _, vf := range valueFlags {
			if unknownFlagCouldBe(tok, vf) {
				r.takes = true
				break
			}
		}
	} else {
		r.operand = true
	}
	if k := knownText(tok); k != "" {
		kr := readWord(k, valueFlags)
		r.operand = r.operand || kr.operand
		r.flag = r.flag || kr.flag
		r.takes = r.takes || kr.takes
	}
	return r
}

// arrival is one place the walk to command position reaches: the word at idx
// is read there as a program name.
type arrival struct {
	idx int
	// noglob records that zsh's `noglob` was stepped on the way, so the words
	// from here on are compared literally.
	noglob bool
	// wrapper records that the word is a wrapper the walk steps through: the
	// command it runs is further on, and the word itself is no entry's command.
	wrapper bool
}

// The walk's modes: at a word that may be a program name, inside a wrapper's
// own options, and stepping a wrapper's mandatory operands.
const (
	walkArrive = iota
	walkOptions
	walkOperands
)

// someWrapper is the wrapper an unknown program name may be: one whose grammar
// is unknown, so each of its options may take a value, and whose one optional
// operand may come before the command it runs. It is the union of every
// wrapper's grammar (wrappers, wrapperValueFlags, wrapperOperands).
const someWrapper = unknownText

// commandArrivals walks a segment's tokens to command position and returns
// every place the walk can arrive at, in token order. Environment assignments
// and reserved words are stepped; a wrapper is stepped with its own options and
// operands; an unknown word is a program of unknown name (an arrival), and is
// also some wrapper (someWrapper), and, when it may print nothing, no word at
// all. Each option word is read by readWord, so an unknown one is read both as
// a flag and as a value flag. The walk visits each (position, mode, wrapper)
// state once, so its cost is linear in the tokens whatever they hold.
func commandArrivals(tokens []string) []arrival {
	out, _ := walkToCommand(tokens)
	return out
}

// maxUnknownSites bounds how many words of unknown name one segment's walk
// reads as its command. Each is every program, so each costs every entry and
// every payload family a read of the words after it; an everyday command has
// one at most. Past the bound the walk stops following them and says so
// (walkToCommand), and Check refuses the segment (unknownSitesBlockSignal).
const maxUnknownSites = 8

// walkToCommand is commandArrivals, and whether it stopped at maxUnknownSites.
func walkToCommand(tokens []string) (out []arrival, capped bool) {
	unknownSites := 0
	type state struct {
		pos     int
		mode    int
		wrapper string
		left    int
		noglob  bool
	}
	seen := map[state]bool{}
	var stack []state
	push := func(st state) {
		if st.pos <= len(tokens) && !seen[st] {
			seen[st] = true
			stack = append(stack, st)
		}
	}
	// operands enters a wrapper's mandatory operands, or command position when
	// it takes none.
	operands := func(pos int, w string, noglob bool) {
		left := wrapperOperands[w]
		if w == someWrapper {
			left = 1
			push(state{pos: pos, mode: walkArrive, noglob: noglob})
		}
		if left == 0 {
			push(state{pos: pos, mode: walkArrive, noglob: noglob})
			return
		}
		push(state{pos: pos, mode: walkOperands, wrapper: w, left: left, noglob: noglob})
	}
	found := map[arrival]bool{}
	push(state{})
	for len(stack) > 0 {
		st := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		tally(1)
		if st.pos >= len(tokens) {
			continue
		}
		tok := tokens[st.pos]
		switch st.mode {
		case walkArrive:
			if isAssignment(tok) || reserved[tok] {
				push(state{pos: st.pos + 1, noglob: st.noglob})
				continue
			}
			if tok == "coproc" {
				push(state{pos: skipCoproc(tokens, st.pos+1), noglob: st.noglob})
				continue
			}
			// The wrapper name is folded to lower case before lookup: on a
			// case-insensitive filesystem (macOS's default) `SUDO`/`ENV`/`NICE`
			// resolve to and run the real binary (gh-315).
			w := strings.ToLower(path.Base(tok))
			a := arrival{idx: st.pos, noglob: st.noglob, wrapper: wrappers[w]}
			if !found[a] {
				if isUnknown(tok) && !a.wrapper {
					if unknownSites == maxUnknownSites {
						capped = true
						continue
					}
					unknownSites++
				}
				found[a] = true
				out = append(out, a)
			}
			switch {
			case wrappers[w]:
				push(state{pos: st.pos + 1, mode: walkOptions, wrapper: w, noglob: st.noglob || w == "noglob"})
			case isUnknown(tok):
				if vanishable(tok) {
					push(state{pos: st.pos + 1, noglob: st.noglob})
				}
				push(state{pos: st.pos + 1, mode: walkOptions, wrapper: someWrapper, noglob: st.noglob})
			}
		case walkOptions:
			if tok == "--" {
				// End of the wrapper's options: everything after it is the command.
				operands(st.pos+1, st.wrapper, st.noglob)
				continue
			}
			if tok == "-" {
				operands(st.pos, st.wrapper, st.noglob)
				continue
			}
			r := readWord(tok, wrapperValueFlags[st.wrapper])
			if st.wrapper == someWrapper && (r.flag || r.takes) {
				r.flag, r.takes = true, true
			}
			next := state{pos: st.pos + 1, mode: walkOptions, wrapper: st.wrapper, noglob: st.noglob}
			if r.vanish || r.flag {
				push(next)
			}
			if r.takes {
				next.pos++
				push(next)
			}
			if r.operand {
				operands(st.pos, st.wrapper, st.noglob)
			}
			// An unknown dash-word's output is split into words when it stands
			// unquoted, so it can print the wrapper's mandatory operands after
			// its own flags: `timeout --$(x) pkill` runs pkill when x prints
			// `foreground 5` (iss-2609260543090196). Each count of operands it
			// may print leaves the rest to the words after it.
			if isUnknown(tok) && strings.HasPrefix(tok, "-") {
				for left := wrapperOperands[st.wrapper] - 1; left >= 0; left-- {
					if left == 0 {
						push(state{pos: st.pos + 1, mode: walkArrive, noglob: st.noglob})
					} else {
						push(state{pos: st.pos + 1, mode: walkOperands, wrapper: st.wrapper, left: left, noglob: st.noglob})
					}
				}
			}
		case walkOperands:
			next := state{pos: st.pos + 1, mode: walkOperands, wrapper: st.wrapper, left: st.left, noglob: st.noglob}
			if vanishable(tok) {
				push(next)
			}
			if next.left--; next.left == 0 {
				push(state{pos: st.pos + 1, mode: walkArrive, noglob: st.noglob})
			} else {
				push(next)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].idx != out[j].idx {
			return out[i].idx < out[j].idx
		}
		return !out[i].noglob && out[j].noglob
	})
	return out, capped
}

// unknownSitesBlockSignal is the fail-closed verdict for a segment whose walk
// to command position met more words of unknown name than maxUnknownSites.
func unknownSitesBlockSignal() payloadSignal {
	return payloadSignal{
		id:      substitutionEntryID,
		verdict: VerdictBlock,
		family:  familySubstitution,
		reason: "This command puts more command substitutions where its program name could be than the guard follows (" +
			itoa(maxUnknownSites) + "), so which program runs, and what it is handed, is not something it has checked.",
		successor: "Name the program the command runs, and keep a substitution's output in a variable, " +
			"so the guard checks the command that actually runs.",
	}
}

// unknownProgramEntryID is the reserved id a command is reported under when
// the only entries it fired are ones its unknown program name can be. No
// registry entry may claim it.
const unknownProgramEntryID = "program-name-unknown"

// unknownProgramSignal is the substitution family's verdict on a command whose
// program name a substitution prints and whose words fit the entry id. Nothing
// fixes the name, so it can be the program id names, and the lesson is the
// substitution's: spell the name, and the guard checks the program that runs.
func unknownProgramSignal(v Verdict, id string) payloadSignal {
	return payloadSignal{
		id:      unknownProgramEntryID,
		verdict: v,
		family:  familySubstitution,
		reason: "This command's program name is the output of a command substitution, so it can be any program, " +
			"and with the words after it the command reads as one the registry refuses (" + id + ").",
		successor: "Spell the program's name, and keep a substitution's output in a variable if you need it, " +
			"so the guard checks the program that actually runs.",
	}
}

// arrivalsOf is commandArrivals for a segment, read from its cache when Check
// has set one (walkSegments).
func arrivalsOf(s segment) []arrival {
	if s.walked {
		return s.arrivals
	}
	return commandArrivals(s.tokens)
}

// walkSegments caches each segment's walk to command position, so the
// pre-passes, every entry's match and every after_cd look back read it without
// walking again. A segment already walked keeps its walk.
func walkSegments(segs []segment) {
	for i := range segs {
		if !segs[i].walked {
			segs[i].arrivals, segs[i].walkCapped = walkToCommand(segs[i].tokens)
			segs[i].walked = true
		}
	}
}

// commandSites is every arrival that can be the segment's command: the ones
// that are not a wrapper stepped through.
func commandSites(s segment) []arrival {
	var out []arrival
	for _, a := range arrivalsOf(s) {
		if !a.wrapper {
			out = append(out, a)
		}
	}
	return out
}

// commandNamed reports whether the word at an arrival can run the program
// called name: by nameCouldBe, or as a glob bash expands to it (the pattern is
// read as the pattern it is, GHSA-3w99-pgv4-8g55), unless noglob holds.
func commandNamed(s segment, a arrival, name string) bool {
	tok := s.tokens[a.idx]
	if nameCouldBe(tok, name) {
		return true
	}
	return !a.noglob && s.globAt(a.idx) && globMatches(strings.ToLower(path.Base(tok)), strings.ToLower(name))
}

// sitesNamed returns the command sites that can run the program called name.
func sitesNamed(s segment, name string) []arrival {
	var out []arrival
	for _, a := range commandSites(s) {
		if commandNamed(s, a, name) {
			out = append(out, a)
		}
	}
	return out
}

// operandWant is what an entry asks of a command's operands: operand 0 and 1
// by name, a count, an argument prefix and a resource path carried by some
// operand.
type operandWant struct {
	sub, sub2 string
	min       int
	prefixes  []string
	paths     []PathArg
}

// operandAcceptance returns, for each index i of tokens, whether some reading
// of tokens[i:] as a command's arguments meets want, so the answer for a
// command at site s is accept[s+1]. Each word is read by readWord, every way it
// can be: an unknown dash-word both stands alone and takes the next word, a
// word that may print nothing both is and is not an operand. One reading must
// satisfy every clause together — operand 0 and 1 are the same reading's — so
// the table's state is (word, operands so far, clauses met), filled from the
// end once: linear in the words, whatever the number of places a command can
// sit.
func operandAcceptance(tokens, valueFlags []string, want operandWant, glob func(int) bool) []bool {
	need := want.need()
	nb := uint(len(want.prefixes) + len(want.paths))
	full := 1<<nb - 1
	width := (need + 1) << nb
	n := len(tokens)
	acc := make([]bool, (n+2)*width)
	at := func(i, k, met int) int { return i*width + k<<nb + met }
	// Past the last word a reading is done: it met want or it did not. A value
	// flag written last takes a word that is not there, and lands one further.
	acc[at(n, need, full)] = true
	acc[at(n+1, need, full)] = true
	for i := n - 1; i >= 0; i-- {
		tally(1) // a token the match walks (work.go's unit); its states are the entry's constant
		a := tokens[i]
		if a == "--" {
			copy(acc[i*width:(i+1)*width], acc[(i+1)*width:(i+2)*width])
			continue
		}
		r := readWord(a, valueFlags)
		hits := 0
		if r.operand {
			for j, prefix := range want.prefixes {
				if argPrefixMatches(prefix, []string{a}) {
					hits |= 1 << j
				}
			}
			for j, pa := range want.paths {
				if pathArgMatches(pa, []string{a}) {
					hits |= 1 << (len(want.prefixes) + j)
				}
			}
		}
		for k := 0; k <= need; k++ {
			operand := r.operand &&
				!(k == 0 && want.sub != "" && !operandIs(a, want.sub, glob(i))) &&
				!(k == 1 && want.sub2 != "" && !operandIs(a, want.sub2, glob(i)))
			k2 := k + 1
			if k2 > need {
				k2 = need
			}
			for met := 0; met <= full; met++ {
				v := (r.vanish || r.flag) && acc[at(i+1, k, met)]
				v = v || (r.takes && acc[at(i+2, k, met)])
				v = v || (operand && acc[at(i+1, k2, met|hits)])
				acc[at(i, k, met)] = v
			}
		}
	}
	out := make([]bool, n+1)
	for i := range out {
		out[i] = acc[at(i, 0, 0)]
	}
	return out
}

// need is how many operands a reading must place: the count asked for, and at
// least one more than the last subcommand position named.
func (w operandWant) need() int {
	n := w.min
	if w.sub != "" && n < 1 {
		n = 1
	}
	if w.sub2 != "" && n < 2 {
		n = 2
	}
	return n
}

// operandNeed is operandWant.need for an entry's pattern.
func operandNeed(p Pattern) int {
	return operandWant{sub: p.Subcommand, sub2: p.Subcommand2, min: p.MinOperands}.need()
}

// operandIs reports whether an operand can be want — literally, as a word its
// glob pattern can produce, or as an unknown word that can print it.
func operandIs(tok, want string, glob bool) bool {
	return wordCouldBe(tok, want) || (glob && globMatches(tok, want))
}

// maxOperandReadings bounds the readings operandReadings enumerates, and
// maxOperandStates the states it visits to find them. Past either the answer is
// incomplete, and each caller fails closed on that.
const (
	maxOperandReadings = 64
	maxOperandStates   = 4096
)

// operandReadings returns every distinct placing of args's first need operands,
// each as the operands' indexes into args in order, reading every word as
// readWord does. It stops a reading at need operands, so a line's later words
// multiply nothing, and it visits each (word, operands placed) state once, so
// two readings that agree from a word on are walked from it once. complete is
// false when the readings or the states ran past their bound, and a caller then
// takes the fail-closed answer.
func operandReadings(args, valueFlags []string, need int) (readings [][]int, complete bool) {
	type state struct {
		i   int
		ops []int
	}
	visited := map[string]bool{}
	emitted := map[string]bool{}
	var stack []state
	push := func(st state) bool {
		key := itoa(st.i) + ":" + intsKey(st.ops)
		if visited[key] {
			return true
		}
		visited[key] = true
		stack = append(stack, st)
		return len(visited) <= maxOperandStates
	}
	push(state{})
	for len(stack) > 0 {
		st := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		tally(1)
		if st.i >= len(args) || len(st.ops) == need {
			key := intsKey(st.ops)
			if !emitted[key] {
				emitted[key] = true
				readings = append(readings, st.ops)
				if len(readings) > maxOperandReadings {
					return readings, false
				}
			}
			continue
		}
		a := args[st.i]
		ok := true
		if a == "--" {
			ok = push(state{st.i + 1, st.ops})
		} else {
			r := readWord(a, valueFlags)
			if r.vanish || r.flag {
				ok = push(state{st.i + 1, st.ops}) && ok
			}
			if r.takes {
				ok = push(state{st.i + 2, st.ops}) && ok
			}
			if r.operand {
				ok = push(state{st.i + 1, append(append([]int(nil), st.ops...), st.i)}) && ok
			}
		}
		if !ok {
			return readings, false
		}
	}
	sort.Slice(readings, func(i, j int) bool { return intsKey(readings[i]) < intsKey(readings[j]) })
	return readings, true
}

// itoa spells an index for a state key.
func itoa(i int) string { return fmt.Sprint(i) }

// firstOperands returns, in order, every index of args that can be operand 0
// in some reading. When the readings run past their bound, every word that can
// be an operand is returned: the fail-closed answer.
func firstOperands(args, valueFlags []string) []int {
	readings, complete := operandReadings(args, valueFlags, 1)
	var out []int
	if !complete {
		for i, a := range args {
			if readWord(a, valueFlags).operand {
				out = append(out, i)
			}
		}
		return out
	}
	seen := map[int]bool{}
	for _, r := range readings {
		if len(r) > 0 && !seen[r[0]] {
			seen[r[0]] = true
			out = append(out, r[0])
		}
	}
	sort.Ints(out)
	return out
}

// firstOperandLimit returns how far the options before operand 0 can run: the
// furthest operand 0 any reading places, or len(args) when a reading places
// none (or the readings ran past their bound).
func firstOperandLimit(args, valueFlags []string) int {
	readings, complete := operandReadings(args, valueFlags, 1)
	if !complete || len(readings) == 0 {
		return len(args)
	}
	limit := 0
	for _, r := range readings {
		if len(r) == 0 {
			return len(args)
		}
		if r[0] > limit {
			limit = r[0]
		}
	}
	return limit
}

// intsKey spells a list of indexes as a key that sorts as the list does.
func intsKey(xs []int) string {
	var b strings.Builder
	for _, x := range xs {
		fmt.Fprintf(&b, "%08d,", x)
	}
	return b.String()
}

// unknownOperandOnPath reports whether an unknown operand can name a resource
// path of exactly pa.Segments segments under pa.Root. Its separators are
// fixed text, so the count of `/`-parts it spells is a floor on its depth
// (an output can only add slashes); a part that is wholly a substitution at
// either end may print nothing and be trimmed away, so it does not count. The
// first part must be the root, or be unknown itself.
func unknownOperandOnPath(pa PathArg, op string) bool {
	parts := strings.Split(strings.Trim(pathOf(op), "/"), "/")
	floor := len(parts)
	if vanishable(parts[0]) {
		floor--
	}
	if len(parts) > 1 && vanishable(parts[len(parts)-1]) {
		floor--
	}
	if floor > pa.Segments {
		return false
	}
	first := parts[0]
	return first == pa.Root || (isUnknown(first) && strings.HasPrefix(pa.Root, knownLead(first)))
}
