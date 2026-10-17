---
id: dcn-2608050000000008
slug: iss-118-decisions-md-and
date: 2026-08-05
---
2026-08-05 — iss-118: DECISIONS.md and ACKNOWLEDGEMENTS.md gain `merge=union` in `.gitattributes`, the same remedy already in place for CHANGELOG.md. Both are the identical shape — append-only ledgers of anonymous, dated, order-independent entries that never need identity — so a concurrent-append conflict has nothing to actually disagree on; the union driver keeps both sides instead of stopping the merge. Prompted directly by the bug-hunt loop's `bugfix/iss-184-…` branch hitting exactly this conflict against `.abcd/work/DECISIONS.md` on PR #199, which iss-118 had already diagnosed (filed after an earlier merge hit the same conflict) but left unresolved pending this design call. Full atomicisation (per-decision records, a new id family, an armed uniqueness detector) was the other option iss-118 named; not pursued here as disproportionate to a minor, low-traffic hotspot. Impact recorded as `internal`, no CHANGELOG entry — matching the iss-80 precedent for a process/tooling change with no user-facing behaviour.
