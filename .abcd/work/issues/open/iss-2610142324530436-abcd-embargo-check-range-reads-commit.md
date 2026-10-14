---
schema_version: 1
id: "iss-2610142324530436"
slug: "abcd-embargo-check-range-reads-commit"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security-review-feat/embargo-steps-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/check.go"
remedy: "Add a repeatable --ref <name> that checks a ref's name and, for an annotated tag, its message."
---

abcd embargo check --range reads commit messages, added lines and added file names only, never ref names or annotated tag messages, so a branch name or a tag annotation naming an embargoed advisory goes out on a push unchecked.
