---
id: itd-2610301428378830
slug: when-abcd-refuses-an-agent-s-command-the
spec_id: spc-2610301431108175
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610301346085246]
severity: minor
refines: [itd-121]
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# abcd tells an agent what to do next, after a refusal and after a success

## Press Release

> When abcd refuses an agent's command, the refusal gives a short code, says in one line what is wrong and gives the next thing to try; asking about the code explains it. When a command succeeds, it says what usually comes next.

## Why This Matters

Split out of itd-2610301346085246 on 2026-10-30 at the product thinker's ruling after the planning interview, keeping every decision and criterion the interview confirmed; this record builds on it, since a refusal code is explained through `abcd explain`, and refines itd-121 (type an id, get the next move), claiming the next command on a refused call and on a successful result rather than on a record id.

A state-of-the-art pass on 2026-10-30 found no study that measures an
`explain` verb or a structured refusal on its own; the measured support is
indirect and points the same way:

- A narrow view beats a whole one: in SWE-agent's ablations (GPT-4 Turbo,
  SWE-bench Lite), a 100-line file window resolved 18.0% against 12.7% for the
  full file, and the edit-time lint guardrail 18.0% against 15.0% (Yang et al.
  2024, https://arxiv.org/abs/2405.15793, ablation table; the abstract's
  headline figure is a different measure).
- Fix-oriented documentation beats reference documentation: troubleshooting
  documents raised kubectl tool-call repair by about 10% over the official
  documentation (Tsay et al. 2025, https://arxiv.org/abs/2510.17874).
- More text is not better text: augmented tool descriptions raised task
  success by a median 5.85 points but cost 67% more steps and regressed in a
  sixth of cases (Hasan et al. 2026, https://arxiv.org/abs/2602.14878), and
  context files did not generally improve success while raising cost by over
  20% (Gloaguen et al. 2026, https://arxiv.org/abs/2602.11988).
- Practice: `kubectl explain` reads its field documentation from the server
  it talks to, rustc's JSON diagnostics carry a code that `rustc --explain`
  resolves, and the MCP specification (2025-11-25) classes an input-validation
  error as feedback a model uses to correct itself.

## Mechanism

- We expect a refusal that names its fix to bring the agent back on track in
  one step. Shown wrong if a refused call is still followed by more than one
  further refusal before it succeeds.

## Scope Conditions

- Agents that run abcd's commands and read their output as text. <!-- cond: cond-2610301431105187 -->
- Any host that can run the abcd program, with the plugin or without it. <!-- cond: cond-2610301431108565 -->
- The agent asks the running program before it fills in an input, as each <!-- cond: cond-2610301431102802 -->
  command page's first line tells it to.

## Acceptance Criteria

Each confirmed by the product thinker on 2026-10-30, one at a time.

- **Given** any refusal, **when** its output is read through `--json`,
  **then** the refusal carries a code, a one-line hint and the next command to
  try, and `abcd explain <code>` prints that code's explanation; a test proves
  every refusal path carries a registered code, every code resolves, and no
  code is reused or renumbered.
- **Given** a command that succeeded, **when** its output is read through
  `--json`, **then** it names what usually comes next, and a test proves every
  suggested next step is a command the binary has.

## Decisions

Ruled by the product thinker on 2026-10-30, in the planning interview of
itd-2610301346085246 (its decisions 3, 9 and 10), and carried here when it was
split the same day:

1. **Refusals carry a code, a one-line hint and the command to try next**,
   chosen over leaving them out; the reviews' cost stands with it: a migration
   across today's refusals and a code that is never renumbered once published.
2. **A command that succeeds says what usually comes next**: each successful
   `--json` result can carry the next step, each suggestion held by a test.
   Chosen over leaving it to the planned what-next menu
   (itd-2610031215002409, which keeps both: that menu is the person's, under
   the board; this is the agent's, on each result) and over not at all.
3. **Every refusal carries a code before this intent ships**, chosen over
   covering refused inputs first and the rest later; the intent owns as many
   specs as that migration needs (brief invariant 17).

## Open Questions

- resolved: that a published refusal code is never renumbered is a trust
  rule, so it is minted as an ADR when the spec that introduces codes is
  written, not stated here (routed by the itd-84 decomposition, 2026-10-30).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: agents are abcd's main users, and on 2026-10-30 one built an input from an out-of-date page and was refused; we expect fewer such refusals once agents ask the running program itself, and we are wrong if agents still build inputs from pages.
