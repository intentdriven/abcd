---
schema_version: 1
id: "iss-2610262030322001"
slug: "context-keeps-intent-ship-as-a-design"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 12 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:11"
remedy: "none (filed automatically)"
finding_ref: cfk-440184740afdc1cd
---

Context keeps `intent ship` as a design target that drives a spec to completion; the scope page says closing the spec ships the intent and there is no separate ship command

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:11` — "`ship` drives that spec to completion (or the full pipeline if from drafts/)"
- End B: `.abcd/development/brief/01-product/04-scope.md:40` — "closing the spec is what ships the intent, and there is no separate ship command."

The context page describes an `intent ship` sub-verb that drives the spec to completion, or the full pipeline from drafts/, and keeps it among the remaining design targets. Under the truth rule a design target is decided. The scope page says closing the spec is what ships the intent and there is no separate ship command. The record cannot both plan a ship command and rule one out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 12.
