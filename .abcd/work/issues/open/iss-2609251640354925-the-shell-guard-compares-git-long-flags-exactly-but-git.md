---
schema_version: 1
id: "iss-2609251640354925"
slug: "the-shell-guard-compares-git-long-flags-exactly-but-git"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

The shell guard compares git long flags exactly, but git accepts any unambiguous prefix of a long option, so the no-verify flag of commit and push, and the with-lease and if-includes force flags of push, each spelled a few letters short, run as the full flag and are allowed. Found by review-guard finding 4 (pre-existing).
