---
schema_version: 1
id: "iss-2610262033006141"
slug: "the-glossary-keeps-disposition-apart"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 347 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/disposition.md:25"
remedy: "none (filed automatically)"
finding_ref: cfk-2031e21b3947f005
---

The glossary keeps 'disposition' apart from 'fixed'; launch uses fixed, invalid and deferred as the dispositions of a release-gate finding

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/disposition.md:25` — "Do not call a disposition a resolution: resolved means fixed"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:151` — "disposition every finding as fixed, invalid or deferred"

The glossary limits disposition to answering a reading item, in four states (accepted, rejected, declined, held), or to a scope condition, and it warns against calling a disposition a resolution because 'resolved means fixed'. The launch receipts protocol has every finding dispositioned as fixed, invalid or deferred. That is a third vocabulary for the term, and it includes the one value the glossary sets against it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 347.
