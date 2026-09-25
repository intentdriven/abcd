package lint

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// personaAttrRe matches a press-release quote attribution: `said <Name>,`.
// The trailing comma anchors the persona-attribution form ("said Kira, a
// maintainer") and keeps ordinary prose ("as we said above") out of scope.
// The name class is Unicode-wide (letters, marks, apostrophes, hyphens) so
// compound and non-ASCII names (O'Brien, Anne-Marie, Zoë) cannot slip past
// as silent non-matches.
var personaAttrRe = regexp.MustCompile(`\bsaid (\p{Lu}[\p{L}\p{M}'’-]*),`)

// PersonaAttribution returns the first persona name text attributes words to
// in the `said <Name>,` form the persona_registry rule reads, and whether it
// found one. The release page refuses one in headline prose, where no quote is
// verified against its source.
func PersonaAttribution(text string) (string, bool) {
	m := personaAttrRe.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// loadPersonaRoster reads the personas registry and returns the set of
// registered names. The registry is the single source of truth for persona
// names (selection is by role; the role's registered name is used).
func loadPersonaRoster(repoRoot, rel string) (map[string]bool, error) {
	if rel == "" {
		return nil, fmt.Errorf("persona_registry: rule enabled but \"registry\" is not set")
	}
	data, err := readRepoFile(repoRoot, rel, maxRepoFileBytes)
	if err != nil {
		return nil, fmt.Errorf("persona_registry: reading roster %s: %w", rel, err)
	}
	var reg struct {
		Personas []struct {
			Name string `json:"name"`
		} `json:"personas"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("persona_registry: parsing roster %s: %w", rel, err)
	}
	if len(reg.Personas) == 0 {
		return nil, fmt.Errorf("persona_registry: roster %s has no personas — misconfigured registry, refusing to flag the whole record", rel)
	}
	roster := make(map[string]bool, len(reg.Personas))
	for _, p := range reg.Personas {
		roster[p.Name] = true
	}
	return roster, nil
}

// checkPersonaRegistry flags quote attributions whose persona name is not in
// the registry roster. Fenced-code lines are skipped via the caller's mask;
// content-exempt files (historical record) are the caller's concern.
func checkPersonaRegistry(rel string, lines []string, mask []bool, roster map[string]bool, cfg RuleConfig) []Finding {
	var out []Finding
	for i, line := range lines {
		if mask[i] {
			continue
		}
		for _, m := range personaAttrRe.FindAllStringSubmatch(line, -1) {
			if roster[m[1]] {
				continue
			}
			out = append(out, Finding{
				File:     rel,
				Line:     i + 1,
				RuleID:   "persona_registry",
				Severity: cfg.Severity,
				Message:  fmt.Sprintf("persona %q is not in the registry (%s); personas are selected by role and use the role's registered name", m[1], cfg.Registry),
			})
		}
	}
	return out
}

// PersonaFindingsInText runs the persona_registry rule, exactly as cfg arms it,
// over one text that sits outside cfg.Roots. The release page is the case: it is
// written to RELEASE.md at the repository root, which record-lint's roots do not
// reach (the root cannot be listed there, because a configured root that does
// not exist is a refusal and the page arrives with the first feature cut). The
// cut runs this over the rendered page before writing it. A rule that is not
// armed returns nothing; a roster that cannot be read is an error.
func PersonaFindingsInText(cfg Config, repoRoot, rel, text string) ([]Finding, error) {
	rc, on := cfg.Rules["persona_registry"]
	if !on || !rc.Enabled {
		return nil, nil
	}
	roster, err := loadPersonaRoster(repoRoot, rc.Registry)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(text, "\n")
	return checkPersonaRegistry(rel, lines, fenceMask(lines), roster, rc), nil
}
