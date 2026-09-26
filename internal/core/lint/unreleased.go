package lint

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
)

// ruleChangelogUnreleasedEmpty keeps `## [Unreleased]` empty. The changelog is
// derived: `launch ship` composes a dated section from the records that reached
// a terminal folder since the last tag and inserts it beneath the Unreleased
// anchor, and the ingest refuses a section that already holds prose, because a
// derived cut never folds a hand-written entry into a generated one. Nothing
// stopped the entry landing in the first place, so a merged hand entry wedged
// the next cut until someone converted it into a record by hand. This rule is
// that refusal moved to the change that writes it (iss-256).
const ruleChangelogUnreleasedEmpty = "changelog_unreleased_empty"

// checkChangelogUnreleasedEmpty reports the first line under `## [Unreleased]`,
// read through the same predicate the release ingest reads, and an absent
// anchor, which the ingest refuses too. A changelog that cannot be read is a
// configuration error: an armed gate with nothing to read is never a pass.
func checkChangelogUnreleasedEmpty(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	path := cfg.Changelog
	if path == "" {
		path = "CHANGELOG.md"
	}
	data, err := readRepoFile(repoRoot, path, maxChangelogBytes)
	if err != nil {
		return nil, &configError{ruleChangelogUnreleasedEmpty + ": reading " + path + ": " + err.Error()}
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	_, first, found := changelog.UnreleasedSection(lines)
	switch {
	case !found:
		return []Finding{{
			File: path, Line: 1, RuleID: ruleChangelogUnreleasedEmpty, Severity: cfg.Severity,
			Message: "no `## [Unreleased]` heading: it is the anchor a derived cut inserts its dated section " +
				"beneath, and `launch ship` refuses a changelog without one",
		}}, nil
	case first >= 0:
		return []Finding{{
			File: path, Line: first + 1, RuleID: ruleChangelogUnreleasedEmpty, Severity: cfg.Severity,
			Message: "`## [Unreleased]` is not empty: the changelog is derived from records, and a hand-written " +
				"entry here blocks the next cut. Announce the change with a record instead — resolve its issue " +
				"(abcd capture resolve) or close its spec and ship its intent (abcd spec close) in this change — " +
				"and remove the entry",
		}}, nil
	}
	return nil, nil
}
