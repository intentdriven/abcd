---
id: dcn-2609300000000014
slug: correcting-one-name-in-the-entry-above
date: 2026-09-30
---
2026-09-30 — Correcting one name in the entry above that applies rulings CJ1 and CJ1b (lane installerMeta2, autonomous run A; Refs: iss-2609291942520919): the session start's single write is not `cache/update-shown` but `cache/update-shown-<new>`, one claim per release, named for the new tag and written beside the cache's `binary-meta`; in the degraded per-root mode it is `.update-shown-<new>` beside the plugin root's own `.binary-meta` (`updateShownPrefix` in `internal/core/ahoy/unseen_update.go`, as the fix round of that lane built it). The ruling and the rest of the entry stand.
