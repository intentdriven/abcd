---
schema_version: 1
id: "iss-2610151050390126"
slug: "the-detector-wiring-check-only-matches"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.15.0 run 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint"
remedy: "Run the release-gate detector's fill over the committed manifest in a test (through node where it is available, or by porting the substitution) and assert the rendered context names every roster entry with no placeholder left, instead of asserting that two substrings occur in the script's source."
---

The check that the release-gate detector fills its ${commands} and ${agents} placeholders only asserts that two substrings occur in brief-surface-crosscheck.js. With the substitution code commented out the check still passes, so a detector that sends its checkers an unfilled prompt would ship green (review note on chore/merge-hotspots, 2026-10-15).
