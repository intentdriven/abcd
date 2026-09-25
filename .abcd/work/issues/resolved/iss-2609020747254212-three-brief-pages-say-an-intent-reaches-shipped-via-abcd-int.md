---
schema_version: 1
id: "iss-2609020747254212"
slug: "three-brief-pages-say-an-intent-reaches-shipped-via-abcd-int"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "release-v0.7.1-docs-currency"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
resolution: "already fixed at tip: 7c3fe74c (batch D) removed the nonexistent intent ship verb; the scope, verification-matrix and out-of-scope pages name spec close"
impact: internal
resolved_by:
  commit: "7c3fe74c2de40ed2c53cc2d6fa152004d081ed42"
---

Three brief pages say an intent reaches shipped/ via '/abcd:intent ship <itd-N>': 01-product/04-scope.md line 44, 06-delivery/02-verification-matrix.md line 42 and 03-out-of-scope.md line 120. No such verb exists; abcd intent offers audit, link, new, plan and ready, and the route that ships an intent is 'abcd spec close <spc-N>', which commands/intent.md documents as CLI-only. The verification matrix is a page whose whole job is to be checkable, and it qualifies the v0.7.1 line that the brief describes the binary that shipped. Found by the v0.7.1 docs-currency pass; deferred from the receipt to this record.

## Grounds

- pursued: the brief names spec close as the only route to shipped/; shown wrong if a brief page again names an intent ship verb as if it existed
