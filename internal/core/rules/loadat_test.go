package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const baseRules = `{"schema_version":1,"disabled":false,"domains":{"BASEONE":{"recall":["word"],"rules":["the rule at the base"]}}}`
const laneRules = `{"schema_version":1,"disabled":false,"domains":{"LANEONE":{"recall":["word"],"rules":["a rule the lane wrote"]}}}`

// TestLoadAtReadsTheRulesAsTheyStoodAtTheBase: what a lane wrote, committed or
// not, reaches nobody reading at its base, and a base that names no commit reads
// nothing.
func TestLoadAtReadsTheRulesAsTheyStoodAtTheBase(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/rules.json", baseRules)
	repo.Commit("the base")
	base := strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(repo.Root(), ".abcd", "rules.json"), []byte(laneRules), 0o600); err != nil {
		t.Fatal(err)
	}
	rs, err := LoadAt(repo.Root(), base)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rs.Domains["BASEONE"]; !ok {
		t.Fatalf("the base's rules are read: %v", strings.Join(domainNames(rs), ","))
	}
	if _, ok := rs.Domains["LANEONE"]; ok {
		t.Fatal("a rule the lane wrote was read")
	}
	if _, err := LoadAt(repo.Root(), "not-a-commit"); err == nil {
		t.Fatal("a base that names no commit reads nothing")
	}
}

func domainNames(rs RuleSet) []string {
	var out []string
	for k := range rs.Domains {
		out = append(out, k)
	}
	return out
}
