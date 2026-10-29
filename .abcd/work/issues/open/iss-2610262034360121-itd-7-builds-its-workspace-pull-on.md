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
remedy: "In itd-7 drop the claim that dev-sync is 'already in brief' (Pack bullet) and state it as a prerequisite owed by the draft itd-13, as the waiting note at the top already does."
related_intents: [itd-7]
finding_ref: cfk-994db1cfb7b3e6cc
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-7 builds its workspace pull on dev-sync, a pre-pack step it says is already in the brief; the board chapter says no dev-sync verb or code exists anywhere.

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38` — "dev-sync runs as a pre-pack step (already in brief)"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:583` — "no `dev-sync` verb is on any shipped surface, and no dev-sync code exists in the tree"

itd-7's pull runs on dev-sync, its schema is validated by abcd dev-sync, and its pack relies on dev-sync running as a pre-pack step that it calls already in the brief. 08-abcd states that the dev-sync work migration surface is a design target, that no dev-sync verb is on any shipped surface and no dev-sync code exists in the tree, and that no store records when it last ran. The planned intent depends on a surface no record here delivers, and its own scope does not build it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 508.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
