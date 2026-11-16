package rules

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
)

// Load is LoadWith at abcd's default question limits with no note, the
// bundled GRILL domain: the form the package's own tests load with.
func Load(repoRoot string) (RuleSet, error) {
	return LoadWith(repoRoot, question.Default, question.LimitsNote{})
}

// Inject is InjectWithin with nothing reserved, the budget of a turn with no
// status or away block.
func Inject(rs RuleSet, prompt string, prev SessionState, backstop int, bracket Bracket) InjectResult {
	return InjectWithin(rs, prompt, prev, backstop, bracket, RulesBudget(0))
}

// Render is every domain in full under the one header: what the hook sends when
// all of it fits. An empty list renders to zero bytes.
func Render(domains []ResolvedDomain) string {
	if len(domains) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# abcd rules — %d domain(s) active\n", len(domains))
	for _, d := range domains {
		b.WriteString(renderDomain(d))
	}
	return b.String()
}
