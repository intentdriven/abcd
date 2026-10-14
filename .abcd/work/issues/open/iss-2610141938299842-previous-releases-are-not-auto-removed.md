---
schema_version: 1
id: "iss-2610141938299842"
slug: "previous-releases-are-not-auto-removed"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "v0.14.1 release follow-up"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/retention.go"
remedy: "Change the retention rule (brief launch chapter, draft itd-70, ComputeRetention) to keep only the newest release before 1.0, and have the release workflow delete every earlier Release and its assets after the new one publishes, keeping tags and history, reporting what it removed."
---

Before 1.0 a release cut leaves every earlier release on offer: nothing removes the previous Releases and their downloads, and the recorded rule keeps the newest release of each version line rather than only the newest release. The product thinker ruled on 2026-10-14 that nothing older than the newest release is offered before 1.0, and the eight earlier Releases (v0.11.1 to v0.14.0) were deleted by hand that day, tags kept. iss-282 records the missing executor for the older newest-per-line rule.
