---
id: dcn-2608230000000001
slug: abcd-site-check-excludes-docs-at-the
date: 2026-08-23
---
2026-08-23 — `abcd site check` excludes `docs/` at the page walk, rather than
teaching the parser to tolerate HTML comments on SSG-authored pages
(iss-2608230838592910's own candidate fix, rejected). Tolerating comments
would admit mkdocs-material output into gates whose rules were written for
generator output — the provenance walk and banned-token gate already exempt
the tree via `isComposedSurface`, so only the mobile and figure-label gates
would newly run over markup this repo cannot act on. Those pages never
entered the page list on any prior revision either (the parser has always
refused comments), so the exclusion removes no coverage that existed; it
states the boundary the overflow audit already draws. The cost is recorded
honestly in the gate's own scope comment: a green report describes the pages
abcd renders and no others, and `docs/` layout stays the SSG's own concern.
