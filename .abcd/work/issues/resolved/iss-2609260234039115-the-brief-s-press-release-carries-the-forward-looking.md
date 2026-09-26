---
schema_version: 1
id: "iss-2609260234039115"
slug: "the-brief-s-press-release-carries-the-forward-looking"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:23"
resolution: "The stale first copy of the Forward-looking bullet is deleted; the copy that matches the tree (consistency shipped per itd-48, grill and shape not yet built) stands unchanged."
impact: internal
resolved_by:
  commit: "da449c7c"
---

The brief's press release carries the Forward-looking discipline bullet twice under What's In Scope, and the two copies contradict each other: the first (re-added by c4fc433e) says /abcd:intent grill is a sibling sub-verb of refine and that /abcd:intent consistency shipped in spc-29, while the second (from 7c3fe74c) says grill, consistency and shape are designed and not yet built. Neither refine nor grill is a registered sub-verb, and spc-29 is a predecessor-store id. Which bullet stands is the product thinker's call, since the page is the headline product in the product thinker's words; the stale copy should then be removed.

## Grounds

- pursued: the press release states each intent verb's standing once, in agreement with the tree; shown wrong if What's In Scope again carries two Forward-looking bullets or names a verb as shipped that is not registered
