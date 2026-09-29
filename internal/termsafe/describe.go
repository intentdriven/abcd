package termsafe

import "strconv"

// DescribeRefused says what a refused closed-set value looks like without
// quoting it: its length, or that it is empty. It is for a refusal of a value
// that arrived in a host-composed payload and was not redacted on the way in —
// an enum member, a version string, a mode — where quoting it would carry
// whatever was pasted there (a token, a home path) into the terminal, a log and
// the session transcript. Sanitize is no substitute: it strips control
// sequences and redacts nothing. The length and the listed set beside it are
// enough to find a typo.
func DescribeRefused(value string) string {
	if value == "" {
		return "an empty value"
	}
	return "a " + strconv.Itoa(len(value)) + "-byte value, not quoted"
}
