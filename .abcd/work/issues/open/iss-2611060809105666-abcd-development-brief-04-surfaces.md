---
schema_version: 1
id: "iss-2611060809105666"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md"
remedy: "Drop the restart from that sentence in 34-build.md."
---

.abcd/development/brief/04-surfaces/34-build.md:533 says a route override is refused beside a release, a discard or a restart, which hand no role out. internal/surface/cli/build.go:1224-1232 refuses it only beside `--release` and `--discard`: a `--restart` takes the route and starts the restarted role on it at once. `abcd implement step --help` and the chapter's own later sentence on a restart with a route in force say the same.
