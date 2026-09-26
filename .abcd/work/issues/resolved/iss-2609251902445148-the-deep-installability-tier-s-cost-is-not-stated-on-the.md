---
schema_version: 1
id: "iss-2609251902445148"
slug: "the-deep-installability-tier-s-cost-is-not-stated-on-the"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
resolution: "The launch command page states the deep tier's cost and the render at the tag's; the materialisation root is created before the copy loop."
impact: internal
resolved_by:
  commit: "ddfc6622"
---

The deep installability tier's cost is not stated on the launch command page: each preview that asks for it and every staging cut makes a shared clone of the baseline tag and checks it out (about three seconds and a tree the size of the checkout), with no opt-out on the cut. Also smokeDeepOverBundle (internal/core/launch/deepsmoke.go) creates the materialisation root after the copy loop rather than before it.

## Grounds

- pursued: commands/launch.md names both costs and the cut's lack of an opt-out; a page that leaves either unstated would show it wrong
