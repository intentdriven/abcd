---
schema_version: 1
id: "iss-2610040758579292"
slug: "an-intent-s-blocked-by-edges-are-read-by-the-build-gate"
severity: "minor"
category: "ux"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/startcheck.go"
remedy: "Add an intent verb that adds and removes blocked_by and builds_on edges, mirroring capture link and refusing an id no intent store holds; test: linking two fixture intents writes the edge, an unknown id is refused, and the build gate then refuses the blocked intent."
resolution: "Added abcd intent edge <itd-N> (--blocked-by/--unblock/--builds-on/--drop-builds-on), mirroring capture link: writes an intent's blocked_by and builds_on in place under the store lock, refusing an id no intent store holds, a self-edge, and removal of an absent edge, with nothing written."
impact: additive
resolved_by:
  commit: "8110adb91e44bee4538becbc8b355a4eb21ab9e8"
---

An intent's blocked_by edges are read by the build gate (internal/core/intent/startcheck.go, startBlockedRow, and the implement loop's check) but no verb writes them: abcd capture link adds and removes blocked_by on an issue, and abcd intent has no counterpart (its link sub-verb links a spec), so a downstream project hand-typed every edge between its intents.

## Grounds

- pursued: a person can now link two intents with one command and the build gate refuses the blocked one; shown wrong if an edge written by the verb is not read by startBlockedRow or record-lint refuses a record the verb wrote
