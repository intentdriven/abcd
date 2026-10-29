---
schema_version: 1
id: "iss-2610262037301480"
slug: "the-build-intent-says-each-invocation"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 728 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:21"
remedy: "In the build intent say each invocation performs one move (one stage of a lane) instead of one step, per BU1 (iss-2609291313276243): a spec's steps and a lane's stages are two words; the host-facing verb name implement step stays."
related_intents: [itd-2609201916151817]
finding_ref: cfk-6bd3c576f4651693
---

The build intent says each invocation does one step; the build brief keeps 'step' for a spec's steps and calls one invocation's work a stage

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:21` — "every invocation reads the state, does one step, writes the state and exits"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:388` — "A spec's steps and a lane's stages are two words for two things"

The build intent says every invocation does one step and exits. Under BU1, the build chapter keeps 'step' for a spec's steps only, and what one invocation performs is a lane's stage, one move of the run. Read with the brief's vocabulary, the intent says each invocation lands a whole spec step, which the loop does not do.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 728.
