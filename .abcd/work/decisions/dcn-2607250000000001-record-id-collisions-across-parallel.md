---
id: dcn-2607250000000001
slug: record-id-collisions-across-parallel
date: 2026-07-25
---
2026-07-25 — Record-id collisions across parallel branches (iss-115, iss-120)
fixed as a class: one canonical allocator primitive (recordid.MaxAcrossRefs)
mints max+1 over the union of the working tree AND every git ref, so a
committed id on another branch is seen; git-unreadable-over-a-repo degrades
loudly to tree-only, a non-repo mints quietly (no refs to collide). Detection
completes the class — a new spec_id_unique record-lint rule via the shared
validateIDUnique primitive (ADR ids left out of scope: this directive is
iss-/itd-/spc- only). Rejected: id-range leases per programme, mint-at-merge,
and non-sequential/hash ids — all trade away human-readable sequential ids
without a maintainer mandate. The residual uncommitted-mint window (both
branches mint before either commits) is accepted behind the armed detectors on
the merged PR union.
