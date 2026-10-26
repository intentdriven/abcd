---
schema_version: 1
id: "iss-2610262036101688"
slug: "the-rewriter-draft-rests-on-abcd-being"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 628 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2610100627454580-an-agent-in-an-abcd-managed-repository.md:61"
remedy: "none (filed automatically)"
related_intents: [itd-2610100627454580]
finding_ref: cfk-b18f3dccfce3561e
---

The rewriter draft rests on abcd being unable to start an agent; 17-guard describes agents abcd starts through its claude runner.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2610100627454580-an-agent-in-an-abcd-managed-repository.md:61` — "abcd cannot start an agent itself"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:385` — "An agent abcd starts through its claude runner has no terminal"

itd-2610100627454580 records a review finding that abcd cannot start an agent itself, so running the rewriter automatically can only mean a rule and a gate. 17-guard says an agent abcd starts through its claude runner has no terminal, and describes how that launch is configured. The premise that rules out an abcd-started rewriter is contradicted by a surface that starts agents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 628.
