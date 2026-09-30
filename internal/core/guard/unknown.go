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
// command line. Nor is a parameter expansion's value (`$X`, `$1`, `$@`,
// `${X:-git}`, iss-2609251824244354). The tokenizer therefore writes
// unknownMark into the word where the output or the value goes, and every
// reader of a token asks this file what the word can be. An arithmetic
// expansion is not unknown in that sense: its output is a number, which no
// flag, subcommand or path the registry names can be, and neither are `$$`,
// `$!`, `$?` and `$#`.
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
// known text only, and an operand an entry names by its exact word
// (arg_values): `rm -rf $(find . -name '*.pyc')` is how an everyday delete
// names its targets, and reading its operand as every target would refuse it
// as a delete of `/`. The residuals are recorded in .abcd/work/DECISIONS.md,
// and a word that is wholly a variable reads the same way (`git push origin
// "$branch"`). A variable's value is read as a flag and a program name, and
// not as data an earlier command carried (variableCarried).

// unknownMark stands, inside a token, for the output of a substitution the
// guard did not run. It is the NUL byte, and it is unforgeable by construction:
// Check drops every NUL the command line itself carries before it reads a
// word, and the one decoder that could make a NUL from other bytes — an ANSI-C
// `$'…'` escape — ends its string at the first decoded NUL, as bash does
// (readAnsiCQuote). No other byte reaches a word decoded.
const unknownMark = '\x00'

// unknownText is unknownMark as a string, for building tokens.
const unknownText = "\x00"

// varMark stands, in the TEXT of a string the guard re-reads as a payload,
// for a variable's value the enclosing shell has already put there
// (payloadView). The tokenizer turns it into unknownMark in the word it lands
// in, and records that the word's unknown part is a variable's
// (segment.variable), wherever it stands: unquoted, inside a quote, behind a
// backslash or in an ANSI-C string. That is what the enclosing shell did to
// the value — the string's own quoting applies to the value, not to a name —
// so `sh -c "git push '--$X'"` is read as the flag of unknown name bash
// builds (review-drainG3 finding 1). Spelling the variable back as `$X` text
// applied that quoting to the name instead, and `'--$X'` read as text.
//
// It is the byte 0x01. An ANSI-C escape that decodes to it stays text
// (readAnsiCQuote), but the byte itself can reach the text read: written raw
// in the line, or carried into a string's text from the level above. There
// it is read as a variable's value — every flag and program name its known
// text allows, less the readings variableCarried drops, each of which a
// literal 0x01 byte in its place cannot produce either: it names no program,
// no stream and no flag. So a byte read as the mark reads no narrower than
// the literal byte bash hands on.
const varMark = '\x01'

// varText is varMark as a string, for spelling a payload's text.
const varText = "\x01"

// varSite is one variable's mark in a word being built: its offset in the
// word, and the texts the expansion can print as the line wrote them (`$HOME`,
// `${PWD}`; `${DIR}` and `$HOME` for `${DIR:-$HOME}`, spellParameter), nil
// for a varMark read from a payload's text, whose name the string no longer
// holds. bare records a name written unquoted and without braces, which the
// unquoted text a brace group places after it runs on from (spellWritten).
type varSite struct {
	at    int
	texts []string
	bare  bool
}

// A written spelling (segment.spelled) is a SET of texts, one for each thing
// the word can print as the line wrote it: `${DIR:-$HOME}` prints DIR's value
// or the home, and is `${DIR}` and `$HOME` (iss-2609290426544292). A word's
// set is every combination of its sites' sets, bounded by maxSpellings, and a
// site's word is followed into its own expansions at most spellWordDepth
// deep. Past either bound the site is spellCapped: a spelling the guard
// stopped reading, which the arg_values compare reads as every value an entry
// names (writtenMatches), so a bound refuses rather than lets the word pass.
const (
	// maxSpellings bounds how many texts one word's written spelling holds,
	// and so how many times a string handed to a shell is re-read to pair
	// them (namedPayloads).
	maxSpellings = 16
	// spellWordDepth bounds how deep a spelling follows a default's or an
	// alternative's word into another expansion.
	spellWordDepth = 8
	// spellCapped is the one text of a spelling past either bound.
	spellCapped = "\x04"
)

