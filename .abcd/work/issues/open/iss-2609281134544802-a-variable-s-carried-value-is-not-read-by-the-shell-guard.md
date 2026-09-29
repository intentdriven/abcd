---
schema_version: 1
id: "iss-2609281134544802"
slug: "a-variable-s-carried-value-is-not-read-by-the-shell-guard"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: lane drainG3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
deferred_after: v0.11.0
deferral_reason: "lane drainG3 (run A, 2026-09-28) built the parameter-expansion reading of iss-2609251824244354 and measured it: reading a variable's carried value as a pid list, a stream path, shell text or a pkill/killall program name refuses ordinary commands (22 program-name and 15 stream blocks in the 4,315-input sweep), and leaving it unread keeps each spelling open; whether each class blocks, warns or allows is a product ruling not yet made, so these four classes, and only they, are carried past v0.11.0."
---

The shell guard reads a parameter expansion as an unknown word (iss-2609251824244354, lane drainG3) for the flags and program names its value can spell, but not as data an earlier command carried into the variable, and four classes stay unread: (1) a pid list a search printed into a variable (p=$(pgrep make); kill $p, or pgrep make | while read p; do kill $p; done), the plain-variable half DECISIONS 2026-09-25 (c) parked on iss-2609251824244354; (2) a stream path in a variable handed to a shell or source as its script (bash "$f" after f=<(curl ...)); (3) shell text a variable holds, run through eval "$X", sh -c "$X", or placed in a string a shell runs, whose value may hold a separator or quotes (eval "$X" allows, as it did before the reading); (4) pkill or killall as a variable's value standing as the program with an operand ($P make), and a variable-named bare interpreter inside a string. Reading each would refuse ordinary commands: the false-positive sweep of 4,315 inputs showed 22 new program-name-unknown blocks ("$GO" build, exec "$BIN", $EDITOR notes.md) and 15 new interpreter-reads-stream blocks (bash "$SCRIPT") before these readings were left out. Classes 1 and 3 carry the same cost, from the sweep's everyday-variable lines, each of which allows today: a reading of class 1 cannot tell a pid a search printed from one the script recorded, so it would refuse kill "$pid", kill -TERM "$PID" and kill -- -"$pg" (a background job's pid, a pidfile's, a process group's); a reading of class 3 cannot tell a hostile string from a composed one, so it would refuse eval "$cmd", the line that runs a command a script built up. Each class is a block, warn or allow call the product thinker has not ruled on.

## Deferral 2026-09-28

Deferred past v0.11.0: lane drainG3 (run A, 2026-09-28) built the parameter-expansion reading of iss-2609251824244354 and measured it: reading a variable's carried value as a pid list, a stream path, shell text or a pkill/killall program name refuses ordinary commands (22 program-name and 15 stream blocks in the 4,315-input sweep), and leaving it unread keeps each spelling open; whether each class blocks, warns or allows is a product ruling not yet made, so these four classes, and only they, are carried past v0.11.0.
