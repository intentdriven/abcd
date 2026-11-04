---
schema_version: 1
id: "iss-2611040555069189"
slug: "06-delivery-01-build-sequence-md-109"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Name the three that ship, limit MCP to a build lane's reviews, and mark a native adapter as a design target."
---

06-delivery/01-build-sequence.md:109 says native, CLI, API and MCP adapters are opt-in for an operator who wants abcd to reach a model directly, unmarked. 05-internals/01-agents.md:190-204 says three ship: the OpenAI-compatible provider API, a model CLI runner, and RepoPrompt over MCP for a build lane's reviews alone. No native adapter exists, and --oracle-backend records a value that routes nothing (internal/core/ahoy/prompt_help.go:124).
