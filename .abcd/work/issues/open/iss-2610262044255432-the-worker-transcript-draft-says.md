---
schema_version: 1
id: "iss-2610262044255432"
slug: "the-worker-transcript-draft-says"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1205 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-59, itd-2609201916056194]
finding_ref: cfk-bcc2e26363bcde84
---

The worker-transcript draft says autonomous passes leave only an ephemeral log, but the shipped runner captures every role's transcript into abcd's store

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-59-autonomous-worker-transcript-capture.md:19` — "emits its instructions and reasoning only to an ephemeral per-pass log that run-dir hygiene prunes"
- End B: `.abcd/development/intents/shipped/itd-2609201916056194-abcd-runs-a-delegated-agent-through-a.md:43` — "the transcript captured into abcd's own store, whatever the harness keeps of its own"

itd-59's premise is that the autonomous worker writes its instructions and reasoning only to a per-pass log that hygiene prunes. itd-2609201916056194 states that a role run through a runner has its transcript captured into abcd's own store, as the host's sub-agent does. The gap the draft sets out to close is described as already closed.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1205.
