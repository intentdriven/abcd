---
schema_version: 1
id: "iss-2609240519413467"
slug: "the-disembark-plugin-page-s-argument"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/disembark.md"
resolution: "the disembark argument-hint lists pack, plan, probe and the five further sub-verbs with the operands the CLI takes"
impact: fix
resolved_by:
  commit: "07af2eb092552739a5360c39e5659457e7b3c734"
---

The disembark plugin page's argument-hint advertises '<source-repo> <dest>', a form the binary rejects with 'unknown command' because packing needs the explicit pack sub-verb; the hint also omits pack and five shipped sub-verbs (coverage, graveyard, press-release, principles, review) and presents plan/probe <source-repo> as required where the CLI makes the repo optional. Found by checker a01 at fa744b41 (finding x-011).

## Grounds

- pursued: every form the hint advertises is one the binary accepts; a hint form the CLI rejects would show it wrong
