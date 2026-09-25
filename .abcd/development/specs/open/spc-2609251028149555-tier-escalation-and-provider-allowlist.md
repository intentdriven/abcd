---
id: spc-2609251028149555
slug: tier-escalation-and-provider-allowlist
intent: itd-2609170822093401
origin: researcher-authored
production_mode: hand-written
---
# A failed fix round escalates one tier, and a provider route stays on its allowlist

## Summary

spc-2609251028149555 delivers what spc-2609180535002478 could not for
itd-2609170822093401. That spec shipped the routing table, its resolution,
`--route` on the delegating verbs, the request block, the receipt and the
install consent. Two acceptance criteria wait on records that do not exist yet,
and this spec holds them:

- AC 10: a lane whose fix round failed runs its next round one tier up, and the
  state file records the switch. The intent's Decision 1 rules this a rule, not
  a router.
- AC 11: a route through a provider adapter is one that provider's allowlist
  admits, or the proposal names the refusal. This is Decision 2 and
  adr-2609221009491186.

## Scope

- **Escalation.** The implement loop's state file records, per lane and round,
  the tier the round ran at. When a round's gates fail, the next round's route
  for the lane's role is `oracle.Resolve`'s row with the tier moved one step up
  the ladder `local` → `economy` → `frontier`. `host-decides` and `frontier`
  do not move. The state file records the round, the failing gate output and
  the switch, and the receipt names the escalated tier as the tier asked.
- **The allowlist.** `oracle.Connections` gains the adapter's allowlist. A tier
  the bundled proposal, a table row or a `--route` would send to a provider is
  checked against it before dispatch. A route the allowlist does not admit is
  refused, naming the provider and the route, instead of being sent.

## Approach

1. Once the implement loop's state file exists, add a `Round` record carrying
   `tier`, `gate_output` and `escalated_from`, and a pure `Escalate(tier)`
   beside `oracle.ParseTier`. Table-test the ladder and its two fixed points.
2. Once the API adapter (itd-2609081951381895) implements `Connections`, extend
   its interface with the allowlist and refuse an unlisted route inside
   `Resolve`, before any settings merge. Test it with an in-memory adapter.

## How the Acceptance Criteria are satisfied

10. A lane whose round N failed resolves round N+1 through `Escalate`, and the
    state file carries the round, the gate output and the switch.
11. `Resolve` consults the connection's allowlist before it returns a provider
    leg. An unlisted route is an error that names the refusal.

## Tests

- `internal/core/oracle`: the escalation ladder; an allowlist refusal naming
  the provider and the route.
- The implement loop: a failed round followed by an escalated round, with the
  state file's record read back.

## Blocked on

- The implement loop's per-lane state file, for AC 10.
- The API adapter's `Connections` and its allowlist (itd-2609081951381895,
  adr-2609221009491186), for AC 11.
