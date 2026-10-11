---
schema_version: 1
id: "iss-2609281134544802"
slug: "a-variable-s-carried-value-is-not-read"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: lane drainG3"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "Waits on ruling AT: in internal/core/guard/unknown.go: if (a), read each of the four carried-value classes as its worst case and block; if (b), emit a Tier 2 warn for each and block none; if (c), block only a pid list a search printed into a variable and warn on the other three; if (d), list the four as named allows in the guard's brief chapter. Prove the choice with one adversarial corpus line per class at its ruled verdict and the 4,315-input false-positive sweep's new block and warn counts recorded against a ceiling test."
deferred_after: v0.11.1
deferral_reason: "lane drainG3 (run A, 2026-09-28) built the parameter-expansion reading of iss-2609251824244354 and measured it: reading a variable's carried value as a pid list, a stream path, shell text or a pkill/killall program name refuses ordinary commands (22 program-name and 15 stream blocks in the 4,315-input sweep), and leaving it unread keeps each spelling open; whether each class blocks, warns or allows is a product ruling not yet made, so these four classes, and only they, are carried past v0.11.1. The ruling is still owed: the product thinker has not ruled on any of the four classes since the v0.11.0 deferral, which lapsed when v0.11.1 was cut."
---

The shell guard reads a parameter expansion as an unknown word (iss-2609251824244354, lane drainG3) for the flags and program names its value can spell, but not as data an earlier command carried into the variable, and four classes stay unread: (1) a pid list a search printed into a variable (p=$(pgrep make); kill $p, or pgrep make | while read p; do kill $p; done), the plain-variable half DECISIONS 2026-09-25 (c) parked on iss-2609251824244354; (2) a stream path in a variable handed to a shell or source as its script (bash "$f" after f=<(curl ...)); (3) shell text a variable holds, run through eval "$X", sh -c "$X", or placed in a string a shell runs, whose value may hold a separator or quotes (eval "$X" allows, as it did before the reading); (4) pkill or killall as a variable's value standing as the program with an operand ($P make), and a variable-named bare interpreter inside a string. Reading each would refuse ordinary commands: the false-positive sweep of 4,315 inputs showed 22 new program-name-unknown blocks ("$GO" build, exec "$BIN", $EDITOR notes.md) and 15 new interpreter-reads-stream blocks (bash "$SCRIPT") before these readings were left out. Classes 1 and 3 carry the same cost, from the sweep's everyday-variable lines, each of which allows today: a reading of class 1 cannot tell a pid a search printed from one the script recorded, so it would refuse kill "$pid", kill -TERM "$PID" and kill -- -"$pg" (a background job's pid, a pidfile's, a process group's); a reading of class 3 cannot tell a hostile string from a composed one, so it would refuse eval "$cmd", the line that runs a command a script built up. Each class is a block, warn or allow call the product thinker has not ruled on.

## Deferral 2026-09-28

Deferred past v0.11.0: lane drainG3 (run A, 2026-09-28) built the parameter-expansion reading of iss-2609251824244354 and measured it: reading a variable's carried value as a pid list, a stream path, shell text or a pkill/killall program name refuses ordinary commands (22 program-name and 15 stream blocks in the 4,315-input sweep), and leaving it unread keeps each spelling open; whether each class blocks, warns or allows is a product ruling not yet made, so these four classes, and only they, are carried past v0.11.0.

## Deferral 2026-09-29

Deferred past v0.11.1: lane drainG3 (run A, 2026-09-28) built the parameter-expansion reading of iss-2609251824244354 and measured it: reading a variable's carried value as a pid list, a stream path, shell text or a pkill/killall program name refuses ordinary commands (22 program-name and 15 stream blocks in the 4,315-input sweep), and leaving it unread keeps each spelling open; whether each class blocks, warns or allows is a product ruling not yet made, so these four classes, and only they, are carried past v0.11.1. The ruling is still owed: the product thinker has not ruled on any of the four classes since the v0.11.0 deferral, which lapsed when v0.11.1 was cut.

## Remedy grounds (2026-09-29)

- Why: ruling AT sets out options (a) to (d) and is unanswered, so the remedy gives the change and proof for each without picking; the sweep counts on this record are the cost evidence the ruling needs.
- Sources (consulted 2026-09-29): the Claude Code sandboxing documentation calls command rules and operating-system isolation 'complementary layers' (https://code.claude.com/docs/en/sandboxing): a static reading of a command line cannot know a variable's runtime value, so the field bounds what such a reading promises instead of chasing values.
- Rejected: taint tracking across lines, which the per-line reader cannot do and no shell guard in the field claims to do.
