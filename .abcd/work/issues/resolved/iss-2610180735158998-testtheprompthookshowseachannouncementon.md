---
schema_version: 1
id: "iss-2610180735158998"
slug: "testtheprompthookshowseachannouncementon"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "review of the fix for iss-2610180712063858"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/status_hook_test.go"
remedy: "Split the test: keep the one-second return check under the shipped budget, and assert the announcement's rendering and show-once behaviour under a stretched read budget."
resolution: "Split the test: TestThePromptHookReturnsInsideItsSecond keeps the shipped one-second bound over three tries in fresh checkouts, logging each and failing when two run over; the show-once test runs under budgets of its own; both decide origin was never reached from a marker file the origin's command leaves."
impact: internal
---

TestThePromptHookShowsEachAnnouncementOnce (internal/surface/cli/status_hook_test.go) runs the prompt hook at the shipped one-second budget with the status read capped at its shipped 600 ms, and asserts the announcement is shown, so a loaded runner can spend the read's share and fail it the way iss-2610180712063858 failed. Its budget is part of what it tests (the hook returns inside its second with an origin that would hang), so stretching the budget there is a choice about what the test proves, not a tolerance fix.
