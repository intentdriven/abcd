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
resolution: "abcd <spc-N> reads a spec with several members through Spec.Members: the links carry intents, a superseded member is passed over and named, and the closed and open moves read every member still in force."
impact: fix
resolved_by:
  commit: "a1c5029fe0edff4ee924f1dac962211b36d9a638"
---

abcd <spc-N> on a bundle's shared spec reads the first member alone (describeSpec in internal/core/record/record.go): Links names only the intent: back-link, the closed branch's 'stays planned until' and 'the linked intent is' lines read that one member, and the readiness gate runs on it alone, so member 2 is never mentioned and, after member 1 is superseded, the page names a superseded record as the linked intent.

## Grounds

- pursued: we expect a bundle spec's page to name every member and never a superseded one as the linked intent; shown wrong if the page for a bundle spec omits a live member or defers to a member that is not the one failing readiness
