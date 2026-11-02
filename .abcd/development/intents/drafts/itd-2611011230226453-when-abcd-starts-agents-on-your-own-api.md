---
id: itd-2611011230226453
slug: when-abcd-starts-agents-on-your-own-api
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-166]
related_issues: [iss-2610100913096346]
severity: major
origin: researcher-authored
production_mode: hand-written
---

# Agents abcd starts on your API key stay within the budget you set

## Press Release

> **When abcd starts agents on your own API key, it spends only what your budget allows, and it pauses rather than surprising you.** Before her first build on a key, Maya sets what abcd may spend in a day, a week and a month, with a command only she can run. Each of her sessions names the one key it uses, and abcd reads that key from her computer's own secret store, never from a file. Every agent abcd launches is limited to what is left of her budget, each step shows what each role has spent against each cap, and when a cap is reached the work pauses until the cap resets or she raises it, unless she has chosen to move to a second key she named for that purpose. "I left a drain running overnight on my key," said Maya, an autonomous-development practitioner. "In the morning it had paused at my daily figure and told me which cap it hit. No surprise on the bill, and nothing ran on my subscription instead."

## Why This Matters

> The first live run of abcd's claude runner on a person's own key (2026-11-01) spent 0.97 USD on one review and 0.33 USD on another, while the run's budget check reported itself skipped because no runner reports a quota. Every route already records the cost it reported (itd-166), but nothing reads it back as a limit, and the runner discards the reported cost of any launch that did not succeed (internal/core/runner/claude.go: a non-success result returns an empty answer), so the launch most likely to overspend is the one whose spend is lost. Today the key reaches the runner only by a person typing it into each command's environment. The product thinker ruled the same day that abcd's agents run on the person's API key by default where a repository is set to that route, so a budget the person sets, and abcd keeps, is the precondition for leaving any work running on a key.

## Mechanism

> We expect spend to stay within the budget because every launch is bounded twice: Claude Code's print mode takes `--max-budget-usd` and ends an agent whose spend reaches it, and abcd hands each launch only the smallest remainder of the day, week and month caps less a headroom, adding each launch's reported `total_cost_usd`, failed launches included, before starting the next. It would be shown wrong by a period whose recorded spend exceeds its cap by more than one response: the harness counts the response that crossed its limit in the reported total, so the overshoot of one launch is bounded by that response, and the headroom exists to absorb it.

## Scope Conditions

- Agents abcd launches through the claude runner under `--bare`, which reads no subscription login, so every such launch is billed per token to the key it is handed.
- Repositories set to the API route by the person's own setting; a repository on the subscription route spends no money abcd can see and needs no budget.
- Claude Code versions whose print mode accepts `--max-budget-usd` and reports `total_cost_usd` in its result, error results included; an older harness is refused for a budgeted launch rather than run unbounded.
- The opencode runner is out of scope here: its cost is summed from its own events and its provider's billing is the person's own arrangement.
- Costs are the figures the harness reports, its own estimate of the provider's price; the provider's invoice is the final figure, and abcd says so wherever it shows a budget.
- The person's keys live in the operating system's secret store (the login keychain on macOS, the Secret Service on Linux); a machine with neither is not offered the API route.

## Acceptance Criteria

