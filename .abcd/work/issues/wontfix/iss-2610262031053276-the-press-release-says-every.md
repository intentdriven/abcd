---
schema_version: 1
id: "iss-2610262031053276"
slug: "the-press-release-says-every"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 83 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:5"
remedy: "none (filed automatically)"
finding_ref: cfk-65e9d98cd4c4a1a0
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Bare land acts on the current branch and is already enumerated as an exception; the press release's universal claim is the single point to correct."
duplicates: [iss-2610262030301422]
---

The press release says every state-mutating action needs an explicit sub-verb, but bare `abcd land` pushes and queues with no sub-verb

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:5` — "state-mutating actions require an explicit sub-verb (the universal abcd convention)"
- End B: `.abcd/development/brief/04-surfaces/38-land.md:27` — "The verb takes at most one operand, the branch, and lands the checkout's current branch without one."

The press release states, as the universal abcd convention, that bare invocation shows status and that state-mutating actions require an explicit sub-verb. The land chapter says the verb takes at most one operand and, given none, lands the checkout's current branch. So bare `abcd land` runs the gate, pushes the branch to origin and arms its pull request. Either the convention is not universal or the land surface breaks it. A person who trusts the press release would run bare `abcd land` expecting a status render.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 83.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030301422. Bare land acts on the current branch and is already enumerated as an exception; the press release's universal claim is the single point to correct.
