---
schema_version: 1
id: "iss-2610262045327872"
slug: "itd-34-says-shape-classification-runs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1281 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90"
remedy: "none (filed automatically)"
related_intents: [itd-34, itd-48]
finding_ref: cfk-19ce938c874fd464
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. No intent shape sub-verb or pre-commit hook exists in the CLI, and itd-34's banner narrows the record to the bundle command and reclassify; itd-48 says on-demand only with the continuous leg deferred."
---

itd-34 says shape classification runs continuously in the pre-commit hook; itd-48 defers continuous pre-commit shape scanning

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90` — "It runs continuously via the pre-commit hook (writing findings to the latest report)"
- End B: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:92` — "Continuous pre-commit shape scanning is **deferred** to a"

itd-34 gives the reviewer's shape-classification role a continuous pre-commit run writing to the latest report, beside the on-demand verb. itd-48, which ships the same role as Role 3, defers continuous pre-commit scanning to a follow-up and calls that surface explicitly out of scope; its decision 1 also hands the shape role back to itd-34. Whether a commit runs the shape pass cannot be both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1281.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. No intent shape sub-verb or pre-commit hook exists in the CLI, and itd-34's banner narrows the record to the bundle command and reclassify; itd-48 says on-demand only with the continuous leg deferred.
