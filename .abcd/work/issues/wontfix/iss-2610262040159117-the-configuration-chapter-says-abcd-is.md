---
schema_version: 1
id: "iss-2610262040159117"
slug: "the-configuration-chapter-says-abcd-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 919 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:809"
remedy: "none (filed automatically)"
related_intents: [itd-69]
finding_ref: cfk-139f79321f9e16d4
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. internal/core/launch/lockstep.go uses TreeDev and TreePublic as check polarities over a source checkout or a rendered payload; 03-configuration's 'no dev-to-public mirror' is about the design record living in one tree."
---

The configuration chapter says abcd is one repository with no dev-to-public mirror; itd-69's checker takes an explicit `--tree dev|public` argument

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:809` — "so the record lives in that repository's tree, and there is no dev-to-public mirror."
- End B: `.abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:62` — "with an explicit `--tree dev|public` argument (no auto-detection)"

03-configuration says the record lives in the one repository's tree and there is no dev-to-public mirror. itd-69's lockstep checker checks plugin metadata across both plugin.json locations, with an explicit `--tree dev|public` argument and a path list pinned per tree. That assumes separate dev and public trees, which the configuration model says do not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 919.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. internal/core/launch/lockstep.go uses TreeDev and TreePublic as check polarities over a source checkout or a rendered payload; 03-configuration's 'no dev-to-public mirror' is about the design record living in one tree.
