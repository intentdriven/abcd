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
remedy: "Let the guard read scripts/abcd.sh as a known launcher (the repository's own wrapper for its built binary, judged as the abcd verb it runs) so one script layer above it stays inspectable, or flatten abcd.sh's inner shell layer so the nesting depth falls within what the guard follows."
---

The guard refuses to run any helper script that calls scripts/abcd.sh, as execute-string-uninspectable ('Execute-a-string wrappers are nested past the depth the guard can follow'), because abcd.sh itself nests a shell layer; bash -c around the same loop is refused the same way. On 2026-10-31 a three-line retry script under .abcd/.work.local/scratch/ was refused, though the SCRATCH rules domain tells an agent to put such helpers exactly there, so in this repository an agent can drive abcd only from an inline loop typed into each tool call.
