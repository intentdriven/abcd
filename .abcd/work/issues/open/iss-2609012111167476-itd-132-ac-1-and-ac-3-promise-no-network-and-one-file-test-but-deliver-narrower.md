---
schema_version: 1
id: "iss-2609012111167476"
slug: "itd-132-ac-1-and-ac-3-promise-no-network-and-one-file-test-but-deliver-narrower"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-132-hook-binary-to-persistent-data-dir.md"
---

Two itd-132 criteria are met with a narrower reading than their words. ac-1 says provisioning a fresh plugin root from the cache involves 'no network fetch'; delivered is no artefact fetch: the online equal-tag path still resolves releases/latest once and fetches checksums.txt to authenticate the cache (adr-46 decision 3), and only the offline path is network-free. ac-3 says a steady-state hook costs 'one file test and no network'; SessionStart always runs bootstrap.sh, whose cache-mode fast path adds a .binary-meta stat, and the cache-mode steady state of a cache-provisioned root has no direct zero-network test. Both are deliberate designs recorded in adr-46; the criteria's wording is what lags. Surfaced by the itd-132 fidelity audit (receipt rcp-acde3e9ce729). Fix is a record edit or a test, not a code change.
