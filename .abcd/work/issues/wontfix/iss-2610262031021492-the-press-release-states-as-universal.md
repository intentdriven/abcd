---
schema_version: 1
id: "iss-2610262031021492"
slug: "the-press-release-states-as-universal"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 77 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-57dbeea3097095ba
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Bare drain performs one move and is already listed as an exception to the bare-renders convention; only the press release's 'universal' claim is wrong."
duplicates: [iss-2610262030301422]
---

The press release states as universal that a bare command shows status and mutations need a sub-verb, but bare `abcd drain` performs a mutating move

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "Bare invocation of any command shows status + suggested next actions; state-mutating actions require an explicit sub-verb (the universal abcd convention)."
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:186` — "each invocation of the bare verb performs one move and exits"

The press release calls it the universal abcd convention that bare invocation of any command shows status and suggested next actions, and that state-mutating actions require an explicit sub-verb. The drain chapter has no sub-verbs. Each bare invocation performs one move: it fetches, reserves on origin, asks judgements and opens lanes. A command whose bare form mutates local and remote state breaks the convention the press release calls universal.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 77.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Bare drain performs one move and is already listed as an exception to the bare-renders convention; only the press release's 'universal' claim is wrong.
