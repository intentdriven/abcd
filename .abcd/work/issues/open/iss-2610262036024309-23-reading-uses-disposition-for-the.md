---
schema_version: 1
id: "iss-2610262036024309"
slug: "23-reading-uses-disposition-for-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 619 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md:319"
remedy: "none (filed automatically)"
related_intents: [itd-2610141956319695]
finding_ref: cfk-6d1c5a87f82f4824
---

23-reading uses disposition for the researcher's recorded answer to a reading item; the deferred-finding discipline uses it for a review finding's fixed/invalid/deferred field in a release receipt.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/23-reading.md:319` — "the confirmed form of a recurrence stays the disposition's `recurs` citation"
- End B: `.abcd/development/intents/disciplines/itd-2610141956319695-a-deferred-finding-is-its-own-captured.md:36` — "`disposition`, spelled exactly, and a deferred finding carries a separate field"

In 23-reading a disposition is the ledger record answering a reading item: the confirmed form of a recurrence stays the disposition's recurs citation, and the comparative position selects items carrying no disposition and no admission. itd-2610141956319695 makes disposition the field of a release-receipt finding whose three words are fixed, invalid and deferred, with a separate field naming the deferred finding's record. One word carries two state vocabularies on two different objects, so the term alone no longer says which record or which state set is meant.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 619.
