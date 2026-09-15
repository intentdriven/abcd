---
schema_version: 1
id: "iss-2609012028144617"
slug: "in-a-source-checkout-of-abcd-itself-the-plugin-root-binary-a"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
resolution: "The convention is in the committed record. AGENTS.md states it under Build, test, and checks: in a source checkout of abcd every abcd invocation is go run ./cmd/abcd from the repo root, with the mechanism (the plugin-root binary and a PATH abcd are the last published version, stale by construction here), the silent shapes it produced, and why the documented resolution ladder never reaches the go run rung. A DOGFOODING domain in .abcd/rules.json carries the same rule with recall keywords capture, launch, changelog, ahoy, version, ship and release; abcd rules DOGFOODING renders it, hook prompt-router injects it on a prompt naming those verbs and not on one that does not, and docs lint reports zero blockers. The command pages' resolution order stays with iss-2608230943088357."
impact: internal
---

In a source checkout of abcd itself, the plugin-root binary and any abcd on PATH are the last published version, so they are stale by construction and fall further behind with every commit, and the failure is not always a refusal. Three hits in one session while cutting v0.7.0: launch ship refused on its surface guard, correctly and loudly; changelog --json returned an empty cut against a tree with 181 shipped records, no error, a plausible wrong answer briefly read as evidence about the release; and capture would have written records through a schema that predates the origin and production_mode disclosure pair the record gates now require. The plugin skill pages document a resolution ladder that reaches the plugin binary first and falls through to go run ./cmd/abcd only when nothing earlier resolves, and in this checkout the first rung exists and answers, so the fallback written for exactly this case is unreachable by following the ladder literally. The convention every agent in this repository needs is that every abcd invocation here is go run ./cmd/abcd from the repo root; it belongs in the committed record, in AGENTS.md and as a loader domain that fires on prompts naming an abcd verb, rather than in a per-machine handover note. Carried from the session handover into autonomous-run-2026-09-01; the skill pages' resolution order is the other half and stays with iss-2608230943088357.

## Grounds

- pursued: a correction every agent in this repository needs belongs in the record and the loader rather than in a per-machine handover note; if the stale-binary shapes recur in a session that read AGENTS.md or received the DOGFOODING injection, the rule's wording or its recall keywords are the defect
