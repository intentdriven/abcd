---
schema_version: 1
id: "iss-2610241427397332"
slug: "the-after-merge-fidelity-audit-of-itd-3"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-3 (receipt rcp-a2e0ab1dca95)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-3-modular-rules-loader.md"
remedy: "fix, then re-run the audit: make ac-5 NOT_MET, ac-6 NOT_MET of itd-3 hold in the delivered code, then re-emit the request with `abcd intent audit itd-3` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-3]
refines: [iss-2610241426406882, iss-2608290822140563, iss-2608290820473197]
wontfix_reason: "The product thinker ruled on 27 October 2026 to amend itd-3's ac-5 and ac-6 to what shipped (recorded in itd-3's Audit Notes): ac-5's line count is obsolete since conventions moved to AGENTS.md, and ac-6's harvest domains were dropped when the defaults shipped. Nothing is owed."
---

The after-merge fidelity audit of itd-3 judged a criterion unmet, so its check is still owed: ac-5 NOT_MET, ac-6 NOT_MET

The audit of itd-3 (receipt rcp-a2e0ab1dca95) judged ac-5 NOT_MET, ac-6 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-3` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.

## Grounds

- declined: The product thinker ruled on 27 October 2026 to amend itd-3's ac-5 and ac-6 to what shipped (recorded in itd-3's Audit Notes): ac-5's line count is obsolete since conventions moved to AGENTS.md, and ac-6's harvest domains were dropped when the defaults shipped. Nothing is owed.
