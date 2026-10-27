---
schema_version: 1
id: "iss-2610262034164290"
slug: "itd-85-has-prepare-this-repo-read-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 483 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-183a67ea409b2853
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033013099. Same draft edit as iss-2610262033013099, set against a different statement of the same current state."
duplicates: [iss-2610262033013099]
---

itd-85 has `prepare-this-repo` read `abcd audit --json`; the board chapter says the install's preparation workflow runs `abcd lint` for its audit half.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:31` — "`prepare-this-repo` Phase 2 consumes `abcd audit --json` in place of the hand-produced gap report"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:148` — "its audit half runs `abcd lint`, and its adoption half is binary-backed"

itd-85 connects onboarding to a new `abcd audit --json` report, read in `prepare-this-repo` Phase 2. 08-abcd.md says the preparation workflow, now on the ahoy page, already does its audit half with `abcd lint`. The two documents disagree about what onboarding's audit step runs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 483.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033013099. Same draft edit as iss-2610262033013099, set against a different statement of the same current state.
