---
schema_version: 1
id: "iss-2610120953445472"
slug: "guard-extended-path-source-unread"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G3, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Search the fixed entries of a PATH value in order and stop at the first entry the line does not fix (the account's PATH where it is $PATH), reading any file found before it; read NAME+=value as appending to the tracked value; prove with Check tests that dir:$PATH and += read dir's file and that $PATH:dir with the name on the account's PATH is unchanged"
resolution: "A PATH value is now read entry by entry: a fixed entry is kept, $PATH or ${PATH} whole is the value before the assignment (the account's PATH where the line set none), $HOME opening an entry is HOME's value, and any other entry the line does not fix stops the search; NAME+=VALUE is read as an append before a command, as a prefix and under export, declare and typeset, and zsh's path=(dir $path) and path+=(dir) read the account's PATH alike. A PATH the line sets is searched strictly, so an unfixed entry reached before the file blocks as script-unread. Verified against /bin/bash 3.2, bash 5 and /bin/zsh 5.9. Proved by TestExtendedPathSourceIsRead, watched failing first, with export PATH=\"$HOME/bin:$PATH\" alone and other ordinary lines allowed."
impact: fix
resolved_by:
  commit: "38236aec8"
---

The guard does not read a sourced file found through a PATH the line extends rather than replaces: after PATH=dir:$PATH, export PATH="dir:$PATH" or PATH+=:dir, a source of a bare name that dir holds is allowed unread, because the search gives up on a PATH value holding a variable instead of searching its fixed entries, and the += form is not read as an assignment at all. Found while fixing the zsh path array (iss-2610111730513071) in security lane G3, 2026-10-12; probe: PATH={d}/x:$PATH; source foo.sh = allow with x/foo.sh holding a blocker, while PATH={d}/x; source foo.sh blocks.
