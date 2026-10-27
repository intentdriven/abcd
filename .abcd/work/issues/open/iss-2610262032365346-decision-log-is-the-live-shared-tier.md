---
schema_version: 1
id: "iss-2610262032365346"
slug: "decision-log-is-the-live-shared-tier"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 293 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:805"
remedy: "none (filed automatically)"
finding_ref: cfk-7f3626302a5444b7
---

'decision log' is the live shared-tier store in configuration, but in ahoy it is a legacy single file kept as frozen history

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:805` — "current orientation, the append-only decision log, the external-contribution runbook"
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:295` — "any single decisions log the repository already keeps left untouched as frozen history"

The configuration chapter's tier table says the shared work tier holds an append-only decision log. The ahoy chapter seeds a decisions folder with one file per decision, and treats any single decisions log as frozen history that is left untouched. One name points to two different things: the current decision store in one chapter and the retired legacy file in the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 293.
