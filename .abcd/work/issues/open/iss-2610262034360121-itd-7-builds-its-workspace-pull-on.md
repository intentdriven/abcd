---
schema_version: 1
id: "iss-2610262034360121"
slug: "itd-7-builds-its-workspace-pull-on"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 508 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-7]
finding_ref: cfk-994db1cfb7b3e6cc
---

itd-7 builds its workspace pull on dev-sync, a pre-pack step it says is already in the brief; the board chapter says no dev-sync verb or code exists anywhere.

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38` — "dev-sync runs as a pre-pack step (already in brief)"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:583` — "no `dev-sync` verb is on any shipped surface, and no dev-sync code exists in the tree"

itd-7's pull runs on dev-sync, its schema is validated by abcd dev-sync, and its pack relies on dev-sync running as a pre-pack step that it calls already in the brief. 08-abcd states that the dev-sync work migration surface is a design target, that no dev-sync verb is on any shipped surface and no dev-sync code exists in the tree, and that no store records when it last ran. The planned intent depends on a surface no record here delivers, and its own scope does not build it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 508.
