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

> **When abcd starts agents on your own API key, it spends only what your budget allows, and it pauses rather than surprising you.** Before their first build on a key, Maya sets what abcd may spend in a day, a week and a month, with a command only they can run. Each of their sessions names the one key it uses by the name they gave it in the credential store, and abcd resolves it from the home they chose for it there (the keychain is the one abcd suggests), never from a repository file and never from a value typed into a command. Every agent abcd launches is limited to what is left of their budget, each step shows what each role has spent against each cap, and when a cap is reached the work pauses until the cap resets or they raise it, unless they have chosen to move to a second key they named for that purpose. "I left a drain running overnight on my key," said Maya, an autonomous-development practitioner. "In the morning it had paused at my daily figure and told me which cap it hit. No surprise on the bill, and nothing ran on my subscription instead."

## Why This Matters

> The first live run of abcd's claude runner on a person's own key (2026-11-01) spent 0.97 USD on one review and 0.33 USD on another, while the run's budget check reported itself skipped because no runner reports a quota. Every route already records the cost it reported (itd-166), but nothing reads it back as a limit, and the runner discards the reported cost of any launch that did not succeed (internal/core/runner/claude.go: a non-success result returns an empty answer), so the launch most likely to overspend is the one whose spend is lost. Today the key reaches the runner only by a person typing it into each command's environment. The product thinker ruled the same day that abcd's agents run on the person's API key by default where a repository is set to that route, so a budget the person sets, and abcd keeps, is the precondition for leaving any work running on a key.

## Mechanism

> We expect spend to stay within the budget because Claude Code's print mode takes `--max-budget-usd` and ends an agent whose spend reaches it, and abcd bounds every launch, running ones included, against what remains of the key's caps: each launch's `--max-budget-usd` is reserved against every cap before it starts, so the bounds handed out never together exceed what remains less a headroom kept for the response that can cross a bound, and a launch held back by reservations is told so, distinct from a cap reached. The spec sets the arithmetic and must bound, and name where it cannot, three residuals, each named in the record when it happens and the first flagged for the product thinker: a response larger than any launch cost recorded before it, a launch whose cost cannot be read, and how many launches may run on one key at once; and a fourth no machine can see when it happens, the same key used by abcd on a second machine, since the store of totals is per machine and each machine's store counts only its own launches, so one key spends up to the cap on each. It would be shown wrong by a period whose recorded spend passes its cap other than through one of those named residuals.

## Scope Conditions

- Agents abcd launches through the claude runner on the key route, each on a key profile (itd-2611040344170206): a login-less profile with abcd's hooks on, so the guard holds and every such launch is billed per token to the key it is handed. The bare launch adr-2610222222326104 names for the key route gives way for stages (dcn-2611041550129278).
- Repositories set to the API route by the person's own setting, and the key a subscription-route repository moves to when a usage window reaches its mark (dcn-2611041550135566): a repository on the subscription route spends no money abcd can see while its stages stay on subscriptions, and the key it names as its fallback is a budgeted key like any other, so naming one needs its budget.
- Claude Code versions whose print mode accepts `--max-budget-usd` and reports `total_cost_usd` in its result, error results included; an older harness is refused for a budgeted launch rather than run unbounded.
- The opencode runner is out of scope here: its cost is summed from its own events and its provider's billing is the person's own arrangement.
- Costs are the figures the harness reports, its own estimate of the provider's price; the provider's invoice is the final figure, and abcd says so wherever it shows a budget.
- The person's key is a named credential in the credential store, in whichever of its homes they chose for it (adr-2609221017021499): a setup outside abcd the store points at, abcd's own owner-only file, or the platform keychain. The keychain is the home abcd suggests, never a requirement, so a machine with no keychain tool is offered the API route like any other (dcn-2611041550126639).
- A cap counts the key across the sessions of one machine: the store of totals is per machine, so the same key used by abcd on a second machine spends up to the cap there too, one of the exceptions the product thinker admitted (dcn-2611041550126639), covered by the provider's own console limit.

## Acceptance Criteria

