---
schema_version: 1
id: "iss-331"
slug: "commands-lint-md-presents-the-abcd-lint"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "commands/lint.md"
resolution: "lint.md now scopes the abcd-lint:allow waiver to privacy-hygiene and names docs-currency's own escape"
impact: fix
---

commands/lint.md presents the abcd-lint:allow line waiver as applying to any finding, but only the privacy-hygiene rule consumes it; a user waiving a docs-currency/three-tier-layout/etc finding gets a silent no-op and a stray committed token, colliding with docs-currency's own differently-spelled escape

## Evidence

- `commands/lint.md` (the waiver sentence) -- scoped to no rule.
- `internal/core/repolint/rule_privacy.go:42,49,112` -- the only consumer of the waiver constants; no other rule file references either.
- `rule_docs.go:79` / `rule_positioning.go:86` emit line-anchored findings where the marker is a no-op; docs-currency's real escape is the differently-spelled docs-lint allow comment.

## Refuter verdict -- CONFIRMED (substantive, lower end)

Every other statement of the waiver in the record scopes it to privacy-hygiene (brief 16-lint.md:52, prepare-this-repo.md:96-99, CHANGELOG). commands/lint.md is the lone outlier; entered unscoped and survived the audit->lint rename. Fix: scope the sentence to privacy-hygiene findings and note other rules have no line waiver.
