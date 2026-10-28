---
schema_version: 1
id: "iss-2610241431221889"
slug: "the-after-merge-fidelity-audit-of-itd-24"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-24 (receipt rcp-2215987d5eb7)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md"
remedy: "fix, then re-run the audit: make ac-2 NOT_MET, ac-7 NOT_MET of itd-24 hold in the delivered code, then re-emit the request with `abcd intent audit itd-24` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-24]
refines: [iss-2610241427397332, iss-2610241426406882, iss-2608290822140563]
resolution: "reflect's seed now treats an OWED review marker as not audited and lists the intents the cut moved to next"
impact: fix
resolved_by:
  commit: "c8ddafcaa"
---

The after-merge fidelity audit of itd-24 judged a criterion unmet, so its check is still owed: ac-2 NOT_MET, ac-7 NOT_MET

The audit of itd-24 (receipt rcp-2215987d5eb7) judged ac-2 NOT_MET, ac-7 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-24` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
