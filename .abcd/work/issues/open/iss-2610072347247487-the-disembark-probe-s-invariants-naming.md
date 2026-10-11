---
schema_version: 1
id: "iss-2610072347247487"
slug: "the-disembark-probe-s-invariants-naming"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "the disembark probe lane (run-2610072231362768), 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat"
remedy: "Have the invariants, naming and personas sources read their own brief file as the open-questions and tradeoffs sources now do (an authored file grounds the section, a stub makes it partial), and print each partial section's reason in the pack's per-section file as the coverage report does."
---

The disembark probe's invariants, naming and personas sources ignore their own brief files, unlike open-questions and tradeoffs since the probe lane, and the pack's per-section file does not print a partial section's reason, which coverage.json and the text render now carry.
