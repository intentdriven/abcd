---
id: dcn-2611201453528321
slug: a-bundle-lane-that-fails-as-a-whole-its
date: 2026-11-20
---
2026-11-20 — A bundle lane that fails as a whole, its fix rounds spent or stuck, is split rather than handed back: no member is tombstoned, each member's reservation is released plainly and its judgement kept, and each is taken up again as a bundle of one that this drain never bundles again, so only an issue that fails its own fix rounds is handed back, for itd-2611201450052326. Grounds: batching merge queues retry a failed batch in smaller parts (bors-ng's batch bisection, GitHub's merge queue removing a failing group, Zuul's dependent pipelines) instead of failing every change in it. Decided by the technical facilitator on delegation, 20 Nov 2026.
