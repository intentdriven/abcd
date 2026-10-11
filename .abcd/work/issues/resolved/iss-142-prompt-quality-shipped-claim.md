---
schema_version: 1
id: "iss-142"
slug: "prompt-quality-shipped-claim"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "itd-100 docs-currency review"
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
resolution: "05-prompt-quality.md marks the PQ linter as a design target, matching the code"
impact: internal
resolved_by:
  commit: "b760a67c"
---

prompt-quality brief self-contradiction: 05-prompt-quality.md states PQ001-PQ006 are shipped by spc-8, while 06-lint.md declares the PQ family a design target with no PromptLinter in internal/core/lint — the code sides with 06-lint. An instance of the iss-37 phantom-enforcement class, surfaced by the itd-100 docs-currency review when the crosswalk nearly repeated the phantom claim. Detector: enforcement-claims-are-facts sweep over the brief's Shipped-by annotations; acceptance corpus: the two contradicting lines.