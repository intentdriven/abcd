package guard

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// segment is one command in command position: the tokens of a single simple
// command, plus the index of the logical line (chain) it belongs to. Segments in
// the same chain are separated by `&&`, `;`, `||`, `|`, `&`, or a subshell
// parenthesis; a newline starts a new chain. The chain is what lets an entry
// require the cd-chain structure (`cd scratch && rm -rf *`) without matching an
// unrelated `rm` on the next line.
//
// A token that carries the output of a command substitution holds unknownMark
// where that output goes (unknown.go): the word is unknown, and every reader
// asks unknown.go what it can be.
type segment struct {
	tokens []string
	chain  int
	// braceGroup records that this command carried an UNQUOTED brace group the
	// tokenizer did not expand: one whose expansion passed the cap
	// (braceexpand.go), or one the look-ahead ran out of budget on. bash
	// rewrites such a group into words the guard never computed, so it cannot
	// say what the argv will be; the flag is how it says so, and Check turns it
	// into a fail-closed block. A group within the cap is expanded instead.
	braceGroup bool
	// heredocUnterminated records that this command opened a here-document
	// whose delimiter line never came, so the tokenizer read the rest of the
	// input as body without knowing whether it WAS body. bash runs such a line
	// (it recovers silently), so it cannot be an error — the hook maps an error
	// to fail-open — and it cannot be an allow either, because a `<<` the
	// classifier misread has swallowed every later command. Check turns the
	// flag into a fail-closed block, the braceGroup precedent.
	heredocUnterminated bool
	// substitutionUnread records a command substitution the tokenizer did not
	// read: one nested inside double quotes past maxQuotedSubstitutionDepth,
	// one whose own text does not tokenize, one whose body holds a case
	// command (whose pattern `)` the span cannot be told from the close by),
	// or one the closing scans ran out of budget on. bash runs its command all
	// the same, so Check turns the flag into a fail-closed block, the
	// braceGroup precedent. It rides on an empty segment of its own, the way an
	// unterminated here-document with no command to hang on does.
	substitutionUnread bool
	// stdinStream records that the command's standard input is a stream of
	// text: a pipe from the command before it, a here-document, or a
	// here-string. A shell reading its script from that stream runs text the
	// guard read as data (iss-2609251640462464).
	stdinStream bool
	// globbed is parallel to tokens and records, per token, that it carried an
	// UNQUOTED, unescaped `*`, `?` or `[` — a word bash expands against the
	// working directory before the command runs, so the bytes here are a
	// PATTERN and the argv may be any word it matches. nil when no token is
	// globbed, which is nearly every segment. Like braceGroup it is a record of
	// what the tokenizer could not resolve, per token instead of per segment,
	// because a glob's expansion IS decidable at the positions an entry
	// constrains (match.go) where a brace group's is not.
	globbed []bool
}

// globAt reports whether token i carried an unquoted glob metacharacter.
func (s segment) globAt(i int) bool {
	return i >= 0 && i < len(s.globbed) && s.globbed[i]
}

// globSlice returns the globbed record for tokens[lo:hi], or nil when nothing in
// the range is globbed — the shape a sub-segment built from a token window
// (Tier 2) carries forward.
func (s segment) globSlice(lo, hi int) []bool {
	if s.globbed == nil {
		return nil
	}
	if hi > len(s.globbed) {
		hi = len(s.globbed)
	}
	if lo >= hi {
		return nil
	}
	for _, g := range s.globbed[lo:hi] {
		if g {
			return s.globbed[lo:hi]
		}
	}
	return nil
}

// tokenize splits a candidate command line into command-position segments,
// honouring shell quoting: single quotes are literal, double quotes take the
// POSIX backslash escapes, and a backslash outside quotes escapes the next
// character. Operators and comments are recognised OUTSIDE quotes only, which is
// exactly what keeps a hazard named inside a quoted argument from ever reaching
// command position.
//
// The tokenizer stops at the token boundary: a command string carried as a DATA
// argument — `sh -c '<payload>'`, `eval '<payload>'`, `bash -lc "<payload>"`,
// `env -S<value>` — stays one opaque token here. Descending into that payload is
// the execute-a-string family's job, done ONCE in Check via expandPayloads
// (payload.go), never in this splitter — so a hazard hidden there is matched
// (iss-200), while an uninspectable payload takes the family's posture.
func tokenize(line string) ([]segment, error) {
	budget := closeScanBudget(len(line))
	return tokenizeAt(line, 0, &budget)
}

// maxQuotedSubstitutionDepth bounds how deeply substitutions nested inside
// double quotes are followed. Each level re-tokenizes its own text, so the
// bound keeps the cost linear in the line. A substitution nested deeper is not
// read, and its command runs all the same, so reaching one raises the
// fail-closed substitutionUnread flag (iss-2609251640353405).
const maxQuotedSubstitutionDepth = 8

const (
	// closeScanPerByte and closeScanFloor size the budget the closing scans
	// (closingParen, closingDoubleQuote, closingBacktick) share across one
	// tokenize call. A scan that finds its close reads its own span, and a
	// byte is read once per double-quoted level around it — at most
	// maxQuotedSubstitutionDepth times, with room to spare here. A scan that
	// finds NO close reads to the end of the line, and each unterminated `$(`
	// inside double quotes started one: quadratic time on a line built of them
	// (review2-guard finding 6). The shared budget makes the total linear, and
	// running it down refuses the substitution as unread.
	closeScanPerByte = 8
	closeScanFloor   = 1 << 12
)

// closeScanBudget is the closing-scan budget for a line of n bytes.
func closeScanBudget(n int) int { return closeScanPerByte*n + closeScanFloor }

// charge spends n units of the closing-scan budget and counts them as work,
// reporting false once the budget cannot cover them.
func charge(budget *int, n int) bool {
	tally(n)
	if *budget < n {
		*budget = 0
		return false
	}
	*budget -= n
	return true
}

// The closing scans answer with an index, or with one of these.
const (
	// closeNone is a span with no close before the input ends: a syntax error
	// bash refuses to run, left as the literal text it is.
	closeNone = -1
	// closeUnread is a span the scan cannot read — a case command in its body,
	// or a spent budget — and is refused, fail-closed, as substitution-unread.
	closeUnread = -2
)

