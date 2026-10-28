---
schema_version: 1
id: "iss-2608210737264758"
slug: "forge-allocator-adapter-has-no-ledger"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "itd-114 ship review"
deferred_after: "v0.11.1"
deferral_reason: "decide later (product thinker, 2026-09-29)"
remedy: "Waits on ruling J12 (forge-backed record numbering, parked as decide later on 2026-09-29) and on the forge adapter seam of itd-129: if built, add an opt-in forge allocator behind recordid.Minter (internal/core/recordid/mint.go), chosen per record family in .abcd/config.json, that has the forge allocate each id in the native timestamp-numeric format, writes the record only to the committed ledger, and on any network or auth failure falls back to the native mint and names the path it took (adr-45 ruling 4), proven by tests against a fake forge that an online mint is allocated there with nothing else stored on it, that an offline mint falls back with the notice, and that an unconfigured mint is unchanged; if declined, close this record as wontfix naming the native mint (adr-45 ruling 1) as sufficient. (The remedy opens Waits on because J12 is in force: a decided remedy would make the drain build what the product thinker parked.)"
---

The optional forge-backed allocator (itd-114 ac-5) is deferred by recorded ruling — spc-33 names it a later adapter behind the same mint seam, adr-45 ruling 4 fixes its posture (allocates and never stores per itd-129's ledger-canonical line; offline falls back to native loudly) — but no open work item holds the seat, so nothing tracks building it. The fidelity verdict on itd-114 carries the criterion as NOT_MET/deferred-by-ruling; this capture is the deferred work's ledger seat, builds on itd-129's adapter seam when that lands