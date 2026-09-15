package guard

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// segment is one command in command position: the tokens of a single simple
// command, plus the index of the logical line (chain) it belongs to. Segments in
// the same chain are separated by `&&`, `;`, `||`, `|`, `&`, or a subshell
// parenthesis; a newline starts a new chain. The chain is what lets an entry
// require the cd-chain structure (`cd scratch && rm -rf *`) without matching an
// unrelated `rm` on the next line.
type segment struct {
	tokens []string
	chain  int
	// braceGroup records that this command carried an UNQUOTED brace group —
	// text bash rewrites into several words before the child ever sees it. The
	// tokenizer does not expand it, so it cannot say what the argv will be; the
	// flag is how it says so, and Check turns it into a fail-closed block.
	braceGroup bool
	// heredocUnterminated records that this command opened a here-document
	// whose delimiter line never came, so the tokenizer read the rest of the
	// input as body without knowing whether it WAS body. bash runs such a line
	// (it recovers silently), so it cannot be an error — the hook maps an error
	// to fail-open — and it cannot be an allow either, because a `<<` the
	// classifier misread has swallowed every later command. Check turns the
	// flag into a fail-closed block, the braceGroup precedent.
	heredocUnterminated bool
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
	var (
		segs    []segment
		toks    []string
		cur     []byte
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
		// braceBudget is the look-ahead braceExpansionAt may spend across this
		// whole call. See braceScanBudget: without a shared cap the per-`{`
		// forward scan is quadratic in the length of one word.
		braceBudget = braceScanBudget
		// lastList records that the previous operator was a list operator
		// (`&&`, `||`, `|`), whose newline is a line continuation rather than a
		// new command — `cd scratch &&\nrm -rf *` is one chain, not two.
		lastList bool
		// parens is the stack of grouping constructs still open at this point,
		// innermost last. It exists for one question — whether a `<<` reached
		// here is an arithmetic shift or a here-document redirection — and that
		// question is answered by what ENCLOSES the operator, never by the bytes
		// after it. See inArithmetic and the `<<` branch.
		parens []parenFrame
	)
	// inArithmetic reports whether the innermost construct that can change how a
	// `<<` reads is an arithmetic one. A plain `(` is skipped rather than
	// answered on: inside `$(( … ))` it is sub-expression grouping, and at the
	// top level it is a subshell, whose own enclosing context is what decides —
	// either way the frame below it has the answer. A `$(` or a backtick stops
	// the walk, because it starts a FRESH command string, where a here-document
	// is possible again.
	inArithmetic := func() bool {
		for n := len(parens) - 1; n >= 0; n-- {
			switch parens[n].kind {
			case parenArithmetic:
				return true
			case parenCommandSub, parenBacktick:
				return false
			}
		}
		return false
	}
	flushToken := func() {
		if hasCur {
			toks = append(toks, string(cur))
			globs = append(globs, curGlob)
			cur = nil
			hasCur = false
			curGlob = false
		}
	}
	flushSegment := func() {
		flushToken()
		if len(toks) > 0 {
			segs = append(segs, segment{tokens: toks, chain: chain, braceGroup: braceGroup, globbed: globsOrNil(globs)})
			toks = nil
			globs = nil
			braceGroup = false
		}
	}

	for i := 0; i < len(line); {
		c := line[i]
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
			cur = append(cur, line[i+1])
			hasCur = true
			lastList = false
			i += 2
		case c == '\'':
			j := i + 1
			for j < len(line) && line[j] != '\'' {
				j++
			}
			if j >= len(line) {
				return nil, fmt.Errorf("%w: unterminated single quote", ErrUnparsableCommand)
			}
			cur = append(cur, line[i+1:j]...)
			hasCur = true
			lastList = false
			i = j + 1
		case c == '"':
			j := i + 1
			closed := false
			for j < len(line) {
				if line[j] == '\\' && j+1 < len(line) {
					switch line[j+1] {
					case '"', '\\', '$', '`':
						cur = append(cur, line[j+1])
					case '\n':
						// Line continuation inside double quotes: both dropped.
					default:
						// Backslash is literal before any other character.
						cur = append(cur, '\\', line[j+1])
					}
					j += 2
					continue
				}
				if line[j] == '"' {
					closed = true
					break
				}
				cur = append(cur, line[j])
				j++
			}
			if !closed {
				return nil, fmt.Errorf("%w: unterminated double quote", ErrUnparsableCommand)
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
				chain++
			}
		case c == '#' && !hasCur:
			// A comment starts only at a word boundary (POSIX): `url/#frag` is
			// part of the token, a bare `#` runs to the end of the line.
			for i < len(line) && line[i] != '\n' {
				i++
			}
		case c == '<' && strings.HasPrefix(line[i:], "<<<"):
			// A herestring, not a heredoc: its payload is an ordinary argument
			// token, so the operator is kept as plain token text.
			cur = append(cur, '<', '<', '<')
			hasCur = true
			lastList = false
			i += 3
		case c == '<' && strings.HasPrefix(line[i:], "<<"):
			// A heredoc redirection (`<<`, `<<-`) — but only when nothing
			// arithmetic encloses the operator and a delimiter word follows.
			// `$((1<<20))` is an arithmetic shift, and taking it for a heredoc
			// would swallow every later line as body text and silently unguard
			// them.
			//
			// WHAT ENCLOSES the `<<` is what tells the two apart. Inside an
			// arithmetic context — a `$(( … ))` expansion or the bare `(( … ))`
			// command — bash has no redirection at all, so a `<<` there is a
			// shift, full stop; outside one, a delimiter-shaped word opens a
			// document. Deciding instead on the bytes AFTER the delimiter word
			// ("does a paren pair close right here?") reads only the flattest
			// shift: `$(( (1 << n) + 1 ))` closes its sub-expression with a
			// SINGLE `)`, so `n` was taken for a delimiter — and a later line
			// equal to `n` then swallowed every command between the two with no
			// signal at all, while a bit mask with no such line blocked as an
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
				cur = append(cur, '<', '<')
				hasCur = true
				lastList = false
				i += 2
				continue
			}
			hd, next, err := readHeredocDelim(line, i+2)
			if err != nil {
				return nil, err
			}
			// A word that cannot start an unquoted delimiter — `20` in a
			// `$((1<<20))` reached outside any paren — is not one.
			if !hd.quoted && !isDelimStart(hd.delim) {
				cur = append(cur, '<', '<')
				hasCur = true
				lastList = false
				i += 2
				continue
			}
			flushToken()
			pending = append(pending, hd)
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
				cur = append(cur, c)
				hasCur = true
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
				cur = nil
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
				return nil, err
			}
			cur = append(cur, decoded...)
			hasCur = true
			lastList = false
			i = next
		case c == '$' && i+1 < len(line) && line[i+1] == '"':
			// bash locale quoting: $"..." is a plain double-quoted word with the
			// `$` stripped (the translation is identity for a tokenizer). Skip the
			// `$` so the double-quote branch reads the string; the same silent
			// allow as $'...' otherwise.
			i++
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
			flushSegment()
			switch c {
			case '(':
				// `((` and `$((` — two parens with NOTHING between them — open
				// an arithmetic context, which is how bash lexes them too;
				// `( (cmd) )` and `$( (cmd) )`, which have a separator, do not.
				// The inner paren converts the frame the outer one pushed, so
				// both halves close it and `$(((a))` reads as arithmetic plus
				// one ordinary group.
				kind := parenGroup
				if i > 0 && line[i-1] == '$' {
					kind = parenCommandSub
				}
				if n := len(parens); n > 0 && parens[n-1].pos == i-1 && parens[n-1].kind != parenArithmetic {
					parens[n-1].kind = parenArithmetic
					kind = parenArithmetic
				}
				parens = append(parens, parenFrame{kind: kind, pos: i})
			case ')':
				if n := len(parens); n > 0 {
					parens = parens[:n-1]
				}
			case '`':
				// A backtick is its own closer, so it toggles: an open one on
				// the stack is popped, anything else pushes a fresh frame.
				if n := len(parens); n > 0 && parens[n-1].kind == parenBacktick {
					parens = parens[:n-1]
				} else {
					parens = append(parens, parenFrame{kind: parenBacktick, pos: i})
				}
			}
			if (c == '&' || c == '|') && i+1 < len(line) && line[i+1] == c {
				lastList = true
				i += 2
				continue
			}
			// A single pipe continues the list across a newline; `;`, `&`, the
			// grouping parens, and a backtick boundary do not.
			lastList = c == '|'
			i++
		case c == '{' && braceExpansionAt(line, i, &braceBudget):
			// An unquoted brace group is EXPANSION, not text: bash rewrites
			// `git push {--force,} origin main` into byte-identical `--force`
			// argv, while this tokenizer read the literal token `{--force,}`,
			// which no blocker matches — a silent allow of a Tier-1 hazard, the
			// same mutate-the-flag-token shape the redirection branch closes.
			// Expanding it properly (the Cartesian product of the alternatives,
			// nested groups, `{a..z}` ranges) is a bounded expander this round
			// does not have, so the group is REFUSED instead: a token whose argv
			// the guard cannot compute is a token it cannot check, and refusing
			// what cannot be read is what fail-closed means here.
			//
			// The refusal rides on the segment rather than returning
			// ErrUnparsableCommand, which is the obvious route and the wrong
			// one: the `guard check` verb maps a tokenize error to a blocking
			// exit, but the pre-tool-use hook maps it to fail-OPEN, so the
			// bypass would have survived on the surface that matters. Check
			// folds the flag into a real VerdictBlock, which blocks on both.
			// The bytes stay in the word so nothing else about the line's
			// tokenization changes.
			braceGroup = true
			cur = append(cur, c)
			hasCur = true
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
			cur = append(cur, c)
			hasCur = true
			lastList = false
			i++
		}
	}
	flushSegment()
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
	// parenArithmetic is one half of a `((` or `$((` pair.
	parenArithmetic
)

