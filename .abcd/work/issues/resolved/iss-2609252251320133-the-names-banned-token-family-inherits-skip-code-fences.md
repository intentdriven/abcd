---
schema_version: 1
id: "iss-2609252251320133"
slug: "the-names-banned-token-family-inherits-skip-code-fences"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/config.go"
resolution: "A names/ banned token with no skip_code_fences declaration now reads inside fenced code (the documentation family keeps skipping fences), and the banlist surface chapter states the default."
impact: fix
resolved_by:
  commit: "1bf76c8f"
---

The names/ banned-token family inherits skip_code_fences' default of true, so a banned name inside a fenced code block in AGENTS.md or any markdown under roots or name_roots passes the name gate. The default is a writing rule for the documentation family (an example in a fence is not prose); the name gate is documented as reaching the whole public surface, and a fence is published as readily as prose. Nothing records fence masking as deliberate for the names family.

## Grounds

- pursued: a banned name inside a fenced block in any markdown the name gate reaches is reported; a fenced name in AGENTS.md or under docs passing lint with no finding (TestNameBansReadInsideCodeFences) would show it wrong
