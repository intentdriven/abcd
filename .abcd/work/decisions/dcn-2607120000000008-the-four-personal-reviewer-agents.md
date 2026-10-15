---
id: dcn-2607120000000008
slug: the-four-personal-reviewer-agents
date: 2026-07-12
---
2026-07-12 — The four personal reviewer agents (ruthless, security, docs-currency,
sota-researcher) MOVED from the machine-global `~/.claude/agents/` into abcd's
plugin `agents/` and deleted at source; they now resolve as `abcd:<name>` in every
repo with the plugin enabled, versioned in-repo and reviewable by PR. Frontmatter
key is `prompt_version` (itd-5's name), not `version` — intent-fidelity-reviewer
renamed. Colour encodes the DOMAIN EXAMINED, never rank or taste: red=trust
boundary, orange=code correctness, blue=documentation truth, green=the record,
purple=external evidence; cyan reserved for artefact-producing (non-verdict) agents.
Accepted cost: the reviewers no longer resolve in repos without abcd installed.
