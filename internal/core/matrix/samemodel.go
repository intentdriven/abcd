package matrix

// samemodel.go answers whether a reviewer's model may count as a model other
// than the one that wrote a change (itd-2610301750162657, spc-2610301925429250
// step 4). The names are written by agents and configuration, so both are
// untrusted, and two names count as other models only when that is shown:
// a change recorded as reviewed by another model while the writer's own
// reviewed it is the costly failure, and a reviewer refused that was in truth
// another model only costs a switch of class or a hand-back. It is the
// opposite lean to sameModel's (export.go), which must never read a
// substitution as the asked model, and it changes none of sameModel's answers.
//
// A name is read for one model family, from the closed list below, and its
// short version numbers. Two names are other models only when each names
// exactly one family and the families differ, or the family is the same and
// both carry a version, and the versions differ with neither a prefix of the
// other. Every other pair is the same model: a qualifier word (`thinking`,
// `fast`, `mini`), a carrier or vendor, a snapshot or date stamp, an alias
// with no family word (`default`, `best`, `opusplan`), and a name holding two
// families. A family not on the list reads as no family, so a name of it is
// unknown and counts as the writer's own.

import (
	"regexp"
	"slices"
	"strings"
)

// maxComparedModel bounds a name the comparison reads; a longer one is
// unknown, and so counts as the writer's.
const maxComparedModel = 200

// modelFamilies is the closed list of model families a name is read for:
// Anthropic's (opus, sonnet, haiku, fable, mythos), OpenAI's gpt and the
// o-series (written `o` and a digit, o1, o3, o4, read as the family "o"),
// Google's gemini and gemma, Meta's llama, Mistral's mistral, mixtral and
// codestral, xAI's grok, deepseek, qwen, Moonshot's kimi, Zhipu's glm and
// Microsoft's phi. A vendor word (claude, anthropic, openai) is no family:
// it names who serves the model, not which model it is. codex is left off:
// OpenAI writes it beside gpt (gpt-5-codex), so a listed codex would make
// those names hold two families, and a name of codex alone is unknown. A
// family not listed here is unknown to the comparison, and a name of it
// counts as the writer's own model.
var modelFamilies = []string{
	"opus", "sonnet", "haiku", "fable", "mythos",
	"gpt",
	"gemini", "gemma",
	"llama",
	"mistral", "mixtral", "codestral",
	"grok", "deepseek", "qwen", "kimi", "glm", "phi",
}

// oSeries is the family an `o` written straight before a digit names.
const oSeries = "o"

// cloudRevisionRe is a cloud's revision tag at the end of a name, as a
// Bedrock id writes it (`-v1:0`): it numbers the listing, not the model.
var cloudRevisionRe = regexp.MustCompile(`[-.]v[0-9]+:[0-9]+$`)

// maxVersionDigits bounds a number read as a version; a longer one is a date
// stamp or a snapshot tag, and is dropped.
const maxVersionDigits = 3

// OtherModel reports whether reviewer is shown to name a model other than
// writer: both names are known (KnownModel), and either their families
// differ, or the family is the same and both carry a version, the versions
// differ, and neither one's digits are a prefix of the other's. Every other pair, an unknown
// name on either side included, counts as the same model. It is symmetric,
// and OtherModel(s, s) is false for every s.
func OtherModel(writer, reviewer string) bool {
	w, ok := readModel(writer)
	if !ok {
		return false
	}
	r, ok := readModel(reviewer)
	if !ok {
		return false
	}
	if w.family != r.family {
		return true
	}
	if w.version == "" || r.version == "" {
		return false
	}
	n := min(len(w.version), len(r.version))
	return w.version[:n] != r.version[:n]
}

// KnownModel reports whether s is a name OtherModel can compare: ASCII, with
// no control character, no longer than maxComparedModel, and naming exactly
// one model family of modelFamilies. A space is a separator here, so `claude
// opus` is known; the loop keeps no name holding a space as a writer's or a
// reviewer's (its modelWritten), so for it such a name stays unknown.
func KnownModel(s string) bool {
	_, ok := readModel(s)
	return ok
}

// readName is a name as OtherModel compares it: its one family, and its
// version's digits in order.
type readName struct {
	family  string
	version string
}

// readModel reads s for its family and version, and false for an unknown
// name. The name is lower-cased; a bracketed suffix (`[1m]`) is dropped, and
// a cloud's trailing revision tag (`-v1:0`); the whole of what remains is
// split on every character that is not an ASCII letter or digit and at each
// boundary between letters and digits (`opus5` gives `opus`, `5`). A number
// of more than three digits (a date stamp, a snapshot tag), one of two or
// three digits opening with a zero (`001`), and one written straight before a
// letter (a size or a tag: `1m`, `200k`, `70b`, `4o`) are dropped; the digits
// of the rest, in order, are the version, so `gpt51` and `gpt-5-1` carry the
// same one. The family is the one token on modelFamilies, or an `o` written
// straight before a digit; a name with none, or with two different ones, is
// unknown.
func readModel(s string) (readName, bool) {
	if s == "" || len(s) > maxComparedModel {
		return readName{}, false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c >= 0x7f {
			return readName{}, false
		}
	}
	s = strings.ToLower(s)
	if i := strings.IndexByte(s, '['); i >= 0 {
		s = s[:i]
	}
	s = cloudRevisionRe.ReplaceAllString(s, "")
	var out readName
	families := 0
	family := func(f string) {
		if f != out.family {
			families++
			out.family = f
		}
	}
	for _, chunk := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		toks := splitDigits(chunk)
		for i, t := range toks {
			switch {
			case isDigits(t):
				if len(t) <= maxVersionDigits && (len(t) == 1 || t[0] != '0') && i+1 == len(toks) {
					out.version += t
				}
			case t == oSeries && i+1 < len(toks) && isDigits(toks[i+1]):
				family(oSeries)
			case slices.Contains(modelFamilies, t):
				family(t)
			}
		}
	}
	if families != 1 {
		return readName{}, false
	}
	return out, true
}

// splitDigits splits an alphanumeric chunk at each boundary between letters
// and digits.
func splitDigits(chunk string) []string {
	var out []string
	start := 0
	for i := 1; i < len(chunk); i++ {
		if isDigit(chunk[i]) != isDigit(chunk[i-1]) {
			out = append(out, chunk[start:i])
			start = i
		}
	}
	return append(out, chunk[start:])
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isDigits(s string) bool { return s != "" && isDigit(s[0]) }
