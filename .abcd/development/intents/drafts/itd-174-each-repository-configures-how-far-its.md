---
id: itd-174
slug: each-repository-configures-how-far-its
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Each repository configures how far its facilitator is consulted and when it escalates

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The facilitator is a role, not a fixed amount of involvement, and how much of it a project wants varies more than the role itself does. A repository that is one person's tool wants the loop to run without stopping. A repository whose output other people rely on wants more consultation, and possibly a person doing the consulting.

That makes the facilitator's involvement a per-repository setting rather than a property of the framework: whether the role is filled by a machine or a person, whether the agents consult it at every step or only when they are stuck, and how uncertain they must be before the question travels further.

The important part is what sits at the top of that setting. The facilitator is the source of advice for the agents, translating what they are working on into what the facilitator knows the product thinker wants. Only when the facilitator is genuinely unsure does the question reach the product thinker, who answers in their own medium and their own time. Escalation is therefore a threshold rather than a rule, and the threshold is configurable too.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [elicitation interviews SOTA note](../../research/notes/2026-10-23-elicitation-interviews-sota.md).

- Both ends of the dial fail measurably: frontier agents at 75 to 89 per cent
  pass rate with complete information fall to 4 to 24 per cent when they must
  decide whether to ask, and Ask-F1 "captures the tension between over-asking
  and silent guessing" (HiL-Bench, 2026, a vendor benchmark); an
  uncertainty-aware multi-agent scaffold reached 69.40 per cent on
  underspecified tasks, not distinguishable from the full-information 70.80
  (p = 0.458) nor from a baseline required to query the user before executing
  (70.40 for one model; 47.20 for another, which misused its completion tool to
  ask) (Edwards and Schuster, 2026). "Only when stuck" under-asks unless gated;
  whether asking at every step costs accuracy is not shown by these sources.
- The model's stated confidence is not the threshold: verbalised confidence is
  "overconfident most of the time" and its reliability "strongly depends on how
  the model is asked", though calibrated scores are possible with particular
  prompts (Groot and Valdenegro-Toro, 2024; Yang, Tsai and Yamada, 2024).
- Practitioner guidance keys escalation on action class and failure count: "Two
  primary triggers typically warrant human intervention", exceeding failure
  thresholds and actions that are "sensitive, irreversible, or have high stakes"
  (OpenAI's agent guide); "fail closed if review times out", serialise state and
  resume the same run; a standing approval "persist[s] in … state" and is "not
  bound to the agent or tool instance that created" it (Microsoft), a warning
  against a one-off answer becoming a standing rule unnoticed.
- Metrics exist for tuning the dial per repository: ask precision and recall
  (HiL-Bench), "average turns to clarify" and "key question coverage" (ClarEval).
- Defaults are what ships: under 5 per cent of surveyed word-processor users had
  changed any setting (a consultant's survey, 2011); 91 per cent of 392 home
  routers ran on defaults (CHI 2025). The default consult list and threshold are
  the product.
- Addressing by role with a timeout is incident practice (an escalation policy is
  "an ordered set of rules"; "acknowledgment does not resolve the incident"), and
  the stop itself is the stop record the autonomous-loops pass recommends.
- The approver decays (automation bias; see itd-97), so the design records what
  was asked, answered and skipped.

**Path 2**: a per-repository ordered escalation rule shaped to the two-trigger
guidance, action class first (irreversible, outside the footprint, a new
dependency, a product-thinker decision), then a stuck count, then an optional
calibrated signal named per prompt template and off by default; the stop is the
stop record, the product thinker answers through the itd-90 path, an answer is
never silently promoted to a standing rule, and each repository logs ask
precision and recall. The 2026-08-29 ruling's settings map onto it as follows:
machine or human is itd-97's mode; consult at each step or not is the consult
list; ask when unsure or not is the stuck count; how unsure is the calibrated
signal, which here ships off by default as a proposal for the product thinker
to confirm, since the ruling names it as a setting. No Go dependency.

The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** The settings live where a repository is configured. As a start: machine or human, consult at each step or not, ask when unsure or not, and how unsure the facilitator must be before the product thinker is asked.

- (SOTA 2026-10-23) Which action classes escalate to the product thinker in every
  repository regardless of the setting, and which the setting may lower.
- (SOTA 2026-10-23) Whether a facilitator's one-off answer may become a standing
  rule, and if so by what explicit act.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
