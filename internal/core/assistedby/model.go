package assistedby

import "regexp"

// modelValuePattern is the value half of the gate's TRAILER_RE, the model form
// `<Vendor>:<model-version>` with an optional bracketed suffix. The gate decides
// it and carries the reasoning for every character; this is its one Go copy,
// tied to the gate by TestModelValueGrammarMatchesTheGate.
const modelValuePattern = `[A-Za-z][A-Za-z0-9._-]*:[A-Za-z0-9._-]+(\[[A-Za-z0-9._-]+\])?`

var (
	modelValueRe = regexp.MustCompile(`^` + modelValuePattern + `$`)
	modelHeadRe  = regexp.MustCompile(`^` + modelValuePattern)
)

// IsModelValue reports whether a trailer value is, whole, the model form the
// gate accepts.
func IsModelValue(value string) bool { return modelValueRe.MatchString(value) }

// ModelHead is the model form a trailer value starts with, or "" when it starts
// with none: a conforming value whole, or the conforming head of a value that
// trails free text after it.
func ModelHead(value string) string { return modelHeadRe.FindString(value) }
