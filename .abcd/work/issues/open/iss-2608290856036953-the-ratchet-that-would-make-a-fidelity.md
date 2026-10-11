---
schema_version: 1
id: "iss-2608290856036953"
slug: "the-ratchet-that-would-make-a-fidelity"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "role-clarification-run"
found_at: "internal/core/intent/audit.go"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: the tracked-verdict intent this waits on (itd-165) is still in drafts, so no verdict corpus exists to baseline, and whether the ratchet gates at audit time or at merge is unsettled. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
remedy: "Waits on itd-165 shipping and the audit-position ruling (audit time or merge): once one audited intent exists, record failed acceptance criteria in the one shared shrink-only baseline of ruling BT3, so a new NOT_MET fails and a fixed one must shrink the file; if it gates at audit time, the ratchet fails intent audit ingest on the closing lane; if at merge, it fails the check of the change carrying the verdict, never a later one; either proven by a test that a new failed criterion fails and a cleared one demands a shrink."
---

The ratchet that would make a fidelity verdict binding waits on a corpus of real verdicts, because there are none. A baseline ratchet is how this repository already converts an advisory signal into a red gate: today's failures are baselined, a new one fails, and a fixed one invites a shrink. Applied to the intent audit it would make a newly failed acceptance criterion fail a gate rather than sit in prose. It is deliberately split out of the intent that gives a failed verdict a tracked record, and sequenced behind it, because a ratchet baselines whatever number it finds and no intent has been audited even once: seeding it now would baseline a number nobody has looked at. It also needs the audit's position relative to the merge settled first, since a ratchet on a post-merge verdict fails the next unrelated change rather than the one that caused the failure, and it needs its blocking behaviour declared for both facilitator modes.

## Remedy grounds (2026-09-29)

Why: ruling BT3 (2026-09-29) already chose one shared shrink-only baseline for warn rules, so the verdict ratchet reuses it instead of adding a second mechanism. SOTA check: betterer keeps a committed results file, fails when a result gets worse, updates it when better, and in CI mode fails on any difference from the committed file (https://phenomnomnominal.github.io/betterer/docs/running-betterer, read 2026-09-29). Rejected: seeding the baseline now, which would baseline a number nobody has read, as the record says.
