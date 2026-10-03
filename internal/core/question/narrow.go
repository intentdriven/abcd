package question

import "strings"

// Matches is the narrowing rule (spc-2610030911534855, "The answer loop"):
// the typed text, matched case-insensitively as a substring of the option's
// label or of its value. Empty text matches every option. The plain-Terminal
// list narrows with it and so does guided connect's session
// (spc-2610031241482088), so a fragment narrows the same in both places.
func Matches(filter string, o Option) bool {
	if filter == "" {
		return true
	}
	f := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(o.Label), f) || strings.Contains(strings.ToLower(o.Value), f)
}

// Narrow returns the indices of the options filter matches, in order.
func Narrow(filter string, opts []Option) []int {
	out := make([]int, 0, len(opts))
	for i, o := range opts {
		if Matches(filter, o) {
			out = append(out, i)
		}
	}
	return out
}
