---
schema_version: 1
id: "iss-2610262033250826"
slug: "06-capture-says-the-filing-time-match"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 402 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:123"
remedy: "In 06-capture.md lines 123-130 list the audit ingest's owed-check filer among the writers that run the filing-time match, saying what it compares (the check's own title) and leaves out; reconcile 'two unattended writers' with the reading ingest named in the next sentence."
finding_ref: cfk-678f6b912630b161
---

06-capture says the filing-time match runs on the ledger's two unattended writers, but 05-intent's audit ingest is a third writer that files through the match.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:123` — "The same match runs on the ledger's two unattended writers, through the same"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:330` — "related to the intent, with the filing-time match"

06-capture names exactly two unattended writers that run the match (promoted inbox reports and the consistency pass) and says what each must leave out of the comparison. 05-intent says a failed or undecided audit ingest captures an issue through the ledger's filer, related to the intent, with the filing-time match. 06-capture's list omits that writer, and with it any rule on which of its fixed lines stay out of the comparison.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 402.
