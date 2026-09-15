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
---

Three brief pages say an intent reaches shipped/ via '/abcd:intent ship <itd-N>': 01-product/04-scope.md line 44, 06-delivery/02-verification-matrix.md line 42 and 03-out-of-scope.md line 120. No such verb exists; abcd intent offers audit, link, new, plan and ready, and the route that ships an intent is 'abcd spec close <spc-N>', which commands/intent.md documents as CLI-only. The verification matrix is a page whose whole job is to be checkable, and it qualifies the v0.7.1 line that the brief describes the binary that shipped. Found by the v0.7.1 docs-currency pass; deferred from the receipt to this record.
