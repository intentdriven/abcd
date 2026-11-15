---
schema_version: 1
id: "iss-2611151551240397"
slug: "the-after-merge-fidelity-audit-of"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-2609091014076309 (receipt rcp-01d317e17525)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md"
remedy: "fix, then re-run the audit: make ac-7 NOT_MET of itd-2609091014076309 hold in the delivered code, then re-emit the request with `abcd intent audit itd-2609091014076309` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-2609091014076309]
refines: [iss-2610241431221889, iss-2610241551259104, iss-2610241613191748]
---

The after-merge fidelity audit of itd-2609091014076309 judged a criterion unmet, so its check is still owed: ac-7 NOT_MET

The audit of itd-2609091014076309 (receipt rcp-01d317e17525) judged ac-7 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-2609091014076309` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
