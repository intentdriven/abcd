---
schema_version: 1
id: "iss-2608271804499169"
slug: "code-declares-config-paths-the-tree"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: "internal/core/repolint/rule_privacy.go"
resolution: "The .abcd/README.md index lists pii.json and scripts-closure.json as optional overrides this checkout does not carry, with what each does when absent and where its schema is stated, and adds the config members and root baseline it had missed. The broader parity sweep between code path literals and documented namespace is not built; nothing here claims it."
impact: internal
resolved_by:
  commit: "eb65d9391"
---

the binary declares two per-repo config paths the tree never instantiates: rule_privacy reads .abcd/config/pii.json and the launch includes-closure reads .abcd/config/scripts-closure.json, but .abcd/config/ holds neither and no doc mentions them. Both read as optional overrides, so nothing is broken — but code-declared record paths and tree-instantiated ones have no reconciliation check in either direction. Document the two optional files where the config/ members get their index entry, and consider a parity sweep between code path literals and the documented namespace.

## Grounds

- pursued: every path the binary reads under .abcd/ config is named in the namespace index; a code-declared .abcd/config path absent from the index would show it wrong