// tokenizeAt is tokenize at a double-quoted substitution depth, spending the
// closing-scan budget of the tokenize call it belongs to.
func tokenizeAt(line string, depth int, budget *int) ([]segment, error) {
	tally(len(line))
	var (
		segs []segment
		toks []string
		cur  []byte
		// hasCur records that a word is being built, which an empty quoted
		// pair (`''`) makes true with no byte in cur.
		hasCur  bool
		chain   int
		pending []heredoc
		// braceGroup rides with the segment being built: an unquoted brace group
		// anywhere in it makes the whole command unexpandable, so the flag is
		// raised once and lands on the segment flushSegment emits.
		braceGroup bool
		// curGlob rides with the WORD being built and globs with the segment:
		// an unquoted `*`, `?` or `[` marks the word as a pattern bash expands.
		// Only the default branch sets it — bytes that arrive through a quote,
		// a backslash or an ANSI-C decode are literal to bash too.
		curGlob bool
		globs   []bool
		// curMask is parallel to cur and records, per byte, whether it reached
		// the tokenizer unquoted (wordStruct) and whether it began its word
		// (wordRawStart) — what the brace expander needs to read a word the way
		// bash does, since a quoted `{`, `,` or `}` is text, not structure.
		curMask []byte
		// curBrace records that the word being built holds a `{` the look-ahead
		// took for a brace group, so flushToken hands it to the expander.
		curBrace bool
		// braceLim bounds what brace expansion may produce and scan across this
		// whole call; past it a word stays unexpanded and its segment is refused.
		braceLim = newBraceLimits()
		// braceBudget is the look-ahead braceExpansionAt may spend across this
		// whole call. See braceScanBudget: without a shared cap the per-`{`
		// forward scan is quadratic in the length of one word.
		braceBudget = braceScanBudget
		// lastList records that the previous operator was a list operator
		// (`&&`, `||`, `|`), whose newline is a line continuation rather than a
		// new command — `cd scratch &&\nrm -rf *` is one chain, not two.
		lastList bool
		// parens is the stack of grouping constructs still open at this point,
		// innermost last. It answers two questions — whether a `<<` reached
		// here is an arithmetic shift or a here-document redirection, and
		// whether a `)` read here can close a substitution — and both are
		// answered by what ENCLOSES the byte, never by the bytes after it. See
		// inArithmetic and inSubstitution.
		parens []parenFrame
		// chainSeq is the highest chain number handed out so far. A newline
		// takes the next one rather than incrementing chain, because a
		// substitution restores its enclosing command's chain when it closes:
		// counting up from the restored value would hand a later line a number
		// an inner line already holds, and precededByCD would read the two as
		// one chain.
		chainSeq int
		// procSubNext records that the redirection branch just read the `<`/`>`
		// of a process substitution, so the `(` that follows opens one.
		procSubNext bool
		// curStdin rides with the segment being built: its standard input is a
		// here-document or a here-string. pipeNext records that the next
		// command emitted reads a pipe. Both land on segment.stdinStream.
		curStdin bool
		pipeNext bool
	)
	// inArithmetic reports whether the innermost construct that can change how a
	// `<<` reads is an arithmetic one. A plain `(` is skipped rather than
	// answered on: inside `(( … ))` it is sub-expression grouping, and at the
	// top level it is a subshell, whose own enclosing context is what decides —
	// either way the frame below it has the answer. A `$(` or a backtick stops
	// the walk, because it starts a FRESH command string, where a here-document
	// is possible again.
	inArithmetic := func() bool {
		for n := len(parens) - 1; n >= 0; n-- {
			switch parens[n].kind {
			case parenArithmetic, parenArithExp:
				return true
			case parenCommandSub, parenBacktick:
				return false
			}
		}
		return false
	}
	// inSubstitution reports whether any construct open here is a command or
	// process substitution, whose close a stray `)` would take early.
	inSubstitution := func() bool {
		for _, p := range parens {
			if p.saved != nil {
				return true
			}
		}
		return false
	}
	// unread raises the fail-closed flag for a substitution the tokenizer
	// could not read, on an empty segment of its own. Once per call is enough:
	// the verdict is the whole command's.
	unreadRaised := false
	unread := func() {
		if !unreadRaised {
			unreadRaised = true
			segs = append(segs, segment{chain: chain, substitutionUnread: true})
		}
	}
	// fail is how the tokenizer refuses a line. Once a substitution has been
	// refused as unread, the quoting after it was read without its span, and a
	// quote that seems to run to the end may be one the span held: bash may
	// well run the line. An error there is mapped to fail-OPEN by the hook, so
	// the refusal stands in its place — the segments read so far, and the flag.
	fail := func(err error) ([]segment, error) {
		if unreadRaised {
			return segs, nil
		}
		return nil, err
	}
	// addCur appends bytes to the word being built with one mask value for all
	// of them: wordStruct for bytes read unquoted, zero for quoted, escaped or
	// decoded ones.
	addCur := func(b []byte, mask byte) {
		cur = append(cur, b...)
		for range b {
			curMask = append(curMask, mask)
		}
		hasCur = true
	}
	flushToken := func() {
		if !hasCur {
			return
		}
		// A case command inside a substitution: its pattern's `)` is read as the
		// substitution's close, and the rest of the case command as the
		// enclosing command's words, so the span is wrong from here on
		// (review2-guard finding 3). bash runs it, so it is refused, not guessed.
		if string(cur) == "case" && inSubstitution() && allReserved(toks) {
			unread()
		}
		// A word holding a brace group is expanded into the words bash would
		// produce, each checked as an argument in its own right
		// (iss-2608282026038930). An assignment in assignment position is the one
		// word bash does not brace-expand (`x={a,b} cmd` sets x to `{a,b}`). A
		// word past the expansion cap stays as written and refuses its segment.
		if curBrace && !(isAssignment(string(cur)) && allAssignments(toks)) {
			if words, ok := expandBraces(bword{b: cur, m: curMask}, &braceLim); ok {
				for _, w := range words {
					toks = append(toks, string(w.b))
					globs = append(globs, w.globbed())
				}
				cur, curMask, hasCur, curGlob, curBrace = nil, nil, false, false, false
				return
			}
			braceGroup = true
		}
		toks = append(toks, string(cur))
		globs = append(globs, curGlob)
		cur, curMask, hasCur, curGlob, curBrace = nil, nil, false, false, false
	}
	flushSegment := func() {
		flushToken()
		if len(toks) > 0 {
			segs = append(segs, segment{
				tokens: toks, chain: chain, braceGroup: braceGroup, globbed: globsOrNil(globs),
				stdinStream: curStdin || pipeNext,
			})
			toks = nil
			globs = nil
			braceGroup = false
			pipeNext = false
		}
		curStdin = false
	}
	// follow reads the text of a command substitution the scan found whole —
	// inside double quotes, or inside an arithmetic expansion — as commands of
	// their own, emitted now because they run first, in this command's chain.
	// Past the depth budget, or when the text does not tokenize, it raises the
	// fail-closed flag instead (iss-2609251640353405).
	follow := func(text string) {
		if depth >= maxQuotedSubstitutionDepth {
			unread()
			return
		}
		isegs, err := tokenizeAt(text, depth+1, budget)
		if err != nil {
			isegs = []segment{{substitutionUnread: true}}
		}
		for _, is := range isegs {
			is.chain = chain
			segs = append(segs, is)
		}
	}
	// arithmetic reads the body of an arithmetic expansion. The expression is
	// not commands — `( 1+2 ) * 3` is grouping and multiplication, never a
	// subshell and a glob (review2-guard finding 5) — but a command
	// substitution inside it runs, so each one is followed.
	arithmetic := func(body string) {
		tally(len(body))
		for j := 0; j < len(body); {
			switch {
			case body[j] == '\\':
				j += 2
			case body[j] == '`':
				k := closingBacktick(body, j+1, budget)
				if k < 0 {
					unread()
					return
				}
				follow(body[j+1 : k])
				j = k + 1
			case body[j] == '$' && j+1 < len(body) && body[j+1] == '(':
				if j+2 < len(body) && body[j+2] == '(' {
					end := arithmeticEnd(body, j, budget)
					if end == closeUnread {
						unread()
						return
					}
					if end >= 0 {
						j += 3 // a nested expansion: its body is read in this same pass
						continue
					}
				}
				k := closingParen(body, j+2, budget)
				if k < 0 {
					unread()
					return
				}
				follow(body[j+2 : k])
				j = k + 1
			default:
				j++
			}
		}
	}
	// openSubstitution suspends the command being built when a command or
	// process substitution opens inside it. The substitution's own command is
	// read as a fresh segment, and closeSubstitution resumes the enclosing one
	// where it stopped — so the argv written AFTER a substitution stays in the
	// enclosing command (`rm $(true) -rf *` is `rm -rf *`, iss-148) instead of
	// becoming a command called `-rf`.
	openSubstitution := func(kind parenKind, pos int, procSub bool) {
		saved := &enclosing{
			toks: toks, globs: globs, cur: cur, curMask: curMask, hasCur: hasCur, curGlob: curGlob,
			curBrace: curBrace, braceGroup: braceGroup, chain: chain, procSub: procSub,
			curStdin: curStdin, pipeNext: pipeNext,
		}
		toks, globs, cur, curMask, hasCur, curGlob, curBrace, braceGroup = nil, nil, nil, nil, false, false, false, false
		curStdin, pipeNext = false, false
		parens = append(parens, parenFrame{kind: kind, pos: pos, saved: saved})
	}
	// closeArithmetic resumes the command an arithmetic expansion suspended,
	// with the number it prints in the word it sat in. What the loop gathered
	// while it stepped the expression is dropped: none of it is a word. The
	// bare `(( … ))` command prints nothing and leaves no word.
	closeArithmetic := func(f parenFrame) {
		e := f.saved
		toks, globs, cur, curMask, hasCur, curGlob, curBrace, braceGroup, chain =
			e.toks, e.globs, e.cur, e.curMask, e.hasCur, e.curGlob, e.curBrace, e.braceGroup, e.chain
		curStdin, pipeNext = e.curStdin, e.pipeNext
		if !f.bare {
			addCur([]byte(arithmeticOperand), 0)
		}
		lastList = false
	}
	// closeSubstitution resumes a suspended enclosing command. What a command
	// substitution prints is unknowable here, so it leaves unknownMark in the
	// word it sat in (unknown.go): standing alone it is a word of its own, one
	// that may also vanish; glued to text it makes that word unknown. A process
	// substitution always contributes exactly one word, the /dev/fd path the
	// shell hands the command, so the operands after it keep their positions.
	closeSubstitution := func(e *enclosing) {
		flushSegment()
		toks, globs, cur, curMask, hasCur, curGlob, curBrace, braceGroup, chain =
			e.toks, e.globs, e.cur, e.curMask, e.hasCur, e.curGlob, e.curBrace, e.braceGroup, e.chain
		curStdin, pipeNext = e.curStdin, e.pipeNext
		if e.procSub {
			addCur([]byte(procSubOperand), 0)
		} else {
			addCur([]byte{unknownMark}, 0)
		}
		lastList = false
	}

	for i := 0; i < len(line); {
		c := line[i]
		// Inside an arithmetic expansion only a command substitution is read:
		// every other byte is expression, stepped over up to the final `)`,
		// which resumes the enclosing command with the number in its word.
		if n := len(parens); n > 0 && parens[n-1].kind == parenArithExp {
			top := parens[n-1]
			switch {
			case i >= top.end:
				parens = parens[:n-1]
				closeArithmetic(top)
				if i == top.end {
					i++
				}
				continue
			case c == '\\':
				i += 2
				continue
			case c == '`':
				openSubstitution(parenBacktick, i, false)
				i++
				continue
			case c == '$' && i+1 < len(line) && line[i+1] == '(' && !(i+2 < len(line) && line[i+2] == '('):
				openSubstitution(parenCommandSub, i+1, false)
				i += 2
				continue
			case c != '"' && c != '\'':
				i++
				continue
			}
		}
		switch {
		case c == '\\':
			if i+1 >= len(line) {
				// A backslash as the LAST byte is bash grammar, not a parse
				// fault: bash 3.2 (the macOS /bin/bash and /bin/sh) and zsh drop
				// it and run the line. bash 5.3 and dash keep it as a literal
				// word instead, under which a hazard flag spelled `--force\`
				// would not run — so "drop" is the reading under which the hazard
				// executes, and the fail-safe one. Returning an error here was
				// worse than either: the pre-tool-use hook maps a tokenizer
				// error to fail-OPEN, so one appended byte walked any command
				// past every blocker (GHSA-5wx3-2c86-fjpx).
				i++
				continue
			}
			if line[i+1] == '\n' {
				// Line continuation: the newline is removed, the chain continues.
				i += 2
				continue
			}
			addCur([]byte{line[i+1]}, 0)
			lastList = false
			i += 2
		case c == '\'':
			j := i + 1
			for j < len(line) && line[j] != '\'' {
				j++
			}
			if j >= len(line) {
				return fail(fmt.Errorf("%w: unterminated single quote", ErrUnparsableCommand))
			}
			addCur([]byte(line[i+1:j]), 0)
			lastList = false
			i = j + 1
		case c == '"':
			j := i + 1
			closed := false
			// A substitution inside double quotes runs as an unquoted one does,
			// and double quotes are its idiomatic spelling, so its command is
			// read as a segment of its own, emitted now because it runs first,
			// in this command's chain (iss-2609251144159533). What bash puts in
			// the word is the substitution's OUTPUT, joined onto the text beside
			// it, so the word holds unknownMark there (unknown.go): under the
			// vanish reading a flag glued to an empty substitution is the flag
			// (iss-2609251640353993), and a dash glued to one is a flag of
			// unknown name. An execute-a-string payload carrying the mark is
			// one the guard cannot read, which the payload reading sees there.
			// One whose end cannot be found stays literal text, and the scan
			// stops looking for more in this string, which keeps it linear;
			// that one is a syntax error bash refuses.
			//
			// An arithmetic expansion is read as one: its output is a number,
			// and only a command substitution inside it runs a command.
			followSubs := true
			for j < len(line) {
				if followSubs && line[j] == '$' && j+2 < len(line) && line[j+1] == '(' && line[j+2] == '(' {
					end := arithmeticEnd(line, j, budget)
					if end == closeUnread {
						unread()
						followSubs = false
						continue
					}
					if end >= 0 {
						arithmetic(line[j+3 : end-1])
						addCur([]byte(arithmeticOperand), 0)
						j = end + 1
						continue
					}
				}
				if followSubs && (line[j] == '`' || (line[j] == '$' && j+1 < len(line) && line[j+1] == '(')) {
					open, inner := j+2, closeNone
					if line[j] == '`' {
						open = j + 1
						inner = closingBacktick(line, open, budget)
					} else {
						inner = closingParen(line, open, budget)
					}
					if inner < 0 {
						if inner == closeUnread {
							unread()
						}
						followSubs = false
						continue
					}
					follow(line[open:inner])
					addCur([]byte{unknownMark}, 0)
					j = inner + 1
					continue
				}
				if line[j] == '\\' && j+1 < len(line) {
					switch line[j+1] {
					case '"', '\\', '$', '`':
						addCur([]byte{line[j+1]}, 0)
					case '\n':
						// Line continuation inside double quotes: both dropped.
					default:
						// Backslash is literal before any other character.
						addCur([]byte{'\\', line[j+1]}, 0)
					}
					j += 2
					continue
				}
				if line[j] == '"' {
					closed = true
					break
				}
				addCur([]byte{line[j]}, 0)
				j++
			}
			if !closed {
				return fail(fmt.Errorf("%w: unterminated double quote", ErrUnparsableCommand))
			}
			hasCur = true
			lastList = false
			i = j + 1
		case c == ' ' || c == '\t' || c == '\r':
			flushToken()
			i++
		case c == '\n':
			flushSegment()
			i++
			// A pending heredoc body starts on the NEXT LINE, and is DATA, not
			// commands: writing a document that names a hazard must never read
			// as running one. It starts there even when the redirection line
			// ends in a list operator and the command list continues after the
			// document — bash collects the bodies at the end of the PHYSICAL
			// line that carried the `<<`, so `cat <<EOF &&` / body / `EOF` /
			// `echo ok` runs `echo ok` with the body as data. Waiting for the
			// list to complete read the body as command text instead: an
			// apostrophe in a document became ErrUnparsableCommand, which the
			// hook maps to fail-OPEN, and a delimiter line reached early
			// swallowed the real commands that followed it as body.
			if len(pending) > 0 {
				next, ok := skipHeredocBodies(line, i, pending)
				if !ok {
					// The delimiter line never came. bash RUNS this (it recovers
					// silently, taking input-to-EOF as the body), so an error is
					// the wrong route for the same reason as the brace group: the
					// hook maps it to fail-open, and `<hazard> <<EOF` plus a
					// newline walked past every blocker (GHSA-5wx3-2c86-fjpx).
					// Succeeding quietly is wrong too — a `<<` the classifier
					// misread has just swallowed every later line as body, and
					// the classifier has been wrong twice (iss-184). The command
					// that opened the document carries the flag; Check turns it
					// into a fail-closed block on both front doors.
					markHeredocUnterminated(&segs, chain)
				}
				i = next
				pending = nil
			}
			// lastList is NOT cleared here: a blank or comment-only line after a
			// list operator does not end the list, and every token-producing
			// branch clears the flag as soon as real content arrives.
			if !lastList {
				chainSeq++
				chain = chainSeq
				pipeNext = false
			}
		case c == '#' && !hasCur:
			// A comment starts only at a word boundary (POSIX): `url/#frag` is
			// part of the token, a bare `#` runs to the end of the line.
			for i < len(line) && line[i] != '\n' {
				i++
			}
		case c == '<' && strings.HasPrefix(line[i:], "<<<"):
			// A herestring, not a heredoc: its payload is an ordinary argument
			// token, so the operator is kept as plain token text. It is the
			// command's standard input.
			addCur([]byte("<<<"), wordStruct)
			curStdin = true
			lastList = false
			i += 3
		case c == '<' && strings.HasPrefix(line[i:], "<<"):
			// A heredoc redirection (`<<`, `<<-`) — but only when nothing
			// arithmetic encloses the operator and a delimiter word follows.
			// `(( x = 1<<20 ))` is an arithmetic shift, and taking it for a
			// heredoc would swallow every later line as body text and silently
			// unguard them.
			//
			// WHAT ENCLOSES the `<<` is what tells the two apart. Inside an
			// arithmetic context — the bare `(( … ))` command, or a `$(( … ))`
			// expansion, which is read whole before its bytes reach here —
			// bash has no redirection at all, so a `<<` there is a shift, full
			// stop; outside one, a delimiter-shaped word opens a document.
			// Deciding instead on the bytes AFTER the delimiter word ("does a
			// paren pair close right here?") reads only the flattest shift:
			// `(( (1 << n) + 1 ))` closes its sub-expression with a SINGLE `)`,
			// so `n` was taken for a delimiter — and a later line equal to `n`
			// then swallowed every command between the two with no signal at
			// all, while a bit mask with no such line blocked as an
			// unterminated document.
			//
			// The check comes BEFORE readHeredocDelim so an arithmetic
			// expression can never reach that reader's unterminated-quote error,
			// which the pre-tool-use hook maps to fail-OPEN. Where the enclosing
			// context stays ambiguous — an arithmetic frame left open on an
			// earlier construct, a `$(` in between — the reading falls to the
			// here-document side, and skipHeredocBodies' fail-closed block below
			// is the answer, never an error.
			if inArithmetic() {
				addCur([]byte("<<"), wordStruct)
				lastList = false
				i += 2
				continue
			}
			hd, next, err := readHeredocDelim(line, i+2)
			if err != nil {
				return fail(err)
			}
			// A word that cannot start an unquoted delimiter — `20` in a
			// `$((1<<20))` reached outside any paren — is not one.
			if !hd.quoted && !isDelimStart(hd.delim) {
				addCur([]byte("<<"), wordStruct)
				lastList = false
				i += 2
				continue
			}
			flushToken()
			pending = append(pending, hd)
			curStdin = true
			i = next
		case c == '>' || (c == '<' && !strings.HasPrefix(line[i:], "<<")):
			// A redirection operator (`>`, `>>`, `>|`, `>&`, `<`, `<>`, `<&`),
			// optionally prefixed by an fd digit that sits in cur. `<<`/`<<<`
			// are recognised above; process substitution `>(...)`/`<(...)` is
			// not a redirection and keeps its prior handling. A redirection
			// terminates the current word and its target is a filename or fd,
			// never a command in command position — so both the operator and
			// the target are dropped. Without this, gluing a redirection onto a
			// token (`git push --force>/dev/null`) mutated the flag token so
			// every blocker missed and the verdict was a silent allow, and a
			// leading redirection (`>/dev/null git push --force`) displaced the
			// command out of position and degraded a Tier-1 block to a warn.
			if i+1 < len(line) && line[i+1] == '(' {
				// Process substitution: the `(` that follows opens it, and the
				// operator byte is not part of any word.
				procSubNext = true
				lastList = false
				i++
				break
			}
			opEnd := i + 1
			if c == '>' {
				if opEnd < len(line) && (line[opEnd] == '>' || line[opEnd] == '|' || line[opEnd] == '&') {
					opEnd++
				}
			} else if opEnd < len(line) && (line[opEnd] == '>' || line[opEnd] == '&') {
				opEnd++
			}
			// A pure-digit cur immediately before the operator is the fd prefix
			// (`2>`, `1>&2`), part of the redirection rather than a token; drop
			// it. Otherwise flush the real word the operator terminates.
			if hasCur && isAllDigits(cur) {
				cur, curMask = nil, nil
				hasCur = false
				curGlob = false
			} else {
				flushToken()
			}
			i = skipRedirectTarget(line, opEnd)
			lastList = false
		case c == '&' && i+1 < len(line) && line[i+1] == '>':
			// bash's `&>` / `&>>`: redirect both stdout and stderr. It has to be
			// recognised BEFORE the list-operator case below, which would read
			// the leading `&` as a background/`&&` operator and call
			// flushSegment -- splitting one simple command into two segments and
			// dropping the command's own dangerous flags out of command
			// position, so every blocker missed and the verdict was a silent
			// allow (`git push &>/dev/null --force origin main`). Unlike `>`/`<`,
			// a digit before `&>` is NOT an fd prefix in bash (`f 2&>x` passes
			// `2` to `f` and still redirects both streams), so the preceding word
			// is a real token and is flushed, never dropped.
			opEnd := i + 2
			if opEnd < len(line) && line[opEnd] == '>' {
				opEnd++
			}
			flushToken()
			i = skipRedirectTarget(line, opEnd)
			lastList = false
		case c == '$' && i+1 < len(line) && line[i+1] == '\'':
			// bash ANSI-C quoting: $'...' contributes its escape-decoded body to
			// the SAME word, exactly as '...' contributes its raw body. Without
			// this the leading `$` fell through to the default word branch and
			// prefixed the token (`$'--force'` tokenised to `$--force`), so a
			// blocker naming `--force` missed — a silent allow of the very argv
			// bash hands the child. Quoting must not change argument semantics
			// (doc.go): `git push $'--force'` fires, like `git push '--force'`.
			decoded, next, err := readAnsiCQuote(line, i+2)
			if err != nil {
				return fail(err)
			}
			addCur(decoded, 0)
			lastList = false
			i = next
		case c == '$' && i+1 < len(line) && line[i+1] == '"':
			// bash locale quoting: $"..." is a plain double-quoted word with the
			// `$` stripped (the translation is identity for a tokenizer). Skip the
			// `$` so the double-quote branch reads the string; the same silent
			// allow as $'...' otherwise.
			i++
		case c == '$' && i+2 < len(line) && line[i+1] == '(' && line[i+2] == '(':
			// An arithmetic expansion (review2-guard finding 5): its expression
			// is not commands, and the number it prints is no flag, subcommand
			// or path an entry names. It suspends the enclosing command like a
			// substitution, and the loop skips its bytes (the parenArithExp
			// step above) except where a command substitution inside it opens
			// — that one runs, and is read as a command here, in this loop, so
			// a here-document it opens takes its body from the lines below as
			// any other does. `$((` that does not close as an expansion is
			// bash's other reading, a command substitution opening with a
			// subshell, and falls to the `(` branch below.
			end := arithmeticEnd(line, i, budget)
			if end == closeUnread {
				unread()
			}
			if end < 0 {
				addCur([]byte{c}, wordStruct)
				lastList = false
				i++
				break
			}
			openSubstitution(parenArithExp, i, false)
			parens[len(parens)-1].end = end
			lastList = false
			i += 3
		case c == '&' || c == '|' || c == ';' || c == '(' || c == ')' || c == '`':
			// A backtick is command substitution, identical to `$( … )`: the inner
			// command EXECUTES before its output is used. `$( … )` already splits
			// into command position via the `(`/`)` operators above; a backtick is
			// the same hazard in its other spelling, so it splits the same way —
			// both the opening and the closing backtick end the current segment,
			// leaving the substituted command as its own command-position segment.
			// Without this the byte fell into the default word branch and a
			// top-level `` `gh repo delete owner/repo` `` was a silent allow while
			// its `$( … )` twin blocked (gh-312). Inside single quotes the byte is
			// literal and never reaches here, matching the shell.
			//
			// A substitution that OPENS here suspends the enclosing command
			// rather than ending it (openSubstitution), so its inner command is
			// its own segment and the enclosing one resumes when it closes.
			procSub := procSubNext
			procSubNext = false
			if c == '(' && (procSub || (i > 0 && line[i-1] == '$')) {
				if !procSub && hasCur && len(cur) > 0 && cur[len(cur)-1] == '$' {
					// The `$` introducer is not part of the word.
					cur, curMask = cur[:len(cur)-1], curMask[:len(curMask)-1]
					hasCur = len(cur) > 0
				}
				openSubstitution(parenCommandSub, i, procSub)
				lastList = false
				i++
				continue
			}
			if c == '`' && !(len(parens) > 0 && parens[len(parens)-1].kind == parenBacktick) {
				openSubstitution(parenBacktick, i, false)
				lastList = false
				i++
				continue
			}
			flushSegment()
			if c == '(' && i+1 < len(line) && line[i+1] == '(' {
				// The bare arithmetic command `(( … ))` is read as the
				// expansion is, when its parens close as one: an expression,
				// not a subshell holding commands and globs.
				end := arithmeticClose(line, i, budget)
				if end == closeUnread {
					unread()
				}
				if end >= 0 {
					openSubstitution(parenArithExp, i, false)
					parens[len(parens)-1].end = end
					parens[len(parens)-1].bare = true
					lastList = false
					i += 2
					continue
				}
			}
			switch c {
			case '(':
				// `((` — two parens with NOTHING between them — opens an
				// arithmetic context, which is how bash lexes it too; `( (cmd) )`
				// and `$( (cmd) )`, which have a separator, do not. The inner
				// paren converts the frame the outer one pushed, so both halves
				// close it and `(((a))` reads as arithmetic plus one ordinary
				// group. A `$((` reaches here only when it did not close as an
				// expansion (the `$((` branch above), and is read the same way.
				kind := parenGroup
				if n := len(parens); n > 0 && parens[n-1].pos == i-1 && parens[n-1].kind != parenArithmetic {
					parens[n-1].kind = parenArithmetic
					kind = parenArithmetic
				}
				parens = append(parens, parenFrame{kind: kind, pos: i})
			case ')', '`':
				// A backtick is its own closer: reaching this branch means the
				// innermost open frame is a backtick (an opening one was taken
				// above), so both bytes pop. A frame that suspended an enclosing
				// command resumes it; a plain group or an arithmetic half does
				// not, which is what keeps a nested bare `(` inside `$( … )` from
				// closing the substitution early.
				if n := len(parens); n > 0 {
					top := parens[n-1]
					parens = parens[:n-1]
					if top.saved != nil {
						closeSubstitution(top.saved)
					}
				}
			}
			if c == '|' && i+1 < len(line) && line[i+1] == '&' {
				// `|&` pipes stdout and stderr both: a pipe.
				pipeNext = true
				lastList = true
				i += 2
				continue
			}
			if (c == '&' || c == '|') && i+1 < len(line) && line[i+1] == c {
				pipeNext = false
				lastList = true
				i += 2
				continue
			}
			switch c {
			case '|':
				pipeNext = true
			case ';', '&':
				pipeNext = false
			}
			// A single pipe continues the list across a newline; `;`, `&`, the
			// grouping parens, and a backtick boundary do not.
			lastList = c == '|'
			i++
		case c == '{':
			// An unquoted brace group is EXPANSION, not text: bash rewrites
			// `git push {--force,} origin main` into byte-identical `--force`
			// argv, and reading the literal token `{--force,}` let a Tier-1
			// hazard through as a silent allow. The look-ahead decides whether
			// this brace can open a group; flushToken expands the finished word
			// the way bash does (braceexpand.go) and checks every word it
			// produces. A look-ahead that runs out of budget can no longer tell
			// a group from a literal, so the segment is refused — raised on the
			// segment, never as ErrUnparsableCommand, which the pre-tool-use hook
			// maps to fail-OPEN. The bytes stay in the word either way.
			group, exhausted := braceExpansionAt(line, i, &braceBudget)
			mask := wordStruct
			if !hasCur && (i == 0 || isWordBreak(line[i-1])) &&
				(i+1 >= len(line) || line[i+1] == '}' || isWordBreak(line[i+1])) {
				mask |= wordNotOpener
			}
			switch {
			case exhausted:
				braceGroup = true
			case group:
				curBrace = true
			}
			addCur([]byte{c}, mask)
			lastList = false
			i++
		case c == unknownMark:
			// A payload re-read from a word that carried a substitution's
			// output: the mark stays where the output goes, and the word it
			// lands in is unknown (unknown.go).
			addCur([]byte{c}, 0)
			lastList = false
			i++
		default:
			// An unquoted glob metacharacter makes the word a PATTERN: bash
			// expands it against the working directory before exec, so
			// `pus?` is `push` whenever a file called push exists. The bytes are
			// kept — the matcher compares them as a pattern where an entry
			// constrains the position (GHSA-3w99-pgv4-8g55) — and the record
			// is what lets it tell this `*` from a quoted one.
			if c == '*' || c == '?' || c == '[' {
				curGlob = true
			}
			addCur([]byte{c}, wordStruct)
			lastList = false
			i++
		}
	}
	flushSegment()
	// A substitution still open when the input ends is a syntax error bash
	// refuses to run, but the guard reads it fail-safe all the same: every
	// suspended enclosing command is resumed and emitted, so no token written
	// before an unterminated `$(` or backtick escapes the check.
	for n := len(parens) - 1; n >= 0; n-- {
		switch {
		case parens[n].kind == parenArithExp:
			closeArithmetic(parens[n])
		case parens[n].saved != nil:
			closeSubstitution(parens[n].saved)
		default:
			continue
		}
		flushSegment()
	}
	// A here-document still pending when the INPUT ends is in the same state as
	// one whose delimiter line never came, and takes the same fail-closed
	// verdict. Reaching the end of the input without ever crossing a newline
	// dropped the pending document silently, which split the two front doors:
	// `guard hook` sees the candidate as the host sent it (`cat <<EOF` and its
	// newline) and blocked, while `guard check` trims a trailing newline off
	// stdin and cleared the very same command. The verdict belongs to the
	// command, not to whether its last byte is a newline.
	if len(pending) > 0 {
		markHeredocUnterminated(&segs, chain)
	}
	return segs, nil
}

