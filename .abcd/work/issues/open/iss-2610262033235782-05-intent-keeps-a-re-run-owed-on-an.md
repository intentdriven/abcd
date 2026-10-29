---
schema_version: 1
id: "iss-2610262033235782"
slug: "05-intent-keeps-a-re-run-owed-on-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 398 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:327"
remedy: "Edit 08-abcd.md line 100 so an ingested review whose verdict left a check owed reports the owed check and its re-emit; make shippedReviewMove in internal/core/record/record.go name it (ReviewEntry already carries AuditOwed and the re-emit)."
finding_ref: cfk-54033ef1764f0f97
---

05-intent keeps a re-run owed on an ingested review that left a check owed, but 08-abcd says an ingested review leaves nothing to do.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:327` — "An ingested review whose verdict left a check owed (ruling DQ1c) is listed under its own heading as reviewed and flagged"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:100` — "is reported unreviewed with its reason; an ingested one leaves nothing to do."

05-intent treats an ingested review whose verdict left a check owed (ruling DQ1c) as a separate case. It is listed as reviewed and flagged, it carries a remedy and a re-emit that rewrites its request for the re-run, and the same reader supplies the record dispatcher's next move for a shipped intent. 08-abcd's account of `abcd <itd-N>` for a shipped intent says an ingested review leaves nothing to do. That drops the re-run still owed after a NOT_MET or INCONCLUSIVE verdict.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 398.
