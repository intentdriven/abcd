---
schema_version: 1
id: "iss-2610241518233773"
slug: "the-product-thinker-asked-how-a-memory"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "product thinker's question on PR #982, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ingest.go"
remedy: "Hold every memory page to the recall's per-turn budget where the page is made: the ingest's distil step writes pages at or under that size, splitting a long source into several pages, and the page writer refuses one over it, naming the budget; the recall's set-aside then never fires for a page abcd wrote."
---

The product thinker asked how a memory page larger than the recall's per-turn budget (about 64 KB) could come about, and why one would be needed. Pages are written by abcd library ingest, which fetches up to 10 MiB of a source, distils it into pages through a model, and writes them, and by hand; nothing holds a page to the recall's budget, and the store reads pages up to 4 MiB. So a long paper or web page can become one oversized page that the recall then sets aside, as PR #982 builds it. No such page is needed: the limit belongs where the page is made, not where it is read.
