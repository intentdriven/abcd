---
schema_version: 1
id: "iss-2610290717018523"
slug: "the-load-gate-now-refuses-a-test-stage"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "fix of iss-2610281502581386"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload"
remedy: "Decide whether abcd may trim the Go build cache; if it may, have the worktree store's reclaim delete cache entries unused for a set age (one or two days) when it removes worktrees and say what it removed, and if it may not, record that decision and close this."
resolution: "Decided in dcn-2611041338454498. abcd ahoy worktree prune, when it removes at least one worktree, also deletes Go build-cache entries unused for two days and says how many entries and bytes went; a dry run says the same and deletes nothing. The trim acts only on the directory go env GOCACHE reports, when that is a real directory this account owns, has a Go build cache's shape and is not the home or above it, and only on <hex>-a and <hex>-d files in two-hex-digit subdirectories; it follows no symlink and removes no directory, and a skip or a failure never fails the prune. This does not bound a cache whose entries are all recent: the age is fixed, so a machine using every entry within two days frees little, and the size or free-space target the 3 Nov note asks for is not built here."
impact: additive
---

The load gate now refuses a test stage when the volume holding the Go build cache or the worktree store has less than its floor free (iss-2610281502581386), but nothing keeps the cache from growing back. Go trims only entries unused for five days, and every store worktree, race build and pinned-toolchain build adds entries, so a machine running many worktrees for days still fills its disk and then stands until a person trims the cache by hand; on 29 Oct 2026 the cache on the development machine had grown back to about 291 GiB. The first issue's remedy also asked the worktree store's reclaim to trim cache entries older than a day when it removes worktrees; that was left out of its fix because abcd deleting another tool's cache is a trust change that needs its own decision.

2026-11-03: the Go build cache reached about 323 GB with every entry used within two days, leaving 5 GiB free; the race gate refused a preflight under its 8 GiB floor, and the find it suggests (entries unused for two days) freed nothing. Deleting entries unused for twelve hours freed about 270 GB. A fixed age cannot free the floor on a machine this busy; the trim needs to work to a size or a free-space target.
