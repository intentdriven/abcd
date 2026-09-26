---
schema_version: 1
id: "iss-2609091911066372"
slug: "an-exported-owner-resolution-helper-has-no-callers-and-its-l"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "fidelity audit of the recovery intent"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/ingest.go"
resolution: "Session ownership has one definition, ownerIndex.owner, called by ingest's placement pass; the uncalled exported SessionOwner is removed."
impact: internal
resolved_by:
  commit: "57acad25"
---

An exported owner-resolution helper has no callers and its logic is duplicated inline at the one place that needs it. The function resolves which store already claims a session and is exported from the history package, but nothing in the tree calls it: the ingest path reimplements the same walk inline instead. That is two copies of one rule, which is the shape this repository's one-canonical-primitive principle exists to prevent, and it is also dead scaffolding on a package boundary, which the wired-or-it-isn-t-done rule forbids. The duplication is the more expensive half: a later change to how a session's owner is resolved has two homes to find, and the inline copy is the one that actually runs, so a fix applied to the exported helper alone would appear to work and change nothing. Either make the inline site call the helper, or delete the helper and let the inline walk be the only definition. The spec already flags this as an uncertainty; it shipped unresolved.

## Grounds

- pursued: a change to the ownership rule now changes both the lookup and ingest placement; a second copy of the one-store rule reappearing in ingest.go would show it wrong
