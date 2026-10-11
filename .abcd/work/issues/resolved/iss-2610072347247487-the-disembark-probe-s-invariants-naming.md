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
resolution: "The invariants, naming and personas native adapters now read the section's own brief file through nativeWithBriefFile, as open-questions and tradeoffs do: an authored file grounds the section, a stub is cited and lifts a blank to partial with a reason. The pack's per-section brief file prints a partial section's reason as 'Why partial:', cleaned like every other untrusted field."
impact: fix
resolved_by:
  commit: "aa51824ccdcbc36b188ae2e7f4dc27c5b8af7109"
---

The disembark probe's invariants, naming and personas sources ignore their own brief files, unlike open-questions and tradeoffs since the probe lane, and the pack's per-section file does not print a partial section's reason, which coverage.json and the text render now carry.

## Grounds

- pursued: a repository whose only signal for invariants, naming or personas is an authored brief file probes grounded for that section (a stub probes partial with a reason), and every partial section's packed brief file states its reason; TestNativeConstraintAndPersonaSectionsGroundFromTheirBriefFiles, TestPackSectionFileStatesWhyPartial and TestBriefSectionDocCleansItsPartialReason would fail if an adapter ignored its brief file or a packed partial omitted its reason
