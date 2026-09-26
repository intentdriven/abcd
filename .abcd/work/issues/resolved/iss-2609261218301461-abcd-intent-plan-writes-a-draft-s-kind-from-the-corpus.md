---
schema_version: 1
id: "iss-2609261218301461"
slug: "abcd-intent-plan-writes-a-draft-s-kind-from-the-corpus"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/lifecycle.go"
resolution: "Plan parses the draft from the bytes it reads under the intent mint lock and judges those, so a kind a reclassify wrote in the window is kept rather than overwritten."
impact: fix
resolved_by:
  commit: "1c8f0ed12eb00e75142c757ebc3412b7050891cd"
---

abcd intent plan writes a draft's kind from the corpus loaded before the intent mint lock (Plan in internal/core/intent/lifecycle.go reads it.Kind and it.SpecID from the pre-lock corpus while the bytes it rewrites are read under the lock), so a reclassify landing in the window that makes the draft a bundle-member is overwritten: the planned record carries kind: standalone beside bundle: <name>, a pairing no verb otherwise writes.

## Grounds

- pursued: we expect Plan's result to equal the result of the reclassify and the plan run one after the other; shown wrong if a planned record ever carries kind: standalone beside a bundle name
