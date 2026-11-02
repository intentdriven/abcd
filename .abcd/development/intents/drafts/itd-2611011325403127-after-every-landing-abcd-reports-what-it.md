---
id: itd-2611011325403127
slug: after-every-landing-abcd-reports-what-it
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-166, itd-2610262140595217]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every landing shows what it cost and the next run's estimate

## Press Release

> After every landing abcd reports what it cost: the hosted CI minutes by operating system and their price, the local gate time, and what the agents spent, and before a run it says what to expect. The product thinker never learns the cost of a week's work from a bill.

## Why This Matters

> Hosted CI is billed by the minute in a private repository, and abcd's agents land work far more often than a person does. On 2026-11-01 abcd's own repository ran 102 merge-queue entries and 95 pull-request checks in 2.5 days, about 6,800 Linux and 6,160 macOS job-minutes; at GitHub's 2026 private-repository prices (Linux $0.006, macOS $0.062 a minute) that pace would cost about $5,080 a month, 90% of it macOS. A managed repository's own bill is its CI minutes per run times the hosted runs per landing times how often abcd lands, and nothing in abcd today tells the product thinker any of those three numbers. The agents' own spend is recorded per launch (itd-166), but nothing adds it up, and the local gate's time, an hour or two of queueing on a busy day, is reported nowhere. A product thinker who is the only human relying on automated gates should know what a week of that automation costs before the bill arrives, and every saving the other drafts promise needs a measure to prove it.

## Mechanism

> We expect the figures to be exact enough to act on because each comes from a source that already records it: the forge's own job timings (the Actions jobs API gives each job's start, end and runner labels, from which minutes per operating system are derived; the run-timing endpoint is being withdrawn and reports nothing for a public repository, so it is at most a cross-check while it lasts), the per-launch cost and tokens every route records (itd-166), and and the preflight's own wait and run times, which nothing records today: they are what itd-2610262140595217 records, and this intent reads that record rather than measuring the gate itself. It would be shown wrong if the forge's figures and the provider's invoice disagreed by more than rounding, which the report names as the check to make.

## Scope Conditions

- Repositories hosted on GitHub, where Actions job timings and billing are readable with the person's own token; another forge reports what its API offers and says what it cannot.
- Prices are the forge's published per-minute rates, read with their date; a runner billed elsewhere (a third-party provider, a self-hosted box) is shown by its minutes and whatever price the person declares.
- Agent spend covers the routes that report it (itd-166); a role the host session runs reports no cost at landing time and is shown as unknown (host session), with the history store's capture (itd-2609292107351737) as the later route to its tokens.
- Network reads happen only on an explicit act: job timings are fetched by the landing's own forge poll and stored, published rates live in a dated table refreshed on request, and the weekly summary and the estimate read the disk alone.

## Acceptance Criteria

- Given a landing has merged, when its report is shown, then it names the hosted job-minutes by operating system, their price at the published rate with its date, the local gate's wall-clock time and queue wait, and the agents' recorded spend by role.
- Given a week of landings, when the product thinker asks for the summary, then abcd shows the totals and the three largest costs, with the share of hosted minutes spent on changes that touched no code.
- Given a run about to start, when abcd shows its plan, then it states the expected hosted minutes, local time and agent spend, drawn from the measured history of comparable landings, and says when there is too little history to estimate.
- Given a figure abcd cannot read (a route that reports no cost, a forge API that refuses), when it is reported, then it is shown as unknown, never as zero.

## Open Questions

1. Where does the weekly summary live: the status board, the morning summary, the dashboard, or a verb of its own?
2. Does the estimate before a run gate anything (a warning past a threshold the person set), or only inform?
3. Is the price of a self-hosted machine (electricity, the person's time) worth modelling, or only its minutes?

## Decomposition

Hand-run four-piece table, 2026-11-01, one of five drafts filed together from the investigation of hosted CI and local compute costs (a private ideate verdict, reframed). The product thinker ruled on 2026-11-01 that the work is filed as separate intents in this order: measure (itd-2611011325403127), fewer tests (itd-2611011325421304), self-attestation (itd-2611011325428560), an independent checker (itd-2611011325430933), and the check profiles new repositories are offered (itd-2611011325449761), the fifth at the product thinker's own request the same day.

| Part | Kind | Home |
|---|---|---|
| per-landing and weekly cost and time report, the estimate before a run | capability | this intent |
| an unknown figure is shown as unknown, never as zero | trust rule | already standing in invariant 16 (an attestation never states more than its examination established); cited, no new ADR |
| reading job timings and billable milliseconds from the forge | plumbing | the brief chapter that owns forge reads |
| (none) | stance | none |

Verdict: FILE-AS-IS. Typed links: `builds_on` itd-166; the spending cap of itd-2611011230226453 reads the agent-spend half of the same figures.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
