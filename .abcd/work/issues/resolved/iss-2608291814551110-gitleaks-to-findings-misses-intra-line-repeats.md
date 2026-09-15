---
schema_version: 1
id: "iss-2608291814551110"
slug: "gitleaks-to-findings-misses-intra-line-repeats"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/adapter/gitleaks/gitleaks.go"
resolution: "toFindings advances a cursor past each match and keys one Finding per (line, column, value) span; a test feeds two same-value reports on one line and asserts two distinct columns and a clean redaction"
impact: fix
---

ultra-v0.6.8 A3: gitleaks toFindings in internal/adapter/gitleaks/gitleaks.go uses strings.Index(ln, needle) per line and never advances a cursor, so when gitleaks reports the same secret twice on one line both reports yield a Finding at the first occurrence; dedupFindings collapses them, sealLine masks one span, and the second occurrence survives verbatim. Retry logs and request+response echoes are exactly this shape. Fix: advance a cursor past each match so every occurrence on the line becomes its own Finding.