// parenFrame is one unclosed grouping construct: its kind, and the offset of the
// byte that opened it — which is what lets the next `(` see that it is adjacent
// and convert the pair into an arithmetic context.
type parenFrame struct {
	kind parenKind
	pos  int
}

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
// unquoted brace group. It is a BLOCK rather than a warn because the group can
// carry any flag at all — the reported shape, `{--force,}`, expands to argv a
// Tier-1 blocker names — and the guard has no way to tell a harmless expansion
// from that one without expanding it.
func braceExpansionBlockSignal() payloadSignal {
	return payloadSignal{
		id:      braceEntryID,
		verdict: VerdictBlock,
		family:  familyBrace,
		reason: "This command carries an unquoted brace group, which the shell expands into different words before the command runs, " +
			"so the arguments the guard can read are not the arguments that would be passed.",
		successor: "Spell the words out (`git push --force origin main`), or quote the braces if they are meant literally, " +
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
func braceExpansionAt(line string, i int, budget *int) (group bool) {
	// `${…}` is parameter expansion — unless the `$` is ITSELF escaped, which
	// makes it a literal dollar and leaves the brace group behind it live:
	// bash expands `\${a,b}` to `$a $b`. So the exemption needs the raw
	// preceding byte to be a `$` that is not escaped.
	if i > 0 && line[i-1] == '$' && !escapedAt(line, i-1) {
		return false
	}
	// The look-ahead is capped by the shared budget rather than by the line, so
	// the total scanning across one tokenize call is linear however many `{`
	// bytes the line holds.
	end, truncated := len(line), false
	if *budget < end-i {
		end, truncated = i+*budget, true
	}
	j := i
	defer func() { *budget -= j - i }()

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
				return true
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
			return false
		default:
			j++
		}
	}
	// Running out of line means no closing brace, which bash leaves unexpanded.
	// Running out of BUDGET means the scan no longer knows, and a guard that
	// cannot tell a group from a literal says group.
	return truncated
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
func readAnsiCQuote(line string, start int) ([]byte, int, error) {
	var out []byte
	for i := start; i < len(line); {
		switch {
		case line[i] == '\'':
			return out, i + 1, nil
		case line[i] == '\\' && i+1 < len(line):
			decoded, next := decodeAnsiCEscape(line, i+1)
			out = append(out, decoded...)
			i = next
		default:
			out = append(out, line[i])
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
