---
schema_version: 1
id: "iss-2609262259360005"
slug: "kill-readings-leave-variable-file-brace-group-and-port-spellings"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
---

The kill-by-search and by-owner readings (iss-2609251640452031) leave these spellings of a kill by name or selector unseen: a pid list carried through a file (pgrep make > p; xargs kill < p), out of a brace group ({ pgrep make; } | xargs kill, because the semicolon inside the group starts a new pipeline in the tokenizer), from a ps | grep | awk chain, or from a shell string inside the substitution (kill $(sh -c 'pgrep make')); a kill behind xargs and then an unknown launcher (pgrep make | xargs myrunner kill, where the Tier 2 window starting at kill no longer holds the xargs); a kill inside a shell string xargs runs (pgrep make | xargs -I{} sh -c 'kill {}', pgrep make | xargs bash -c 'kill "$@"' _), which the review of the first reading found allowed silently, because the string's commands carry none of the pipe that feeds xargs; a selector value attached to its flag that holds a byte other than a letter or digit (pkill -tpts/3, pkill -ubob.smith), which the flag matcher does not read as a short cluster; and pkill -Ubob or -Gstaff attached, left out because the signal names -HUP, -USR1, -QUIT and -SEGV hold the same upper-case letters. Two kills by selector have no entry at all: kill by port through lsof -t (kill $(lsof -t -i :8080)) and fuser -k, which stop whatever holds a port or file, a peer session server included. A pid list carried through a variable or a while-read loop (pids=$(pgrep make); kill $pids) is not this record's: it is the plain-variable half of iss-2609251824244354, ruled in DECISIONS 2026-09-25 (c) and deferred there.
