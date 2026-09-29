---
schema_version: 1
id: "iss-276"
slug: "the-capture-source-enum-lacks-a-value-for-external-contribut"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/capture/capture.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: issueschema.Sources holds ten values (managed-repo joined for inbox reports) and none fits a finding that arrives from an outside pull request's triage, and .abcd/work/intake.md names no --source value. Which value intake uses is owed together with the advisory source of iss-2609020716571275. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

The capture source enum lacks a value for external-contribution intake: validSources in internal/core/capture/capture.go names nine sources, none fitting a finding that arrives from an outside PR's triage; the intake pipeline (.abcd/work/intake.md) will need one