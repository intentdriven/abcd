---
schema_version: 1
id: "iss-2610262031196440"
slug: "03-mental-model-says-a-bundle-s-shared"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 115 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:65"
remedy: "Edit 03-mental-model.md (bundle-member bullet): the shared spec carries `intents:` (every member) and `bundle:` beside the scalar `intent:`, not a list-valued `intent:`."
finding_ref: cfk-9b175bf0febf1c2c
---

03-mental-model says a bundle's shared spec carries intent: [itd-A, itd-B]; the verification matrix says it carries intents: and bundle:.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:65` — "creates a single spec with `intent: [itd-A, itd-B]`"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:80` — "`intents:` and `bundle:` on the spec"

03-mental-model says /abcd:intent plan, given several intent ids, 'creates a single spec with `intent: [itd-A, itd-B]`'. The matrix's itd-34 row names the shared spec's fields as '`intents:` and `bundle:` on the spec'. Two field names are given for the one link from a bundle's spec to its members, so a spec written from one document carries a key the other does not read.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 115.