// spellWritten is a word as the line wrote its variables (segment.spelled):
// each variable's mark replaced by each text its expansion can print, each
// substitution's mark dropped as knownText drops it, and a variable whose
// text is not known kept as unknownMark. The result is every combination of
// the sites' texts, in word order; a site already past a bound, or one that
// would make more than maxSpellings of them, is spellCapped in each. A simple name the next byte
// kept would extend is braced (`"$A"B` is `${A}B`, not `$AB`), so the
// spelling reads as the same expansions when it is read again (spelledViews).
// sites is in word order.
//
// mask is nil for a word as the line wrote it. For a word a brace group made
// it is the word's bword.m, and a bare name directly followed by unquoted
// name bytes is not braced: bash expands the group first and reads the name
// after, so `$HO{ME,}` makes `$HOME` (iss-2609290419119456). A quote or
// escape between them leaves the byte quoted, and the name ends there.
func spellWritten(word []byte, sites []varSite, mask []byte) []string {
	outs := [][]byte{nil}
	add := func(c byte) {
		for i := range outs {
			outs[i] = append(outs[i], c)
		}
	}
	k := 0
	isVar := func(p int) bool { return k < len(sites) && sites[k].at == p }
	for p := 0; p < len(word); p++ {
		if !isVar(p) {
			if word[p] != unknownMark {
				add(word[p])
			}
			continue
		}
		site := sites[k]
		k++
		if len(site.texts) == 0 {
			add(unknownMark)
			continue
		}
		if capped(site.texts) || len(outs)*len(site.texts) > maxSpellings {
			// The site is past a bound: it is spellCapped in every text, and
			// the text around it is kept, so a string handed to a shell still
			// pairs its words (spellPayload).
			add(spellCapped[0])
			continue
		}
		// Only a simple name is braced: a default's or an alternative's word
		// is spelled as written (`$HOME/`, `~`), and bracing that would
		// change it.
		braced := false
		next := p + 1
		for next < len(word) && word[next] == unknownMark && !isVar(next) {
			next++
		}
		if next < len(word) && word[next] != unknownMark && isNameByte(word[next]) {
			runsOn := mask != nil && site.bare && next == p+1 && mask[next]&wordStruct != 0
			braced = !runsOn
		}
		grown := make([][]byte, 0, len(outs)*len(site.texts))
		for _, o := range outs {
			for n, text := range site.texts {
				if braced && len(text) > 1 && text[0] == '$' && simpleParamEnd(text, 1) == len(text) {
					text = "${" + text[1:] + "}"
				}
				b := o
				if n < len(site.texts)-1 {
					b = append([]byte(nil), o...)
				}
				grown = append(grown, append(b, text...))
			}
		}
		outs = grown
	}
	texts := make([]string, 0, len(outs))
	for _, o := range outs {
		tally(len(o))
		texts = appendText(texts, string(o))
	}
	return texts
}

// appendText adds text to a spelling's texts unless it is already there.
func appendText(texts []string, text string) []string {
	for _, t := range texts {
		if t == text {
			return texts
		}
	}
	return append(texts, text)
}

// capped reports whether a spelling is past a bound (spellCapped).
func capped(texts []string) bool {
	for _, t := range texts {
		if strings.Contains(t, spellCapped) {
			return true
		}
	}
	return false
}

// paramText is a parameter expansion's text as bash reads it: without the
// backslash-newlines it drops before it reads a name (simpleParamEnd) or the
// text between a `${` and its `}`.
func paramText(text string) string { return strings.ReplaceAll(text, "\\\n", "") }

