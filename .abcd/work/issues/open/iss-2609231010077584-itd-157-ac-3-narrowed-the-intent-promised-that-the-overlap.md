---
schema_version: 1
id: "iss-2609231010077584"
slug: "itd-157-ac-3-narrowed-the-intent-promised-that-the-overlap"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/check.go"
---

itd-157 ac-3 narrowed: the intent promised that the overlap gate 'flags the by-links overlap as a red result', and the delivery (PR #555, 64ac809c) widens the count to both arrangements and prints it (internal/surface/cli/site.go:201 'N overlapping bubbles across both arrangements'), but nothing outside the test suite goes red on a non-zero count — 'site build' exits 0 with the number in its summary, and 'site check' (internal/core/site/check.go) never reads Layout.Overlaps, so the make site-render gate passes a record whose by-links picture overlaps. The only red is TestBuildLayoutDoesNotOverlap over the test fixture and TestByLinksNeverOverlaps over synthetic corpora, neither of which sees a user's record. Either 'site check' should refuse on a non-zero published count or the intent's criterion should say the gate is the test suite
