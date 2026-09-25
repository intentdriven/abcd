---
schema_version: 1
id: "iss-2608210737264758"
slug: "forge-allocator-adapter-has-no-ledger-seat"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-114 ship review"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Give the forge-backed allocator adapter (deferred by adr-45 ruling 4) a planned intent?"
---

The optional forge-backed allocator (itd-114 ac-5) is deferred by recorded ruling — spc-33 names it a later adapter behind the same mint seam, adr-45 ruling 4 fixes its posture (allocates and never stores per itd-129's ledger-canonical line; offline falls back to native loudly) — but no open work item holds the seat, so nothing tracks building it. The fidelity verdict on itd-114 carries the criterion as NOT_MET/deferred-by-ruling; this capture is the deferred work's ledger seat, builds on itd-129's adapter seam when that lands