// arithmeticOperand is the word an arithmetic expansion leaves in its
// command: a number, which is all `$(( … ))` can print. Its value is not
// modelled, and needs not be: no flag, subcommand or path an entry names is a
// number.
const arithmeticOperand = "0"

// allReserved reports whether every token so far is a reserved word, so the
// next word stands in command position.
func allReserved(toks []string) bool {
	for _, t := range toks {
		if !reserved[t] {
			return false
		}
	}
	return true
}

// arithmeticEnd returns the index of the final `)` of the arithmetic expansion
// whose `$((` begins at line[dollar], closeNone when the bytes do not close as
// one, or closeUnread when the budget ran out. bash's own test is the one read
// here: the expansion's two `)` are adjacent and close the two `(` — `$((a) +
// (b))` closes its first group early and is a command substitution instead.
func arithmeticEnd(line string, dollar int, budget *int) int {
	return arithmeticClose(line, dollar+1, budget)
}

// arithmeticClose is arithmeticEnd for the `((` whose first paren is at
// line[open]: the bare arithmetic command, or an expansion past its `$`.
func arithmeticClose(line string, open int, budget *int) int {
	outer := closingParenMode(line, open+1, budget, true)
	if outer < 0 {
		return outer
	}
	inner := closingParenMode(line, open+2, budget, true)
	if inner == closeUnread {
		return closeUnread
	}
	if inner != outer-1 {
		return closeNone
	}
	return outer
}

