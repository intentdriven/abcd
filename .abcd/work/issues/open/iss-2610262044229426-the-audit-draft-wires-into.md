---
schema_version: 1
id: "iss-2610262044229426"
slug: "the-audit-draft-wires-into"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1201 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-85, itd-2610090831227812]
finding_ref: cfk-d5c6a9ff1cee9954
---

The audit draft wires into prepare-this-repo Phase 2, but prepare-this-repo has folded into ahoy install with no alias

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:31` — "Wiring: `prepare-this-repo` Phase 2 consumes `abcd audit --json` in place of the hand-produced gap report."
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:48` — "`prepare-this-repo` folds into `ahoy install`; `consult`, `ingest`, `memory` and `source` merge into `library`"

itd-85's wiring depends on `prepare-this-repo` Phase 2 consuming `abcd audit --json`. itd-2610090831227812 folds `prepare-this-repo` into `ahoy install`, and under adr-40 no old name survives. The surface the draft wires into no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1201.
