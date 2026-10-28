---
schema_version: 1
id: "iss-2610262035065340"
slug: "itd-80-names-the-verdict-ingest-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 547 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-436e6f36daa44e9c
---

itd-80 names the verdict-ingest verb abcd intent review ingest; the intent chapter's command tree names it abcd intent audit ingest

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:32` — "`abcd intent review ingest` ingests a verdict"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:733` — "Sub-verbs: `abcd intent audit ingest`."

itd-80 says abcd intent review ingest ingests a verdict. The chapter's generated appendix lists the only ingest under abcd intent audit as abcd intent audit ingest, and no review sub-verb exists. The chapter also keeps the word review for change-judgement verbs, apart from the audit verb that writes these verdicts. So one operation carries two verb names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 547.
