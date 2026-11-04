---
schema_version: 1
id: "iss-2611040554556207"
slug: "05-internals-03-configuration-md-3-5"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Change the count to three questions and describe the oracle backend as recorded without asking."
refines: [iss-2610031236155833, iss-2609261447395216]
---

05-internals/03-configuration.md:3-5 says installing asks four questions and :49-50 lists visibility, the docs target, the oracle backend and the deep scan. internal/core/ahoy/prompt_help.go:136 keeps the backend question off while only one choice has an adapter, recording host-delegated with a note; 05-internals/01-agents.md and commands/ahoy.md say the same. --oracle-backend still sets the value.
