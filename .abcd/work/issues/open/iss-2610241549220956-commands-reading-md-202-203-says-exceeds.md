---
schema_version: 1
id: "iss-2610241549220956"
slug: "commands-reading-md-202-203-says-exceeds"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b03 reading iss-2609210748032488"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reading.md"
remedy: "Reword commands/reading.md:202-203 to say the window eval logs a breach as a warning and this report states it as exceeds_window, never a failure, and record the 2026-10-12 ruling with abcd decide so the page and the test cite one decision; proven by docs-lint and a grep finding no page under commands/ or docs/ that says the eval fails on a breach."
---

commands/reading.md:202-203 says exceeds_window is what the cold-reading eval lane fails on, but since c738e111e the window eval logs a breach as a warning and passes (evals/coldreading_window_test.go:4-10, :73), on the product thinker's ruling of 2026-10-12. That ruling has no decision record; it lives only in the commit message and the test's comment.
