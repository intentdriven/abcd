---
schema_version: 1
id: "iss-2610262033179679"
slug: "itd-67-keeps-a-version-x-y-z-override-on"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 385 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-0fb30706ef30d089
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/surface/cli/ship.go registers changelog-json, payload-dir, allow-dirty and fetch-baseline only."
---

itd-67 keeps a `--version <x.y.z>` override on ship; 04-launch says there is no version flag at all

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:56` — "`--version <x.y.z>` remains the override / major-bump escape hatch, not the norm"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:185` — "There is no version flag at all: the version is derived, never authored"

itd-67 keeps `--version <x.y.z>` as the ship verb's override and major-bump escape hatch. 04-launch says there is no version flag at all, because the version is derived and never authored, and its generated appendix lists no such flag on `abcd launch ship`. The override cannot both exist and not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 385.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/surface/cli/ship.go registers changelog-json, payload-dir, allow-dirty and fetch-baseline only.
