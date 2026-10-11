---
schema_version: 1
id: "iss-2610090642371836"
slug: "when-a-release-cut-is-refused-for-an"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/emit.go"
remedy: "Raise the open-records refusal as its own entry whenever FindingGuard.Uncommitted is non-empty, carrying UncommittedReason and naming the paths, and test the three-way case (unfixed, deleted and uncommitted together) watched fail first."
---

When a release cut is refused for an unfixed or deleted finding and also for an issue record under open/ that differs from HEAD, the open-records refusal gets no refusal entry of its own: emit.go raises the unfixed and deleted refusals and adds the guard's verdict as a backstop only when both lists are empty, so the uncommitted-records reason survives only in the findings verdict. A front door that acts on the refusal list fixes the findings, retries, and only then meets the second refusal.
