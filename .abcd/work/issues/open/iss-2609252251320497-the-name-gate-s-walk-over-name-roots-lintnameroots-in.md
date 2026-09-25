---
schema_version: 1
id: "iss-2609252251320497"
slug: "the-name-gate-s-walk-over-name-roots-lintnameroots-in"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

The name gate's walk over name_roots (lintNameRoots in internal/core/lint/lint.go) skips a file silently when the post-resolve os.Stat fails (for example a file under a directory that can be listed but not searched), so a leak gate passes a file it never read and says nothing. The adjacent guarded read fails loud; the Stat failure should too (loud staging).
