---
schema_version: 1
id: "iss-2609260149217402"
slug: "the-reviews-charter-abcd-work-reviews-readme-md-enforcement"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/reviews/README.md"
resolution: "The charter and the gate header now name scripts/check-reviews.sh as where the RD codes are defined."
impact: internal
resolved_by:
  commit: "d831f43b3bfcf2a7fca8e8a87c829362741bcbec"
---

The reviews charter (.abcd/work/reviews/README.md, Enforcement) says the RD codes are defined in .abcd/development/brief/05-internals/06-lint.md, and scripts/check-reviews.sh's header points there for the RD family, but 06-lint.md defines no RD code and says the lint engine carries no numbered catalogue; the codes are defined only in scripts/check-reviews.sh, so both pointers send a reader to a chapter that does not hold them.

## Grounds

- pursued: a reader following the charter's pointer lands on the definitions; shown wrong if a pointer to the RD codes again names a file that holds none
