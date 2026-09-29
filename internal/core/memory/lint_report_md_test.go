package memory

import (
	"strings"
	"testing"
)

// memory lint's run-log report.md is markdown, and a finding's file, message and
// suggestion carry names the store or the per-repo scanner override supplied.
// Sanitize alone defangs a terminal and leaves an HTML comment opener or link
// syntax live, so each field is cleaned with the file-write cleaner and a file
// is set off as a code span (iss-2609262148072415).
func TestLintReportMDCleansEveryFindingField(t *testing.T) {
	md := renderLintReportMD(map[string]any{
		"store_path": ".abcd/memory<!-- s",
		"summary":    map[string]any{"blockers": 1},
		"findings": []any{map[string]any{
			"code": "MR001", "severity": "blocker", "line": 3,
			"file":       "topic_<!--x_[a](http://example.com).md",
			"message":    "pattern <script>evil</script> is unavailable",
			"suggestion": "see [docs](http://example.com) <!-- hidden",
		}},
	})
	for _, h := range []string{"<!--", "<script", "](http"} {
		if strings.Contains(md, h) {
			t.Errorf("report.md carries a live %q from an untrusted finding field:\n%s", h, md)
		}
	}
	if !strings.Contains(md, "`topic_") {
		t.Errorf("report.md does not set the finding's file off as a code span:\n%s", md)
	}
}
