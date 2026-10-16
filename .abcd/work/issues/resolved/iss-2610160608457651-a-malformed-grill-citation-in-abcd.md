---
schema_version: 1
id: "iss-2610160608457651"
slug: "a-malformed-grill-citation-in-abcd"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/record.go"
remedy: "Add dcn-N to the refusal message at internal/core/ideate/record.go:497."
resolution: "The ideate refusal of a malformed grill citation now names the citable forms from recordid.CitedIDForms, derived from the same list CitedIDRe is built from, so it lists dcn-N with the other four families."
impact: fix
---

A malformed grill citation in abcd ideate is refused with '(want adr-N, itd-N, iss-N, or spc-N)' (internal/core/ideate/record.go:497), but CitedIDRe (internal/core/recordid/resolve.go:41) accepts dcn-N as well, commands/ideate.md lists dcn-N as citable, and the brief (.abcd/development/brief/04-surfaces/18-ideate.md:57-58) says a decision record is cited by its dcn- id like any other record. The refusal tells the caller dcn ids are not citable.
