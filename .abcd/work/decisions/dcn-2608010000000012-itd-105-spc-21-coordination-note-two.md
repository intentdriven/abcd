---
id: dcn-2608010000000012
slug: itd-105-spc-21-coordination-note-two
date: 2026-08-01
---
2026-08-01 — itd-105 / spc-21, coordination note: two independently-running instances of the v0.5.0 loop each reviewed PR #184 at its merge gate concurrently, found different real findings (a self-referential allowlist bypass; a curl config-surface bypass), and pushed both fix commits to `v050/itd-105` minutes apart — rebased together here rather than either overwriting the other. Same collision class the state issue's round-9 comment already disclosed for `v050/iss-159`; recorded again per that precedent, no new action taken beyond the rebase.
