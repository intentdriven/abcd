---
schema_version: 1
id: "iss-2609262259360005"
slug: "kill-readings-leave-variable-file-brace"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Waits on ruling AS: per class in internal/core/guard/defaults/guard.json: if covered, add kill-by-holder entries for kill $(lsof -t ...) and fuser -k, read a ps | grep | awk source as a search only when grep or awk filters by a name (so ps -o pid= -p N | xargs kill stays allowed), and read a pid file written and read in one line; if left uncovered, name each spelling as a known limit in the guard's brief chapter. Either way add each spelling to the adversarial corpus with its ruled verdict and keep the repo-mined false-positive ceiling green."
deferred_after: "v0.11.1"
deferral_reason: "every ordinary spelling this record named is read on fix/drain-guard (4cc129009, lane drainG2 of run A); what remains asks a ruling the lane may not take (ruling AS, owed by the product thinker): whether the guard follows a pid list across lines through a file, which source commands of a ps | grep | awk chain count as a search when ps -o pid= -p N | xargs kill is the everyday form of a recorded-pid kill, and whether a kill by the holder of a port or file (lsof -t, fuser -k) belongs to the kill-by-name family at all."
---

The kill-by-search and by-owner readings (iss-2609251640452031) left spellings of a kill by name or selector unseen. The ordinary ones are read on fix/drain-guard (4cc129009): a brace or paren group ({ pgrep make; } | xargs kill), a search inside a shell string (kill $(sh -c 'pgrep make')), a kill inside a shell string xargs runs or one that reads the pipe (pgrep make | xargs -I{} sh -c 'kill {}', pgrep make | xargs bash -c 'kill "$@"' _, pgrep make | sh -c 'xargs kill'), a kill behind xargs and an unknown launcher (pgrep make | xargs myrunner kill, now a Tier 2 warn), a selector value attached with a byte other than a letter or digit (pkill -tpts/3, pkill -ubob.smith), and pkill -Ubob or -Gstaff attached, read once pkill's signal word is read as a signal first.

Three remain, and each waits on ruling AS: (1) a pid list carried through a file across commands (pgrep make > p; xargs kill < p), a data flow between lines the per-line reading does not follow; (2) a pid list taken from a ps | grep | awk chain, where reading ps or grep as a search source would also block ps -o pid= -p N | xargs kill, the recorded-pid kill; (3) a kill by the holder of a port or file, kill $(lsof -t -i :8080) and fuser -k, which stop whatever holds it, a peer session's server included, and have no entry and no statement in the brief. A pid list carried through a variable or a while-read loop (pids=$(pgrep make); kill $pids) is not this record's: it is the plain-variable half of iss-2609251824244354, ruled in DECISIONS 2026-09-25 (c) and deferred there.

## Remedy grounds (2026-09-29)

- Why: the three open spellings are scope choices the record names, so each answer is written with its entry and its test; ruling AS is unanswered and none is picked.
- Sources (consulted 2026-09-29): fuser(1), which documents -k as killing every process accessing the file and 'fuser -km /home' as killing everything on that file system (https://man7.org/linux/man-pages/man1/fuser.1.html), so a kill by holder is a kill by selector in effect; the Claude Code sandboxing documentation treats command rules and operating-system isolation as 'complementary layers' (https://code.claude.com/docs/en/sandboxing), which is why an uncovered spelling is stated as a limit rather than left silent.
- Rejected: reading ps or grep as a search source unconditionally, which blocks the recorded-pid kill the record names as everyday.
