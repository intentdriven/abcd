---
schema_version: 1
id: "iss-2609251507443137"
slug: "the-reading-exclusion-floor-admits-two-line-shapes-every"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
---

The reading exclusion floor admits two line shapes every renderer reads and its line split does not, and the excluded section travels with a nil error. A byte-order mark before an ATX heading on line 0: floorATXRe is matched against the raw line with no frontmatter.TrimBOM, so the BOM hides the heading from the verifier while a renderer drops it. And a lone carriage return: CommonMark treats a CR not followed by LF as a line ending, but the floor splits only on newline, so a CR-only document (or a lone CR inside a CRLF one) is one line to the floor and many to a renderer. Both leak at base and on fix/one-fence-rule. Each is a one-line refusal.
