---
schema_version: 1
id: "iss-2610262032542009"
slug: "the-lifeboat-licence-gate-s-allowlist-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 333 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:483"
remedy: "none (filed automatically)"
finding_ref: cfk-b95bb4d4aa6a7848
---

the lifeboat licence gate's allowlist is named .abcd/launch-allowlist.json; launch says that allowlist and release packaging are two mechanisms never given one name

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:483` — "Two mechanisms, never one name."
- End B: `.abcd/development/brief/05-internals/09-provenance-substrate.md:89` — "unless `.abcd/launch-allowlist.json` explicitly names it"

04-launch says the restrictive-licence gate is the lifeboat's, not the release artefact's, and that the gate's evaluation-input allowlist and packaging are two mechanisms, never one name. 09-provenance names that lifeboat gate's allowlist .abcd/launch-allowlist.json, beside a .abcd/launch.allow, so the lifeboat's mechanism carries launch's name, the conflation launch's rule exists to prevent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 333.
