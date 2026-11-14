---
schema_version: 1
id: "iss-2611060810148228"
slug: "abcd-agents-is-new-since-v0-16-1-and"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Have the delivering change carry a record; name the verb in the next cut's changelog."
refines: [iss-2611040555455335, iss-2611040555499163, iss-2610080620372731]
resolution: "abcd agents lists the agents abcd started and adds them up, from the launch ledger on this computer (~/.abcd.noindex/agents/): one line per launch naming the session or run that asked for it, the work, the billing route, the model, the tokens, the cost, the API time and how it ended. Bare, it lists this session's launches; --run <id> lists one run's, --session <id> one session's and --all every one, in text or --json. Money is counted for launches billed to a named key alone: a subscription's or an unplaced launch's cost is shown as an estimate, which is no money spent, and every cost is the harness's own estimate. It reads only and writes nothing. With nothing on record it says so; it exits 2 where launches are on record and no session or scope says whose, or where two scopes are given at once. The verb shipped in v0.16.2 with no changelog line (part of the work of itd-2611031342011261, still planned), so this record names it in the next cut."
impact: additive
resolved_by:
  commit: "c54599f7d463802042eba776ca456373e77e6938"
---

`abcd agents` is new since v0.16.1 and read-only, with no line in the v0.16.2 changelog, because no record in the cut covers it.

## Grounds

- pursued: the next cut's Added section names abcd agents from this record; a cut whose changelog does not name the verb would show it wrong
