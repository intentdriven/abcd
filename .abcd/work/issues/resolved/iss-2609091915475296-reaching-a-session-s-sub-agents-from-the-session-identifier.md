---
schema_version: 1
id: "iss-2609091915475296"
slug: "reaching-a-session-s-sub-agents-from-the-session-identifier"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "fidelity audit of the capture intent"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/history.go"
resolution: "A session's whole set is reachable from the session identifier through the set verb rather than the one-record verb: history list gains --session, which is ListForSession's front door, and an empty result names the session so a mistyped id never reads as an empty store. The human listing now carries the agent and its type, and renders a known-but-untyped agent as such rather than blank, because that is a real state and not an omission. Show gains the agent, its parent, its depth and a pointer to the set, on sub-agent records only, and keeps returning one record because the record calls that behaviour deliberate. The plugin page documents the flag and the lineage fields, which is the half the JSON-only route was missing. Smoked against the real corpus: a session returns 74 records, main thread first, each sub-agent naming its type."
impact: additive
resolved_by:
  intent: "itd-2609090559376002"
  spec: "spc-2609090624222051"
---

Reaching a session's sub-agents from the session identifier works in the core and has no operator surface, so the promise is met only through a JSON field the plugin page does not document. The store gained a listing that returns every record for a session, main thread first, and it is tested. But the show verb deliberately returns the main-thread record alone, the human render of list and show prints neither the agent identifier nor the agent type, and the only caller of the session listing outside the tests is the reconstruction path. The operator's actual route is therefore to take the machine-readable listing and filter it by hand on the session field, which works because the envelope marshals the whole record, but it is not documented on the surface page's field list and it is not what the acceptance criterion describes a person doing. By this repository's own rule that a capability is not delivered until it is reachable from both front doors, this is the first gap to close: either the human render carries the lineage columns and show grows a way to ask for a session's whole set, or the surface page documents the JSON route explicitly. Records still filed under the old composite identifier stay unreachable from the real session identifier either way; repairing those is a different piece of work and is already done.

## Grounds

- pursued: we expect the set to belong on the list verb rather than the show verb because list is already the set verb and show is already the one-record verb, so the capability lands where a reader would look for it; it is shown wrong if operators keep reaching for show and finding one record
