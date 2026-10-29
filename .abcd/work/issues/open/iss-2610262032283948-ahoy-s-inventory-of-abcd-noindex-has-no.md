---
schema_version: 1
id: "iss-2610262032283948"
slug: "ahoy-s-inventory-of-abcd-noindex-has-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 273 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:420"
remedy: "Add the ideas/ store (private ideate verdicts, owner-only, one folder per root-commit SHA) to the user-scope tree in 01-ahoy.md and to the mirrored inventory prose in 05-internals/03-configuration.md."
finding_ref: cfk-4a2c40fd04a21936
---

ahoy's inventory of ~/.abcd.noindex/ has no ideas store; ideate writes private verdicts to ~/.abcd.noindex/ideas/< root-sha>/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:420` — "USER SCOPE — one per machine (machine-local state only)"
- End B: `.abcd/development/brief/04-surfaces/18-ideate.md:87` — "verdict is written owner-only under `~/.abcd.noindex/ideas/<root-sha>/` and nothing else"

01-ahoy.md's user-scope tree, presented as the inventory of ~/.abcd.noindex/, has no ideas/ entry. 18-ideate.md writes every private verdict owner-only under ~/.abcd.noindex/ideas/< root-sha>/. A reader of the inventory would not know this store holds verdicts the person kept private. Either the inventory is incomplete, or the private verdict store lives elsewhere.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 273.
