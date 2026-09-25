---
schema_version: 1
id: "iss-2609250955209513"
slug: "internal-core-lint-fencemask-lint-go-which-about-a-dozen"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

internal/core/lint fenceMask (lint.go), which about a dozen rules use to skip example text (citations, persona, context currency, delivery state, cross-store, subverbs and the rules in lint.go), is a private backtick-only toggle that diverges from mdrecord.Mask. It flips on any line whose trimmed text starts with three backticks. So a tilde-fenced example is linted as live prose, which gives false findings. And a three-backtick line inside a ~~~ block flips the mask on, so the live prose after the block is masked until the next backtick line, and a real finding there is silently missed. Probe at 70daf701: in lines [~~~, three-backtick-go, ~~~, blank, LIVE PROSE LINE], fenceMask marks the live line masked, and in [~~~, example prose, ~~~] it marks the example unmasked. Moving onto mdrecord.Mask is not a mechanical swap. mdrecord also masks HTML comments, so every rule would stop reading commented text, and it recognises only the 0-3 space fence indent. Both are per-rule decisions.
