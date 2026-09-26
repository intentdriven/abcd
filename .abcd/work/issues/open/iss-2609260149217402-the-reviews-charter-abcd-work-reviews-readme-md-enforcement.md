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
---

The reviews charter (.abcd/work/reviews/README.md, Enforcement) says the RD codes are defined in .abcd/development/brief/05-internals/06-lint.md, and scripts/check-reviews.sh's header points there for the RD family, but 06-lint.md defines no RD code and says the lint engine carries no numbered catalogue; the codes are defined only in scripts/check-reviews.sh, so both pointers send a reader to a chapter that does not hold them.
