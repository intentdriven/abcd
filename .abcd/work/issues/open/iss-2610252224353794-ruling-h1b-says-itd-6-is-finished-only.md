---
schema_version: 1
id: "iss-2610252224353794"
slug: "ruling-h1b-says-itd-6-is-finished-only"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "gap audit of the owed fidelity reviews, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/repoprompt/mcptest/mcptest.go"
remedy: "Ask the person to run the adapter once against the real RepoPrompt app and record the run (date, app version, what was asked and returned) in a research note linked from itd-6's Audit Notes; until then the intent carries the check as owed."
---

Ruling H1b says itd-6 is finished only after one live run against the real RepoPrompt app, a check owed to a person. itd-6 moved to shipped/ with every test on a test-owned MCP server (internal/adapter/repoprompt/mcptest/mcptest.go) and no record of the run, and no ledger record tracks it; the command-line runner's equivalent is tracked as iss-2609301519558538.
