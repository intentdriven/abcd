---
schema_version: 1
id: "iss-2609240519413467"
slug: "the-disembark-plugin-page-s-argument-hint-advertises"
severity: "minor"
category: "documentation"
source: "drift-detection"
found_during: "v0.10.0 release gate: brief-surface crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/disembark.md"
---

The disembark plugin page's argument-hint advertises '<source-repo> <dest>', a form the binary rejects with 'unknown command' because packing needs the explicit pack sub-verb; the hint also omits pack and five shipped sub-verbs (coverage, graveyard, press-release, principles, review) and presents plan/probe <source-repo> as required where the CLI makes the repo optional. Found by checker a01 at fa744b41 (finding x-011).
