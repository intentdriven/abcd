---
schema_version: 1
id: "iss-2610222212301498"
slug: "route-parity-test-checks-the-prediction"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "review pair of feat/route-names-runner-model-r2, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/route_parity_test.go"
remedy: "Add a parity case that drives the real validateStage and implementStage on a fixture lane and compares the role each hands out with predictHand's, so a change to either side fails the test"
---

The --route prediction parity test (internal/core/implement/loop/route_parity_test.go) checks predictHand against fake stage bodies, not the real validateStage and implementStage. If validateStage's hand-out order changed (openRound's roles reordered, or a Stay or Goto returned before its Await), a --route for a reviewer role would be accepted and then silently not used: no refusal, and no route_unused entry, since nothing is unsure without a landing ahead. The real bodies agree with the prediction today; the gap is that no test would notice them drifting apart.
