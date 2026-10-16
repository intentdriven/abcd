---
schema_version: 1
id: "iss-2610160608415797"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md"
remedy: "Narrow the sentence at 16-lint.md:140 to bare abcd lint, and name the docs, site and outbound targets the gates run."
resolution: "16-lint.md narrows the no-gate sentence to bare abcd lint and names where the docs, site and outbound targets are gated"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/16-lint.md:140 says of the lint verb 'No gate in this repository invokes it'. The Makefile's docs-lint and site-render targets and CI (.github/workflows/ci.yml:505) run `abcd lint docs` and `abcd lint site`, and the commit-msg hook and scripts/check-attribution.sh run `abcd lint outbound`; only bare `abcd lint`, the v1 rule set, runs in no gate. The same page says at lines 71-73 that outbound is wired into CI.
