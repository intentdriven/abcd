---
schema_version: 1
id: "iss-2611040712399867"
slug: "a-fixed-test-budget-can-hide-behind-an"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.16.1 cut, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/gate_test.go"
remedy: "Have one nightly run shuffle package and test order with go test -shuffle, to surface tests that lean on what ran before."
---

A fixed test budget can hide behind an earlier test warming something up: TestAPreflightPastItsBoundIsKilledWithItsGroup's two-second bound passed for months because another test ran make first, and failed every time once the macOS suite was sharded, since a cold make on a fresh runner goes through xcrun. Sharding is likely to surface more of these.
