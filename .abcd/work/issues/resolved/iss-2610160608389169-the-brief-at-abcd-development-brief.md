---
schema_version: 1
id: "iss-2610160608389169"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
remedy: "Add dcn to senderIDRe in internal/core/report/inbox.go:714 and to the family list at 30-inbox.md:105-112."
resolution: "The inbox's promotion rewrite now takes its families from recordid.CitedFamilies, the list CitedIDRe is built from, so a sender's dcn-N is written as one word like adr, itd, iss and spc; the inbox brief names dcn and says the list is the cited grammar's."
impact: fix
---

The brief at .abcd/development/brief/04-surfaces/30-inbox.md:105-112 says a sender's id in a record family abcd resolves (adr, itd, iss, spc) is written as one word, so it never cites abcd's own record. abcd also resolves dcn ids (recordid.CitedIDRe; record-lint's prose_citation_resolves, internal/core/lint/prosecitations.go:138), but the promotion rewrite's senderIDRe (internal/core/report/inbox.go:714) covers only adr|itd|iss|spc, so a sender's dcn-N is copied as written and cites abcd's own dcn-N.