// closingParen returns the index of the `)` that closes a `$(` whose body
// starts at i, closeNone when none does, or closeUnread when the span cannot
// be read. It reads the body's own grammar, not only its quoting — single
// quotes, double quotes with their own substitutions, backticks, backslashes,
// a `#` comment to the end of its line, an arithmetic expansion, and a
// here-document's body — so a `)` inside any of them is not the close
// (review2-guard finding 3). A body holding a case command is refused: its
// patterns end in an unbalanced `)`, and guessing which one closes the span is
// how a flag glued after it got through.
func closingParen(line string, i int, budget *int) int {
	return closingParenMode(line, i, budget, false)
}

// closingParenMode is closingParen reading either a command body or, with
// arith, an arithmetic expression, which has no comment, no here-document and
// no case command: `16#ff` is a number and `1 << 2` a shift.
func closingParenMode(line string, i int, budget *int, arith bool) int {
	start := i
	depth := 1
	var pending []heredoc
	for i < len(line) {
		if !charge(budget, 1) {
			return closeUnread
		}
		c := line[i]
		switch {
		case c == '\\':
			i += 2
			continue
		case c == '\'':
			k := strings.IndexByte(line[i+1:], '\'')
			if k < 0 {
				return closeNone
			}
			if !charge(budget, k+1) {
				return closeUnread
			}
			i += k + 2
			continue
		case c == '"':
			k := closingDoubleQuote(line, i+1, budget)
			if k < 0 {
				return k
			}
			i = k + 1
			continue
		case c == '`':
			k := closingBacktick(line, i+1, budget)
			if k < 0 {
				return k
			}
			i = k + 1
			continue
		case c == '$' && !arith && i+2 < len(line) && line[i+1] == '(' && line[i+2] == '(':
			if end := arithmeticEnd(line, i, budget); end >= 0 {
				i = end + 1
				continue
			} else if end == closeUnread {
				return closeUnread
			}
		case arith:
			// An arithmetic expression: only the parens below are structure.
		case c == '#' && (i == start || isWordBreak(line[i-1])):
			k := strings.IndexByte(line[i:], '\n')
			if k < 0 {
				return closeNone
			}
			if !charge(budget, k) {
				return closeUnread
			}
			i += k
			continue
		case c == 'c' && keywordAt(line, start, i, "case"):
			return closeUnread
		case c == '<' && strings.HasPrefix(line[i:], "<<<"):
			i += 3
			continue
		case c == '<' && strings.HasPrefix(line[i:], "<<"):
			hd, next, err := readHeredocDelim(line, i+2)
			if err != nil {
				return closeNone
			}
			if hd.quoted || isDelimStart(hd.delim) {
				pending = append(pending, hd)
			}
			if !charge(budget, next-i) {
				return closeUnread
			}
			i = next
			continue
		case c == '\n' && len(pending) > 0:
			next, ok := skipHeredocBodies(line, i+1, pending)
			if !ok {
				return closeNone
			}
			if !charge(budget, next-i) {
				return closeUnread
			}
			pending = nil
			i = next
			continue
		}
		switch c {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i
			}
		}
		i++
	}
	return closeNone
}

