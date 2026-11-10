package coord

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	// maxBodyBytes is the largest body parseDependsOn reads. A larger one is
	// refused, never truncated, so no dependency line is lost past the cut.
	maxBodyBytes = 65536
	// maxDepends is the most distinct dependencies one body may name.
	maxDepends = 10
)

var (
	// dependsLineRe is the one accepted shape, the shape of the declaration
	// lines in internal/core/landing/prbody.go: exact case, a number of one to
	// nine digits with no leading zero, so every number fits an int.
	dependsLineRe = regexp.MustCompile(`^Depends-On:[ \t]+#[1-9][0-9]{0,8}([ \t]*,[ \t]*#[1-9][0-9]{0,8})*[ \t]*$`)
	dependsNumRe  = regexp.MustCompile(`#([1-9][0-9]{0,8})`)
	// dependsWordRe finds a line that means to declare a dependency in any
	// case, anywhere on it, so a line of the wrong shape refuses rather than
	// being read past, which would queue its dependant unheld.
	dependsWordRe = regexp.MustCompile(`(?i)depends-on`)
)

// parseDependsOn reads the dependencies a pull request body declares
// (dcn-2610180720362692): each line exactly `Depends-On: #N` or
// `Depends-On: #N, #N, ...`, the body split on newlines with one trailing
// carriage return stripped from each line. The numbers of every accepted line
// are joined, ascending and once each, nil for none.
//
// The body is untrusted, so the parse fails closed and no text of it reaches
// the error: any other line naming depends-on, in any case and anywhere on the
// line, is refused by its 1-based line number alone; more than maxDepends
// distinct numbers, and a body over maxBodyBytes, are refused too.
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
			if dependsWordRe.MatchString(line) {
				return nil, fmt.Errorf(`coord: line %d names depends-on but is not a line of the form "Depends-On: #N, #N"`, i+1)
			}
			continue
		}
		for _, m := range dependsNumRe.FindAllStringSubmatch(line, -1) {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				return nil, fmt.Errorf("coord: line %d carries a number that cannot be read", i+1)
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
