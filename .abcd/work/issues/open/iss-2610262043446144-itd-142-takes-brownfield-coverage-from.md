---
schema_version: 1
id: "iss-2610262043446144"
slug: "itd-142-takes-brownfield-coverage-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1158 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:38"
remedy: "none (filed automatically)"
related_intents: [itd-142, itd-88]
finding_ref: cfk-c4d6254eb411a70d
---

itd-142 takes brownfield coverage from embark probe; itd-88 ships the coverage report as disembark probe

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:38` — "pre-populated from `embark probe` coverage"
- End B: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:18` — "abcd ships `/abcd:disembark probe <repo>` — point it at any repository"

itd-142 fills the brief interview in advance from 'embark probe' coverage. itd-88 makes /abcd:disembark probe the verb that reads a repository and reports what each section can be grounded from. The coverage a brownfield interview needs comes from the disembark probe, so a builder following itd-142 would wire the interview to the wrong probe.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1158.
