---
id: itd-2610301428390406
slug: before-an-agent-submits-a-filled-in-form
spec_id: spc-2610301431131636
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# An agent can practise a filled-in form before it counts

## Press Release

> Before an agent submits a filled-in form to abcd, it can do a practice run that checks the form and shows what would be written, writing nothing.

## Why This Matters

Split out of itd-2610301346085246 on 2026-10-30 at the product thinker's ruling after the planning interview, keeping every decision and criterion the interview confirmed; this record depends on no other: the practice run goes through the same validator each writing command already has.

Today the only way an agent learns whether a filled-in input is right is to
submit it: a wrong one is refused and nothing is written, but the agent finds
out only by trying, as the 2026-10-30 `ideate` refusal showed. The nearest
measured evidence is SWE-agent's edit-time lint guardrail, which resolved
18.0% of SWE-bench Lite against 15.0% without it in the paper's ablations
(GPT-4 Turbo; Yang et al. 2024, https://arxiv.org/abs/2405.15793): checking an
action before it takes effect helped the agent.

## Mechanism

- We expect agents to submit a wrong input less often, because a practice run
  shows the refusal before the input counts. Shown wrong if refused writes per
  session do not fall where a practice run is available.

## Scope Conditions

- Agents that run abcd's commands and read their output as text. <!-- cond: cond-2610301431133725 -->
- Any host that can run the abcd program, with the plugin or without it. <!-- cond: cond-2610301431137567 -->
- The agent asks the running program before it fills in an input, as each <!-- cond: cond-2610301431135398 -->
  command page's first line tells it to.

## Acceptance Criteria

Each confirmed by the product thinker on 2026-10-30, one at a time.

- **Given** a filled-in input, **when** an agent runs the writing command as a
  practice run, **then** the binary checks it through the same validator the
  real write uses, reports what it would write or why it would refuse, and
  writes nothing: the working tree and every abcd store are byte-identical
  afterwards.

## Decisions

Ruled by the product thinker on 2026-10-30, in the planning interview of
itd-2610301346085246 (its decision 8), and carried here when it was split the
same day:

1. **A practice run is wanted**: a command that writes a record from a
   filled-in input can check it and show what it would write, writing
   nothing, through the same validator the real write uses. Chosen over not
   at all.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: agents are abcd's main users, and on 2026-10-30 one built an input from an out-of-date page and was refused; we expect fewer such refusals once agents ask the running program itself, and we are wrong if agents still build inputs from pages.
