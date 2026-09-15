---
schema_version: 1
id: "iss-2608290856036953"
slug: "the-ratchet-that-would-make-a-fidelity-verdict-binding-waits"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "role-clarification-run"
found_at: "internal/core/intent/audit.go"
---

The ratchet that would make a fidelity verdict binding waits on a corpus of real verdicts, because there are none. A baseline ratchet is how this repository already converts an advisory signal into a red gate: today's failures are baselined, a new one fails, and a fixed one invites a shrink. Applied to the intent audit it would make a newly failed acceptance criterion fail a gate rather than sit in prose. It is deliberately split out of the intent that gives a failed verdict a tracked record, and sequenced behind it, because a ratchet baselines whatever number it finds and no intent has been audited even once: seeding it now would baseline a number nobody has looked at. It also needs the audit's position relative to the merge settled first, since a ratchet on a post-merge verdict fails the next unrelated change rather than the one that caused the failure, and it needs its blocking behaviour declared for both facilitator modes.