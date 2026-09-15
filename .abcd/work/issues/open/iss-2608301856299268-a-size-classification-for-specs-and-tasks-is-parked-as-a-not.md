---
schema_version: 1
id: "iss-2608301856299268"
slug: "a-size-classification-for-specs-and-tasks-is-parked-as-a-not"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "phase-boundary-parking"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes"
---

a size classification for specs and tasks is parked as a note and owes a design and filing decision in a future session

Parked on the maintainer's instruction and armed here so it is revisited, on the
same reasoning as iss-2608301744251874: a note carrying a trigger line is read
only by whoever happens to open it.

The proposal and its evidence are in
`.abcd/development/research/notes/2026-08-30-spec-size-classification-gap.md`.

The gap: nothing lets a spec or task be called S, M, L, XL or XXL before it is
built, so nothing can be budgeted or compared against what it became. `itd-29`
assumes the gap away with a flat per-task constant, and cycle 1's measurements
say no such constant exists -- the same corpus holds a five-commit intent and a
fifty-commit one.

The finding that should shape the scheme, and the reason this is a design task
rather than a table: **depth does not track volume.** itd-177 touched 97 files
and converged in three review rounds; itd-188 touched 16 and took four; itd-181
and itd-182 took none at all. What the deep intents share is the number of ways
they can be quietly wrong, not their size.

Two constraints bind any answer. The roadmap forbids time estimates, so the
classes cannot be denominated in days. And the scheme must be checkable against
actuals or it becomes another unfalsifiable constant -- which makes it a pair
with iss-2608301744251874, the token metering park: one predicts, the other
measures, and neither is worth much alone.

**Trigger: a future session, ideally the one that takes the metering park.**
Two legal closes, and record which:

- designed and filed, closing this with the minted record id;
- rejected, becoming a wontfix carrying the grounds, so a later reader finds
  the decision rather than the silence.

Closing it by concluding the note is enough is not one of them.

