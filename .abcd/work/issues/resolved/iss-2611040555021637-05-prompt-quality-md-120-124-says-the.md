---
schema_version: 1
id: "iss-2611040555021637"
slug: "05-prompt-quality-md-120-124-says-the"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/05-prompt-quality.md"
remedy: "Correct the list of entries, or state the delta in those entries and keep the sentence."
resolution: "The prompt-quality chapter's calibration-band bullet now names all ten agent-changelog entries that never record the delta as unmeasured (the five 2026-08-19 entries, cold-reading-widening 0.2.0, cold-reading-entailment 0.1.3, cold-reading-detection 0.1.4, and the renames intent-auditor 0.1.1 and lifeboat-reviewer 0.1.1), and a lint-package test holds the bullet to the changelog."
impact: internal
resolved_by:
  commit: "ede9ee0dfee9709b913e23abb05af91deb9e22ff"
---

05-prompt-quality.md:120-124 says the entries that state no delta anywhere are the five prompts hardened on 2026-08-19 and one cold-reading bump. In .abcd/development/agents/CHANGELOG.md, cold-reading-widening 0.2.0 states none, cold-reading-entailment 0.1.3 and cold-reading-detection 0.1.4 only point at widening's entry, and the renames intent-auditor 0.1.1 and lifeboat-reviewer 0.1.1 state none in the entry or its section preamble.

## Grounds

- pursued: the brief's list matches the entries in .abcd/development/agents/CHANGELOG.md whose text and section preamble both lack 'unmeasured'; TestBriefCalibrationBandNamesEveryEntryThatStatesNoDelta would show it wrong by failing on any silent entry the bullet does not name or a count that differs.
