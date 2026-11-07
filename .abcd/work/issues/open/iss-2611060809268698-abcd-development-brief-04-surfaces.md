---
schema_version: 1
id: "iss-2611060809268698"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-session.md"
remedy: "Add exit 130 for a question stopped at a terminal, keeping any answers already given."
---

`.abcd/development/brief/04-surfaces/37-session.md:234` lists the exit codes of the session verbs as 0, 1 and 2. `abcd session start`, `abcd session blueprint` and `abcd session setup` also exit 130 when the live brief session question is stopped with Ctrl-C at a terminal (`internal/surface/cli/session.go:224`, `:485`, `:730`; `internal/surface/cli/setup_interview.go:489`). The ahoy and reflect chapters name this exit for their own questions.
