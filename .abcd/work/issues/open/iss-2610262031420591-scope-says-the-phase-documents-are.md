---
schema_version: 1
id: "iss-2610262031420591"
slug: "scope-says-the-phase-documents-are"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 166 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:3"
remedy: "none (filed automatically)"
related_intents: [itd-78]
finding_ref: cfk-94f9db24433c81cd
---

Scope says the phase documents are history, but itd-78's graph lint fails on a phase that scopes an intent before its blocker's phase.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:3` — "The phase documents under `roadmap/phases/` are history."
- End B: `.abcd/development/intents/drafts/itd-78-intent-dependency-graph.md:17` — "a phase that scopes an intent before its blocker's phase"

04-scope says no stored unit sits above the intent for sequence: order comes from blocked_by/builds_on and the shelf, and the phase documents under roadmap/phases/ are history. itd-78 lists 'a phase that scopes an intent before its blocker's phase' among its deterministic lint failures, which is a check over a unit that no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 166.
