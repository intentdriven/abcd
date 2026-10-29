---
id: dcn-2610222219297487
slug: a-withdrawn-merge-is-re-armed-in-the
date: 2026-10-22
---
2026-10-22 — A withdrawn merge is re-armed in the form it was taken from (auto-merge with its merge method, or a merge-queue entry), pinned with expectedHeadOid to the head it had when withdrawn; one whose head moved, or that was closed or merged meanwhile, is skipped and named on the tombstone and in peers for its owner to re-arm (iss-2610222014550692, choice 3A over re-arming the current head). Grounds: the re-arm then merges exactly the commits its author armed, never commits pushed while the cut held the branch that nobody armed. Decided by the coordinating session on the product thinker's delegation while the product thinker is away, on the technical facilitator's recommendation.