// keywordAt reports whether line[i:] is the reserved word kw standing in
// command position of a command body that starts at start: bounded by word
// breaks, and preceded by the body's start, a command separator, or another
// reserved word (`then case …`).
func keywordAt(line string, start, i int, kw string) bool {
	if !strings.HasPrefix(line[i:], kw) {
		return false
	}
	if end := i + len(kw); end < len(line) && !isWordBreak(line[end]) {
		return false
	}
	p := i - 1
	for p >= start && (line[p] == ' ' || line[p] == '\t') {
		p--
	}
	if p < start {
		return true
	}
	switch line[p] {
	case ';', '&', '|', '(', '\n', '{', '!':
		return true
	}
	if p+1 == i {
		return false // glued to the word before it
	}
	q := p
	for q >= start && !isWordBreak(line[q]) {
		q--
	}
	return reserved[line[q+1:p+1]] || line[q+1:p+1] == "time"
}

// closingDoubleQuote returns the index of the `"` that closes a double-quoted
// string whose body starts at i, stepping over escapes and the substitutions
// inside it, or one of closeNone and closeUnread.
func closingDoubleQuote(line string, i int, budget *int) int {
	for i < len(line) {
		if !charge(budget, 1) {
			return closeUnread
		}
		switch {
		case line[i] == '\\':
			i += 2
			continue
		case line[i] == '"':
			return i
		case line[i] == '$' && i+1 < len(line) && line[i+1] == '(':
			k := closingParen(line, i+2, budget)
			if k < 0 {
				return k
			}
			i = k + 1
			continue
		case line[i] == '`':
			k := closingBacktick(line, i+1, budget)
			if k < 0 {
				return k
			}
			i = k + 1
			continue
		}
		i++
	}
	return closeNone
}

