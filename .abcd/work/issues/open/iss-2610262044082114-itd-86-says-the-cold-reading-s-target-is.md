---
schema_version: 1
id: "iss-2610262044082114"
slug: "itd-86-says-the-cold-reading-s-target-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1185 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-86-cold-reading-surface.md:41"
remedy: "In itd-86 replace the design-document target and the 'reviewing code' out-of-scope bullet with itd-184's Detection object (the shipped tree against the claim record), or supersede the draft by itd-184."
related_intents: [itd-86, itd-184]
finding_ref: cfk-19bb3b7a364b5066
---

itd-86 says the cold reading's target is a design document and not code; itd-184 makes the detection reading the shipped tree against the claim record

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:41` — "The target is a design document. `ruthless-reviewer` owns diffs."
- End B: `.abcd/development/intents/shipped/itd-184-four-cold-reading-definitions-one.md:35` — "Shipped tree against the claim record"

itd-86 defines the cold reading's detection as reading one design document against the constraints visible in it, and puts reviewing code out of scope. itd-184's Detection definition reads the shipped tree, which includes source and tests, against the claim record. The two disagree on what the detection reading's object is.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1185.
