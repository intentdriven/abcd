---
schema_version: 1
id: "iss-2610141647314635"
slug: "a-stale-published-abcd-binary-in-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lab lab-261014142726-9c4d759 (a hand-run of /abcd:ideate on one external write-up, 2026-10-14)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli"
remedy: "Have a stale binary (vintage behind the checkout tip it is run in) say so on every verb that writes a record, not only on a refusal, and decide staleness from the checkout it writes into."
---

A stale published abcd binary in a source checkout can answer a verb with a plausible wrong result instead of refusing: iss-2608230943088357 resolved only the refusal half (an unknown verb or flag now names the staleness), and the same binary reports 'staleness: up to date' when run outside a checkout, so the flag cannot warn a runner who resolves it from elsewhere (lab finding F-3).
