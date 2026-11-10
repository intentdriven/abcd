package coord

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

const (
	// maxBodyBytes is the largest body parseDependsOn reads. A larger one is
	// refused, never truncated, so no dependency line is lost past the cut.
	maxBodyBytes = 65536
	// maxDepends is the most distinct dependencies one body may name, and the
	// most one change in a snapshot may carry.
	maxDepends = 10
)

var (
	// dependsLineRe is the one accepted shape, the shape of the declaration
	// lines in internal/core/landing/prbody.go: exact case, a number of one to
	// nine digits with no leading zero, so every number fits an int.
	dependsLineRe = regexp.MustCompile(`^Depends-On:[ \t]+#[1-9][0-9]{0,8}([ \t]*,[ \t]*#[1-9][0-9]{0,8})*[ \t]*$`)
	dependsNumRe  = regexp.MustCompile(`#([1-9][0-9]{0,8})`)
	// dependsWordRe is the scan for a line that looks like a dependency line
	// in a wrong shape, matched against the line after scanForm: depends-on
	// anywhere, or depend or depends, any run of hyphens, underscores, spaces,
	// tabs, Unicode dashes (Pd), minus signs (U+2212) or hyphen bullets
	// (U+2043), on, then optional spaces or tabs and a colon, in any case.
	dependsWordRe = regexp.MustCompile(`(?i)depends-on|depends?[-_ \t\p{Pd}\x{2212}\x{2043}]*on[ \t]*:`)
)

// parseDependsOn reads the dependencies a pull request body declares
// (dcn-2610180720362692): each line exactly `Depends-On: #N` or
// `Depends-On: #N, #N, ...`, the numbers separated by commas with optional
// spaces or tabs around each, the body split on newlines with one trailing
// carriage return stripped from each line. The numbers of every accepted line
// are joined, ascending and once each, nil for none.
//
// The body is untrusted, so the parse fails closed and no text of it reaches
// the error: more than maxDepends distinct numbers, and a body over
// maxBodyBytes, are refused, and so is any line that is not an accepted line
// but looks like one in a wrong shape, by its 1-based line number alone.
//
// To find those, each such line is first stripped by scanForm of every
// character in Unicode categories Cf (format), Mn (non-spacing marks) and Me
// (enclosing marks), of the Hangul fillers U+115F, U+1160 and U+3164, and of
// every NUL and backslash. The tests pin these of them: the zero-width space,
// zero-width joiner, word joiner, soft hyphen and byte-order mark (Cf); the
// combining grapheme joiner U+034F, the variation selector U+FE0F and the
// Mongolian free variation selector U+180B (Mn); the enclosing circle U+20DD
// (Me); the three fillers; NUL; and the backslash. The stripped line is then
// refused when it contains depends-on in any case anywhere, or, in any case,
// depend or depends, any run of hyphens, underscores, ASCII spaces, tabs,
// Unicode dashes (category Pd), minus signs (U+2212) or hyphen bullets
// (U+2043), on, optional spaces or tabs, and a colon.
//
// Nothing else is decoded, so these are read past as no dependency, each
// pinned by a test: an HTML entity (Depends&#45;On); a letter from another
// script that looks like a Latin one; a fullwidth letter; any space but an
// ASCII space or tab between the words; a non-ASCII space between on and the
// colon in a form with no ASCII hyphen (Depends On, a no-break space, then a
// colon); a form with no colon (Depends on #5); Depends: alone; an invisible
// character outside the stripped set, such as the halfwidth Hangul filler
// U+FFA0; and the word split by a newline, a lone carriage return, a line
// separator U+2028, a next-line character U+0085 or a byte that is not valid
// UTF-8. The scan is for mistakes, not for an author set against it: an author
// who wants no hold can leave the line out.
//
// The scan also refuses honest lines that are no dependency line, in the safe
// direction and by line number: ordinary prose with a colon after the words
// (The order depends on: the queue), a quoted or indented compose key
// (depends_on:), a path or URL holding depends-on, a word that begins with it
// (depends-only), a Windows path such as C:\depends\on: once its
// backslashes are dropped, and a word written with a decomposed accent, such
// as dépend on: with its accent a combining mark (Mn), once the mark is
// dropped; the same word with its accent precomposed (U+00E9) is read past.
// The error tells the author to reword that line or write it in the accepted
// form.
//
// Lines are judged one at a time, with no regard for markdown. A well-formed
// line inside a code fence or an HTML comment that spans several lines is read
// as a dependency, which holds its change rather than freeing it. A line
// quoting a placeholder that names depends-on in the wrong shape, such as
// <!-- Depends-On: #N -->, is refused, so a body that keeps a template's
// placeholder is refused until the placeholder is edited or removed.
//
// It is staged code like the pass: nothing outside the package calls it yet.
// Step 2a's landing wiring exports it and wires its first caller in one
// commit, and it is never pushed or landed without that wiring
// (dcn-2611101309224327).
func parseDependsOn(body string) ([]int, error) {
	if len(body) > maxBodyBytes {
		return nil, fmt.Errorf("coord: the body is %d bytes, above the most of %d", len(body), maxBodyBytes)
	}
	var nums []int
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if !dependsLineRe.MatchString(line) {
			if dependsWordRe.MatchString(scanForm(line)) {
				return nil, fmt.Errorf(`coord: line %d looks like a Depends-On line but is not in the accepted form "Depends-On: #N, #N"; reword that line, or write it in that form`, i+1)
			}
			continue
		}
		for _, m := range dependsNumRe.FindAllStringSubmatch(line, -1) {
			// The accepted shape gives one to nine ASCII digits, so the number
			// is folded with no error path and always fits an int.
			n := 0
			for _, d := range m[1] {
				n = n*10 + int(d-'0')
			}
			nums = append(nums, n)
		}
	}
	slices.Sort(nums)
	nums = slices.Compact(nums)
	if len(nums) > maxDepends {
		return nil, fmt.Errorf("coord: the body names %d dependencies, above the most of %d", len(nums), maxDepends)
	}
	return nums, nil
}

// scanForm is the line the fail-closed scan reads: the line with every
// character of categories Cf, Mn and Me, the Hangul fillers U+115F, U+1160 and
// U+3164, NUL and backslash dropped, and nothing else changed. A byte that is
// not valid UTF-8 is kept, as U+FFFD.
func scanForm(line string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == 0, r == '\\', r == '\u115f', r == '\u1160', r == '\u3164',
			unicode.In(r, unicode.Cf, unicode.Mn, unicode.Me):
			return -1
		}
		return r
	}, line)
}