- Given a repository on the API route and a session that names a key in the person's machine configuration, when abcd starts a routed role, then it reads that key from the operating system's secret store and passes it to that launch alone, never writing it anywhere and never into the environment of anything else it starts.
- Given a session that names no key, or a name the secret store does not hold, when a routed role is handed out, then nothing is launched on a key, and abcd names the key it looked for, the store it looked in, and the command that sets it.
- Given a launch, when it ends, then the run record names the key it was handed by its configured name (never its value) and the credential source the harness reported.
- Given a repository on the API route with no budget set, when a run would start a routed role, then it refuses to start and names the command that sets the budget.
- Given day, week and month caps set by the person, when abcd starts a launch, then it hands it a `--max-budget-usd` equal to the smallest remaining cap less the reservations of launches still running and less the headroom, and when that is not above zero it starts nothing and the work pauses, naming the cap reached and when it resets.
- Given a launch that ends, successfully or not, when its result reports `total_cost_usd`, then that figure is added to the day, week and month totals; and given a launch whose cost cannot be read, then it is charged the whole `--max-budget-usd` it was handed.
- Given a cap reached during a run, when the next role is due, then the repository's at-cap setting decides: by default the run waits, resuming when the period resets or the person raises the cap; set to move, it continues on a second key the person named for that purpose; it never hands the role to the host session or the subscription.
- Given an agent's session, when it runs the command that sets or raises a budget, sets the at-cap behaviour, or names the second key, then abcd refuses it, and the person's own run of the same command sets it; all three are kept in the person's own machine configuration under ~/.abcd.noindex/, never in a repository file an agent's change can reach, with the same residual as the totals: they are writable by the person's own account.
- Given a run, when each step reports, then it shows what each role has spent against each cap, by the key's configured name.
- Given several sessions on one machine using the same key, when any of them launches on it, then its spend is added to that key's day, week and month totals in one store every session on the machine reads, so a cap counts the whole key across sessions; reaching it pauses every session on that key together. Before a launch starts, its bound is reserved against the key's totals in that store under a lock, so launches still running count against the cap and two sessions can never be handed the same remainder; a launch starts only while every remainder, less the reservations, exceeds a headroom sized from the largest single response cost yet recorded on the key, and before any response is recorded the headroom is the whole per-launch bound. Recorded spend then does not pass the cap; the one residual is a response larger than any recorded before it, which the record names when it happens.
- Given a key's spend reaching three quarters of any of its caps, when the next launch is made, then every session using that key is told, naming the cap, the spend and what is left.
- Given any launch abcd makes on a key (a build or drain role, the planning interviewer, the drain's judge, a review abcd starts), when it ends, then its cost is added to the key's totals, so no launch abcd makes goes uncounted; spend on the same key outside abcd's launches (a command a person or an agent runs by hand) is neither counted nor bounded, and the record of totals is writable by the person's own account, which is the residual the person's own console limit covers.

## Open Questions

_All resolved on 2026-11-01 by the product thinker; the rulings are in the Decisions section._

## Decisions

- 2026-11-01, the product thinker: each session names the one key it uses; abcd reads it from the operating system's secret store (macOS keychain, Linux Secret Service) by the name in the machine configuration, never from a per-command variable.
- 2026-11-01, the product thinker: caps are configurable per day, per week and per month; reaching any one pauses the work and waits, never falls back to the host or the subscription.
- 2026-11-01, the product thinker: a key route with no cap set refuses to start.
- 2026-11-01, the product thinker: a cap is set or raised through a command abcd refuses to agents; a touch to raise it is a later intent.
- 2026-11-01, the product thinker (dcn-2611012102280663, recorded by another session): every agent abcd hands out runs on the person's API key where the repository is set to the API route, the route being a per-repository choice of the person's with the subscription as the shipped default.
- 2026-11-02, the product thinker: a key's caps count the whole key; every session on that key pauses together at a cap, and spend never passes it. This replaces the 2026-11-01 reading that a limit stops only the session that hit it (dcn-2611012123470476). Because the harness can overshoot its own limit by the response that crosses it, 'never passes' is kept by refusing a launch whose remainder is not above a headroom sized from the largest response recorded on the key; a response larger than any before it is the residual, flagged for the product thinker.
- 2026-11-02, the product thinker: what happens at a cap is a per-repository setting, pausing by default, with moving to a second key named for that purpose as the alternative, and the subscription never (settles iss-2611012148181861).
- 2026-11-01, the product thinker: this draft may be planned unattended overnight through `abcd build --auto-plan`, after its two plan reviews, and reviewed by them in the morning (recorded as a decision the same night).

## Evidence

- 2026-11-01, the night run by hand (reported by the session that ran on the second key): the product thinker set a per-key ceiling for the night, with work moving to the second key when the first reached it. Three sessions spent concurrently, two on the main key and one on the second, so one key's cap was shared across sessions and a total kept per session undercounted it. Before moving keys a session had to ask the others what was left, since nothing shared held the figure. Sessions warned each other at about three quarters of the ceiling. Each session tallied cost from its own run states by a hand-written script, so spend outside the build loop went uncounted: the planning interviewer's launch on the main key, and review pairs run by hand through `claude --bare` in text output, which reports no cost. The per-key store, the warning and the every-launch count above come from this evidence; moving to a second named key at a cap, which the night practised, is captured separately for the product thinker, since their ruling for this intent then was one key per session and a pause at the cap; the at-cap setting ruled on 2026-11-02 now admits it.

## Decomposition

Hand-run four-piece table, 2026-11-01, revised the same evening after two pre-interview reviews (design and feasibility; record discipline) and the product thinker's six rulings on keys and budgets.

| Part | Kind | Home |
|---|---|---|
| reading the session's named key from the secret store, the budget per day, week and month, the per-launch bound, the pause, the spend shown per role | capability | this intent |
| abcd never spends a person's money past a budget they set by more than one response, an unread cost counts as the whole bound, and reaching a cap pauses rather than falls back | trust rule | adr-2611011231015519 (proposed), and a brief invariant when it is accepted |
| keeping a failed launch's reported cost, the remainder arithmetic, the key's credential source in the run record | plumbing | the runner and the build and drain brief chapters |
| raising a budget only with the person's touch | capability | itd-2611012123486967, a later intent |

Verdict: FILE-AS-IS after the revision. Typed links: `builds_on` itd-166; `related_issues` iss-2610100913096346 (the budget check that always skips); the ADR `refines` decisions 17 and 34 of itd-2610161733316518 (advisory, for the product thinker to confirm); the pause honours adr-2610222222326104's rule that a role with no host to take it is handed back, never run elsewhere. The live proof, every role of a build on the key under a real budget and one step under a deliberately low one, is this intent's grounds, not a criterion.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
