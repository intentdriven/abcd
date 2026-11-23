---
schema_version: 1
id: "iss-2610262035308321"
slug: "the-lint-chapter-gives-the-record-lint"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 576 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:168"
remedy: "In 06-lint.md:12 say a rule's severity resolves from its config entry as blocker or warn, and that info is carried only by a code-pinned report rule (internal/core/lint/config.go severityPinnedRules)."
finding_ref: cfk-8b1aec720d7040f4
resolution: "06-lint.md line 12 now says a rule's severity resolves from its config entry as blocker or warn, and that info is never a config choice: it is set in code, for reports (reading_outstanding, pinned in severityPinnedRules, and the prose_citation_pending and prose_citation_baseline_stale findings of the prose_citation_resolves check)"
impact: internal
resolved_by:
  commit: "85b4597f8ae850f28ab17089b7a2092504a315f8"
---

The lint chapter gives the record-lint engine two severities, `blocker|warn`; the lint contract gives the same engine three, `blocker`, `warn` and `info`.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:168` — "That is a different vocabulary from the record-lint engine's (`internal/core/lint`) `blocker|warn`"
- End B: `.abcd/development/brief/05-internals/06-lint.md:12` — "Each rule carries a severity (`blocker`, `warn`, or `info`) resolved from its config entry"

16-lint contrasts its own `error|warn|off` vocabulary with the `internal/core/lint` engine's `blocker|warn`. 06-lint says every rule of that engine carries `blocker`, `warn` or `info`, and gives `info` its own row in the severity model. A rule set at `info` has no place in 16-lint's account of the vocabulary its boundary mapping translates.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 576.

## Grounds

- pursued: internal/core/lint/config.go validateSeverities refuses an enabled rule's severity other than blocker or warn, unless the rule is in severityPinnedRules, which holds only reading_outstanding pinned to info; prosecitations.go:291 and :320 set prose_citation_pending and prose_citation_baseline_stale to severityInfo in code. Shown wrong if a config value of info were accepted for a rule outside severityPinnedRules, if that set gained an entry, or if another finding were set to info in code
