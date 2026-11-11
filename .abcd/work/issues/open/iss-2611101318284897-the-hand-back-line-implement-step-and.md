---
schema_version: 1
id: "iss-2611101318284897"
slug: "the-hand-back-line-implement-step-and"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Name what stopped the lane (stuck, design finding, closed) whenever the verdict is empty, and never print an empty field; a test renders a hand-back with no verdict and finds a kind named."
---

The hand-back line implement step and implement receipt print leaves the hand-back's kind blank when the hand-back carries no verdict. renderStepResult (internal/surface/cli/build.go) prints the verdict field where the kind belongs, so on 7 Nov a lane's hand-back read stopped as, then two spaces, then after 0 fix round(s).
