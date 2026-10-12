---
schema_version: 1
id: "iss-2610120916220240"
slug: "guard-file-carry-false-block"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "Fable review of guard rounds G2, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Skip the wild file read for a token whose unknown parts are only parameter expansions the line did not assign; prove with a Check test that a logged search plus a recorded-pid xargs kill allows and the search-to-file-to-kill form still blocks"
---

On branch fix/sec-g2-261011, fileCarries (internal/core/guard/carried.go) treats every unfixed word of a later command as a read of every file the line wrote, so after a search's output is merely logged to a file, the guard's own sanctioned successor piped through xargs (a recorded pid echoed into xargs kill) blocks as kill-by-search. Rare false block; the ruling says a variable the line did not set reads as before.