// spellParameter is the written spelling (segment.spelled) of a `${…}`
// expansion whose text between the braces is body: every text it can print
// as the line wrote it. Where the expansion can print its variable's value
// unchanged, the set holds that variable, so arg_values reads `${HOME%/}` as
// the `${HOME}` it can be (iss-2609290419119456):
//
//   - a default or an assignment, with or without the colon (`${DIR:-w}`,
//     `${DIR=w}`), prints the value when the variable is set and its word w
//     when it is not, so the set holds the variable and every text w can
//     print, through its own expansions (spellWord): `${DIR:-$HOME}` is
//     `${DIR}` and `$HOME` (iss-2609290426544292);
//   - an error message (`${HOME:?x}`) prints the value or nothing: the
//     message goes to the standard error, never into the word;
//   - a trimmed prefix or suffix and a pattern replacement (`${HOME%/}`,
//     `${HOME#x}`, `${HOME/x/y}`): the value when the pattern does not match,
//     and what a suffix trim leaves otherwise is the path above it. What
//     else it can print is read from its pattern's shape (trimTexts,
//     replacementTexts): a pattern that can take any remainder of the value
//     can leave only the `/` an absolute path begins with (`${X%${X#?}}` is
//     `${X}`, `/` and nothing), or the whole value, which a replacement's
//     string takes the place of (`${X/*/$HOME}` and `${X/?*/$HOME}` are
//     `${X}` and `$HOME`; iss-2609292320015665, iss-2609300009581165);
//   - a substring (`${HOME:0}`), whose offset is arithmetic and can be 0 or
//     past the end, and which can print the `/` an absolute path begins with
//     (`${PWD:0:1}` is `${PWD}`, `/` and nothing);
//   - a case change (`${HOME^^}`, `${HOME@U}`), which names the same directory
//     on a case-insensitive disk, and `@E` and `@P`, which change no path;
//   - a subscript (`${HOME[0]}`, `${HOME[x[0]]}`), which can be 0, read to
//     its matching `]`. bash 3.2, the /bin/sh and /bin/bash of macOS, steps
//     over any text after it to the first operator byte
//     (subscriptOperators): an alternative there prints its word, a default
//     or an assignment the value or its word (`${X[0]]-$HOME}` is the home
//     with X unset), a trim, a replacement or a substring what it prints
//     after a name and also nothing, which is what bash 3.2 prints for one
//     after a scalar's subscript (`${X[0]%x}` with X=/a/b;
//     iss-2609300009506126), and anything else the value (`${HOME[0]]}`,
//     `${HOME[0]@Q}`). A subscript with no `]` cannot be read further.
//
// An alternative (`${X:+w}`, `${X+w}`) prints w or nothing, and is the texts
// w can print and the empty text. Every other expansion keeps its text as written and names no
// variable an entry names: a length (`${#HOME}`), an indirection (`${!X}`),
// and `@Q` and the other transforms.
//
// split reports that the expansion stands unquoted, where bash splits a
// default's or an alternative's word on whitespace: each unquoted whitespace
// run in it is spelled fieldMark, which the compare splits on
// (argValueMatches).
func spellParameter(body string, split bool) []string {
	return spellParameterAt(paramText(body), 0, split)
}

// fieldMark stands in a spelling where bash splits a word into fields: at an
// unquoted whitespace run in an unquoted alternative's word (`${X:+$HOME }`
// hands rm the home). Only the arg_values compare splits on it; a payload
// re-read reads it as the space it was (spelledViews).
const fieldMark = '\x02'

// fieldText is fieldMark as a string.
const fieldText = "\x02"

// quotedFieldMark stands where a double-quoted alternative's word holds
// unquoted whitespace (`sh -c "rm -rf ${X:+$HOME x}"`): the word is one
// field here, which the compare reads as a space, but a shell re-reading the
// string splits it there, so a payload re-read takes it for fieldMark
// (spelledViews), and the string's words pair with its marked reading's.
const quotedFieldMark = '\x03'

// quotedFieldText is quotedFieldMark as a string.
const quotedFieldText = "\x03"

