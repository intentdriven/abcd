package gitutil_test

import (
	"errors"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestGitAtLeastNamesTheVersionInDigitsAlone: the floor check says whether
// git is at least major.minor, and names its version as major.minor, digits
// alone; a probe that fails, or a line that does not parse, is below every
// floor and names none.
func TestGitAtLeastNamesTheVersionInDigitsAlone(t *testing.T) {
	for _, c := range []struct {
		line    string
		err     error
		ok      bool
		version string
	}{
		{"git version 2.28.1 (Apple Git-1)", nil, false, "2.28"},
		{"git version 2.29.0", nil, true, "2.29"},
		{"git version 3.0.0.windows.1", nil, true, "3.0"},
		{"git version two", nil, false, ""},
		{"", errors.New("no git"), false, ""},
	} {
		restore := gitutil.SwapGitVersionForTest(func() (string, error) { return c.line, c.err })
		ok, version := gitutil.GitAtLeast(2, 29)
		restore()
		if ok != c.ok || version != c.version {
			t.Errorf("GitAtLeast(2, 29) under %q = %v, %q; want %v, %q", c.line, ok, version, c.ok, c.version)
		}
	}
}
