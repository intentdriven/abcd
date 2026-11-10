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
	// dependsWordRe is the scan for a line that names a dependency in a wrong
	// shape, matched against the line after scanForm: depends-on anywhere, or
	// depend or depends, any run of hyphens, underscores, spaces, tabs or
	// Unicode dashes, on, then optional spaces or tabs and a colon, in any case.
	dependsWordRe = regexp.MustCompile(`(?i)depends-on|depends?[-_ \t\p{Pd}]*on[ \t]*:`)
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
// but names a dependency in a wrong shape, by its 1-based line number alone.
// To find those, each such line is first stripped of every Unicode format
// character (category Cf: zero-width characters, the soft hyphen, the
// byte-order mark), every NUL and every backslash, and is then refused when it
// contains depends-on in any case anywhere, or, in any case, depend or depends,
// any run of hyphens, underscores, ASCII spaces, tabs or Unicode dashes
// (category Pd), on, optional spaces or tabs, and a colon.
//
// Nothing else is decoded, so these are read past as no dependency: an HTML
// entity (Depends&#45;On), a letter from another script that looks like a
// Latin one, a fullwidth letter, any space but an ASCII space or tab between
// the words, a form with no colon (Depends on #5), Depends: alone, and a word
// split across two lines. The scan is for mistakes, not for an author set
// against it: an author who wants no hold can leave the line out.
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
				return nil, fmt.Errorf(`coord: line %d names depends-on but is not a line of the form "Depends-On: #N, #N"`, i+1)
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

// scanForm is the line the fail-closed scan reads: the line with every Unicode
// format character (Cf), NUL and backslash dropped, and nothing else changed.
func scanForm(line string) string {
	return strings.Map(func(r rune) rune {
		if r == 0 || r == '\\' || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, line)
}
