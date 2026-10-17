package guard

// approvegroup.go checks the text an approval hands back (runFrom): the
// command inside a group that a change into the working tree guards. The
// check reads that text with the guard's own tokenizer and approves it only
// when its top level is exactly `cd -- <dir>`, joined by && to one group
// that holds the whole command and ends the text. A command that would close
// the group early (a } at its own top level), or leave the text unparsable
// as one group (a ) or a closing keyword with nothing open, a backslash
// joining the group's } to its last line), is refused instead.
//
// The tokenizer splits commands without building the shell's grammar, so the
// check rebuilds the nesting from the words at the start of each command: the
// words the shell can read as opening or closing a group, an if, a loop or a
// case. It reads more of them than the shell does, never fewer, and asks for
// the outer group to close at the very end and nowhere before, so a word it
// misreads can refuse a command but cannot approve one. It counts a word only
// when the line leaves it wholly unquoted (segment.bareWord): a quoted { or
// keyword (`"{"`, `'{'`, `\{`, `{""`, `'then'`) is a command's name to the
// shell, and the words after it are its arguments, so counting one as an
// opener would pair it with a } that in fact closes the outer group.

import "slices"

// The words that close the constructs a command opens.
const (
	endGroup = "}"
	endIf    = "fi"
	endLoop  = "done"
	endCase  = "esac"
)

// openers maps a word that opens a construct at the start of a command to
// the word that closes it.
var openers = map[string]string{
	"{": endGroup, "if": endIf, "while": endLoop, "until": endLoop, "for": endLoop, "select": endLoop, "case": endCase,
}

// leadWords are the words after which the next word still stands where the
// shell may read a keyword: the reserved words, read generously, and the
// words that close a construct.
var leadWords = map[string]bool{
	"if": true, "then": true, "elif": true, "else": true, "do": true, "while": true, "until": true,
	"{": true, "!": true, "time": true, endGroup: true, endIf: true, endLoop: true, endCase: true,
}

// groupWord reports whether w is a word the walk below reads at the start of
// a command: an opener, a lead word, a close or `function`. The tokenizer
// records where one stands unquoted (segment.bareWord).
func groupWord(w string) bool {
	_, open := openers[w]
	return open || leadWords[w] || w == "function"
}

// allStruct reports whether every byte of a word reached the tokenizer
// unquoted and unescaped.
func allStruct(mask []byte) bool {
	for _, m := range mask {
		if m&wordStruct == 0 {
			return false
		}
	}
	return true
}

// runFromIntact is runFrom(dir, command) and whether that text is intact: the
// change of directory and one group holding the whole command. A dir that is
// not absolute is not wrapped, and the command is returned as it is.
//
// Besides the closes the shells share, zsh also reads a } after a command's
// words as closing a group (`{ echo a }`), so every unquoted } word must be
// one the walk counts as a close. Openers, lead words and closes are counted
// only where they stand unquoted, the test the tokenizer applies to that }.
func runFromIntact(dir, command string) (string, bool) {
	text := runFrom(dir, command)
	if text == command {
		return text, true
	}
	segs, err := tokenize(text)
	if err != nil || len(segs) < 3 {
		return text, false
	}
	for _, s := range segs {
		if s.heredocUnterminated || s.substitutionUnread || s.braceGroup {
			return text, false
		}
	}
	first, open, last := segs[0], segs[1], segs[len(segs)-1]
	if !slices.Equal(first.tokens, []string{"cd", "--", dir}) || len(first.redirects) > 0 ||
		!slices.Equal(open.tokens, []string{"{"}) || len(open.redirects) > 0 || !open.afterAnd || open.chain != first.chain ||
		!slices.Equal(last.tokens, []string{"}"}) || len(last.redirects) > 0 || last.end != len(text) {
		return text, false
	}
	// Each ) that closed nothing must be a case pattern's: it ends a command
	// read while a case is the innermost construct open.
	strays := map[int]bool{}
	if first.home != nil {
		for _, at := range first.home.strayParens {
			strays[at] = true
		}
	}
	var stack []string
	for k := 1; k < len(segs); k++ {
		s := segs[k]
		toks := s.tokens
		counted := map[int]bool{}
		closed := false
		for i := 0; i < len(toks); i++ {
			t := toks[i]
			if !s.bareWord[i] {
				break // a quoted word is a command's name, not a keyword
			}
			if end, ok := openers[t]; ok {
				if closed {
					return text, false // an opener straight after a close
				}
				stack = append(stack, end)
			} else if t == endGroup || t == endIf || t == endLoop || t == endCase {
				if len(stack) == 0 || stack[len(stack)-1] != t {
					return text, false
				}
				stack = stack[:len(stack)-1]
				if len(stack) == 0 && k != len(segs)-1 {
					return text, false // the outer group closed early
				}
				counted[i], closed = true, true
			}
			// `function NAME {` opens its body after the name.
			if i == 0 && t == "function" && len(toks) > 2 && toks[2] == "{" && s.bareWord[2] {
				stack = append(stack, endGroup)
				break
			}
			if !leadWords[t] {
				break
			}
		}
		for i := range s.bareWord {
			if toks[i] == endGroup && !counted[i] {
				return text, false
			}
		}
		if strays[s.end] {
			if len(stack) == 0 || stack[len(stack)-1] != endCase {
				return text, false
			}
			delete(strays, s.end)
		}
	}
	return text, len(stack) == 0 && len(strays) == 0
}
