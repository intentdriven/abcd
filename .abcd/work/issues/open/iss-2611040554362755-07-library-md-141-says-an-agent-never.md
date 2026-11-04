---
schema_version: 1
id: "iss-2611040554362755"
slug: "07-library-md-141-says-an-agent-never"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md"
remedy: "Add that the shell guard refuses an agent's flip where the host runs the guard's hook."
---

07-library.md:141 says an agent never runs the flip and, since the binary cannot tell a person from an agent, that the rule lives in the command pages. The shell guard also refuses it: entry abcd-source-ledger-flip (internal/core/guard/personacts.go:57), taught by abcd rules SHELL and named at commands/library.md:219.
