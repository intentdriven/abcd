---
schema_version: 1
id: "iss-363"
slug: "commands-lint-md-and-prepare-this-repo"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "commands/lint.md"
resolution: "Added identity positioning to the lint-convention enumerations in commands/lint.md and commands/prepare-this-repo.md, with the conditional-skip note."
impact: fix
---

commands/lint.md and prepare-this-repo.md enumerate the conventions abcd lint checks as five, presented as exhaustive, but DefaultRules ships six including identity-positioning; prepare-this-repo.md itself arms that rule
## Evidence

- `internal/core/repolint/rules.go:6-15` — `DefaultRules()` returns six: threeTierLayout, conventionsRouter, decisionDurability, docsCurrency, privacyHygiene, **identityPositioning** (RuleID `identity-positioning`, `rule_positioning.go:54`); ships by default.
- `commands/lint.md:3` and `commands/prepare-this-repo.md:73-76` enumerate five ("the conventions the binary checks"), no "e.g."/"such as" — exhaustive framing.
- Internal contradiction: `prepare-this-repo.md:156-167` itself arms identity-positioning (`abcd identity init` → `.abcd/positioning.json` → ":164 abcd lint reports any surface that drifts").

## Adversarial verdict

CONFIRMED (substantive). identity-positioning is conditional in the same shape as the listed docs-currency, so its omission is inconsistent. Fix: add it to both enumerations.
