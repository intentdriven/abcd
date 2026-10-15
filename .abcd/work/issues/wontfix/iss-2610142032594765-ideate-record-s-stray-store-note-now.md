---
schema_version: 1
id: "iss-2610142032594765"
slug: "ideate-record-s-stray-store-note-now"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "PR #991 review pair"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ideate.go"
remedy: "Leave it: the defect that laid such stores is fixed and every release since has reported them; a stray-store note belongs to the store the verb writes now."
wontfix_reason: "The defect that laid stray research/notes stores below a checkout root was fixed in September 2026 and every release since has reported such stores; ideate now writes research/ideas, so its stray-store note looks for that store, and keeping a check for a retired store would guard a defect that no longer exists."
---

ideate record's stray-store note now looks for a stray research/ideas store below the checkout root, so a stray research/notes store that the long-fixed subdirectory defect (iss-2609091729516940) left below a root is no longer reported when ideate runs. Raised by the Opus reviewer of PR #991.

## Grounds

- declined: The defect that laid stray research/notes stores below a checkout root was fixed in September 2026 and every release since has reported such stores; ideate now writes research/ideas, so its stray-store note looks for that store, and keeping a check for a retired store would guard a defect that no longer exists.
