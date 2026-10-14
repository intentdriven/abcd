---
schema_version: 1
id: "iss-2610121623088421"
slug: "fsutil-casefoldingfs-decides-whether"
severity: "minor"
category: "observation"
source: "impl-review"
found_during: "security review round 3 of fix/quota-deadline-and-folded-branch, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
remedy: "probe the volume (create a probe name and look it up in another case, as pathconf _PC_CASE_SENSITIVE does on darwin) where the answer decides a deletion, keeping the platform guess only where over-matching is the safe side"
---

fsutil.CaseFoldingFS decides whether names fold by the operating system (darwin or windows), not by the volume a path is on. A case-sensitive APFS volume on macOS only over-matches, the safe side; a case-insensitive mount on Linux (SMB, exFAT) under-matches, so every check that folds through it (the loop's checked-out check, the FoldPath callers) reads two spellings of one name there as distinct.
