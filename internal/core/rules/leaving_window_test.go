package rules_test

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// TestLeavingRuleNamesTheModesLongestWindow holds the LEAVING rule's longest
// window to the away mode's own cap, so the rule cannot promise a window the
// mode refuses (iss-2611211733065807). It lives outside package rules because
// autonomy imports rules.
func TestLeavingRuleNamesTheModesLongestWindow(t *testing.T) {
	if autonomy.MaxWindow != 7*24*time.Hour {
		t.Fatalf("autonomy.MaxWindow is %s; restate the LEAVING rule's seven days to match it", autonomy.MaxWindow)
	}
	d, ok := rules.Defaults().Domains["LEAVING"]
	if !ok || len(d.Rules) == 0 {
		t.Fatal("no bundled LEAVING rule")
	}
	if !strings.Contains(d.Rules[0], "seven days at most") {
		t.Errorf("the LEAVING rule does not name the mode's seven-day cap:\n%s", d.Rules[0])
	}
}