- Given a repository on the API route and a session that names a key in the person's machine configuration, when abcd starts a routed role, then it resolves that key by name through the credential store, from the home the person chose for it, and hands it to that launch alone; after the launch no environment variable, argument, file or helper exposes the key, to the agent's own shell included, and nothing else abcd starts carries it. This does not stop an agent spending through the harness itself, which is what the caps bound (dcn-2611041550126639).
- Given a session that names no key, or a name the credential store does not resolve, when a routed role is handed out, then nothing is launched on a key, and abcd names the key it looked for, the home it looked in, and the walkthrough that sets it.
- Given a launch, when it ends, then the run record names the key it was handed by its configured name (never its value) and the credential source the harness reported.
- Given a repository on the API route with no budget set, when a run would start a routed role, then it refuses to start and names the command that sets the budget.
- Given day, week and month caps set by the person, when abcd starts a launch, then abcd bounds every launch, running ones included, against what remains of the key's caps: each launch's `--max-budget-usd` is reserved against every cap before it starts, so the bounds handed out never together exceed what remains less a headroom kept for the response that can cross a bound, and a launch held back by reservations is told so, distinct from a cap reached. The spec sets the arithmetic and must bound, and name where it cannot, three residuals, each named in the record when it happens and the first flagged for the product thinker: a response larger than any launch cost recorded before it, a launch whose cost cannot be read, and how many launches may run on one key at once.
- Given a launch that ends, successfully or not, when its result reports `total_cost_usd`, then that figure is added to the day, week and month totals, which are summed from the launch ledger (itd-2611031342011261), whose retention never falls below the longest cap period plus one day; and given a launch whose cost cannot be read, then it is charged the whole `--max-budget-usd` it was handed.
- Given a cap reached during a run, when the next role is due, then the repository's at-cap setting decides: by default the run waits, resuming when the period resets or the person raises the cap; set to move, it continues on a second key the person named for that purpose; it never hands the role to the host session or the subscription.
- Given an agent's session, when it runs the command that sets or raises a budget, sets the at-cap behaviour, or names the second key, then abcd refuses it, and the person's own run of the same command sets it; all three are kept in the person's own machine configuration under ~/.abcd.noindex/, never in a repository file an agent's change can reach, with the same residual as the totals: they are writable by the person's own account.
- Given a run, when each step reports, then it shows what each role has spent against each cap, by the key's configured name.
- Given several sessions on one machine using the same key, when any of them launches on it, then its spend is added to that key's day, week and month totals in one store every session on the machine reads, so a cap counts the whole key across the sessions of that machine; reaching it pauses every session on that key on that machine together.
- Given a key's spend reaching three quarters of any of its caps, when a session using that key next launches or steps, then that session is told from the shared store, naming the cap, the spend and what is left, so every session on the key hears it at its next move.
- Given how many launches may run on one key at once, when the person has set no figure, then one launch at a time is the shipped default and the person raises it in their own machine configuration, never a repository file; a key route with a budget set and no figure set is never refused for the figure's absence. (The drafter's own, from the landing reviews of 2026-11-02, under the standing stance that abcd ships a sensible default the person configures, dcn-2611041550129278.)
- Given a remainder above zero but below the headroom with nothing running on the key, when a launch is due, then abcd names that state apart from a cap reached and from a launch held back by reservations, says the remainder and the headroom, and says what lifts it: the period's reset, a raised cap, or a smaller largest-recorded response. (The drafter's own, from the same reviews.)
- Given any launch abcd makes on a key (a build or drain role, the planning interviewer, the drain's judge, a review abcd starts), when it ends, then its cost is added to the key's totals, so no launch abcd makes goes uncounted; spend on the same key outside abcd's launches (a command a person or an agent runs by hand) is neither counted nor bounded, and the record of totals is writable by the person's own account, which is the residual the person's own console limit covers.

## Open Questions

_None open. The rulings of 2026-11-01 and 2026-11-02 and the planning interview's two rulings of 2026-11-04 are in the Decisions section._

## Decisions

- 2026-11-01, the product thinker (dcn-2611012123470476): each session names the one key it uses, by the name in the machine configuration, never from a per-command variable. The reading that the key lives in the operating system's secret store alone is replaced by the ruling of 2026-11-04 below.
- 2026-11-01, the product thinker (dcn-2611012123470476): caps are configurable per day, per week and per month; reaching any one pauses the work and waits, never falls back to the host or the subscription.
- 2026-11-01, the product thinker (dcn-2611012123470476): a key route with no cap set refuses to start.
- 2026-11-01, the product thinker (dcn-2611012123470476): a cap is set or raised through a command abcd refuses to agents; a touch to raise it is a later intent.
- 2026-11-01, the product thinker (dcn-2611012102280663, recorded by another session): every agent abcd hands out runs on the person's API key where the repository is set to the API route, the route being a per-repository choice of the person's with the subscription as the shipped default.
- 2026-11-02, the product thinker (dcn-2611020715491928): a key's caps count the whole key; every session on that key pauses together at a cap, and spend never passes it. This replaces the 2026-11-01 reading that a limit stops only the session that hit it (dcn-2611012123470476). Because the harness can overshoot its own limit by the response that crosses it, 'never passes' is kept by refusing a launch whose remainder is not above a headroom sized from the largest response recorded on the key; a response larger than any before it is the residual, flagged for the product thinker; the spec keeps this residual and sizes the headroom from recorded launch costs, the only cost the harness reports.
- 2026-11-02, the product thinker (dcn-2611020715491928): what happens at a cap is a per-repository setting, kept in the person's machine configuration keyed by repository, pausing by default, with moving to a second key named for that purpose as the alternative, and the subscription never (settles iss-2611012148181861).
- 2026-11-01, the product thinker (dcn-2611012137380016): this draft may be planned unattended overnight through `abcd build --auto-plan`, after its two plan reviews, and reviewed by them in the morning.
- 2026-11-04, the product thinker (dcn-2611041550126639): the key is a named credential resolved through the credential store from whichever of its homes the person chose (adr-2609221017021499); the keychain is suggested, not required. The press release, the scope conditions and the first two criteria read so.
- 2026-11-04, the product thinker (dcn-2611041550126639): spend never passing a cap admits the exceptions the record names (a response larger than any recorded before it, a launch whose cost cannot be read, several launches sharing one key at once, and the same key used by abcd on a second machine); an overrun abcd can see is recorded against the key and named in the run's record, and the provider's console limit is the outer stop. On that ruling adr-2611011231015519 is accepted, and its refinement of decisions 17 and 34 of itd-2610161733316518 is no longer advisory.
- 2026-11-04, the product thinker (dcn-2611041550126639): the key route and the subscription route compose by one rule. A key at its cap pauses or moves to a second key the person named and never falls to a subscription; a subscription at its usage mark never opens a route the person has not named; both out at once is a wait, never a third route.
- 2026-11-04, the product thinker (dcn-2611041550126639): the key reaches the harness by a channel the agent's own shell cannot read, proved to work with the harness first; that proof is the plan's first step. Criterion 1 carries it.
- 2026-11-04, the product thinker (dcn-2611041550129278): a stage on the key route runs on a key profile with abcd's hooks on (itd-2611040344170206), so the first scope condition no longer names the bare launch.

## Evidence

- 2026-11-01, the night run by hand (reported by the session that ran on the second key): the product thinker set a per-key ceiling for the night, with work moving to the second key when the first reached it. Three sessions spent concurrently, two on the main key and one on the second, so one key's cap was shared across sessions and a total kept per session undercounted it. Before moving keys a session had to ask the others what was left, since nothing shared held the figure. Sessions warned each other at about three quarters of the ceiling. Each session tallied cost from its own run states by a hand-written script, so spend outside the build loop went uncounted: the planning interviewer's launch on the main key, and review pairs run by hand through `claude --bare` in text output, which reports no cost. The per-key store, the warning and the every-launch count above come from this evidence; moving to a second named key at a cap, which the night practised, is captured separately for the product thinker, since their ruling for this intent then was one key per session and a pause at the cap; the at-cap setting ruled on 2026-11-02 now admits it.

## Inputs for the spec

The plan's first step proves, with the harness itself, that the key reaches it by a channel the agent's own shell cannot read; nothing else in the plan starts before that proof (the product thinker, 2026-11-04).

The landing reviews of 2026-11-02 tried several exact budget models and found constraints the spec must meet (the two that name a user-visible state or a refusal are criteria above, as the drafter's own, since the reviews are no record):
- The harness reports cost once per launch (`total_cost_usd` on the result), not per response, so any headroom is sized from launch costs, which only makes it more conservative.
- A headroom fixed per key and multiplied by the number of launches allowed at once admits only one launch at a time once a cost is recorded; letting several run needs the remainder divided among them, rechecked so the bounds plus their crossing responses stay within the cap.
- A launch whose cost cannot be read is charged its whole bound, and the overshoot it may still hide is named as a residual.
- The spec says whether a run paused through a weekly or monthly reset stays alive or hands back with its resume time, within adr-2610222222326104's rule for lanes abcd launches, and names the marker that tells an agent's session from the person's for the refusal in criterion 8.

## Decomposition

Hand-run four-piece table, 2026-11-01, revised the same evening after two pre-interview reviews (design and feasibility; record discipline) and the product thinker's six rulings on keys and budgets; revised again on 2026-11-04 after the two plan reviews and the planning interview (dcn-2611041550126639).

| Part | Kind | Home |
|---|---|---|
| resolving the session's named key through the credential store, the budget per day, week and month, the reserved bound of each launch, the pause, the spend shown per role | capability | this intent |
| abcd's launches spend a person's money only within the budget they set, past it only through the exceptions the record names, and reaching a cap pauses rather than falls back to the host or the subscription | trust rule | adr-2611011231015519 (accepted, dcn-2611041550126639); a brief invariant owed in a change of its own |
| keeping a failed launch's reported cost, the remainder arithmetic, the key's credential source in the run record, the channel the key reaches the harness by | plumbing | the runner and the build and drain brief chapters |
| raising a budget only with the person's touch | capability | itd-2611012123486967, a later intent |

Verdict: FILE-AS-IS after the revision. Typed links: `builds_on` itd-166; `related_issues` iss-2610100913096346 (the budget check that always skips); the ADR `refines` decisions 17 and 34 of itd-2610161733316518 (confirmed, dcn-2611041550126639); the pause honours adr-2610222222326104's rule that a role with no host to take it is handed back, never run elsewhere. The live proof, every role of a build on the key under a real budget and one step under a deliberately low one, is this intent's grounds, not a criterion.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the product thinker's first priority is efficiency, which asks for far more agents than today running together on keys held under caps (dcn-2611031549087617), and the rulings that bound a key's spend (dcn-2611012123470476, dcn-2611020715491928, dcn-2611041550126639) make the cap abcd's to keep; we expect a run's spend on a key to stay within what its caps allow, and a key's billed spend passing a cap other than through a named exception, or a run stalling at a cap with budget left, would show it wrong
