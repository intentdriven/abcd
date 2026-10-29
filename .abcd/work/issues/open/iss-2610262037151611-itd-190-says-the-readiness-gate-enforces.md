---
schema_version: 1
id: "iss-2610262037151611"
slug: "itd-190-says-the-readiness-gate-enforces"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 710 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:39"
remedy: "In itd-190 state the shipped staging: the mechanism and scope-conditions rows are advisory at the readiness gate (reported, never gating) and bind at a build run (34-build.md); reword 'the gate exits non-zero' to match."
related_intents: [itd-190]
finding_ref: cfk-4dda363b4287b78c
---

itd-190 says the readiness gate enforces the claim-recording rules; 34-build says the readiness gate only reports them as advisory and enforcement happens at a run.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:39` — "the gradient binds standalone intents at the readiness gate from adoption on"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:65` — "The readiness gate reports both as advisory; a run is where they bind"

itd-190's rule says the gradient binds standalone intents at the readiness gate, and that a blank scope-conditions section or an empty mechanism section is a gate fault. 34-build says the readiness gate reports the mechanism prompt and the scope conditions as advisory, and that they bind only when a build starts. Either the gate refuses, or it only advises; the two pages say opposite things.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 710.
