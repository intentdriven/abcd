---
schema_version: 1
id: "iss-2610262035213062"
slug: "the-guard-chapter-describes-agents-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 565 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md:385"
remedy: "Rewrite the premise at 27-implement.md:129: abcd's runners start agents (34-build.md, internal/core/runner), so say the session ceiling counts the agent_start lines a session declares, with runner-started agents held to the run's lane slots instead of those lines."
finding_ref: cfk-297b72709229390c
---

The guard chapter describes agents abcd starts through its claude runner, but the implement chapter counts agents against the ceiling on the premise that abcd runs no agent

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/17-guard.md:385` — "An agent abcd starts through its claude runner has no terminal, so nobody can"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:129` — "abcd runs no agent, so the count it holds"

17-guard describes 'an agent abcd starts through its claude runner', launched with the lane's base commit. 27-implement holds a session's agent ceiling only against the session's own agent_start lines 'because abcd runs no agent'. If abcd starts agents itself, that premise is false, and agents abcd launches can sit outside the count the ceiling is held against.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 565.