// closingBacktick returns the index of the unescaped backtick that closes one
// whose body starts at i, or one of closeNone and closeUnread.
func closingBacktick(line string, i int, budget *int) int {
	for i < len(line) {
		if !charge(budget, 1) {
			return closeUnread
		}
		switch line[i] {
		case '\\':
			i += 2
			continue
		case '`':
			return i
		}
		i++
	}
	return closeNone
}

// parenKind names what an unclosed `(` opened, to the one precision the
// tokenizer needs: whether a `<<` inside it is an arithmetic shift.
type parenKind uint8

const (
	// parenGroup is a plain `(`: a subshell at the top level, sub-expression
	// grouping inside arithmetic. It decides nothing on its own.
	parenGroup parenKind = iota
	// parenCommandSub is the `(` of a `$( … )` — a fresh command string.
	parenCommandSub
	// parenBacktick is an open backtick: command substitution in its other
	// spelling, and its own closer.
	parenBacktick
	// parenArithmetic is one half of a `((` pair, or of a `$((` that did not
	// close as an expansion.
	parenArithmetic
	// parenArithExp is an arithmetic expansion `$(( … ))` whose close the scan
	// found (arithmeticEnd): the loop steps its expression and reads only the
	// substitutions inside it.
	parenArithExp
)

// parenFrame is one unclosed grouping construct: its kind, and the offset of the
// byte that opened it — which is what lets the next `(` see that it is adjacent
// and convert the pair into an arithmetic context.
type parenFrame struct {
	kind parenKind
	pos  int
	// saved is the enclosing command a substitution suspended, resumed when
	// this frame closes; nil for a frame that suspends nothing (a subshell or
	// grouping paren, an arithmetic half).
	saved *enclosing
	// end is the offset of an arithmetic expansion's final `)`, and bare
	// records that it is the `(( … ))` command, which leaves no word.
	end  int
	bare bool
}

// enclosing is the state of a command suspended by a substitution opening
// inside it: its tokens so far, the word in progress, and the chain it belongs
// to, which a newline inside the substitution must not change.
type enclosing struct {
	toks       []string
	globs      []bool
	cur        []byte
	curMask    []byte
	hasCur     bool
	curGlob    bool
	curBrace   bool
	braceGroup bool
	chain      int
	// procSub records that the substitution is a process substitution, which
	// leaves one /dev/fd operand in the word it sat in.
	procSub bool
	// curStdin and pipeNext are the enclosing command's own standard-input
	// record (tokenizeAt), suspended with the rest of it.
	curStdin bool
	pipeNext bool
}

// procSubOperand is the word a process substitution leaves in the enclosing
// command: the /dev/fd path bash hands it (the descriptor number varies; the
// shape does not). It is an operand, never a flag, so an entry's flag scan
// passes over it and its operand positions stay where the shell puts them.
const procSubOperand = "/dev/fd/63"

// globsOrNil returns the per-token glob record, or nil when no token in it is
// globbed — the common case, kept allocation-free for the matcher's compares.
func globsOrNil(globs []bool) []bool {
	for _, g := range globs {
		if g {
			return globs
		}
	}
	return nil
}

const (
	// braceEntryID is the reserved id an unexpandable brace group is reported
	// under. Like syntheticEntryID it names a verdict the Pattern language
	// cannot express — "this word is not the word that will run" — so no
	// registry entry may claim it and it must never index Registry.Entries.
	braceEntryID = "brace-expansion-unexpanded"

	familyBrace = "brace expansion"

	// heredocEntryID is the reserved id an unterminated here-document is
	// reported under: another verdict the Pattern language cannot express ("the
	// rest of this input may be commands or may be a document"), so no registry
	// entry may claim it and it must never index Registry.Entries.
	heredocEntryID = "heredoc-unterminated"

	familyHeredoc = "here-document"

	// substitutionEntryID is the reserved id a command substitution the guard
	// stopped reading is reported under: one nested inside double quotes past
	// maxQuotedSubstitutionDepth, or one whose text does not tokenize. Its
	// command runs all the same, so the verdict is another the Pattern language
	// cannot express, and no registry entry may claim the id.
	substitutionEntryID = "substitution-unread"

	familySubstitution = "command substitution"

	// braceScanBudget bounds the TOTAL look-ahead braceExpansionAt may spend
	// across one tokenize call. The scan reads forward from every structural
	// `{`, so a word made of nothing but `{` re-reads the same tail once per
	// byte: a megabyte of them — well inside the guard's own stdin cap — took
	// minutes, which is a hang on the PreToolUse path reachable by any command
	// an agent can be asked to run. The budget is generous next to any real
	// command and small next to that, and exhausting it is fail-closed.
	braceScanBudget = 1 << 16
)

// braceExpansionBlockSignal is the fail-closed verdict for a command carrying an
// unquoted brace group the guard did not expand — one past the expansion cap.
// It is a BLOCK rather than a warn because the group can carry any flag at all
// — `{--force,}` expands to argv a Tier-1 blocker names — and an unexpanded
// group is one the guard has not read.
func braceExpansionBlockSignal() payloadSignal {
	return payloadSignal{
		id:      braceEntryID,
		verdict: VerdictBlock,
		family:  familyBrace,
		reason: "This command carries an unquoted brace group that expands into more words than the guard reads " +
			"(its cap is " + strconv.Itoa(braceMaxWords) + " words per command line), so the arguments that would be passed are ones it has not checked.",
		successor: "Split the command so each part expands to fewer words, spell the words out, or quote the braces if they are meant literally, " +
			"so the guard checks the command that actually runs.",
	}
}

