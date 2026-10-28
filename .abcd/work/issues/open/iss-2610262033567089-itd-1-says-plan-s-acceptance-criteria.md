---
schema_version: 1
id: "iss-2610262033567089"
slug: "itd-1-says-plan-s-acceptance-criteria"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 455 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-1]
finding_ref: cfk-d0d9ff460ff62f1c
---

itd-1 says plan's acceptance-criteria refusal is a record-lint hard block; the intent surface says it is the intent package's own check, not internal/core/lint

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17` — "`/abcd:intent plan` refuses to promote an intent without it (hard block via the record lint, `internal/core/lint`)."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:198` — "Refuses to promote if `## Acceptance Criteria` is missing/malformed (the intent package's own hasAcceptanceCriteria check, not internal/core/lint)"

itd-1 places the refusal to promote an intent without acceptance criteria in the record lint (`internal/core/lint`). The intent surface's lifecycle says the plan step's refusal is the intent package's own hasAcceptanceCriteria check and explicitly not internal/core/lint. Both cannot say where the gate lives: a reader trusting itd-1 expects the record lint to hold committed intents to the section, which the intent surface says no record-lint rule does.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 455.
