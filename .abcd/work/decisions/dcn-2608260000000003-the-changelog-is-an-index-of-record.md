---
id: dcn-2608260000000003
slug: the-changelog-is-an-index-of-record
date: 2026-08-26
---
2026-08-26 — The changelog is an INDEX of record transitions, not a curated
release note: every record reaching a terminal folder earns its line, with no
inclusion judgement. Rationale lives in the record; the changelog is the trace
from record to version, which is why a record does not carry its own release.
Consequences: `impact` keeps its version arithmetic and LOSES its silencing
job; two families the cut ignores today (principles, ADRs) come into scope;
bundling stays, because fewer lines carrying the same information is the
"better" half, while dropping a record for dullness is not. `shipped_in`
(iss-2608241612087533) is reclassified as MIGRATION-ONLY: abcd was built before
RS001 made resolution ride the fixing commit, and a repo managed from its first
commit should never set it. Not yet implemented — captured, so the change lands
behind a record like everything else.
