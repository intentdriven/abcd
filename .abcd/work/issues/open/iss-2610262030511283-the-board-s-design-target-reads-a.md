---
schema_version: 1
id: "iss-2610262030511283"
slug: "the-board-s-design-target-reads-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 54 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:84"
remedy: "none (filed automatically)"
finding_ref: cfk-e19323c5109fc692
---

The board's design target reads a recent-logbook section; the naming register retires the logbook and forbids re-minting it

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:84` — "It is retired and must not be re-minted"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:543` — "**Recent logbook**: the last five entries by modification time, the render descending exactly one bounded level into each category directory"

08-abcd's staged six-section board depends on reading the five most recent logbook entries from its category directories, while 04-naming says .abcd/logbook/ is retired, must not be re-minted, and a test fails the build if source names it. The design depends on a location the constraints forbid from existing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 54.
