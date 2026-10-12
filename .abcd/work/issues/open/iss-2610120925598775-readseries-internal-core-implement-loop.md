---
schema_version: 1
id: "iss-2610120925598775"
slug: "readseries-internal-core-implement-loop"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "build-next review round 2 (run-2610110908001704)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/nextseries.go"
remedy: "Have readSeries run parsePaceFlags on the stored flags and refuse the file at the pick stage on error, as it already refuses an outcome it never wrote; a test feeds a series whose flags carry U+202E and U+009B and expects the refusal, with nothing raw on the --json stream."
---

readSeries (internal/core/implement/loop/nextseries.go) checks a series file's intent, run_id and outcome but not its stored pace flags (flags.pace, flags.sub_agents, flags.fix_rounds), so a hand-written or repository-shipped .abcd/.work.local/run/next.json can carry control or bidi runes (for example U+202E, U+009B) in them; on a series move that picks nothing (window or cap), those strings reach 'abcd build next --json' unvalidated and encoding/json leaves them raw. Low severity: display spoofing of the JSON text on a terminal, no read, write or exec. Found by the security reviewer, round 2 of run-2610110908001704 lane-1.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The series file, its reader and the `--json` stream that echoes it are all abcd's own, so abcd added it and fixes it; the fix refuses a malformed file abcd itself reads and changes nothing about how the repository works.
