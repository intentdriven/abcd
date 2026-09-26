package changelog

import (
	"strings"
	"testing"
)

func TestUnreleasedSection(t *testing.T) {
	cases := []struct {
		name              string
		text              string
		heading, firstRow int
		found             bool
	}{
		{"empty section", "# C\n\n## [Unreleased]\n\n## [0.1.0] - 2026-01-01\n- x\n", 2, -1, true},
		{"entry under it", "# C\n\n## [Unreleased]\n\n- a hand entry\n\n## [0.1.0] - 2026-01-01\n", 2, 4, true},
		{"a subheading counts", "## [Unreleased]\n### Added\n## [0.1.0] - 2026-01-01\n", 0, 1, true},
		{"trailing space and CR", "## [Unreleased]  \r\n\r\n- x\r\n", 0, 2, true},
		{"no anchor", "# C\n## [0.1.0] - 2026-01-01\n", -1, -1, false},
		{"entry at end of file", "## [Unreleased]\n- x", 0, 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, first, found := UnreleasedSection(strings.Split(c.text, "\n"))
			if h != c.heading || first != c.firstRow || found != c.found {
				t.Fatalf("got (%d, %d, %v), want (%d, %d, %v)", h, first, found, c.heading, c.firstRow, c.found)
			}
		})
	}
}
