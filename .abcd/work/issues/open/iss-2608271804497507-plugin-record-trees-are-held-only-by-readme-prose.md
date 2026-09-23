---
schema_version: 1
id: "iss-2608271804497507"
slug: "plugin-record-trees-are-held-only-by-readme-prose"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "agents/README.md"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: a record-lint root over the plugin trees (agents/, commands/, hooks/) or an explicit prose-governed ruling). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the plugin record trees are held only by README prose: agents/ is a second record family with its own constitution (itd-5 frontmatter per prompt, an injection-canary fixture for every untrusted-input agent, a per-agent changelog) and commands/, hooks/, .claude-plugin/ carry similar structure, but no lint root or gate checks any of it — currently conformant, enforced by nothing. Decide the gate: either a record-lint root over the plugin trees with rules for the frontmatter/fixture/changelog invariants, or an explicit ruling that these trees stay prose-governed.