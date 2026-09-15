---
schema_version: 1
id: "iss-2608311206480573"
slug: "the-ingest-verb-s-missing-body-field-guard-is-mutation-vacuo"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "TestMissingBodyFieldRefusesTheItem walks every declared detection body field in both the empty and the absent form; removing the guard turns it red."
impact: internal
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb's missing-body-field guard is mutation-vacuous: removing it (checkItem's len(missing) > 0 branch) leaves every test in internal/core/reading green. No case builds an item that carries the position's own body but leaves one declared field empty or absent, because TestWrongPositionBodyIsUndecodable supplies a FOREIGN body and the unknown-key check fires first. The guard is the one that would catch a reading returning a partial item at its own position.

## Grounds

- pursued: a guard proved by mutation rather than by a passing test, so the case fails without it; a later mutation run finding it survived again would show this wrong
