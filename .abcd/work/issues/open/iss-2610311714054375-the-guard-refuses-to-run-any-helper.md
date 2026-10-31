---
schema_version: 1
id: "iss-2610311714054375"
slug: "the-guard-refuses-to-run-any-helper"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "retrying the product thinker's selected builds"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/abcd.sh"
remedy: "The layer past the depth limit is the two string-action EXIT traps in the abcd.sh launcher script, each a shell payload the guard counts in its payload reader. Either flatten them into explicit cleanup on each exit path (cost: an interrupted build leaves a per-pid binary behind in the local tier), or stop charging a depth layer for a trap payload that tokenizes as one plain command the guard reads and judges in full. Either way the guard keeps reading and judging every command in the script, and nothing is trusted by its path."
---

The guard refuses to run any helper script that calls scripts/abcd.sh, as execute-string-uninspectable ('Execute-a-string wrappers are nested past the depth the guard can follow'), because abcd.sh itself nests a shell layer; bash -c around the same loop is refused the same way. On 2026-10-31 a three-line retry script under .abcd/.work.local/scratch/ was refused, though the SCRATCH rules domain tells an agent to put such helpers exactly there, so in this repository an agent can drive abcd only from an inline loop typed into each tool call.
