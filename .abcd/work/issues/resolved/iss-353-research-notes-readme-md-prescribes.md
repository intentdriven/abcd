---
schema_version: 1
id: "iss-353"
slug: "research-notes-readme-md-prescribes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/research/notes/README.md"
resolution: "notes README naming section rewritten to the dated adr-30 form; undated pre-adr-30 files declared legacy"
impact: internal
---

research/notes/README.md prescribes topic-kind note naming while adr-30 and development/README.md ratify date-prefixed naming and the record-lint successor plus ideate both emit dated notes; the notes README is the outlier and 29 of 41 notes are dated
## Evidence

- `research/notes/README.md:29-35` prescribes `<topic>-<kind>.md` with no date slot; adr-30:47 (accepted, unsuperseded) rules "Plans, research notes, and the Tier-2 DECISIONS.md log are date-prefixed"; `development/README.md:24-25` restates it; `record-lint.json:17` names the successor as `research/notes/<date>-<slug>.md`; `internal/core/ideate/ideate.go:50` writes dated notes. Disk: 29 dated / 12 not (one numeric-prefixed, a form line 35 disowns).
- Prior art: flagged in the 2026-07-06 review (`05-link-structure.md:160`) but the iss-42 disposition scoped to note *location*, not naming; the drift has since inverted sides.
- Refuter verdict: CONFIRMED substantive; `notes/README.md` yields to the ADR. Fix rewrites the naming section to `<YYYY-MM-DD>-<topic>[-<kind>].md`, keeps the kind vocabulary as an optional suffix, notes pre-ADR-30 undated files as legacy, renames nothing.
