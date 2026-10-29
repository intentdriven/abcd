---
schema_version: 1
id: "iss-2610262033319993"
slug: "the-intent-chapter-defers-the-honoured"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 416 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:553"
remedy: "none (filed automatically)"
finding_ref: cfk-f0058a3c774fe96b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033223107. Same End A (05-intent.md line 553 says the gap buckets are deferred); End B is the agent catalogue, which is correct."
duplicates: [iss-2610262033223107]
---

The intent chapter defers the honoured/diverged/missing prose audit, while the agent catalogue says the shipped fidelity role returns it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:553` — "The broader **press-release prose review** (the `honoured` / `diverged` / `missing` buckets below) and other prose/terminology/PRD-fidelity outputs are **deferred** to a later spec."
- End B: `.abcd/development/brief/05-internals/01-agents.md:141` — "per-criterion verdicts plus a three-bucket prose audit (honoured / diverged / missing), each claim carrying a cited evidence pointer."

05-intent states that the press-release prose review, with its honoured, diverged and missing buckets, is deferred to a later spec. 01-agents describes the shipped single-document fidelity role as returning per-criterion verdicts plus that same three-bucket prose audit, each claim cited. Both documents describe the same shipped role, so they cannot both be right about whether the gap audit is part of its verdict.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 416.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033223107. Same End A (05-intent.md line 553 says the gap buckets are deferred); End B is the agent catalogue, which is correct.
