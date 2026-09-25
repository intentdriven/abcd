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
---

The names/ banned-token family inherits skip_code_fences' default of true, so a banned name inside a fenced code block in AGENTS.md or any markdown under roots or name_roots passes the name gate. The default is a writing rule for the documentation family (an example in a fence is not prose); the name gate is documented as reaching the whole public surface, and a fence is published as readily as prose. Nothing records fence masking as deliberate for the names family.
