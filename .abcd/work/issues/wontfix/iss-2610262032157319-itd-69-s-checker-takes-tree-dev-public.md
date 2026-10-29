---
schema_version: 1
id: "iss-2610262032157319"
slug: "itd-69-s-checker-takes-tree-dev-public"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 243 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:62"
remedy: "none (filed automatically)"
related_intents: [itd-69]
finding_ref: cfk-f452400bca5d64eb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. In internal/core/launch/lockstep.go dev requires the version keys absent in the source tree and public requires them present and agreeing, in the rendered payload or any installed copy; no dev-to-public mirror is implied."
---

itd-69's checker takes `--tree dev|public`; the platform page says there is no dev→public mirror

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:62` — "with an explicit `--tree dev|public` argument (no auto-detection)"
- End B: `.abcd/development/brief/02-constraints/01-platform.md:7` — "ships a **curated release artifact** cut from it — there is no dev→public mirror"

itd-69's lockstep checker takes an explicit `--tree dev|public` argument, which assumes there is a development tree and a separate public tree. 01-platform says abcd ships one curated release artifact cut from one repository and that there is no dev→public mirror. Under that constraint, the public tree the checker selects does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 243.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. In internal/core/launch/lockstep.go dev requires the version keys absent in the source tree and public requires them present and agreeing, in the rendered payload or any installed copy; no dev-to-public mirror is implied.
