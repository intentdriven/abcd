package assistedby

import (
	"regexp"
	"strings"
)

// The pull-request body check (iss-2609061525437969): the attribution gate's
// body arm, check_text over strip_fenced_blocks in scripts/check-attribution.sh,
// in Go. The gate runs in CI without Go and decides it; this is its one Go copy,
// which the shell guard judges a body with before gh posts it and `abcd land`
// judges the body it composes with, tied to the gate by TestJudgeBodyMatchesTheGate.
// The gate's session-URL half is not here: it is scanner.CheckOutbound, which
// both callers already run.

// BodyFault is what the gate's body arm refuses a pull-request body for, the
// first it finds in the gate's own order.
type BodyFault int

const (
	// BodyClean: the gate accepts the body.
	BodyClean BodyFault = iota
	// BodyGeneratedFooter: a tool's "Generated with [...]" footer line.
	BodyGeneratedFooter
	// BodyCoAuthor: a Co-authored-by line, which asserts an authorship no tool
	// holds.
	BodyCoAuthor
	// BodyMalformedLabel: an Assisted-by line naming abcd in a form abcd never
	// writes.
	BodyMalformedLabel
	// BodyNoTrailer: no Assisted-by line in one of the three accepted forms.
	BodyNoTrailer
)

func (f BodyFault) String() string {
	switch f {
	case BodyClean:
		return "clean"
	case BodyGeneratedFooter:
		return "a tool's 'Generated with' footer"
	case BodyCoAuthor:
		return "a Co-authored-by line"
	case BodyMalformedLabel:
		return "an Assisted-by line naming abcd in a form abcd never writes"
	case BodyNoTrailer:
		return "no Assisted-by line in an accepted form"
	}
	return "an unknown fault"
}

// The gate's GENERATED_RE and COAUTHOR_RE, read out of the script by
// TestBodyRefusalPatternsAreTheGates.
const (
	generatedFooterPattern = `^[[:space:]]*(🤖[[:space:]]*)?([_*]{1,3})?(🤖[[:space:]]*)?[Gg]enerated (with|by) \[`
	coAuthorPattern        = `^[[:space:]]*[Cc]o-[Aa]uthored-[Bb]y:`
)

var (
	generatedFooterRe = regexp.MustCompile(generatedFooterPattern)
	coAuthorRe        = regexp.MustCompile(coAuthorPattern)
)

// trailerKey is the trailer's key with the one space the gate's patterns
// require before the value.
const trailerKey = "Assisted-by: "

// JudgeBody judges a pull-request body as the gate's body arm does: fenced
// blocks stripped first, since a forge renders them as examples, then CR
// removed, then each line read on its own.
func JudgeBody(body string) BodyFault {
	lines := strings.Split(strings.ReplaceAll(stripFencedBlocks(body), "\r", ""), "\n")
	for _, l := range lines {
		if generatedFooterRe.MatchString(l) {
			return BodyGeneratedFooter
		}
	}
	for _, l := range lines {
		if coAuthorRe.MatchString(l) {
			return BodyCoAuthor
		}
	}
	for _, l := range lines {
		if v, ok := strings.CutPrefix(l, trailerKey); ok && NamesAbcd(v) && !IsComposed(v) {
			return BodyMalformedLabel
		}
	}
	for _, l := range lines {
		if v, ok := strings.CutPrefix(l, trailerKey); ok && (IsModelValue(v) || v == "None" || v == "none" || IsComposed(v)) {
			return BodyClean
		}
	}
	return BodyNoTrailer
}

// stripFencedBlocks is the gate's strip_fenced_blocks: it drops each markdown
// fenced block, opener and closer included. The gate's comment carries the
// reasoning for every rule, and the corpus in body_test.go holds this copy to
// the gate's behaviour on each: an opener is a run of three or more backticks
// or tildes indented at most three spaces (a backtick run may not carry a
// backtick after it); a closer is a run of the same character at least as long
// with nothing but blanks after it; an unterminated fence, or any line that
// opens an HTML block (a `<` indented at most three spaces), strips nothing.
func stripFencedBlocks(text string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if strings.HasSuffix(text, "\n") {
		lines = lines[:len(lines)-1]
	}
	for _, l := range lines {
		t := strings.TrimLeft(l, " ")
		if len(l)-len(t) <= 3 && strings.HasPrefix(t, "<") {
			return joinLines(lines)
		}
	}
	var kept []string
	inside, openChar, openLen := false, byte(0), 0
	for _, l := range lines {
		c, n, rest, ok := fenceRun(l)
		if !inside {
			if !ok || (c == '`' && strings.Contains(rest, "`")) {
				kept = append(kept, l)
				continue
			}
			inside, openChar, openLen = true, c, n
			continue
		}
		if ok && c == openChar && n >= openLen && strings.Trim(rest, " \t\r") == "" {
			inside = false
		}
	}
	if inside {
		return joinLines(lines)
	}
	return joinLines(kept)
}

// fenceRun is the gate's fenceparse: the fence character a line opens or
// closes with, its run length and the text after the run, or ok false when
// the line can do neither.
func fenceRun(l string) (c byte, n int, rest string, ok bool) {
	t := strings.TrimLeft(l, " ")
	if len(l)-len(t) > 3 || t == "" || (t[0] != '`' && t[0] != '~') {
		return 0, 0, "", false
	}
	c = t[0]
	for n < len(t) && t[n] == c {
		n++
	}
	if n < 3 {
		return 0, 0, "", false
	}
	return c, n, t[n:], true
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
