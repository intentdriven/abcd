---
schema_version: 1
id: "iss-2610262041445723"
slug: "the-loop-glossary-entry-says-abcd-never"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1021 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-2609201925079472]
finding_ref: cfk-aec7d73c7343fbc3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037018200. Same End A. The pacing intent's run state file, lanes and next_eligible_at are abcd's own implement loop, so the edit belongs in loop.md."
duplicates: [iss-2610262037018200]
---

The loop glossary entry says abcd never owns the run loop; the shipped pacing intent describes the loop abcd's own build, drain and implement verbs run.

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."
- End B: `.abcd/development/intents/shipped/itd-2609201925079472-an-autonomous-implementation-run-paces.md:20` — "A run started with `abcd build` (or `abcd drain`, or the `abcd implement` loop they hand over to)"

The loop entry says abcd only defines the run-loop contract (iterate, gate on a receipt, guard) and that a pluggable adapter supplies the loop itself. The shipped pacing intent describes a run started with `abcd build` or `abcd drain` and handed to the `abcd implement` loop. That run opens lanes, writes next_eligible_at into its own state file, checkpoints and refuses an early restart. This is a loop the abcd binary owns and runs, with no adapter. Someone building from the glossary would treat a native loop as forbidden, which contradicts what shipped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1021.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262037018200. Same End A. The pacing intent's run state file, lanes and next_eligible_at are abcd's own implement loop, so the edit belongs in loop.md.
