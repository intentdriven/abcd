---
id: itd-97
slug: the-facilitator-is-a-mode-not-a-person
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817]
severity: major
---

# The Facilitator Is a Mode, Not a Person

## Press Release

> _Facilitator-seeded draft (2026-07-25 role-ownership review) — the product
> thinker owns the final press-release framing._

> **abcd runs solo or duo: the facilitator is a configuration, not a hire.** A
> project chooses **duo** — a human facilitator working alongside the product
> thinker, as today — or **solo**, where abcd itself performs the facilitator's
> work: shaping intents into plans, curating the backlog, firing reviews,
> triaging captures, and deciding *when* each of those happens. The gates do
> not move: solo and duo run the identical deterministic gates and the
> identical product-thinker decision points; the only thing that changes is
> who operates the machinery between them.
>
> "I don't have a facilitator — and it turned out I didn't need to find one,"
> said Alice, a product thinker. "abcd drafted the plan from my intent, queued
> the work, ran the reviews, and every time a decision was genuinely mine it
> arrived as a clear proposal with the price spelled out. Bob still prefers
> duo, with a human translator beside them — same framework, same gates, their
> choice."

## Why This Matters

The README already records the ambition ("in a later version, abcd aims to
offer an automated facilitator") but nothing owns it. The facilitator's
**what** has shipped kernels — the planning interview, `intent ready`, the
run protocol, the fidelity reviewer — with completions planned (itd-27,
itd-50, itd-53, itd-58, itd-78, itd-82). The facilitator's **when** —
acting unprompted at the right moment — exists only as drafts: the schedule
(itd-13), self-firing reviews (itd-83), stage-aware defaults (itd-19), and
the coordination/escalation contract (itd-33). Solo mode is the named
composition of those parts behind one switch; making it first-class is also
the precondition for testing solo against duo at all (itd-98).

The role boundary is already doctrine and does not change with the mode:
steps owned by the product thinker (adoption, adjudication, dependency
sign-off, irreversible actions) stay human in both modes
(verifier-selects-gates-decide); solo automates only facilitator-owned work.

## What's In Scope

- **A per-repo mode declaration** (duo | solo) with duo the default; the
  status board and every run surface name the active mode — no ambient
  ambiguity about who the facilitator is.
- **Solo = the composed "when"**: scheduled sync, self-firing reviews,
  capture triage, backlog curation, and run initiation composed behind the
  mode switch, over the pluggable run seam (itd-29).
- **Duo unchanged**: the same machinery surfaces queued facilitator work to
  the human instead of acting; nothing is removed from today's flow.
- **Gate parity**: mode changes an operator, never a gate — the deterministic
  gates, review requirements, and STOP conditions are byte-identical across
  modes.
- **The escalation contract**: solo stops at every product-thinker decision
  point with a proposal (options, recommendation, price), exactly where duo
  would hand over.

## What's Out of Scope

- **The solo-vs-duo evaluation** — itd-98 owns the measurement.
- **A team of product thinkers** — itd-99 owns plural thinkers; this intent
  assumes one.
- **Any weakening of human gates in solo** — `--dangerously-unattended`-style
  skips are the run protocol's concern and never cross the product-thinker
  line.
- **Model routing and credit contingency** — operational policy, owned by the
  run PLAN contract.

## Acceptance Criteria

- **Given** a repo configured solo, **when** a new capture lands in the
  ledger, **then** triage runs without a human initiating it, and any routing
  that requires judgement lands as a proposal for the product thinker — never
  a silent action.
- **Given** a repo configured duo, **when** the same events occur, **then**
  abcd surfaces the queued facilitator work to the human facilitator and
  performs none of it unprompted.
- **Given** either mode, **when** the same change is driven to a PR, **then**
  the deterministic gates, reviews, and STOP conditions applied are
  identical (a diff of the gate configuration between modes is empty).
- **Given** solo mode reaches a decision reserved for the product thinker
  (adoption, adjudication, new dependency, irreversible action), **when** it
  gets there, **then** it stops with a clear proposal and recorded reasoning
  and does not proceed until adoption.
- **Given** a session starts in either mode, **when** the status board
  renders, **then** it names the active mode.
- **Given** the technical facilitator is set to human (duo), **when** an agent
  would arm auto-merge on a pull request or add it to the merge queue, **then**
  that is refused as a hard rule, whoever asks, and the pull request waits for
  a human to merge it. (The first gate declared in both modes; proposed by the
  product thinker, 2026-10-11.)
- **Given** the technical facilitator is set to auto (solo), **when** a pull
  request's required checks and gates pass, **then** auto-merge is armed on it
  automatically, with no one asked.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [elicitation interviews SOTA note](../../research/notes/2026-10-23-elicitation-interviews-sota.md).

- Autonomy is a design choice on a scale: Sheridan and Verplank's level 5
  "executes that suggestion if the human approves", 6 "allows the human a
  restricted time to veto", 7 "executes automatically, then necessarily informs";
  a 2025 framework names the user's role at each level (operator, collaborator,
  consultant, approver, observer; Feng, McDonald and Zhang). Human mode is the
  approver level, auto-merge the observer level, a merge queue with a veto window
  level 6.
- The coding hosts have converged on classifier-gated auto with hard residual
  refusals: the host abcd ships for never auto-approves its question tool or
  writes to protected configuration paths, pauses after 3 consecutive or 20 total
  blocks, blocks by default "Merging a pull request no human has approved,
  approving Claude's own pull request, or disabling CI checks", and ignores `auto`
  set in a project's settings; another deprecated "Ask Every Time" but still
  prompts for deletion in every mode; a third allows its all-approve mode "only
  … via command line". The residual list (a question to a person, configuration
  paths, deletion, the product thinker's decisions) belongs in both modes.