func spellParameterAt(body string, depth int, split bool) []string {
	raw := []string{"${" + body + "}"}
	n := 0
	for n < len(body) && isNameByte(body[n]) {
		n++
	}
	if n == 0 || body[0] >= '0' && body[0] <= '9' {
		return raw
	}
	name, rest := body[:n], body[n:]
	same := "${" + name + "}"
	value := []string{same}
	// orWord is the value, or the texts the word w can print.
	orWord := func(w string) []string {
		texts := value
		for _, t := range spellWord(w, depth, split) {
			texts = appendText(texts, t)
		}
		if capped(texts) || len(texts) > maxSpellings {
			return []string{spellCapped}
		}
		return texts
	}
	// alternative is the texts the word w can print, or nothing: with the
	// variable unset or empty the expansion prints no text, so
	// `${X:+x}$HOME` is `x$HOME` and `$HOME`. A word the guard cannot read
	// is the expansion as written, which names nothing.
	alternative := func(w string) []string {
		texts := spellWord(w, depth, split)
		if len(texts) == 0 {
			texts = raw
		}
		return appendText(texts, "")
	}
	subscript := false
	if strings.HasPrefix(rest, "[") {
		k := subscriptEnd(rest)
		if k < 0 {
			return value
		}
		rest = rest[k+1:]
		op := strings.IndexAny(rest, subscriptOperators)
		if op < 0 {
			return value
		}
		rest, subscript = rest[op:], true
	}
	if rest == "" {
		return value
	}
	switch {
	case rest[0] == '+':
		return alternative(rest[1:])
	case strings.HasPrefix(rest, ":+"):
		return alternative(rest[2:])
	case rest[0] == '-' || rest[0] == '=':
		return orWord(rest[1:])
	case strings.HasPrefix(rest, ":-") || strings.HasPrefix(rest, ":="):
		return orWord(rest[2:])
	case strings.HasPrefix(rest, ":?"):
		return value
	}
	var texts []string
	switch {
	case rest[0] == ':':
		// A substring: a part of the value, the whole of it at offset 0, the
		// `/` an absolute path begins with, and nothing at an offset past
		// its end (`${X:9}`) or a length of 0.
		texts = []string{same, "/", ""}
	case rest[0] == '/':
		texts = replacementTexts(value, rest[1:], depth, split)
	case rest[0] == '%' || rest[0] == '#':
		texts = trimTexts(value, rest)
	case subscript:
		return value
	case strings.IndexByte("?^,~", rest[0]) >= 0:
		// Every other operator that can print the value unchanged.
		return value
	case rest[0] == '@':
		if len(rest) == 2 && strings.IndexByte("EPULu", rest[1]) >= 0 {
			return value
		}
		return raw
	default:
		return raw
	}
	if subscript {
		// bash 3.2 prints nothing for a trim, a replacement or a substring
		// after a scalar's subscript: `${X[0]%x}` with X=/a/b.
		texts = appendText(texts, "")
	}
	return texts
}

// trimTexts is the written spelling of a trim, whose operator and pattern
// are rest (`%p`, `%%p`, `#p`, `##p`), where value is the variable's own:
// the value, which the trim leaves where its pattern does not match, and
// what the pattern's shape (readPattern) lets it leave whatever the value
// holds (iss-2609292320015665, iss-2609300009506126). The rule, for a value
// that is an absolute path:
//
//   - a suffix trim (`%`, `%%`) can leave only the leading `/` when its
//     pattern can take a remainder of any length (it holds a `*`, unknown
//     text or an extglob group) and its first element past any run of `*`
//     is a glob or unknown text: that element can match the text after the
//     `/`, and the rest of the pattern the remainder (`${X%${X#?}}`,
//     `${X%%[!/]*}`, `${X%$Y}`). A prefix trim (`#`, `##`) can leave only a
//     trailing `/` when the same holds of its last element (`${T#${T%?}}`,
//     `${T##*[!/]}` with T=/tmp/x/). Either can then also leave nothing.
//   - a longest trim (`%%`, `##`) can leave nothing when its pattern can
//     match the whole path: it can take any length, its first element is a
//     `*`, a glob, unknown text or a literal `/`, and its last a `*`, a glob
//     or unknown text (`${X%%*}`, `${X%%/*}`, `${X##/*}`).
//
// Every other trim is the value alone. A pattern whose element at the
// anchored end is literal text, or which matches a fixed width, leaves the
// root or nothing only for a value of one particular content or length, as
// `rm -rf $X` deletes the root only for X=/: `${DIR%/}`, `${f%.txt}`,
// `${f%.*}`, `${p##*/}`, `${p%/*}`, `${p#$HOME/}`, `${X%?}` and `${X#?}`.
// A `*` at that end is stepped over, since it can match nothing, so a
// shortest trim reads as the element behind it; for a longest trim that
// over-reads (`${X%%*[!/]*}` leaves nothing, never `/`), which only adds a
// text. Unknown text is any expansion (`$Y`, `${…}`, `$(…)`, a backtick),
// quoted or not, `$HOME` included: its text is not in the line, and
// `${X%${X#?}}` builds it from the value itself.
func trimTexts(value []string, rest string) []string {
	suffix := rest[0] == '%'
	longest := len(rest) > 1 && rest[1] == rest[0]
	p := rest[1:]
	if longest {
		p = rest[2:]
	}
	sh := readPattern(p, false)
	anchored := sh.lastPast
	if suffix {
		anchored = sh.firstPast
	}
	texts := value
	if sh.wide && roving(anchored) {
		texts = appendText(appendText(texts, "/"), "")
	}
	if longest && sh.whole() {
		texts = appendText(texts, "")
	}
	return texts
}

