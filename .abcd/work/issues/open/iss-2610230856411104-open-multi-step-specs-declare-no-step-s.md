---
schema_version: 1
id: "iss-2610230856411104"
slug: "open-multi-step-specs-declare-no-step-s"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "lane C, the coordinator's ruling on build-lane serialisation, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open"
remedy: "Backfill a needs line on every step of each open multi-step spec, starting with the merge gate's and the model matrix's: name the earlier steps a step truly depends on, or none, so independent steps run in parallel; each backfill is the spec owner's call, recorded as a change to the spec"
---

Open specs with two or more steps declare no step's needs, so every build run of them is fully serial by default (ruling DR6b: a step with no needs line waits for every earlier step). On origin/main on 2026-10-23 only 5 of 169 specs declared any needs line. The open specs most affected are the merge gate's and the model matrix's, which are being built now. No one is adding needs lines to the open specs yet.
