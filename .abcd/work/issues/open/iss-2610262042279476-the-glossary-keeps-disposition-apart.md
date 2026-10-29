---
schema_version: 1
id: "iss-2610262042279476"
slug: "the-glossary-keeps-disposition-apart"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1070 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:25"
remedy: "In itd-87 line 31 replace 'dispositioned entries (resolve, wontfix)' with 'closed entries (resolved or wontfix)'."
related_intents: [itd-87]
finding_ref: cfk-736424e371fdf8bf
---

The glossary keeps disposition apart from the issue ledger's resolved and wontfix; itd-87 calls resolved and wontfixed entries dispositioned

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:25` — "Do not reuse the issue ledger's open, resolved and wontfix."
- End B: `.abcd/development/intents/drafts/itd-87-recurrence-escalation-in-capture.md:31` — "Matching an incoming capture against **dispositioned** entries (`resolve`, `wontfix`), not merely against open ones."

The disposition entry says a disposition is not a resolution and must not reuse the issue ledger's open, resolved and wontfix. itd-87 uses dispositioned to mean exactly an issue resolved or wontfixed, so the term carries two senses across the record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1070.
