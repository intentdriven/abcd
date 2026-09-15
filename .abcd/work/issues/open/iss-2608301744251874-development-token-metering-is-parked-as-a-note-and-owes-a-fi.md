---
schema_version: 1
id: "iss-2608301744251874"
slug: "development-token-metering-is-parked-as-a-note-and-owes-a-fi"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "phase-boundary-parking"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/research/notes"
---

development token metering is parked as a note and owes a file or reject decision at the next phase planning

Parked on the maintainer's instruction, and armed here on their instruction to
ensure it is revisited. The proposal, its evidence and its hand-run routing are
in `.abcd/development/research/notes/2026-08-30-development-token-metering-gap.md`.

The gap in one line: `itd-29` ESTIMATES token cost forward from a hardcoded
per-task constant and nothing ever measures the actuals, so the constant has no
ground truth and can be wrong indefinitely; `itd-98` measures token cost only
inside one experiment's comparison report. Nothing records what a development
act actually costs.

It is not filed as an intent because it is a general abcd capability rather than
cold-reading work, and this cycle's standing authorisation stops at records
outside the workstream's own filings. The filing decision is the maintainer's.

**Trigger: the next phase's planning.** Two legal closes, and record which:

- filed, in which case this closes citing the minted intent id;
- rejected, in which case this becomes a wontfix carrying the grounds, so a
  later reader finds the decision rather than the silence.

Do not close it by concluding the note is enough. A note with a trigger line in
it is precisely what this record exists to compensate for: the same failure mode
as iss-2608301726130926, where an obligation lived in a handover line and in a
verdict's prose and would have expired unread. That one was armed the same day,
for the same reason, which makes this the second instance of one lesson rather
than a one-off.

