---
schema_version: 1
id: "iss-2610151422139704"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
remedy: "Rewrite .abcd/development/brief/05-internals/05-prompt-quality.md:106-108 to match the shipped surface: The agent changelog entry '0.2.0 — 2026-08-19 (iss-278)' (.abcd/development/agents/CHANGELOG.md:752-791) lists FIVE prompts: ruthless-reviewer, security-reviewer, docs-currency-reviewer, intent-auditor, sota-researcher. The same chapter says 'five prompts…"
---

The brief at .abcd/development/brief/05-internals/05-prompt-quality.md:106-108, .abcd/development/brief/05-internals/05-prompt-quality.md:128-129 says: the four review and research prompts enter the changelog at 0.2.0 (the bump that first gave them the untrusted-input contract) The shipped surface differs: The agent changelog entry '0.2.0 — 2026-08-19 (iss-278)' (.abcd/development/agents/CHANGELOG.md:752-791) lists FIVE prompts: ruthless-reviewer, security-reviewer, docs-currency-reviewer, intent-auditor, sota-researcher. The same chapter says 'five prompts hardened on 2026-08-19' at line 119. (v0.14.2 release crosscheck, stale-count, raised by sonnet in a35.)
