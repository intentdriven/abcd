package issuerecord

import "github.com/intentdriven/abcd/internal/termsafe"

// Parse splits a record into its frontmatter map and body. Its error is
// masked (maskedError).
func Parse(text string) (map[string]any, string, error) {
	fm, body, err := parse(text)
	return fm, body, masked(err)
}

// ValidateStrict validates a frontmatter map against the issue schema. It
// special-cases schema_version first (mirrors _validate_strict) and rejects
// unknown keys (additionalProperties:false). Its error is masked
// (maskedError).
func ValidateStrict(fm map[string]any) error { return masked(validateStrict(fm)) }

// ValidateInvariants enforces folder<->field invariants and filename<->id
// match, mirroring _issue_lib._validate_invariants. Assumes ValidateStrict
// ran. Its error is masked (maskedError).
func ValidateInvariants(fm map[string]any, status, path string) error {
	return masked(validateInvariants(fm, status, path))
}

// maskedError is a validator's error with its text passed through
// termsafe.Sanitize (iss-2610121542258412). The validator quotes stored
// values back (schema_version with %v, the rest with %q), and its errors reach
// --json payloads and refusal reasons on several verbs; %q escapes a control
// or bidi rune but leaves a mark Go counts as printable, such as the
// combining grapheme joiner, and %v escapes nothing. Masking the composed
// message holds every one of them, a quote's escapes being plain ASCII
// already. Unwrap keeps the sentinel, so errors.Is still names the class.
type maskedError struct{ err error }

func (e maskedError) Error() string { return termsafe.Sanitize(e.err.Error()) }

func (e maskedError) Unwrap() error { return e.err }

func masked(err error) error {
	if err == nil {
		return nil
	}
	return maskedError{err}
}