// braceExpansionAt reports whether the `{` at line[i] — reached as a structural,
// unquoted byte, so quoted spellings never arrive here — opens a brace group
// bash would EXPAND into several words, rather than an ordinary literal brace.
// Three things separate the two, and each is a shape bash itself does not
// expand:
//
//   - `${…}` is parameter expansion. bash's own brace scanner skips it on
//     exactly this test — the RAW byte before the brace — so `echo ${HOME}` and
//     `${x:-a,b}` are untouched. The raw line is read rather than the decoded
//     word so a QUOTED dollar (`'$'{a,b}`, which bash does expand) cannot buy
//     the exemption.
//   - A group needs an alternative: a comma, or a `..` range. `{a}`, a lone
//     `{`, and `awk {print}` are literals in bash and stay literals here.
//   - A group lives inside ONE word. An unquoted space or operator ends the
//     word, so the reserved-word group command `{ git push --force; }` is not a
//     brace group — its inner command still reaches command position, where the
//     blocker for it already fires.
//
// Everything else is read fail-closed. Nested groups count (`{{a,b}}` expands,
// though the comma is not at the outer group's own level), and an alternative
// found inside quotes still counts — a comma the scan cannot rule out is one it
// must assume bash will act on.
func braceExpansionAt(line string, i int, budget *int) (group, exhausted bool) {
	// `${…}` is parameter expansion — unless the `$` is ITSELF escaped, which
	// makes it a literal dollar and leaves the brace group behind it live:
	// bash expands `\${a,b}` to `$a $b`. So the exemption needs the raw
	// preceding byte to be a `$` that is not escaped.
	if i > 0 && line[i-1] == '$' && !escapedAt(line, i-1) {
		return false, false
	}
	// The look-ahead is capped by the shared budget rather than by the line, so
	// the total scanning across one tokenize call is linear however many `{`
	// bytes the line holds.
	end, truncated := len(line), false
	if *budget < end-i {
		end, truncated = i+*budget, true
	}
	j := i
	defer func() { *budget -= j - i; tally(j - i) }()

	depth, expands := 0, false
	for j < end {
		switch c := line[j]; {
		case c == '\\':
			j += 2
		case c == '\'' || c == '"':
			// Scan the quoted run for structure only: its bytes cannot close the
			// group, but an alternative inside it is still counted.
			for j++; j < end && line[j] != c; j++ {
				if line[j] == ',' || (line[j] == '.' && j+1 < end && line[j+1] == '.') {
					expands = true
				}
				if c == '"' && line[j] == '\\' {
					j++
				}
			}
			j++
		case c == '`', (c == '$' || c == '<' || c == '>') && j+1 < end && line[j+1] == '(':
			// A command or process substitution does NOT end the word — bash
			// brace-expands straight through one, so `{--force,$(true)}` and its
			// backtick twin both expand to `--force` argv. Treating the paren or
			// the backtick as a word terminator (the plain `(`/`)` case below)
			// read "no group" exactly where bash reads one, which left the
			// mutate-the-flag bypass open under a variant of itself.
			var alt bool
			j, alt = skipSubstitution(line, j, end)
			expands = expands || alt
		case c == '{':
			depth++
			j++
		case c == '}':
			// A `}` ends the group only once a separator has been seen at this
			// level. bash does not stop at the first one either: it keeps
			// looking, so `{msg},--no-verify}` is a real group whose
			// alternatives are `msg}` and `--no-verify` (checked against bash
			// 5.3). Stopping at that inner brace read the whole word as inert
			// text and reopened the very bypass this scan closes, four
			// characters wider — and in a form that runs cleanly, since `-m`
			// swallows the junk alternative as the commit message. An
			// unseparated `}` is therefore an ordinary byte inside a group that
			// is still open, which is exactly how bash reads it.
			switch {
			case depth > 1:
				depth--
			case expands:
				return true, false
			}
			j++
		case c == ',':
			expands = true
			j++
		case c == '.' && j+1 < end && line[j+1] == '.':
			expands = true
			j += 2
		case c == ' ' || c == '\t' || c == '\n' || c == ';' ||
			c == '&' || c == '|' || c == '(' || c == ')':
			return false, false
		default:
			j++
		}
	}
	// Running out of line means no closing brace, which bash leaves unexpanded.
	// Running out of BUDGET means the scan no longer knows, and a guard that
	// cannot tell a group from a literal refuses the segment.
	return truncated, truncated
}

// isWordBreak reports whether a byte ends the word before it, so the byte
// after it begins a new word.
func isWordBreak(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', ';', '&', '|', '(', ')', '<', '>':
		return true
	}
	return false
}

// allAssignments reports whether every token so far is a NAME=VALUE prefix, so
// the next assignment-shaped word is still in assignment position.
func allAssignments(toks []string) bool {
	for _, t := range toks {
		if !isAssignment(t) {
			return false
		}
	}
	return true
}

// escapedAt reports whether the byte at p is preceded by an odd number of
// backslashes, which is what makes it escaped rather than structural.
func escapedAt(line string, p int) bool {
	n := 0
	for p--; p >= 0 && line[p] == '\\'; p-- {
		n++
	}
	return n%2 == 1
}

// skipSubstitution scans the substitution beginning at line[j] — a backtick
// pair, or a `$(`/`<(`/`>(` group — and returns the index just past it together
// with whether it held a brace ALTERNATIVE. It is deliberately structural only:
// what the substitution would produce is unknowable here, so a comma or a `..`
// inside one is counted rather than reasoned about, the same fail-closed reading
// the quoted-run branch takes. An unterminated substitution consumes to end.
func skipSubstitution(line string, j, end int) (next int, alt bool) {
	mark := func(k int) {
		if line[k] == ',' || (line[k] == '.' && k+1 < end && line[k+1] == '.') {
			alt = true
		}
	}
	if line[j] == '`' {
		for j++; j < end && line[j] != '`'; j++ {
			mark(j)
		}
		return j + 1, alt
	}
	depth := 0
	for j++; j < end; j++ { // j++ steps over the `$`/`<`/`>` introducer
		switch line[j] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return j + 1, alt
			}
		default:
			mark(j)
		}
	}
	return end, alt
}

// readAnsiCQuote decodes a bash ANSI-C `$'...'` body that begins at start (the
// byte just after the opening quote) and returns the decoded bytes together with
// the index just past the closing quote. Inside `$'...'` a backslash introduces
// an escape — so `\'` does not end the string — and the common escapes are
// resolved so an encoded spelling of a hazard (`$'\x2d\x2dforce'`) tokenises to
// the same bytes bash would hand the child (`--force`).
//
// bash ends the string at the first byte an escape decodes to NUL (`\x00`,
// `\0`, `\u0000`, `\c@`, …): what follows up to the closing quote is read and
// dropped, so `$'\x00'git` is `git`. The guard reads it the same way, and that
// is also what keeps unknownMark unforgeable (unknown.go): no decoded byte is
// ever a NUL.
func readAnsiCQuote(line string, start int) ([]byte, int, error) {
	var out []byte
	ended := false
	for i := start; i < len(line); {
		switch {
		case line[i] == '\'':
			return out, i + 1, nil
		case line[i] == '\\' && i+1 < len(line):
			decoded, next := decodeAnsiCEscape(line, i+1)
			if nul := bytes.IndexByte(decoded, 0); nul >= 0 && !ended {
				out = append(out, decoded[:nul]...)
				ended = true
			}
			if !ended {
				out = append(out, decoded...)
			}
			i = next
		default:
			if !ended {
				out = append(out, line[i])
			}
			i++
		}
	}
	return nil, 0, fmt.Errorf("%w: unterminated $'' quote", ErrUnparsableCommand)
}

// decodeAnsiCEscape resolves one ANSI-C escape whose leading backslash has
// already been consumed; p indexes the character after the backslash. It returns
// the decoded bytes and the index just past the escape. An unrecognised escape
// keeps the backslash and the character, matching bash.
func decodeAnsiCEscape(line string, p int) ([]byte, int) {
	switch c := line[p]; c {
	case 'n':
		return []byte{'\n'}, p + 1
	case 't':
		return []byte{'\t'}, p + 1
	case 'r':
		return []byte{'\r'}, p + 1
	case 'a':
		return []byte{'\a'}, p + 1
	case 'b':
		return []byte{'\b'}, p + 1
	case 'f':
		return []byte{'\f'}, p + 1
	case 'v':
		return []byte{'\v'}, p + 1
	case 'e', 'E':
		return []byte{0x1b}, p + 1
	case '\\', '\'', '"', '?':
		return []byte{c}, p + 1
	case 'x':
		val, n := 0, 0
		for j := p + 1; j < len(line) && n < 2 && isHexDigit(line[j]); j++ {
			val = val*16 + hexValue(line[j])
			n++
		}
		if n == 0 {
			return []byte{c}, p + 1
		}
		return []byte{byte(val)}, p + 1 + n
	case '0', '1', '2', '3', '4', '5', '6', '7':
		val, n := 0, 0
		for j := p; j < len(line) && n < 3 && line[j] >= '0' && line[j] <= '7'; j++ {
			val = val*8 + int(line[j]-'0')
			n++
		}
		return []byte{byte(val)}, p + n
	case 'u', 'U':
		// bash \uHHHH (up to 4 hex) and \UHHHHHHHH (up to 8 hex): a Unicode code
		// point, UTF-8 encoded. Without these an ASCII hazard spelled as
		// `$'--force'` decoded to --force in the shell but not here, so a
		// Tier-1 blocker missed on the same bytes the \x and octal forms already
		// close.
		width := 4
		if c == 'U' {
			width = 8
		}
		val, n := 0, 0
		for j := p + 1; j < len(line) && n < width && isHexDigit(line[j]); j++ {
			val = val*16 + hexValue(line[j])
			n++
		}
		if n == 0 {
			return []byte{c}, p + 1
		}
		r := rune(val)
		if !utf8.ValidRune(r) {
			r = utf8.RuneError
		}
		return utf8.AppendRune(nil, r), p + 1 + n
	case 'c':
		// bash \cX: the control character for X (X with bit 6 cleared, uppercased).
		// `\c` at end of string is left literal.
		if p+1 >= len(line) {
			return []byte{c}, p + 1
		}
		x := line[p+1]
		if x >= 'a' && x <= 'z' {
			x -= 'a' - 'A'
		}
		return []byte{x & 0x1f}, p + 2
	default:
		return []byte{'\\', c}, p + 1
	}
}

