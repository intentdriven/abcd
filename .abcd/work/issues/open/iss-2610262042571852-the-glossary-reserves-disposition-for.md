---
schema_version: 1
id: "iss-2610262042571852"
slug: "the-glossary-reserves-disposition-for"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1105 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-82]
finding_ref: cfk-6605e36ade33f075
---

The glossary reserves 'disposition' for the researcher's response to a reading item; itd-82 uses it for a drain run's per-issue outcome.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:4` — "The researcher's recorded response to one reading item, written as a separate record keyed to the item"
- End B: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:52` — "every issue's disposition (pull request number, draft id, flag with home or reason"

In the ledger bounded context a disposition is 'the researcher's recorded response to one reading item, written as a separate record keyed to the item', with four named states, and the glossary forbids mixing it with issue-ledger vocabulary. itd-82's drain summary records 'every issue's disposition' as a pull request number, a draft id, a flag, ineligible, skipped or stopped. The same word now names two unrelated things that both live on the issue ledger side (reading dispositions are filed under the issues tree), so a reader of a drain summary or a ledger query cannot tell which sense a 'disposition' carries.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1105.
