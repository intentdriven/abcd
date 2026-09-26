package changelog

import (
	"regexp"
	"strings"
)

// unreleasedHeadingRe is the `## [Unreleased]` anchor a derived cut inserts its
// dated section beneath.
var unreleasedHeadingRe = regexp.MustCompile(`^## \[Unreleased\]\s*$`)

// UnreleasedSection locates the `## [Unreleased]` heading in a changelog's lines
// and the first non-blank line under it, before the next `## ` heading. heading
// is the heading's 0-based index and found reports whether there is one;
// firstEntry is -1 when the section is empty.
//
// It is the ONE reading of "the Unreleased section is empty": the release
// ingest refuses a cut over a non-empty section (a derived cut never folds
// hand-written prose into a generated one), and record-lint's
// changelog_unreleased_empty rule refuses the change that would put an entry
// there in the first place, so the two cannot disagree about what counts as an
// entry (iss-256).
func UnreleasedSection(lines []string) (heading, firstEntry int, found bool) {
	heading = -1
	for i, line := range lines {
		if unreleasedHeadingRe.MatchString(strings.TrimRight(line, "\r")) {
			heading = i
			break
		}
	}
	if heading < 0 {
		return -1, -1, false
	}
	for i := heading + 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.TrimSpace(line) != "" {
			return heading, i, true
		}
	}
	return heading, -1, true
}
