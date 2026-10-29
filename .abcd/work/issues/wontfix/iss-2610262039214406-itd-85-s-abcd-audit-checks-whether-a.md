---
schema_version: 1
id: "iss-2610262039214406"
slug: "itd-85-s-abcd-audit-checks-whether-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 854 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-653350ad239b1485
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034151413. Same draft edit as iss-2610262034151413, set against a different statement of the same current state."
duplicates: [iss-2610262034151413]
---

itd-85's `abcd audit` checks whether a repository conforms, which the register already gives to `/abcd:lint`

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:26` — "doctor = tool-setup health; audit = repo conformance"
- End B: `.abcd/development/brief/04-surfaces/README.md:27` — "Check whether this repo still conforms to the working conventions"

itd-85 defines `audit` as repository conformance, meaning whether a repository follows the working conventions, and builds it on `internal/core/lint`. The surfaces register already gives the shipped `/abcd:lint` that purpose: checking whether the repo still conforms to the working conventions. Two verbs claim the same ground, and neither document draws a line between them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 854.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034151413. Same draft edit as iss-2610262034151413, set against a different statement of the same current state.
