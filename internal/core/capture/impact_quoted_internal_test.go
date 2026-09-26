package capture

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// gateImpactVerdict is record-lint's issue_impact_valid reading of one impact
// line, restated from checkIssueImpact over the same shared primitives
// (frontmatter.Fields for the RAW scalar, frontmatter.IsNull, and
// changelog.ParseImpact): a null passes on an open record, anything else must
// parse as an impact exactly as written, quotes included.
func gateImpactVerdict(line string) bool {
	f := frontmatter.Fields([]string{"---", line, "---"})["impact"]
	if frontmatter.IsNull(f.Value) {
		return true
	}
	_, err := changelog.ParseImpact(f.Value)
	return err == nil
}

// TestImpactQuotingReachesOneVerdict is iss-2608261133218490's differential
// test: the capture reader and the committed-ledger gate must reach the same
// verdict on every spelling of one impact value. The gate reads the raw scalar,
// so a quoted legal enum ("fix") and a quoted empty value ("") are refused
// there — and were accepted here, because the parser unquotes before the
// validator sees the value. A record the gate blocks must be one the reader
// refuses, or it loads and resolves cleanly past its own blocker.
func TestImpactQuotingReachesOneVerdict(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{
			"schema_version": 1, "id": "iss-1", "slug": "x", "severity": "minor",
			"category": "bug", "source": "agent-finding", "found_during": "review",
		}
	}
	for _, line := range []string{
		"impact: fix",
		"impact: internal",
		"impact:",
		"impact: null",
		`impact: "fix"`,
		`impact: "internal"`,
		`impact: ""`,
		`impact: "null"`,
		"impact: sideways",
		`impact: "sideways"`,
	} {
		t.Run(line, func(t *testing.T) {
			fm, err := parseFrontmatterBlock([]string{line})
			if err != nil {
				t.Fatalf("parse %q: %v", line, err)
			}
			m := valid()
			m["impact"] = fm["impact"]
			reader := validateStrict(m) == nil
			gate := gateImpactVerdict(line)
			if reader != gate {
				t.Errorf("%q: the reader accepts=%v but the gate accepts=%v — one value, two verdicts", line, reader, gate)
			}
		})
	}
}
