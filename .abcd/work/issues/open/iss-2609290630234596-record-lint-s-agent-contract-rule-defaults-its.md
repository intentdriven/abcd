---
schema_version: 1
id: "iss-2609290630234596"
slug: "record-lint-s-agent-contract-rule-defaults-its"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/agentcontract.go"
---

record-lint's agent_contract rule defaults its prompt-version log to <agents_dir>/CHANGELOG.md when the config names no changelog path, which is inside the directory a harness loads whole: a repository that takes the default gets a spurious CHANGELOG agent, the iss-110 defect this repository fixed by moving its log to .abcd/development/agents/ and configuring the path. The default either moves outside the loader's root or becomes a required key; either way the agent_contract tests that write agents/CHANGELOG.md with a bare rule follow.
