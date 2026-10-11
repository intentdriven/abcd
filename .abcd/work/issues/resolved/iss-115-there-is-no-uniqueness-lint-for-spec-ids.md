---
schema_version: 1
id: "iss-115"
slug: "there-is-no-uniqueness-lint-for-spec-ids"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-95/itd-96 merge to main"
found_at: ".abcd/development/specs"
resolution: "No spec-id uniqueness lint and local-tree-only minting fixed as a class: refs-union allocator (recordid.MaxAcrossRefs) folded into all three id families, plus the new spec_id_unique record-lint rule."
impact: fix
---

There is no uniqueness lint for spec ids, so two concurrent branches always mint the same spc-N. abcd mints the next free id from the local tree, so any two branches cut from the same base collide silently: the itd-73/itd-67 programme and the itd-95/itd-96 programme both minted spc-10 and spc-11 on 2026-07-21. The collision is invisible on each branch and only appears on merge, where it breaks the bidirectional intent-spec link in BOTH directions at once — 'itd-95 names spc-10, but spc-10 claims itd-73' and 'itd-67 names spc-11, but spc-11 claims itd-96' — because whichever spec wins the store lookup leaves the other intent's link dangling. iss-74 added an issue-id uniqueness lint; specs need the same guard, and record-lint is the natural home so CI catches it on the PR rather than a human catching it at merge. Consider also whether minting should reserve ids in a way concurrent branches cannot duplicate.