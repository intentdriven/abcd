---
schema_version: 1
id: "iss-2609231553391212"
slug: "promote-back-links-renamed-to-related-issues-and-related-intents"
severity: "major"
category: "documentation"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues/README.md"
---

The promote join's back-links are renamed: an intent's `promoted_from` becomes `related_issues` and a ledger record's `promoted_to` becomes `related_intents`, in the files and the JSON output. Nothing reads the retired names; a record still carrying `promoted_to` is refused. A managed repository runs `abcd capture migrate --apply` once to rewrite them (without `--apply` it only reports).
