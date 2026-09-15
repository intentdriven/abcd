---
schema_version: 1
id: "iss-2608291814561674"
slug: "settle-re-sorts-placed-bubbles-per-ray"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/site/layout.go"
resolution: "clearOutward no longer sorts its intervals (the fixed-point walk lands on the least uncovered rho whatever their order) and reuses one scratch buffer per pass at all three call sites; BenchmarkSettle over 600 bubbles drops from 92 ms and 281k allocs to 13.7 ms and 6 allocs, and layout_test.go stays green"
impact: internal
---

ultra-v0.6.8 C10: clearOutward in internal/core/site/layout.go allocates a fresh forbidden slice and sort.SliceStable's it on every ray, and settle calls it up to 1+2*settleRays times per bubble, twice per build via byLinks. The fixed-point loop jumps rho to the hi of any interval covering it until none does, and every jump only crosses points that interval covers, so it converges to the least uncovered point at or beyond the start whatever the interval order — the sort is not needed for correctness. Fix: drop the sort and reuse one scratch slice across rays; benchmark before and after.
