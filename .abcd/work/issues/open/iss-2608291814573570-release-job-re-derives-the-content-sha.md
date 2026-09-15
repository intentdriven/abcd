---
schema_version: 1
id: "iss-2608291814573570"
slug: "release-job-re-derives-the-content-sha"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: ".github/workflows/release.yml"
---

ultra-v0.6.8 below-cap eff-2/simp-1: the release job in .github/workflows/release.yml re-derives the content sha (go run ./cmd/record-lint --derive-content-sha) instead of consuming the verify job's output — a second compile plus a full-history walk per release, and a second derivation that can in principle diverge from the one the gate passed. Cleanup: emit the sha as a verify job output and consume it.
