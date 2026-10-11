---
schema_version: 1
id: "iss-2608261550394120"
slug: "rules-render-lets-a-rule-body-forge"
severity: "major"
category: "security"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/rules/rules.go"
resolution: "sanitizeRuleBody strips control characters and space-indents hash-leading lines at the one render site, with the forgery corpus as regression tests; ported from the adaptor-lab fix"
impact: fix
resolved_by:
  commit: "5b114ced"
---

A repo-controlled rule body can forge a domain heading in the rendered rules injection block: renderDomain emits the body verbatim after its bullet prefix, so a body carrying a flush-left '## NAME' continuation line — or one that starts after a lone CR, U+2028 or U+2029, which are line starts to host-side multiline parsers — splits the block and impersonates or replaces a real domain. A red-team execution replaced a bundled domain's content end to end. Control characters, bidi overrides and C1/zero-width runes also reach the rendered block unmasked. A test-first fix (sanitizeRuleBody plus render_sanitize_test.go: separator normalisation before the trailing trim, heading-defusing indent, the canonical termsafe mask, render proven a fixed point across the pipeline boundary) sits unmerged in a local lab worktree cut at 089c0d61; promotion needs only the module-path update.