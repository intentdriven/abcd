---
schema_version: 1
id: "iss-2610151422111866"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/01-agents.md"
remedy: "Rewrite .abcd/development/brief/05-internals/01-agents.md:98 to match the shipped surface: All four agents/cold-reading-*.md carry a sixth top-level section, '## Shell commands the guard refuses' (the guard-lessons block added under iss-2610091942156774, noted in internal/core/reading/definitions_test.go). No test asserts an exact five-part…"
---

The brief at .abcd/development/brief/05-internals/01-agents.md:98 says: A cold-reading definition holds five parts (## Object, ## Question, ## The blindness core, ## Regime, ## Item shape), and a test asserts exactly that composition. The shipped surface differs: All four agents/cold-reading-*.md carry a sixth top-level section, '## Shell commands the guard refuses' (the guard-lessons block added under iss-2610091942156774, noted in internal/core/reading/definitions_test.go). No test asserts an exact five-part composition: definitions_test.go fetches the named sections individually and asserts exactly one fenced JSON block in Item shape, so a sixth section passes. (v0.14.2 release crosscheck, false-claim, raised by opus in a33.)
