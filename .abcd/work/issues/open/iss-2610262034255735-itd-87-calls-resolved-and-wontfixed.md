---
schema_version: 1
id: "iss-2610262034255735"
slug: "itd-87-calls-resolved-and-wontfixed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 494 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:17"
remedy: "In itd-87 replace 'dispositioned' with 'closed (resolved or wontfixed)' throughout; the glossary entry for disposition reserves the word for a researcher's answer to a reading item and says not to reuse the issue ledger's open, resolved and wontfix."
related_intents: [itd-87]
finding_ref: cfk-74489cbb4b5791eb
---

itd-87 calls resolved and wontfixed issues 'dispositioned'; the capture chapter uses 'disposition' only for a researcher's answer to a reading item.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:17` — "matches one already dispositioned — resolved or wontfixed"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:211` — "**A disposition** records the researcher's answer to one reading"

In itd-87 a dispositioned entry is an issue closed by resolve or wontfix. In 06-capture.md a disposition is a separate record keyed to one reading item, with states such as accepted, declined and held, filed in the same ledger. One ledger term has two meanings, so 'matching against dispositioned entries' could also be read as matching against reading dispositions.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 494.
