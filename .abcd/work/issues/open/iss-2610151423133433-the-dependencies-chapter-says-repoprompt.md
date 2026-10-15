---
schema_version: 1
id: "iss-2610151423133433"
slug: "the-dependencies-chapter-says-repoprompt"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 doc-fidelity review, content commit 138ded90b"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md"
remedy: "Drop codex from the list of oracle adapters, or mark it a design target: the shipped adapters are RepoPrompt over MCP and the OpenAI-compatible API client."
---

The dependencies chapter says RepoPrompt and codex are two opt-in oracle adapters, but no codex adapter exists: internal/adapter holds gitleaks, hosting, openaiapi, repoprompt, scanner and tailscale, and the runners are claude and opencode. Raised by the Opus doc-fidelity reviewer for the v0.14.2 cut, outside the two intents it judged.
