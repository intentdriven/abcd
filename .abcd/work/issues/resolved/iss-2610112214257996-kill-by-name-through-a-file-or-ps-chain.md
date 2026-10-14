---
schema_version: 1
id: "iss-2610112214257996"
slug: "kill-by-name-through-a-file-or-ps-chain"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "security fix lane G2a, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Waits on a ruling for each spelling: if covered, read a ps | grep | awk source as a search only when grep or awk filters by a name (so ps -o pid= -p N | xargs kill stays allowed), and read a pid file written and read in one line, in internal/core/guard; if left uncovered, keep each named as a known limit in commands/guard.md and the guard's brief chapter. Either way add each spelling to the adversarial corpus with its ruled verdict."
refines: [iss-2609262259360005]
resolution: "Covered in the spirit of ruling AS (a pid list that came from a search blocks, a recorded pid does not). A command that reads a file the same text wrote, on its standard input or as an operand naming it, carries what the writer printed, so a kill fed through a file written by pgrep or pidof blocks under kill-by-search; a file name the line does not fix can be any file. A pid list lsof or fuser prints, carried the same way, is read as no search: the kill-by-holder entry that would block it belongs to iss-2609262259360005, still open. A grep, or an awk program that matches, filtering a ps listing is read as a search, so kill-by-search blocks it, while ps -o pid= -p N | xargs kill and a recorded pid file stay allowed. Proved by TestAPidListFromASearchBlocksHoweverItTravels, watched failing first, new kill-by-search fixtures and adversarial corpus lines; the false-positive sweep stays at 25 blocks and 29 warns over 5,937 inputs."
impact: fix
---

Two spellings of a kill by name that iss-2609262259360005 named stay unread after ruling AS (product thinker, 2026-10-11) settled its third, the kill by the holder of a port or file: (1) a pid list carried through a file across commands (pgrep make > p; xargs kill < p), a data flow between commands the per-line reading does not follow; and (2) a pid list taken from a ps | grep | awk chain, where reading ps or grep as a search source would also block ps -o pid= -p N | xargs kill, the everyday kill of a recorded pid. The ruling as relayed to lane G2a named only the holder class, so whether these two block is still owed. A pid list carried through a variable is iss-2609281134544802.
