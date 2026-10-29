---
schema_version: 1
id: "iss-2610262040379523"
slug: "itd-25-calls-the-intent-side-back-edge"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 944 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-25]
finding_ref: cfk-8e4663e7aa4937b1
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034017438. Same draft edit as iss-2610262034017438, set against a different statement of the same current state."
duplicates: [iss-2610262034017438]
---

itd-25 calls the intent-side back-edge to promoted issues resolves_issues; the verification matrix calls it related_issues.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:44` — "the intent's frontmatter gains `resolves_issues: [iss-N, iss-NM, iss-NO]` reciprocally"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:68` — "which writes both halves (`related_issues` on the intent, `related_intents` on the record)"

itd-25's N:1 promotion adds resolves_issues to the intent's frontmatter as the reciprocal of related_intents on the issues. The matrix's capture-promote and back-link rows name the shipped intent-side field related_issues. That gives two field names for one join, and itd-25's lint extension would check a field the ledger does not write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 944.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034017438. Same draft edit as iss-2610262034017438, set against a different statement of the same current state.
