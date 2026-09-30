---
schema_version: 1
id: "iss-2609301046433372"
slug: "the-loop-s-records-commit-claims-no-assistance-for"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "take the trailer from the lane receipts' reported models in the vendor form the attribution gate accepts (Claude:<model-id>), refuse the landing when a receipt reports none rather than falling back to None, and make the records commit with the repository's hooks running so the commit-msg gate judges it; the pick commit keeps None, its text being computed (ruling owed, review-buildNext (b))"
---

the loop's records commit claims no assistance for model-written text and skips the commit hooks: land.go stamped the landing's records commit Assisted-by: None and made it through pickGit (core.hooksPath=/dev/null), though its diff carries prose a model composed (the implementer receipt's resolution note and grounds, the audit's verdict ingested into the intent), so the disclosure was false and the commit-msg outbound gate never judged it