// isHexDigit reports whether b is an ASCII hexadecimal digit.
func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// hexValue returns the value of an ASCII hexadecimal digit; the caller guarantees
// isHexDigit(b).
func hexValue(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	default:
		return int(b-'A') + 10
	}
}

// isAllDigits reports whether b is a non-empty run of ASCII digits — the shape
// of a file-descriptor prefix on a redirection (`2>`, `1>&2`).
func isAllDigits(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// skipRedirectTarget consumes optional whitespace and a single redirection
// target word starting at pos, returning the position just past it. The target
// of a redirection is a filename or fd, never a command in command position, so
// the guard drops it. Quotes and backslashes inside the target are honoured so
// an embedded space or operator does not end the target early.
func skipRedirectTarget(line string, pos int) int {
	for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
		pos++
	}
	for pos < len(line) {
		c := line[pos]
		switch {
		case c == '\\':
			if pos+1 >= len(line) {
				return pos + 1
			}
			pos += 2
		case c == '\'':
			j := pos + 1
			for j < len(line) && line[j] != '\'' {
				j++
			}
			if j >= len(line) {
				return j
			}
			pos = j + 1
		case c == '"':
			j := pos + 1
			for j < len(line) {
				if line[j] == '\\' && j+1 < len(line) {
					j += 2
					continue
				}
				if line[j] == '"' {
					break
				}
				j++
			}
			if j >= len(line) {
				return j
			}
			pos = j + 1
		case c == ' ' || c == '\t' || c == '\n' || c == '&' || c == '|' ||
			c == ';' || c == '(' || c == ')' || c == '<' || c == '>':
			return pos
		default:
			pos++
		}
	}
	return pos
}

// heredoc is one pending here-document: the delimiter word that ends its body,
// and whether the `<<-` form allows the delimiter line to be tab-indented.
type heredoc struct {
	delim     string
	stripTabs bool
	// quoted records that the delimiter word carried quotes, which makes it a
	// delimiter beyond doubt however exotic it looks (`<<'---'`).
	quoted bool
}

// isDelimStart reports whether a word can begin an unquoted here-document
// delimiter. bash accepts a WORD there, not an identifier: `cat <<20` reads a
// document ended by a line saying 20, `cat <<$D` reads one ended by whatever
// $D expands to, and `let mask=1<<20` is a here-document too — the shell
// tokenises the redirection before the `let` builtin ever sees an arithmetic
// expression. Restricting the set to letters and `_` therefore sent those to
// the shift branch, where the DOCUMENT was tokenised as commands: one
// apostrophe in the body became ErrUnparsableCommand, which the pre-tool-use
// hook maps to fail-OPEN, and any hazard below it ran unguarded.
//
// The set is the conservative superset of what a delimiter word ordinarily
// starts with — a letter, a digit, `_`, or a `$`-led word. bash accepts more
// (`<<!`, `<</tmp/x`); those stay out, because a `<<` reached in an arithmetic
// context this tokenizer does not model would otherwise swallow the rest of
// the input on the strength of an operator. What actually separates a shift
// from a document is the ENCLOSING context (inArithmetic), which is checked
// before this; the unmodelled contexts that remain — bash's deprecated
// `$[ … ]` — are a named residual, not a discriminator this word test can fix.
func isDelimStart(delim string) bool {
	if delim == "" {
		return false
	}
	c := delim[0]
	return c == '_' || c == '$' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// readHeredocDelim reads the delimiter word after a `<<` at pos, honouring the
// `<<-` variant and any quoting of the word itself (`<<'EOF'`, `<<"EOF"`), and
// returns the position just past it.
func readHeredocDelim(line string, pos int) (heredoc, int, error) {
	hd := heredoc{}
	if pos < len(line) && line[pos] == '-' {
		hd.stripTabs = true
		pos++
	}
	for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
		pos++
	}
	var w []byte
	for pos < len(line) {
		c := line[pos]
		switch c {
		case '\'', '"':
			j := pos + 1
			for j < len(line) && line[j] != c {
				j++
			}
			if j >= len(line) {
				return heredoc{}, 0, fmt.Errorf("%w: unterminated quote in heredoc delimiter", ErrUnparsableCommand)
			}
			w = append(w, line[pos+1:j]...)
			hd.quoted = true
			pos = j + 1
			continue
		case '\\':
			if pos+1 >= len(line) {
				// The twin of tokenize's own trailing-backslash site: dropped,
				// as bash 3.2 reads it, never an error the hook fails open on.
				hd.delim = string(w)
				return hd, pos + 1, nil
			}
			w = append(w, line[pos+1])
			pos += 2
			continue
		case ' ', '\t', '\n', ';', '&', '|', '(', ')', '<', '>':
			hd.delim = string(w)
			return hd, pos, nil
		}
		w = append(w, c)
		pos++
	}
	hd.delim = string(w)
	return hd, pos, nil
}

// markHeredocUnterminated flags the command that opened an unterminated
// here-document: the last segment emitted, or — for a line that holds the
// redirection and nothing else — an empty segment carrying only the flag, so
// the verdict is raised even when there is no command to hang it on.
func markHeredocUnterminated(segs *[]segment, chain int) {
	if n := len(*segs); n > 0 {
		(*segs)[n-1].heredocUnterminated = true
		return
	}
	*segs = append(*segs, segment{chain: chain, heredocUnterminated: true})
}

// substitutionBlockSignal is the fail-closed verdict for a command substitution
// the tokenizer did not read. It is a BLOCK rather than a warn because the
// substitution's command runs before the command around it, whatever it is,
// and the guard has not seen it.
func substitutionBlockSignal() payloadSignal {
	return payloadSignal{
		id:      substitutionEntryID,
		verdict: VerdictBlock,
		family:  familySubstitution,
		reason: "This command nests command substitutions inside double quotes deeper than the guard reads (" +
			strconv.Itoa(maxQuotedSubstitutionDepth) + " levels), or carries one whose text it cannot split — " +
			"a case command inside one, or more nesting than its scan budget covers — " +
			"so a command that runs first is one it has not checked.",
		successor: "Run the inner command on its own and keep its output in a variable, " +
			"so each command the shell runs is one the guard checks.",
	}
}

// heredocBlockSignal is the fail-closed verdict for a here-document whose
// delimiter line never came. It is a BLOCK rather than a warn because the
// tokenizer has just read everything after the redirection as data: if the
// `<<` was a misclassified shift or a typo, the "document" is commands the guard
// did not check, and it has no way to tell the two apart.
func heredocBlockSignal() payloadSignal {
	return payloadSignal{
		id:      heredocEntryID,
		verdict: VerdictBlock,
		family:  familyHeredoc,
		reason: "This command opens a here-document whose delimiter line never comes, so the shell would take the rest of the input as the document — " +
			"and the guard cannot tell whether that rest is a document or commands it did not check.",
		successor: "Terminate the here-document with its delimiter on a line of its own, or quote a `<<` that is not one, " +
			"so the guard checks the command that actually runs.",
	}
}

// skipHeredocBodies consumes the body of every pending here-document, starting
// at pos (the first byte after the newline that ended the command line), and
// returns the position just past the last body. The second return is false if
// any body never finds its terminating delimiter line before the input ends —
// which is either a genuinely truncated heredoc, or a `<<` that isDelimStart
// mistook for one (an identifier-operand arithmetic shift, `$((1<<shift))`,
// reads as a delimiter word). Either way, silently consuming the remainder of
// the line as unchecked "body" text would swallow real commands with no
// signal; the caller flags the opening command so Check fails CLOSED on it,
// never an error, which the hook would turn into a fail-open.
func skipHeredocBodies(line string, pos int, pending []heredoc) (int, bool) {
	for _, hd := range pending {
		found := false
		for pos < len(line) {
			end := pos
			for end < len(line) && line[end] != '\n' {
				end++
			}
			text := line[pos:end]
			if hd.stripTabs {
				text = strings.TrimLeft(text, "\t")
			}
			if end < len(line) {
				pos = end + 1
			} else {
				pos = end
			}
			if text == hd.delim {
				found = true
				break
			}
		}
		if !found {
			return pos, false
		}
	}
	return pos, true
}
