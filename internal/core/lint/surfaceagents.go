package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// agentRow is one parsed row of the surface registry's agents table.
type agentRow struct {
	name string // the agent's name: its page is <agents_dir>/<name>.md
	line int    // 1-based line of the row in the registry file
}

// agentNameRe takes the agent's name from an Agent cell: the first backticked
// token, so a cell may link the name to its page.
var agentNameRe = regexp.MustCompile("`([a-z][a-z0-9-]*)`")

// checkAgentCoverage is surface_coverage's sweep of the agent pages
// (iss-2610110456358519). A harness registers every markdown file directly
// under the agents directory as an invocable agent, exactly as it registers
// every file under commands/ as a command, so the register holds the two to
// the same rule in both directions: every agent page has a row in the
// registry's agents table, and every row names a page that exists. A
// subdirectory is an agent's fixtures, never an agent. Unarmed while
// cfg.AgentsDir is empty; a registry with no agents table leaves every page
// without a row, which is a finding per page rather than a silent pass.
func checkAgentCoverage(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	if cfg.AgentsDir == "" || cfg.Registry == "" {
		return nil, nil
	}
	rows, err := parseAgentRegister(repoRoot, cfg.Registry)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil // registry file absent — nothing to cross-check
	}
	pages, err := agentPages(repoRoot, cfg.AgentsDir)
	if err != nil {
		return nil, err
	}

	var out []Finding
	rowNames := make(map[string]bool, len(rows))
	for _, r := range rows {
		rowNames[r.name] = true
		if !pages[r.name] {
			out = append(out, Finding{
				File: cfg.Registry, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: "agent row '" + r.name + "' names no page: " + filepath.ToSlash(filepath.Join(cfg.AgentsDir, r.name+".md")) + " does not exist",
			})
		}
	}
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !rowNames[name] {
			out = append(out, Finding{
				File: filepath.Join(cfg.AgentsDir, name+".md"), Line: 0, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: "agent page '" + name + "' has no row in the agents table of the brief surface registry (" + cfg.Registry + ")",
			})
		}
	}
	return out, nil
}

// agentPages is the set of agent names the agents directory registers: the
// stem of every *.md file directly under it. A missing directory registers
// none and is not an error.
func agentPages(repoRoot, dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, dir))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	set := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		set[strings.TrimSuffix(e.Name(), ".md")] = true
	}
	return set, nil
}

// parseAgentRegister reads the registry's agents table: the first table outside
// a fence whose header names an Agent column. Each data row's Agent cell names
// the agent by its first backticked token; a row with none is skipped. A
// missing file yields (nil, nil); a present file with no such table yields an
// empty, non-nil slice.
func parseAgentRegister(repoRoot, registry string) ([]agentRow, error) {
	content, err := readRepoFile(repoRoot, registry, maxRepoFileBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	mask := fenceMask(lines)

	col, header := -1, -1
	for i, line := range lines {
		if mask[i] || !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		for j, cell := range tableCells(line) {
			if strings.EqualFold(cell, "agent") {
				col, header = j, i
				break
			}
		}
		if header >= 0 {
			break
		}
	}
	rows := []agentRow{}
	if header < 0 {
		return rows, nil
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
		if col >= len(cells) {
			continue
		}
		m := agentNameRe.FindStringSubmatch(cells[col])
		if m == nil {
			continue
		}
		rows = append(rows, agentRow{name: m[1], line: i + 1})
	}
	return rows, nil
}