- Auto-merge is only a rule if something is required: it "merges a pull request
  automatically after all required reviews and status checks pass" and is offered
  "only on pull requests that cannot be merged immediately" (GitHub). "Checks
  pass" must mean success on the merge-group ref: GitHub "considers a 'neutral'
  check run conclusion as passing" (Kodiak).
- The bots already live under the human-review rule: Renovate "can't automerge
  its own PR until such a review has happened"; GitHub's own coding agent needs
  "Another reviewer" because the requester's approval "won't count"; its
  reviewer's "approval assessment alone does not count toward merge
  requirements", but "When enabled, Copilot can submit an approval that counts
  toward the repository's required-approvals rule", off by default and
  enabled by an administrator, so the forge now lets a model approval satisfy
  a required review where that is switched on. A repository's policy can
  forbid the mode outright (QEMU declines AI-derived
  contributions on sign-off grounds).
- The human gate decays: automation bias "cannot be prevented by training or
  instructions" (Parasuraman and Manzey, 2010); wrong advice raised error risk by
  a pooled 1.26 (Goddard et al., 2011). An audit of auto-merged pull requests
  (random re-review, rejection rate) is part of the mode.
- On "is merging irreversible": no opened source states a position. That
  Renovate and the GitHub agent allow a merge under required reviews and checks
  is consistent with treating a reviewed, queued merge as revertable and
  re-validated, but that reading is this pass's inference, not a field
  position; the question stays with the product thinker, and the 2026-10-15
  refinement reads the same way.

**Path 2**: a native two-point mode shaped to the levels scale and the hosts'
convergence. Repository configuration can forbid auto but should not alone arm
it: the hosts refuse to take their auto mode from project or local settings,
and a committed `.abcd/` file is writable by any agent in the run and by any
pull request (the guard's protected-path list does not include it), so reading
it from the repository root stops no agent from editing it. Either auto is
armed from a user-level setting, with the repository able only to lower it, or
the key is a protected path the guard refuses to write and only the person's
own act changes; the planning interview rules which, and "Where does the mode
live?" stays open. Solo v1 in-session and host-delegated; `pause` a temporary drop to human; "checks
pass" defined as success with a human approval the requester cannot supply in
human mode. No Go dependency.

The independent fit-challenge runs at plan time.

## Open Questions

- **Where does the mode live?** A `.abcd/` config key, or a value of the
  itd-19 stage taxonomy (whose Autonomous/Collaborative axis is the nearest
  existing vocabulary)?
- **v1 initiative scope** — which "when" triggers compose into the first solo
  cut (triage and reviews look smallest; scheduling has the itd-13
  launchd/cron design)?
- **Relationship to itd-29's operator verbs** — is `pause` in solo mode
  equivalent to a temporary drop to duo?
- **Does solo v1 require the run seam**, or can it operate in-session
  (host-delegated) first, per script-first MVP?
- **Is merging an "irreversible action"?** The fourth criterion keeps
  irreversible actions with the product thinker in both modes. If merging a
  pull request counts as one, an auto facilitator could not merge either,
  which contradicts the auto-merge gate the product thinker proposed. Settle
  which actions stay the product thinker's, and whether a revertable merge is
  among them.
- **How the hard rule is enforced.** A gate an agent can talk its way past is
  not a hard rule: the guard registry could refuse `gh pr merge --auto` and an
  enqueue while the facilitator is human, which needs the guard to read the
  facilitator setting.
- (SOTA 2026-10-23) Whether a model reviewer's approval may satisfy a forge's
  required review in auto mode, given the forge's own reviewer does not count by
  default and the requester's approval never does.

## Decisions

- 2026-10-11: the product thinker proposed a toggle that automates the technical facilitator's role, set to human or auto, with auto-merge as the first example: always refused as a hard rule while the facilitator is human, so a human merges; armed automatically when it is auto. Shown that this draft already covers the switch, they chose "Add to the draft" over a new intent, so the example lands here as this intent's first two-mode gate, and other facilitator responsibilities follow the same pattern (adr-2609151528057260: every gate declares its behaviour in both modes). "human" and "auto" are the product thinker's names for duo and solo.
- 2026-10-15, the product thinker, refining the 2026-10-11 decision: with the facilitator automated, every ready pull request auto-merges, or is enqueued, with no human step; with a human facilitator, the facilitator reviews and approves, and the merge waits for them. Automated merging is not unreviewed merging: the reviews are automated by separate agents. A ready pull request has a passing preflight, a review pair of separate agents on different models with fresh context (principle evaluator-outside-the-loop), the stop rule (only a critical or major finding blocks; anything smaller becomes its own capture), and CI and the merge queue on the merged tree. The land defaults built for the next release (iss-2610150524235929, push after a passing preflight; iss-2610150524246898, enqueue every ready pull request) follow this gate rather than fixing either mode in place.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- README § Roles — the recorded automated-facilitator ambition this intent
  gives an owner.
- itd-29 (run seam operator surface, planned); itd-13, itd-19, itd-33,
  itd-83 (the "when" drafts this composes).
- `.abcd/development/principles/verifier-selects-gates-decide.md` — the role
  boundary both modes preserve.
- itd-2610150602547271 (builds on this intent): its first slice, the
  facilitator mode declaration and the merge gate that obeys it.
- itd-98 (the evaluation), itd-99 (plural thinkers) — the sibling drafts
  filed with this one.