// replacementTexts is the written spelling of a pattern replacement, whose
// text after the first `/` is rest (`p/s`, `/p/s`, `#p/s`, `%p/s`), where
// value is the variable's own: the value, and what the pattern's shape
// (readPattern) lets it print whatever the value holds
// (iss-2609300009581165). A pattern that can match the whole of an absolute
// path, by the longest-trim rule of trimTexts, prints the string s in its
// place (`${X/*/$HOME}`, `${X/\/*/$HOME}`, `${X/?*/$HOME}`, `${X/$Y/~}`),
// and one that can match all of it after the leading `/`, by the suffix-trim
// rule and ending in a `*`, a glob or unknown text, prints `/` and s,
// unless `/#` anchors it at the start (`${X/${X#?}}` and `${X//[!\/]*/}`
// are `/`). s is read as a default's word is (spellWord), and one that
// prints nothing the guard reads is the empty text. Every other replacement
// is the value alone: `${X/foo/$HOME}`, `${DIR/#\~/$HOME}`, and
// `${name//[^a-z]/}`, whose pattern matches one byte.
func replacementTexts(value []string, rest string, depth int, split bool) []string {
	p := rest
	anchor := byte(0)
	if p != "" && strings.IndexByte("/#%", p[0]) >= 0 {
		anchor, p = p[0], p[1:]
	}
	sh := readPattern(p, true)
	whole := sh.whole()
	tail := anchor != '#' && sh.wide && roving(sh.firstPast) && anyWidth(sh.last)
	if !whole && !tail {
		return value
	}
	s := ""
	if sh.end < len(p) {
		s = p[sh.end+1:]
	}
	words := spellWord(s, depth, split)
	if len(words) == 0 {
		words = []string{""}
	}
	texts := value
	for _, w := range words {
		if whole {
			texts = appendText(texts, w)
		}
		if tail {
			texts = appendText(texts, "/"+w)
		}
	}
	if capped(texts) || len(texts) > maxSpellings {
		return []string{spellCapped}
	}
	return texts
}

// patElem is one element of a trim's or a replacement's pattern, as far as
// what it can match decides what the expansion can print (readPattern).
type patElem uint8

const (
	// elemNone stands where a pattern has no element: an empty one.
	elemNone patElem = iota
	// elemLiteral is a byte the pattern matches as itself: plain, escaped
	// or quoted, other than `/`.
	elemLiteral
	// elemSlash is a literal `/`.
	elemSlash
	// elemStar is an unquoted `*`, which matches any text, none included.
	elemStar
	// elemOne is a glob of one byte: a `?` or a bracket expression.
	elemOne
	// elemAny is text the line does not spell, of any length: an expansion
	// (`$Y`, `${…}`, `$(…)`, a backtick, an ANSI-C string) or an extglob
	// group (`@(…)`, `*(…)`), or a quote or an expansion that does not close.
	elemAny
)

// roving reports whether an element can match text the line does not spell:
// a one-byte glob or text of any length.
func roving(e patElem) bool { return e == elemOne || e == elemAny }

// anyWidth reports whether an element can match whatever byte ends a value:
// a `*` or a roving element.
func anyWidth(e patElem) bool { return e == elemStar || roving(e) }

// patternShape is what readPattern records of a pattern: its first and last
// element, the same past any run of `*` at that end, whether it can take a
// remainder of any length, and, for a replacement, where the `/` that ends
// it stands (len of the text where none does).
type patternShape struct {
	first, last         patElem
	firstPast, lastPast patElem
	wide                bool
	end                 int
}

// whole reports whether the pattern can match the whole of an absolute
// path, whatever it holds: it can take any length, its first element can
// match the leading `/`, and its last can match whatever byte the path ends
// with.
func (sh patternShape) whole() bool {
	return sh.wide && (sh.first == elemSlash || anyWidth(sh.first)) && anyWidth(sh.last)
}

