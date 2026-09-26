---
schema_version: 1
id: "iss-2609261215160156"
slug: "abcd-spc-n-on-a-bundle-s-shared-spec-reads-the-first-member"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd34"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/record/record.go"
---

abcd <spc-N> on a bundle's shared spec reads the first member alone (describeSpec in internal/core/record/record.go): Links names only the intent: back-link, the closed branch's 'stays planned until' and 'the linked intent is' lines read that one member, and the readiness gate runs on it alone, so member 2 is never mentioned and, after member 1 is superseded, the page names a superseded record as the linked intent.
