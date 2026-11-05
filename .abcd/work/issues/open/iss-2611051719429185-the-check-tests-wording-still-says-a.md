---
schema_version: 1
id: "iss-2611051719429185"
slug: "the-check-tests-wording-still-says-a"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "cmd/check-tests/main.go"
remedy: "Correct the three places to say a cut may take the fresh form and the hosted run reuses nothing."
---

Test reuse's wording still says a release cut reuses nothing, though that is now a person's option: the `-fresh` flag text (`cmd/check-tests/main.go:48`), the run text (`internal/testreuse/run.go:171`) and a lint test (`internal/core/lint/testreuse_test.go:64`).
