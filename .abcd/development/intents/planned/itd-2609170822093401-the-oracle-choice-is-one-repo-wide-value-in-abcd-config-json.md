---
id: itd-2609170822093401
slug: the-oracle-choice-is-one-repo-wide-value-in-abcd-config-json
spec_id: spc-2609180535002478
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609180517121254, itd-2, itd-2609081951381895]
severity: minor
impact: additive
related_adrs: [adr-25]
related_intents: [itd-17]
related_issues: [iss-2609170818061083]
origin: extracted-from-record
production_mode: hand-written
---

# abcd proposes the model each agent deserves, and facilitates it where it can

## Press Release

> **abcd now ships its own proposal for which model tier each of its agents
> deserves, shows it, applies it once the operator accepts it, and makes a
> best effort to satisfy it — a local or cloud provider where one is
> configured and reachable, the harness otherwise, always.** Today the oracle
> backend is one repo-wide value that nothing reads, and every delegated step
> reaches whatever model the harness picks at whatever it costs. The proposal
> is a table with one row per agent: a tier from a small closed set (`local`,
> `economy`, `frontier`, `host-decides`) that survives model churn, and a
> fan-out bound that may tighten the ceiling the agent's own trust contract
> declares but never raise it. Nothing is applied until it is accepted: the
> install step renders the proposal and writes it, on consent, as the
> machine's table under `~/.abcd/`; a repository may commit its own rows at
> `.abcd/config/oracle-routing.json`, and a repo row wins over a machine row,
> which wins over the bundled proposal. Above all three sits a manual
> override for one invocation — `--route <agent>=<tier>[@<connection>]`,
> repeatable, on every delegating verb — so an autonomous experiment can run
> the same agent across backends and configurations and read the outcome
> from receipts that name the override verbatim. A provider's sampling
> settings — temperature, seed, and whatever else that provider accepts —
> default on the connection, and a row or a `--route` may override them for
> one agent or one run; a setting a provider does not accept is refused by
> that provider's adapter, never dropped. At run time abcd resolves the
> winning row against the connections configured on the machine: a
> configured provider that is reachable and can serve the tier is used
> directly; otherwise the step goes to the harness with the tier named in its
> request, one line on stderr says so before the step runs, and the receipt
> records the tier asked, the connection tried, the connection used and the
> model the host or provider reported. A machine with nothing accepted
> behaves exactly as today. The proposal is the surface a later learner
> updates from measurements across repositories; this intent ships the best
> guess and the machinery that carries it.

> "The release gate's semantic review and the cold reading's four positions
> were reaching the same model at the same price, and I could not say which
> steps I had actually read the output of," said Maya, AI/agent researcher.
> "abcd's proposal already had the split right — the verdicts I read at
> frontier, the rest at economy — and once I pointed it at the local server
> the economy rows ran there without my touching a row. The receipt shows me
> what ran next to what was asked." Dave, security engineer, put it more
> simply: "The row says `local`, and if the local server is down the receipt
> says the step went to the cloud instead. I would rather know than guess."

## Why This Matters

Graduated from `iss-2609170818061083`. The host-delegated oracle (adr-25)
made model access the host's problem, which was right, but it left abcd with
no way to express *which* model an agent deserves or how much parallel work
it may spawn, and the repo-wide `oracle.backend` the install step writes is
read by no verb. Cost is undifferentiated: every delegated step is paid for
as if it carried a verdict somebody reads. Privacy is undifferentiated: a
step over sensitive material is routed like one over a public brief, so an
operator with a local provider has no switch that sends the sensitive passes
there. Fan-out is unbounded by anything the record can see.

The operator should not have to write this table from nothing. Which agents
carry judgement a human reads and which are plumbing is a property of the
agents, not of the repository, so abcd can propose the split and be right for
most operators; what a machine can reach is a property of the machine, so
the proposal is resolved against configured connections at run time rather
than written into rows. Acceptance is the opt-in that adr-25 asks for: a
configured provider and an accepted table are the operator's two acts, and
after them abcd routes without asking. The host still owns model choice,
credentials and execution on the harness leg, which is why this refines
adr-25 rather than reversing it; on a provider leg abcd owns the connection,
and that adapter's own intent (`itd-2609081951381895`) records what that
means.

Two intents share the seam. The enforced half — every ingested payload must
name its model and its agent count — is `itd-2609180517121254`, which this
one builds on: the routing table is what those fields are checked against.
Learned routing (itd-17) refines this: the proposal is the floor a learner
adjusts from measurements across repositories, and the invocation-time
override is how those measurements are taken — the same agent run under
several routes, each receipt naming the route that governed it.

## What's In Scope

- The bundled proposal: one row per agent, tier and fan-out bound, shipped
  with the binary.
- Rendering the proposal and writing it only on consent, at machine level
  (`~/.abcd/`) and, offered separately, at repository level
  (`.abcd/config/oracle-routing.json`); repo over machine over bundled.
- Run-time resolution of the winning row against the machine's configured
  connections: provider where reachable and able to serve the tier, harness
  otherwise, the fallback announced on stderr and recorded on the receipt.
- The closed tier vocabulary and the `host-decides` floor.
- The request block carrying the tier and bound on the harness leg; the
  receipt carrying tier asked, connection tried, connection used and model
  reported.
- An orphan row reported by name when a table is read.
- The invocation-time override, `--route <agent>=<tier>[@<connection>]`,
  repeatable, on every delegating verb: it wins over every layer for that
  run alone, and the receipt records it as an override so a measurement run
  is never mistaken for accepted routing.
- Provider settings carried on a route: a connection's defaults, overridden
  per row and per `--route`, passed through to the provider and recorded on
  the receipt as sent.

## What's Out of Scope

