---
schema_version: 1
id: "iss-2611180414027708"
slug: "testawideningofverymanypathsisrefusedato"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "review of the heading-text race fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/widen_paths_test.go"
remedy: "Measure it under -race through the load gate; where the bound is its only assertion and the work runs on the caller's goroutine, give internal/core/implement/loop a race_on/race_off pair and skip the test under -race (as TestTheHiddenHTMLReadersStayLinear does), and name it in the two plain-only guards steps, which plainguards_test.go then requires"
---

TestAWideningOfVeryManyPathsIsRefusedAtOnce bounds the validation, cleaning, deduplication and sorting of 200,000 paths at a fixed 4s of wall-clock time and reads no raceEnabled flag, so it runs under -race on the macOS release shards, the same shape that failed the release run's macOS gate through TestHeadingTextStaysLinear; its headroom under -race on a slow runner is not measured
