package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The agent register's Serves column names every verb that builds an agent's
// input or validates what it returns. The build loop's validate stage hands the
// closing lane to intent-auditor, composes its request and checks its fidelity
// verdict against that request before recording it (34-build.md), so `build`
// serves intent-auditor as much as `intent` does (iss-2610262037002531).
func TestAgentRegisterIntentAuditorServesIntentAndBuild(t *testing.T) {
	path := filepath.Join(repoRootFromPackage, ".abcd", "development", "brief", "04-surfaces", "README.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the agent register: %v", err)
	}
	serves, ok := agentRegisterServes(string(content), "intent-auditor")
	if !ok {
		t.Fatalf("%s: no intent-auditor row in the agent register", path)
	}
	for _, verb := range []string{"intent", "build"} {
		if !strings.Contains(serves, "`"+verb+"`") {
			t.Errorf("%s: intent-auditor's Serves cell is %q; it must name `%s`", path, serves, verb)
		}
	}
}

// agentRegisterServes returns the Serves cell of the named agent's row in the
// first unfenced table whose header carries both an Agent and a Serves column.
func agentRegisterServes(content, agent string) (string, bool) {
	lines := strings.Split(content, "\n")
	mask := fenceMask(lines)
	agentCol, servesCol, header := -1, -1, -1
	for i, line := range lines {
		if mask[i] || !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		a, s := -1, -1
		for j, cell := range tableCells(line) {
			switch {
			case strings.EqualFold(cell, "agent"):
				a = j
			case strings.EqualFold(cell, "serves"):
				s = j
			}
		}
		if a >= 0 && s >= 0 {
			agentCol, servesCol, header = a, s, i
			break
		}
	}
	if header < 0 {
		return "", false
	}
	for i := header + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if mask[i] || !strings.HasPrefix(trimmed, "|") {
			break
		}
		if isTableSeparator(trimmed) {
			continue
		}
		cells := tableCells(lines[i])
		if agentCol >= len(cells) || servesCol >= len(cells) {
			continue
		}
		if m := agentNameRe.FindStringSubmatch(cells[agentCol]); m != nil && m[1] == agent {
			return cells[servesCol], true
		}
	}
	return "", false
}
