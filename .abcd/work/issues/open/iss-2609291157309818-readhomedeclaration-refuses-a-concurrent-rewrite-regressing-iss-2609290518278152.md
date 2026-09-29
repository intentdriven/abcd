---
schema_version: 1
id: "iss-2609291157309818"
slug: "readhomedeclaration-refuses-a-concurrent-rewrite-regressing-iss-2609290518278152"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home.go"
---

fsutil.ReadHomeDeclaration still refuses a home-scoped declaration as replaced between its vetting and its read when a concurrent abcd renames a new version into place inside the lstat-to-open window: the re-vetting fix for iss-2609290518278152 (b342b2b29) went into the path-based ReadDeclaration, and the integration merge that landed it (24e78506b) kept the descriptor-based readDeclarationIn from a07ad672f, which every home-scoped reader (layered.Load, oracle, credential, rules, statusline) goes through and which refuses a replacement on sight; so iss-2609290518278152 regressed and TestConcurrentConnectsKeepEveryKeyAndBlock failed the Linux CI leg of PR 748 with ErrDeclarationSwapped from ~/.abcd/config.json
