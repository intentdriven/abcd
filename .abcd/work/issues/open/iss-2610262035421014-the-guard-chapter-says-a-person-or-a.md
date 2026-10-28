---
schema_version: 1
id: "iss-2610262035421014"
slug: "the-guard-chapter-says-a-person-or-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 591 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md:11"
remedy: "none (filed automatically)"
finding_ref: cfk-4bbd0c4c68b38009
---

The guard chapter says a person or a script asks it about a command line; the surfaces register lists `guard` with the agents, a block it reserves for commands no person types.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/17-guard.md:11` — "A person or a script asks it about one command line."
- End B: `.abcd/development/brief/04-surfaces/README.md:163` — "only a command no person types is listed with the agents"

17-guard names a person or a script as one of its two callers and gives the verdicts as a person receives them. The surfaces README says only a command no person types is listed with the agents, and its help table puts `guard` there. The README's rule and the guard chapter's account of its callers cannot both be right.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 591.
