package recall

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// headerLine heads the recall's block, appended after the rules block.
const headerLine = "## MEMORY\n"

// renderItem renders one item in full: a "- " bullet naming its key and
// title, the body indented under it. The text goes through the rule body's
// render-safety pass (rules.SanitizeRuleBody), so a body line shaped like a
// heading or a bullet stays inside its item, as a rule body does.
func renderItem(it Item) string {
	var b strings.Builder
	b.WriteString(it.Key)
	b.WriteString(": ")
	b.WriteString(it.Title)
	if body := strings.TrimSpace(it.Body); body != "" {
		b.WriteString("\n")
		b.WriteString(body)
	}
	return "- " + rules.SanitizeRuleBody(b.String()) + "\n"
}
