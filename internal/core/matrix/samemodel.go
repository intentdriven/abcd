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

import (
	"regexp"
	"slices"
	"strings"
)

// maxComparedModel bounds a name the comparison reads; a longer one is
// unknown, and so counts as the writer's.
const maxComparedModel = 200

// carrierWords are the words a name may carry that name a vendor, a carrier
// or a region rather than the model: a provider's prefix (`anthropic/`,
// `openai/`), the vendor in an Assisted-by trailer (`Claude:`), a cloud's
// model id (`bedrock/us.anthropic.…`, `google-vertex:…`), and a moving alias
// (`-latest`). Dropping one only makes two names more likely to count as the
// same model, which is the safe side.
var carrierWords = []string{
	"anthropic", "claude", "openai", "google", "vertex", "bedrock", "azure", "aws", "openrouter",
	"us", "eu", "apac", "global", "latest",
}

var (
	dateStampRe    = regexp.MustCompile(`^[0-9]{8}$`)
	versionStampRe = regexp.MustCompile(`^v[0-9]+$`)
)

// OtherModel reports whether reviewer is shown to name a model other than
// writer: both names are known (KnownModel) and neither one's tokens are
// contained in the other's (modelTokens). Every other pair, an unknown name
// on either side included, counts as the same model. It is symmetric, and
// OtherModel(s, s) is false for every s.
func OtherModel(writer, reviewer string) bool {
	w, ok := modelTokens(writer)
	if !ok {
		return false
	}
	r, ok := modelTokens(reviewer)
	if !ok {
		return false
	}
	return !contains(w, r) && !contains(r, w)
}

// KnownModel reports whether s is a name OtherModel can compare: ASCII, with
// no control character, no longer than maxComparedModel, and leaving at least
// one token once the carrier words and stamps are dropped.
func KnownModel(s string) bool {
	_, ok := modelTokens(s)
	return ok
}

// modelTokens reduces a name to the set of tokens OtherModel compares, and
// false for an unknown name. The name is lower-cased; a bracketed suffix
// (`[1m]`) is dropped, and everything up to the last `/`; what remains is
// split on every character that is not an ASCII letter or digit; a trailing
// `v<digits>` token and what follows it (`-v1:0`) is dropped, then any
// trailing eight-digit date stamp, then every carrier word.
func modelTokens(s string) ([]string, bool) {
	if s == "" || len(s) > maxComparedModel {
		return nil, false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c >= 0x7f {
			return nil, false
		}
	}
	s = strings.ToLower(s)
	if i := strings.IndexByte(s, '['); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	for i := len(parts) - 1; i > 0; i-- {
		if versionStampRe.MatchString(parts[i]) {
			parts = parts[:i]
			break
		}
	}
	for len(parts) > 0 && dateStampRe.MatchString(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}
	var out []string
	for _, p := range parts {
		if !slices.Contains(carrierWords, p) && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// contains reports whether every token of small is in big.
func contains(big, small []string) bool {
	for _, t := range small {
		if !slices.Contains(big, t) {
			return false
		}
	}
	return true
}