- Requiring and refusing on the payload's `model` and `agents_used` fields —
  `itd-2609180517121254`.
- The provider adapters themselves, how a connection is configured and
  probed, and which settings each provider accepts — `itd-2609081951381895`
  and `iss-2609081951416843`; this intent resolves against whatever
  connections they establish and hands settings through, and the adapter
  refuses one its provider does not accept.
- Updating the proposal from measurements — itd-17.
- Enforcing a fan-out bound at ingest; here a row carries the number and the
  request block states it.
- A lint rule over the tables; run-time reporting is the gate here.

## Mechanism

- We expect a routing row to change which model a harness-run step uses
  because the harness already exposes model selection to a sub-agent
  request, so a plainly stated tier in the request block is enough; shown
  wrong if receipts show the host ignoring the tier on a material share of
  runs.
- We expect cost to fall without judgement suffering because the steps whose
  output a human reads are a small minority of delegated steps, so tiering
  the rest down changes nothing a person acts on; shown wrong if economy-tier
  outputs fail the ingesting verbs' schema gates or the audit verdicts drift.

## Scope Conditions

- Providers the operator configured at machine level: abcd never reaches a <!-- cond: cond-2609180535004580 -->
  connection nobody set up, and a step on a machine with no provider runs
  through the harness whatever the row says.
- Single-operator repositories: one person or one small team accepts the <!-- cond: cond-2609180535004623 -->
  tables, and no claim is made about a table contested across many
  contributors.
- Hosts that report the model used: on the harness leg the self-report is <!-- cond: cond-2609180535006629 -->
  the only evidence, and a host that cannot name its model cannot be
  checked, so rows for it are advice with a receipt that says so.

## Decisions

Ruled by the product thinker on 2026-09-22, after the research pass on model routing:

1. **Escalation inside a lane is a rule, not a router.** A lane starts at its role's tier; on a failed fix round the next round runs one tier up, and the switch is recorded as a fact in the state file with the round and the gate output that caused it.
2. **A route through a provider adapter must be on that provider's allowlist** (adr-2609221009491186); the tier proposes only routes the resolver admits.
3. **No learned per-request router** (itd-17 superseded by itd-2609221009495079).

## Acceptance Criteria

- **Given** a machine with no accepted table and a repository with no
  routing file, **when** any delegating step runs, **then** it runs through
  the harness at `host-decides` under the agent contract's own fan-out
  ceiling, and no provider is contacted.
- **Given** the install step on a machine with no accepted table, **when** it
  runs, **then** it renders abcd's proposed table, one row per agent with
  tier and fan-out bound, and writes it under `~/.abcd/` only on consent;
  a repository-level file is offered separately and written only on
  consent.
- **Given** an accepted row for an agent naming a tier, and a configured
  provider on this machine that is reachable and can serve that tier,
  **when** a step dispatching that agent runs, **then** abcd routes the
  step to that provider directly.
- **Given** an accepted row for an agent, and no configured provider that is
  reachable and can serve its tier, **when** a step dispatching that agent
  runs, **then** the step goes to the harness with the tier and bound named
  in its request block, one line on stderr names the fallback and its reason
  before the step runs, and the receipt records the tier asked, the
  connection tried, and the connection used.
- **Given** a step's payload is ingested, **when** the receipt is written,
  **then** it carries the tier requested, the connection used and the model
  reported, side by side, verbatim.
- **Given** a repository file and a machine file that disagree on an agent,
  **when** a step dispatching that agent runs, **then** the repository row
  is the one resolved, and the bare `abcd` board shows all three layers for
  that agent with the winning row marked.
- **Given** `--route <agent>=<tier>[@<connection>]` on a delegating verb,
  **when** a step dispatching that agent runs, **then** the flag's row is the
  one resolved regardless of the accepted tables, a second `--route` for a
  different agent applies alongside it, and the receipt records the route as
  an invocation override, verbatim; a `--route` naming an agent the verb
  does not dispatch, or a connection that is not configured, is refused
  before the step runs.
- **Given** a connection carrying default provider settings, a row that
  overrides some of them, and a `--route` that overrides others, **when** a
  step resolves to that connection, **then** the settings sent are the
  connection's defaults with the row's overrides applied and then the
  flag's, and the receipt records the settings as sent; a setting the
  provider's adapter does not accept is refused before the step runs, never
  dropped.
- **Given** a table carrying a row for a name that is not an agent in the
  roster, **when** the table is read, **then** the orphan row is reported by
  name on stderr and the remaining rows still apply.
- **Given** a lane whose fix round failed, **when** the next round starts, **then** it runs one tier up from the role's tier and the state file records the round, the gate output and the switch.
- **Given** a proposed route through a provider adapter, **when** the tier proposes it, **then** it is one the provider's allowlist admits, or the proposal names the refusal instead.

## Open Questions

- Whether a fan-out bound on a row is read by any verb before
  `itd-2609180517121254` gives the payload an `agents_used` field, or only
  carried in the request block until then.
- How the bundled proposal is versioned against the agent roster: an agent
  added without a row, or a row for an agent since removed, must be a lint
  finding on the payload rather than a run-time surprise.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: flexibility first — each delegated step routed to the model it deserves, with privacy as an option (a local model where the material must not leave the machine) and cost as the essential lever: abcd proposes the split between judgement-bearing and plumbing steps, the operator accepts it once, and abcd facilitates it where a configured provider can serve the tier, the harness always the fallback; sub-agent fan-out declared and bounded per agent, because unbounded fan-out is where cost and unpredictability come from. Shown wrong if the locally-routed or bounded steps start failing the binary's schema gates or their verdicts drift from the frontier-routed ones, if accepted tables diverge widely from the proposal, or if bounded runs cost the same and vary as much as unbounded ones.
