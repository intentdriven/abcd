---
schema_version: 1
id: "iss-2610262034345807"
slug: "the-intent-chapter-records-hand"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 506 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:598"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-059ab2ef4fb136d3
---

The intent chapter records hand verification as a separate JSON verification receipt; itd-50 rules it is a grounds entry, not a separate receipt.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:598` — "a separate JSON artifact under `.abcd/.work.local/logs/audit/verify-<ts>/receipt.json`, never merged into `## Audit Notes`"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:101` — "in the product thinker's words, not a separate receipt"

05-intent specifies the product thinker's sign-off as a verification receipt distinct from the machine verdict: a separate JSON artifact in the local logs tier, never merged into Audit Notes, with offered, accepted and rejected_wrong_criteria states. itd-50's decision 4 rules that hand verification is a grounds entry in the product thinker's words, not a separate receipt. One design keeps the sign-off in a local-tier receipt with its own states, the other in the record's append-only grounds; both cannot be what the loop delivers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 506.
