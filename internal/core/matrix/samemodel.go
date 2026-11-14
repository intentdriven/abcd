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
// version: only the short numbers written straight after the family. Two
// names are other models only when each names exactly one family and the
// families differ, or the family is the same and both carry a version, and
// the versions differ with neither a prefix of the other. Every other pair is
// the same model: a qualifier word (`thinking`, `fast`, `mini`), a carrier,
// vendor or region, a snapshot or date stamp, a revision tag, an alias with no
// family word (`default`, `best`, `opusplan`), and a name holding two
// families. A family not on the list reads as no family, so a name of it is
// unknown and counts as the writer's own.

import (
	"slices"
	"strings"
)

// maxComparedModel bounds a name the comparison reads; a longer one is
// unknown, and so counts as the writer's.
const maxComparedModel = 200

// modelFamilies is the closed list of model families a name is read for:
// Anthropic's (opus, sonnet, haiku, fable, mythos), OpenAI's gpt and the
// o-series (written `o` and a digit, o1, o3, o4, read as the family "o"),
// Google's gemini and gemma, Meta's llama, Mistral's mistral, mixtral,
// codestral, devstral, magistral, ministral and pixtral, xAI's grok,
// deepseek, qwen, Moonshot's kimi, Zhipu's glm and Microsoft's phi. A family
// glued to another word is no family (`codellama`, `codegemma`,
// `chatgpt-4o-latest`), so such a name is unknown. A vendor word (claude, anthropic, openai) is no family:
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
	"mistral", "mixtral", "codestral", "devstral", "magistral", "ministral", "pixtral",
	"grok", "deepseek", "qwen", "kimi", "glm", "phi",
}

// oSeries is the family an `o` written straight before a digit names.
const oSeries = "o"

// maxVersionDigits bounds a number read as part of a version; a longer one is
// a date stamp or a snapshot tag, and ends the version.
const maxVersionDigits = 3

// OtherModel reports whether reviewer is shown to name a model other than
// writer: both names are known (KnownModel), and either their families
// differ, or the family is the same and both carry a version, the versions
// differ, and neither one's digits are a prefix of the other's. Every other
// pair, an unknown name on either side included, counts as the same model, so
// `opus-4` and `opus-4-6` are not told apart, nor `opus-5` and `opus-55`. It
// is symmetric, and OtherModel(s, s) is false for every s.
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
// one model family of modelFamilies in its last segment after a slash or, where
// that segment names none or two, in the whole name (readModel). A space is a separator here, so `claude
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

// nameToken is one token of a name: a run of ASCII letters or one of digits.
// glued says the next token is written straight after it, with no separator
// between (`opus5`, `70b`).
type nameToken struct {
	text  string
	glued bool
}

// readModel reads s for its family and version, and false for an unknown
// name. The name is lower-cased and a bracketed suffix (`[1m]`) is dropped.
// The family is read from the last segment after a slash where that segment
// names exactly one, else from the whole name where it names exactly one;
// any other name is unknown. So `mistral/codestral-latest` is codestral, and
// `azure/eastus2/gpt-5` is gpt. The version is read where the family was
// (versionAfter): the numbers written straight after the family's token.
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
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		if out, ok := readSegment(s[i+1:]); ok {
			return out, true
		}
	}
	return readSegment(s)
}

// readSegment reads seg for its one family and that family's version, and
// false where seg names no family or two different ones. seg is split into
// tokens on every character that is not an ASCII letter or digit and at each
// boundary between letters and digits (`opus5` gives `opus`, `5`). A family
// is a token on modelFamilies, or an `o` written straight before a digit
// (`o3`). Where the family's token stands more than once, no version is read.
func readSegment(seg string) (readName, bool) {
	toks := tokens(seg)
	var out readName
	at, seen := -1, 0
	for i, t := range toks {
		f := ""
		switch {
		case t.text == oSeries && t.glued && i+1 < len(toks) && isDigits(toks[i+1].text):
			f = oSeries
		case slices.Contains(modelFamilies, t.text):
			f = t.text
		default:
			continue
		}
		if out.family != "" && f != out.family {
			return readName{}, false
		}
		out.family, at = f, i
		seen++
	}
	if out.family == "" {
		return readName{}, false
	}
	if seen == 1 {
		out.version = versionAfter(toks[at+1:])
	}
	return out, true
}

// versionAfter is the version the tokens straight after a family's token
// carry: the digits, in order, of the run of numbers of at most three digits
// that opens them, so `gpt51` and `gpt-5-1` carry the same one. The run ends
// at the first token that is not such a number: a letter token (`v`, `p`, `b`,
// `latest`), a number of four or more digits (a date stamp or a snapshot tag),
// and a number written straight before a letter (a size or a tag: `1m`,
// `200k`, `70b`, `4o`). So a number written before the family is never the
// version (`us-east-1`, `eastus2`, `claude-3-5-sonnet`), nor one after a
// revision tag (`-v1:0`, `-v2`) or a vendor's `v3p1`, nor any part of a date
// after its year (`-2026-10-14`).
func versionAfter(toks []nameToken) string {
	v := ""
	for _, t := range toks {
		if !isDigits(t.text) || len(t.text) > maxVersionDigits || t.glued {
			break
		}
		v += t.text
	}
	return v
}

// tokens splits s on every character that is not an ASCII letter or digit and
// at each boundary between letters and digits.
func tokens(s string) []nameToken {
	var out []nameToken
	for _, chunk := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		parts := splitDigits(chunk)
		for i, p := range parts {
			out = append(out, nameToken{text: p, glued: i+1 < len(parts)})
		}
	}
	return out
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
