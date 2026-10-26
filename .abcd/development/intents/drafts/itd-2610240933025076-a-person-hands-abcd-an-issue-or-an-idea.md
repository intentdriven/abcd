---
id: itd-2610240933025076
slug: a-person-hands-abcd-an-issue-or-an-idea
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030810370060, itd-2609201916056194]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd plans unattended and asks only what a person owns

## Press Release

> A person hands abcd an issue or an idea and says build it. abcd takes it from there without a terminal of its own: it promotes the issue, conducts the planning interview, plans the intent and starts the build, launching the person's own claude where no host session is driving. Along the way it puts to the person only the decisions that are theirs to make, each in their role's words (outcomes and choices for the product thinker, mechanism and trade-offs for the technical facilitator), and settles everything the record already answers itself, writing down what it settled and why. A decision only a person can make parks the work that depends on it, and nothing else; abcd never adopts a product decision on a person's behalf, and a question left unanswered stays asked.

## Why This Matters

On 2026-10-24 a product thinker asked for one issue to be built (`/abcd:build
iss-2610140652230444`). Getting from that request to a running build took six verbs,
three refusals and two questions about abcd's own record-keeping, and it then stopped at
a planning interview the person could not run: the session sent them to the terminal
verb, the verb needs a runner on a paid API key, their account is a subscription, and
the verb's refusal never said that the session itself may conduct the interview
(`commands/intent.md`, "Planning interview (host-run, with the human present)"). Each refusal was correct by its own rule. Together they
make a person do the orchestration abcd exists to do, and they ask questions (which
category, which link, which route) that are about abcd rather than about the product.

The person's own words: "too complicated. Not working … the user (TF / PT) is only
presented with the options it can answer."

## Mechanism

We expect a person to reach a running build from one request, answering only their own
decisions, because three things change together: (1) the unattended path gets a model
it can actually start: the person's own `claude`, under their own login
(iss-2610240932215918), or the host session already driving (iss-2610240932444000);
(2) deciding *whether to ask* is taken away from the agent that writes the plan and given
to a separate detection pass with a structured output, each open decision carrying an
owner (product thinker, technical facilitator, or the record) and the evidence that
settles it, because models asked to plan almost never ask unprompted, even on badly
underspecified input (Ambig-SWE, ICLR 2026, https://arxiv.org/abs/2502.13069v3; Su and
Cardie 2026, https://arxiv.org/abs/2605.25284), and separating detection from execution
is what measurably helped (https://arxiv.org/abs/2603.26233); and (3) a person-owned
decision becomes a durable parked question that blocks only the work depending on it,
resumed by session id when answered (the deferred-tool round trip:
https://code.claude.com/docs/en/hooks; LangGraph's interrupt model:
https://docs.langchain.com/oss/python/langgraph/interrupts). Shown wrong if, over the
next ten requests run this way, the person is asked a question whose answer the record
already held, or a product decision is found settled without a person's recorded
answer.

## Scope Conditions

- One person's own machine and own model account; abcd starts the person's unmodified
  `claude` binary and never reads, mints or stores a credential.
- Unattended means the person answers within a working day; work longer than that
  between answers is outside what this claim is about.
- The repository is managed by abcd, with its decision store, its issue ledger and its
  intent store, so the record a pass reads to settle a decision exists.
- Claude Code alone: the deferred-tool round trip is its feature, and another runner
  is outside this claim until it offers a structured output naming the open decisions
  and a resume by session id.

## Acceptance Criteria

- **Given** an issue `build <iss-N>` hands back on its category or its severity,
  **when** a person runs it, **then** the refusal also names the route that reaches a
  build from there: `capture promote <iss-N>` when no intent delivers it, or
  `build <itd-N>` naming the intent that does (the promoted refusal already names its
  intent, `internal/core/capture/eligible.go:238-245`).
- **Given** a draft intent and a host session driving, **when** the planning interview
  runs, **then** the session conducts it through the same answers record the terminal
  interview writes, and no runner route is required.
- **Given** a draft intent and no host session, **when** the planning interview runs on
  a machine whose model account is a subscription, **then** abcd starts the person's own
  `claude` under that login and the interview proceeds; an interview whose run did not
  load abcd's question hook is refused before its first turn, naming the hook.
- **Given** a planning pass over a draft, run separately from the agent that writes
  the plan, **when** it finishes, **then** every open decision it found is recorded with
  its owner (product thinker, technical facilitator, or the record) and its evidence;
  each whose owner is the record cites the record id that settles it and is written into
  the intent's Decisions with that id; and only the rest are put to a person.
- **Given** a seeded draft whose one open decision a cited decision record already
  answers, **when** the pass runs, **then** no question is put to a person and the
  Decisions line names that record (the GRILL rule that a decision with one defensible
  answer is recorded, not asked, made checkable by the cited id).
- **Given** a decision whose owner is the product thinker, **when** it is put, **then**
  it is asked in product terms with no record ids or internals, and one whose owner is
  the technical facilitator is asked with the mechanism and ids, as the GRILL register
  rule already says; the new part is that the owner is recorded before the question is
  drafted.
- **Given** a parked question, **when** the person has not answered it, **then** the work
  that depends on it stays parked, other work proceeds, and nothing adopts a default; an
  answer resumes the parked session by its id.
- **Given** a request carried through to a running build this way, **when** the run's
  record is read, **then** it lists every question put to a person, its owner, and every
  decision settled from the record with the id that settled it.

## Decomposition

Hand-run four-piece table (DECOMPOSITION, itd-84), 2026-10-24. Routing awaits a
person's confirmation.

| Piece | Record home | Verdict |
|---|---|---|
| Capability: unattended promote, interview, plan, build with parked person-owned questions | this intent | FILE-AS-IS |
| Trust rule: a machine never adopts a product decision; a parked question never times out into a default; abcd never handles a model credential | ADR + brief invariant (owed) | SPLIT |
| Stance: the human's adoption is the gate | existing principle `verifier-selects-gates-decide` | no new record |
| Plumbing: the subscription runner, the host route, the promotion link | iss-2610240932215918, iss-2610240932444000, iss-2610240932493158 | captured |

## Open Questions

- Does this reverse dcn-2607180000000002 (plan sign-off is implicit in a human running
  `intent plan`; agents are barred from unattended planning)? The intent proposes that
  an agent may run `intent plan` when every product-owned decision in the plan carries a
  person's recorded answer. Reversal flag, advisory: the product thinker rules it.
- Does this reverse ruling RN2 (dcn-2610020000000016, the claude runner on a paid key
  only)? iss-2610240932215918 carries the evidence and the probe it waits on. Reversal
  flag, advisory: RN2 was the technical facilitator's ruling, so the technical
  facilitator rules it.

## Typed Links

- **reverses `dcn-2607180000000002`** (advisory, unconfirmed): agents barred from
  unattended planning.
- **reverses `dcn-2610020000000016`** (RN2, advisory, unconfirmed), and with it the
  `--bare` premise of `dcn-2610161624025493`.
- **refines `iss-2610222000271964`** (a drain run from a plain terminal launches claude
  itself): the same launch, for planning.
- **builds on `itd-2610030810370060`** (plain-Terminal interviews): the written
  interview this intent makes reachable without a paid route.
- **builds on `itd-2609201916056194`** (the command-line runner).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
