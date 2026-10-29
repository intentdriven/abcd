---
schema_version: 1
id: "iss-2610262037262122"
slug: "itd-59-says-the-autonomous-worker-leaves"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 723 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19"
remedy: "Re-scope itd-59 to the unattended passes not yet routed through capture: the runner already stores each delegated role's transcript through history.Capture (internal/core/runner/transcript.go), as the build chapter says; state what the draft adds beyond that, and name the planned loop intent as owner of the loop's own capture, or retire the draft."
related_intents: [itd-59]
finding_ref: cfk-fb6d9b2b499ad6f8
---

itd-59 says the autonomous worker leaves only an ephemeral, pruned log; the build brief says a runner's transcript lands in the history store

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19` — "emits its instructions and reasoning only to an ephemeral per-pass log that run-dir hygiene prunes"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:499` — "the runner's transcript lands in abcd's history store, keyed on the repository's root commit"

itd-59 aims to close a gap. Its premise is that the unattended worker writes its instructions and reasoning only to a per-pass log that run-dir hygiene prunes. The build chapter says the transcript of a runner the step verb starts lands in abcd's history store, keyed on the repository's root commit, and that the record verb stores the run's transcripts there at the end. If the build chapter is right, the gap itd-59 describes is already closed for the loop that ships.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 723.
