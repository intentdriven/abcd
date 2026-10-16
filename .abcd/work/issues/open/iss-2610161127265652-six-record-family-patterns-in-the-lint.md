---
schema_version: 1
id: "iss-2610161127265652"
slug: "six-record-family-patterns-in-the-lint"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "PR5 dcn sweep, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/recordid/resolve.go"
remedy: "Derive each from recordid.CitedFamilies() (adding rdi where the pattern admits it), and test that a family added there reaches each."
---

Six record-family patterns in the lint and site packages keep their own hand-written copy of the cited families (adr, itd, iss, spc, dcn) instead of reading recordid.CitedFamilies: claimedHandleRe in internal/core/lint/crossstore.go:57, proseCitationRe in internal/core/lint/prosecitations.go:138, recordHandleRe and recordHandleFullRe in internal/core/lint/schema.go:60 and :62, the recordHandleKinds message in internal/core/lint/schema.go:73, and bodyHandleRe in internal/core/site/recordjson.go:34. Each includes dcn today, but each is a copy the next family added to recordid's list would miss, which is how the inbox rewrite and four refusal messages came to omit dcn (iss-2610160608389169, iss-2610160608457651).

Out of scope by design: exemptEdgeRe in internal/core/lint/reciprocal.go also hand-lists families and omits dcn, deliberately; the exempt_edges list it reads is frozen to its 155 itd edges (TestBuildsOnExemptionsNeverGrow), so a dcn entry could never be valid there. Its comment says so.
