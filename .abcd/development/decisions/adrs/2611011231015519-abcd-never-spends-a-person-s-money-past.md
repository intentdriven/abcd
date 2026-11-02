---
id: adr-2611011231015519
slug: abcd-never-spends-a-person-s-money-past
status: proposed
date: 2026-11-01
supersedes: null
superseded_by: null
related_intents: [itd-2611011230226453, itd-2610161733316518]
related_rfcs: []
related_adrs: [adr-2610222222326104, adr-2610120937435856]
---

# ADR-2611011231015519: The launches abcd makes spend a person's money only within the budget they set, overshooting by at most one response

## Context

abcd starts agents itself when a role is routed to a command-line runner, and on the claude runner's key route every launch is billed to the person's own API account. The first live run on a person's key (2026-11-01) cost 0.97 USD for one review and 0.33 USD for another; the run's budget check reported itself skipped, because no runner reports a quota. Every route already records the cost it reported (itd-166), so the figure exists; nothing treats it as a limit. The model-matrix intent rules a spending cap for matrix runs alone (itd-2610161733316518, decision 17: a figure the person named, no default abcd supplies, shown before consent, and the run stops when it is reached). Builds and drains spend the same money with no such rule, and a drain is meant to run unattended for a day (itd-2610161016401115).

## Decision

We will spend a person's money, in the launches abcd makes, only within a budget they set; spend on the same key outside abcd's launches is neither counted nor bounded, and the record of totals is writable by the person's own account, residuals the provider's own console limit covers. Any launch that bills a person per token, through any route abcd starts, runs only under a budget of the person's own: caps per day, per week and per month, set in their own configuration through a command abcd refuses to agents (a filter, not a lock, since agents run as the person's own account and could edit the configuration directly), with no default figure abcd supplies; a route with no budget set refuses to start. Each launch is handed no more than the smallest remaining cap, less the bounds reserved by launches still running and less a headroom sized from the largest response recorded on the key (the whole per-launch bound before any is recorded), through the harness's own budget limit, and its own bound is reserved under the shared store's lock before it starts; a launch is started only when that figure is above zero. Each launch's reported cost replaces its reservation when it ends, a failed launch's included, and a launch whose cost cannot be read is charged the whole bound it was handed. The harness counts the response that crossed its limit, so a period may overshoot its cap by at most that one response, which the headroom is there to absorb. When a cap is reached the work pauses until the period resets or the person raises the cap; it never continues on the host session or on another route. A route that bills no money (the host session, the subscription route) needs no budget.

This refines decisions 17 and 34 of itd-2610161733316518 from matrix runs to every route that spends the person's money, keeping 17's person-named figure with no default and 34's rule that no launch starts unless its remainder covers it; the refinement is advisory until the product thinker accepts it.

## Alternatives Considered

- **A budget the person sets, enforced per launch with headroom and summed per period, pausing at the cap (chosen).** It keeps the guarantee where the money is spent, reuses the costs itd-166 records, and bounds the residual to one response. Its residual: a command refused to agents is a filter, not a lock, since agents run as the person's own account; raising a cap with the person's touch (itd-2611012123486967) closes it.
- **Leave it to the provider's console limit.** It stops the key only at the account's limit, across every use of the key, and a run then fails mid-lane with no pause; the person learns of it from the provider. Rejected as the only control; the console limit stays a sensible second fence.
- **A default budget abcd supplies.** Every figure abcd could choose spends money on a number the person never set, which decision 17 already refuses. Rejected.
- **Fall back to the host session when a cap is reached.** It keeps work moving but spends the person's subscription without asking, which the product thinker ruled out on 2026-11-01. Rejected.

## Consequences

- Every route that bills per token must report its cost, failed launches included, and honour the bound before it can be routed; the claude runner must keep the cost of a non-success result it discards today.
- The budget check of a build and a drain stops reporting itself skipped for a budgeted route.
- An unread cost charged as its whole bound means a harness crash can pause the work early; the pause keeps the work, so the cost is time, not loss.
- A person who wants agents unable even to request a raise needs the touch of itd-2611012123486967.
- The brief gains an invariant naming this rule, and the build and drain chapters gain the arithmetic (itd-2611011230226453 carries both).
