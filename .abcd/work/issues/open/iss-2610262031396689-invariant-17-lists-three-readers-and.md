---
schema_version: 1
id: "iss-2610262031396689"
slug: "invariant-17-lists-three-readers-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 158 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-195-a-claim-about-how-the-code-behaves-is.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-195]
finding_ref: cfk-d7bcc3479e0374b8
---

Invariant 17 lists three readers and what each does with no test behind it, the shape itd-195 forbids

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-195-a-claim-about-how-the-code-behaves-is.md:23` — "The forbidden shape is a list: "these are the readers and this is what each"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "the three readers that assumed one spec per intent now ask the 1:n question"

itd-195 names a list of the readers and what each does as the forbidden shape: such a claim is backed by a test that runs, or it is not written. Invariant 17 says the three readers that assumed one spec per intent now ask the 1:n question and then states what each of the three does, citing no test that establishes it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 158.
