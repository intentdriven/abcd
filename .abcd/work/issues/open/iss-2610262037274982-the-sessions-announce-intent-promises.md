---
schema_version: 1
id: "iss-2610262037274982"
slug: "the-sessions-announce-intent-promises"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 724 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:22"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703]
finding_ref: cfk-cd3c36c0dbed36e9
---

The sessions-announce intent promises every session's merge waits out a release cut; the reserve brief says merges armed before the hold still land

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:22` — "A session cutting a release holds merges with a reservation everyone sees, and every session's merge step waits until the cut is done."
- End B: `.abcd/development/brief/04-surfaces/37-reserve.md:357` — "A merge armed, or queued, before the release was reserved (a pull request with auto-merge on, or an entry already in the merge queue) is not stopped by it and lands when its checks pass."

The intent's press release says a release reservation makes every session's merge step wait until the cut is done. The reserve chapter says the hold only stops a merge from being armed. A pull request with auto-merge already armed, or already in the merge queue, when the release is reserved is not stopped and lands when its checks pass. Someone building from the intent would expect no merge to land during a cut, but the brief says some can.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 724.
