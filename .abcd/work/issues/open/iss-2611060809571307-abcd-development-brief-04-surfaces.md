---
schema_version: 1
id: "iss-2611060809571307"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Add `--scores -` and its shared bound to the chapter and the command page."
---

`.abcd/development/brief/04-surfaces/03-embark.md:74-76` describes the scores as handed back in a file, read as a regular file and never through a link. `--scores -` also reads standard input, capped at the same bound (`internal/surface/cli/embark_lessons.go:147-148`). The flag's help says so; the chapter and `commands/embark.md` do not.
