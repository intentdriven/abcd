package guard

import "strings"

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
// The rule is that an unknown word fails closed in every role it could play:
//
//   - A word whose own text begins with a dash (`--$(x)`, `-r$(x)`,
//     `--"$(x)"`) is a flag of unknown name, and stands for every flag
//     alternative its known text can still become (unknownFlagCouldBe). It is
//     never the `--` terminator, which is spelled with no substitution.
//   - As the value of a value flag it fills the value slot, and the value flag
//     never consumes the word after it: operandIndexes steps it over like any
//     other value, because it is a word in the token list.
//   - As an operand it is one operand of unknown value: it counts toward
//     min_operands, it keeps the operands after it in their positions, and a
//     subcommand compare at its position matches whatever the entry names.
//
// A word that is nothing but a substitution, unquoted, may also expand to no
// word at all, and the operand position readers take that reading too
// (vanishable): `git $(true) push --force` is a push.
//
// What stays deliberately outside the rule is a word that is WHOLLY a
// substitution standing where a flag could be: it is read as an operand, not as
// every flag, because that is how an everyday command spells its commit message
// and its branch (`git commit -m "$(cat msg)"`, `git push origin
// "$(git branch --show-current)"`), and reading it as every flag would refuse
// both. The same reason keeps an operand's `+` refspec prefix read from its
// known text only. Both residuals are recorded in .abcd/work/DECISIONS.md.

// unknownMark stands, inside a token, for the output of a substitution the
// guard did not run. It is the NUL byte because no argv word can hold one — an
// argument is a C string — so a real word can never be mistaken for it; Check
// drops any NUL the command line itself carries before it reads a word.
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
