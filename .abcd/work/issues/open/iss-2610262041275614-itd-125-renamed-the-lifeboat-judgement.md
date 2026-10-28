---
schema_version: 1
id: "iss-2610262041275614"
slug: "itd-125-renamed-the-lifeboat-judgement"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1001 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-125-the-lifeboat-verdict-says-what-it-is.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-125]
finding_ref: cfk-886be6b6b5e097d6
---

itd-125 renamed the lifeboat judgement to a review; the build sequence still says disembark runs the host-delegated audit

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-125-the-lifeboat-verdict-says-what-it-is.md:20` — "judgement over the pack, which is a review"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:156` — "runs the host-delegated audit"

itd-125 rules that the judgement over a packed lifeboat is a review, and that the verb, flag, artefact and prose all now say so, ending the one-thing-three-words problem. The build sequence's disembark milestone still says the pack runs the host-delegated audit. The brief keeps the word the intent retired for this act, and that blurs the lint / review / audit split the record draws.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1001.