// readPattern reads the pattern p of a trim or, with replacement, of a
// replacement, once and left to right, recording its shape (patternShape).
// Its quotes and escapes make literal text; an expansion in it is stepped
// over to its close without being read again, so the cost is p's length.
// A replacement's pattern ends at its first unescaped `/`, which bash 3.2
// reads as the end even inside quotes and brackets (`${X/[/]/c}` replaces
// `[`).
func readPattern(p string, replacement bool) patternShape {
	tally(len(p))
	sh := patternShape{end: len(p)}
	add := func(e patElem) {
		if sh.first == elemNone {
			sh.first = e
		}
		if e != elemStar && sh.firstPast == elemNone {
			sh.firstPast = e
		}
		sh.last = e
		if e != elemStar {
			sh.lastPast = e
		}
		if e == elemStar || e == elemAny {
			sh.wide = true
		}
	}
	literal := func(c byte) {
		if c == '/' {
			add(elemSlash)
		} else {
			add(elemLiteral)
		}
	}
	// unread marks the rest of the pattern as text the guard does not read.
	unread := func() patternShape {
		add(elemAny)
		return sh
	}
	budget := 4*len(p) + 16
	dq := false
	for i := 0; i < len(p); {
		c := p[i]
		switch {
		case replacement && c == '/':
			sh.end = i
			return sh
		case c == '\\':
			if i+1 < len(p) {
				literal(p[i+1])
			} else {
				literal(c)
			}
			i += 2
		case c == '"':
			dq = !dq
			i++
		case c == '\'' && !dq:
			k := strings.IndexByte(p[i+1:], '\'')
			if k < 0 {
				return unread()
			}
			for j := i + 1; j < i+1+k; j++ {
				if replacement && p[j] == '/' {
					sh.end = j
					return sh
				}
				literal(p[j])
			}
			i += k + 2
		case c == '$' && i+1 < len(p) && p[i+1] == '{':
			end := closingDolBrace(p, i+2, &budget)
			if end < 0 {
				return unread()
			}
			add(elemAny)
			i = end + 1
		case c == '$' && i+1 < len(p) && p[i+1] == '(':
			end := closingParen(p, i+2, &budget)
			if end < 0 {
				return unread()
			}
			add(elemAny)
			i = end + 1
		case c == '`':
			end := closingBacktick(p, i+1, &budget)
			if end < 0 {
				return unread()
			}
			add(elemAny)
			i = end + 1
		case c == '$' && !dq && i+1 < len(p) && p[i+1] == '\'':
			k := i + 2
			for k < len(p) && p[k] != '\'' {
				if p[k] == '\\' {
					k++
				}
				k++
			}
			if k >= len(p) {
				return unread()
			}
			add(elemAny)
			i = k + 1
		case c == '$':
			if end := simpleParamEnd(p, i+1); end > 0 {
				add(elemAny)
				i = end
				continue
			}
			literal(c)
			i++
		case dq:
			literal(c)
			i++
		case strings.IndexByte("*?+@!", c) >= 0 && i+1 < len(p) && p[i+1] == '(':
			end := extglobEnd(p, i+1)
			if end < 0 {
				return unread()
			}
			add(elemAny)
			i = end + 1
		case c == '*':
			add(elemStar)
			i++
		case c == '?':
			add(elemOne)
			i++
		case c == '[':
			if end := bracketEnd(p, i, replacement); end > 0 {
				add(elemOne)
				i = end + 1
				continue
			}
			literal(c)
			i++
		default:
			literal(c)
			i++
		}
	}
	return sh
}

// bracketEnd returns the index of the `]` that closes the bracket
// expression opening at p[i], or -1 where none does: a `]` directly after
// the `[` or its `!` or `^` is a member, and a backslash quotes the next
// byte. In a replacement's pattern a `/` ends the pattern first
// (readPattern), and the `[` is then literal.
func bracketEnd(p string, i int, replacement bool) int {
	j := i + 1
	if j < len(p) && (p[j] == '!' || p[j] == '^') {
		j++
	}
	if j < len(p) && p[j] == ']' {
		j++
	}
	for j < len(p) {
		switch p[j] {
		case '\\':
			j += 2
			continue
		case ']':
			return j
		case '/':
			if replacement {
				return -1
			}
		}
		j++
	}
	return -1
}

