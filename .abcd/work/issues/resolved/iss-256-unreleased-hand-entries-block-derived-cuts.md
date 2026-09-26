---
schema_version: 1
id: "iss-256"
slug: "unreleased-hand-entries-block-derived-cuts"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "v0.5.1 derived cut"
found_at: "CHANGELOG.md"
resolution: "record-lint arms changelog_unreleased_empty (internal/core/lint/unreleased.go) as a blocker: any line under ## [Unreleased] fails the record gate naming the record-first flow, read through changelog.UnreleasedSection, the predicate the release ingest now shares. TestChangelogUnreleasedMustStayEmpty and TestChangelogUnreleasedArmedInRealConfig pin it."
impact: internal
resolved_by:
  commit: "1a9dc218"
---

Post-cutover, a hand-written CHANGELOG Unreleased entry blocks every derived cut: ingest refuses a non-empty Unreleased section by design, but nothing stops the entry landing in the first place — the contributor habit predates the derived flow, and the first collision happened the day after the cutover (a merged PR added an Unreleased entry for work that had no resolved record, wedging the next cut until the entry was converted into a record by hand). Detector: a record-lint or CI rule that blocks a PR adding lines under '## [Unreleased]'; acceptance: such a PR fails its record gate with a message naming the record-first flow.

## Grounds

- pursued: a pull request adding lines under ## [Unreleased] fails record-lint before it merges; one that merges green would show it wrong
