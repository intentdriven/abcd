---
schema_version: 1
id: "iss-2608271711539855"
slug: "docs-lint-harness-rules-miss-host-tokens"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "docs/how-to/install.md"
resolution: "harness/claude-code also matches the host's dotted plugin directory and an upper-case environment variable prefixed with its name, and harness/codex and harness/gemini gain the environment-variable arm; watched RED against the old config and watched firing on both install page sites, which are then rephrased in generic terms. No allow marker was added: whether install pages may name a host stays with iss-216."
impact: internal
resolved_by:
  commit: "cdacf598d"
---

the docs-lint harness rules miss bare host-name tokens in paths and env vars: docs/how-to/install.md names the host harness twice in hand-written prose — a .claude-plugin/ directory link and the $CLAUDE_PLUGIN_DATA cache variable — with no docs-lint finding and no sanctioned allow escape. Fix the detector first: widen the harness/claude-code pattern in .abcd/docs-lint.json so a product-name token inside a path or env var is caught, and watch it fire on both install.md sites before deciding the prose remedy. Whether install pages get a per-line allow marker is already an open decision on iss-216 — do not pre-empt it here.

## Grounds

- pursued: a host named through a path or an environment variable in user-facing docs is a blocker; such a token in docs/ that abcd lint docs does not report would show it wrong
