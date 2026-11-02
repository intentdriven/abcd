---
schema_version: 1
id: "iss-2611022247147314"
slug: "a-merge-queue-drop-is-silent-in-abcd"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/land.go"
remedy: "Have land (bare, on the branch) and peers name a pull request that left the queue unmerged, with the required job that failed or was cancelled and its run, and offer the re-arm."
---

A merge-queue drop is silent in abcd: #1347 left the queue twice on 2026-11-02, once when its merge-group macOS job was cancelled and once on a flaky test, and neither land, peers nor status said the entry had left or why; the coordinator learned it only by polling the forge and reading the run's jobs.
