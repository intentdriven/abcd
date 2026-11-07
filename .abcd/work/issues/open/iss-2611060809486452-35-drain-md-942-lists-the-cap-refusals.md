---
schema_version: 1
id: "iss-2611060809486452"
slug: "35-drain-md-942-lists-the-cap-refusals"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/35-drain.md"
remedy: "Either refuse `--max 0` as `build next` does, or state in the chapter that zero means no cap and correct the remedy's wording and the printed cap line."
---

`35-drain.md:942` lists the cap refusals and line 12 says the cap bounds the lanes a drain opens, but the chapter never says that `abcd drain --max 0` is accepted and opens every lane. The run prints `cap:   none (--max not given: all)` although the flag was given, and writes `"max": 0`. The refusal of `--max -1` at `internal/core/implement/loop/drain.go:369` asks for a whole number of lanes, 1 or more, which excludes the 0 it accepts, and `abcd build next` refuses `--max 0` (`internal/surface/cli/build.go:430`).
