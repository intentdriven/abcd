---
id: dcn-2611041338454498
slug: the-product-thinker-ruled-on-2026-11-04
date: 2026-11-04
---
2026-11-04 — The product thinker ruled on 2026-11-04 that worktree prune may trim the Go build cache on its own: on every prune that removes a worktree it deletes cache entries unused for two days and says what it freed, because a full disk otherwise stops unattended work until a person acts, and Go rebuilds a deleted entry at the cost of one slower build.