// extglobEnd returns the index of the `)` that closes the extglob group
// whose `(` is at p[i], counting the groups nested in it, or -1 where none
// does.
func extglobEnd(p string, i int) int {
	depth := 0
	for j := i; j < len(p); j++ {
		switch p[j] {
		case '\\':
			j++
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return j
			}
		}
	}
	return -1
}

// subscriptOperators are the bytes bash 3.2 stops at in the text after a
// subscript's `]`: an operator, or a backslash, which quotes the next byte.
// A `+` or `:+` there reads an alternative, and a `-`, `:-`, `=` or `:=` a
// default or an assignment: with X unset, `${X[0]]-$HOME}`,
// `${X[0]]:-$HOME}` and `${X[0]]=$HOME}` print the home on bash 3.2 and
// /bin/sh, and with X set they print X's value. Any other byte there prints
// the value: `${X[0]]?$HOME}`, and `${X[0]]\+$HOME}`.
const subscriptOperators = "-=?+%#/:\\"

// subscriptEnd returns the index of the `]` that closes the subscript opening
// at s[0], counting the brackets nested in it (`[x[0]]`), or -1 where none
// does.
func subscriptEnd(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

// spellWord is the texts a default's or an alternative's word w can print,
// at depth expansions deep. w is spelled as it is written: its quotes and
// escapes removed, and each expansion in it a site spelled as a word's own
// are (spellWritten), `${…}` through spellParameterAt one level deeper.
// `${X:+$HOME/}` is `$HOME/`, `${X:+/}` is `/` and `${X:+"${HOME%/}"}` is
// `${HOME}`. A command substitution in it is its unknown output, which
// spellWritten drops as knownText does (`${X:+$(true)$HOME}` is `$HOME`).
// Where split is set, each unquoted whitespace run is fieldMark, where bash
// splits the word (`${X:+$HOME }` is `$HOME`). A word holding a quote or an
// expansion that does not close, and a word that spells to nothing, print no
// text the guard reads, and are nil. A word at spellWordDepth is not read,
// and is spellCapped: the bound refuses, never passes (writtenMatches).
func spellWord(w string, depth int, split bool) []string {
	if depth >= spellWordDepth {
		return []string{spellCapped}
	}
	tally(len(w))
	var word []byte
	var sites []varSite
	budget := 4*len(w) + 16
	dq := false
	for i := 0; i < len(w); {
		switch c := w[i]; {
		case c == '"':
			dq = !dq
			i++
		case c == '\\':
			// Inside double quotes a backslash escapes only `$`, a
			// backtick, `"` and itself, and stays text before any other.
			if i+1 < len(w) && (!dq || strings.IndexByte("$`\"\\", w[i+1]) >= 0) {
				i++
			}
			word = append(word, w[i])
			i++
		case c == '\'' && !dq:
			k := strings.IndexByte(w[i+1:], '\'')
			if k < 0 {
				return nil
			}
			word = append(word, w[i+1:i+1+k]...)
			i += k + 2
		case c == '$' && i+1 < len(w) && w[i+1] == '{':
			end := closingDolBrace(w, i+2, &budget)
			if end < 0 {
				return nil
			}
			sites = append(sites, varSite{at: len(word), texts: spellParameterAt(w[i+2:end], depth+1, split && !dq)})
			word = append(word, varMark)
			i = end + 1
		case c == '$' && i+1 < len(w) && w[i+1] == '(':
			end := closingParen(w, i+2, &budget)
			if end < 0 {
				return nil
			}
			word = append(word, unknownMark)
			i = end + 1
		case c == '`':
			end := closingBacktick(w, i+1, &budget)
			if end < 0 {
				return nil
			}
			word = append(word, unknownMark)
			i = end + 1
		case !dq && (c == ' ' || c == '\t' || c == '\n'):
			mark := byte(quotedFieldMark)
			if split {
				mark = fieldMark
			}
			if len(word) == 0 || word[len(word)-1] != mark {
				word = append(word, mark)
			}
			i++
		case c == '$':
			end := simpleParamEnd(w, i+1)
			if end < 0 {
				return nil
			}
			sites = append(sites, varSite{at: len(word), texts: []string{w[i:end]}})
			word = append(word, varMark)
			i = end
		default:
			word = append(word, c)
			i++
		}
	}
	if dq || len(word) == 0 {
		return nil
	}
	return spellWritten(word, sites, nil)
}

// isNameByte reports whether c can continue a shell variable's name.
func isNameByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// writtenMatches reports whether the word at i names one of an entry's
// arg_values as the line wrote it: some text of its written spelling
// (segment.spelled) where it holds a variable, else its known text
// (argValueMatches). It is read by nothing else, so `rm -rf $HOME` names
// `$HOME` to arg_values while every other reading takes the variable as the
// unknown word it is (iss-2609290321312087), and `rm -rf ${DIR:-/}` names the
// root as its default's word does (iss-2609290426544292). A spelling past its
// bound (spellCapped) names every value: the guard stopped reading it, and a
// bound refuses rather than passes.
func writtenMatches(values []string, tokens []string, spelled map[int][]string, i int) bool {
	texts, ok := spelled[i]
	if !ok {
		return argValueMatches(values, knownText(tokens[i]))
	}
	if capped(texts) {
		return true
	}
	for _, w := range texts {
		if argValueMatches(values, w) {
			return true
		}
	}
	return false
}

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
// word that is wholly unknown. The tokenizer reads a `${…}` whole where it
// finds its `}` (parameterExpansion), so a `${` reaches here only as text it
// could not close, or behind an escaped `$`, where reading the rest as
// unknown is the fail-closed side.
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

// variableCarried reports whether the word at index i of s is unknown only
// because it holds parameter expansions (segment.variable): its value is a
// variable's, set before the line ran. Such a word is read as every flag and
// every program name its known text allows, as a substitution's output is,
// with two exceptions, each because a variable is how ordinary commands
// carry a path and a program between commands, and reading it the other way
// refuses them (iss-2609251824244354's false-positive sweep). As a shell's or
// `source`'s script it is not a stream (`bash "$script"`, `. "$ENV_FILE"`): a
// stream path in a variable is data an earlier command carried, the half
// DECISIONS 2026-09-25 (c) defers for a pid list. As a program name nothing
// fixes, it fires no entry that names only its program and a count of
// operands (namesOnlyItsProgram): every command with an operand fits one, so
// `"$GO" build` would read as a pkill.
func variableCarried(s segment, i int) bool {
	_, ok := s.variable[i]
	return ok && i < len(s.tokens) && isUnknown(s.tokens[i])
}

// namesOnlyItsProgram reports whether an entry's pattern constrains nothing
// but its program and how many operands follow it (pkill-by-pattern,
// killall-by-name).
func namesOnlyItsProgram(p Pattern) bool {
	return p.Subcommand == "" && p.Subcommand2 == "" && len(p.Flags) == 0 && len(p.FlagValues) == 0 &&
		len(p.ArgPaths) == 0 && len(p.ArgPrefixes) == 0 && len(p.ArgsFrom) == 0 && p.AfterCD == nil
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
// operand, and one of a set of exact words standing as some operand.
type operandWant struct {
	sub, sub2 string
	min       int
	prefixes  []string
	paths     []PathArg
	values    []string
}

// operandAcceptance returns, for each index i of tokens, whether some reading
// of tokens[i:] as a command's arguments meets want, so the answer for a
// command at site s is accept[s+1]. Each word is read by readWord, every way it
// can be: an unknown dash-word both stands alone and takes the next word, a
// word that may print nothing both is and is not an operand. One reading must
// satisfy every clause together — operand 0 and 1 are the same reading's — so
// the table's state is (word, operands so far, clauses met), filled from the
// end once: linear in the words, whatever the number of places a command can
// sit. spelled is the segment's segment.spelled, read by the arg_values
// clause alone (writtenMatches).
func operandAcceptance(tokens []string, spelled map[int][]string, valueFlags []string, want operandWant, glob func(int) bool) []bool {
	need := want.need()
	nv := 0
	if len(want.values) > 0 {
		nv = 1 // the values are one clause: any one of them meets it
	}
	nb := uint(len(want.prefixes) + len(want.paths) + nv)
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
			if nv > 0 && writtenMatches(want.values, tokens, spelled, i) {
				hits |= 1 << (len(want.prefixes) + len(want.paths))
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
