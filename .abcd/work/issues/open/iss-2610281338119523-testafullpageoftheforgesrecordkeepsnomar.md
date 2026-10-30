---
schema_version: 1
id: "iss-2610281338119523"
slug: "testafullpageoftheforgesrecordkeepsnomar"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "preflight of PR 1164, 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/status_forge_page_test.go"
remedy: "Let the test raise the status fetch's budget: claim.pusherBudget (10 s, internal/core/claim/status.go) bounds both the forge lookup and the git rev-list that places its pushes, and the cli test has no way to set it, so give the claim package a test-only setter (or a budget field on its options) that the test raises to a minute; the production budget stays 10 s."
---

TestAFullPageOfTheForgesRecordKeepsNoMarkForWhatItDidNotCover (internal/surface/cli/status_forge_page_test.go) fails when the machine is slow enough. It runs abcd status with the production budget for each forge lookup, and in a preflight's race stage on 2026-10-28, at a one-minute load near 180, a git rev-list ran past that budget. So the reply said the forge's record was not placed in full, where the test wants it said to be read in part. The branch under test changed only records, and the same test passed in the stage before.
