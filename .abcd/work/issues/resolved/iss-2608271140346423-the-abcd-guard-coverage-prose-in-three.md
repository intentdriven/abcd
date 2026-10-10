---
schema_version: 1
id: "iss-2608271140346423"
slug: "the-abcd-guard-coverage-prose-in-three"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "v0.6.7-release-gate-docs-currency"
resolution: "Corrected the guard backtick-coverage prose on all four coverage surfaces — the guard.go 'guard check --help' string, the regenerated docs/reference/cli/commands.md, commands/guard.md, and the brief 04-surfaces/17-guard.md blind-spot list (the last surfaced by the release-gate cross-check) — to state a top-level backtick substitution is now followed into command position like the dollar-paren form, matching the gh-312 tokenizer fix and the v0.6.7 Security changelog. The brief entry re-frames the residual disclosed gap as the flags-after-substitution truncation that affects both forms, citing iss-148 (narrowed in place to that deep sub-part). Regenerated the surface snapshot."
impact: internal
---

The abcd guard coverage prose in three user-facing places still describes the pre-gh-312 backtick limitation: guard.go's 'guard check --help' string and the generated docs/reference/cli/commands.md say the top-level backtick substitution is not followed (a disclosed v1 limit, iss-148), and commands/guard.md lists 'one inside a backtick substitution' among what an allow does not see. The gh-312 fix now follows top-level backticks into command position like the dollar-paren form, which the v0.6.7 Security changelog advertises — so the shipped help and docs contradict the release's own changelog and under-report a security capability.