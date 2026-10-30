---
id: itd-2610301346085246
slug: an-agent-working-in-an-abcd-managed
spec_id: spc-2610301431075964
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2609212113220149, itd-2610090831227812, itd-111]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# An agent learns how to call abcd from the abcd it runs

## Press Release

> An agent in a project that uses abcd learns how to use abcd from the abcd it is running. It asks for one command's instructions and gets them in a few lines, with the exact form any input must take and one example that works, instead of reading a long page that may be older than the program.
>
> Every fact about a command (its options, the forms it accepts and their versions, its examples) is written once, in the program. The instruction pages, a plain list for each command and the program's own answers are all made from that one place and checked at every build, so a page can no longer tell an agent one thing while the program expects another.

_The product thinker's accepted text (2026-10-30, planning interview); the paragraph on practice runs, refusals and next steps moved to itd-2610301428390406 and itd-2610301428378830 when the intent was split the same day._

## Why This Matters

It refines three shipped records: itd-2609212113220149 (the one does / writes /
refuses sentence every verb's help opens with), itd-2610090831227812 (the
`--help --agent` expansion, whose lines name the page to read next), itd-111
(a stale abcd never answers silently, which already stamps a version);
the split-out itd-2610301428378830 carries the refusal codes and next steps,
and itd-2610301428390406 the practice run. The machine-readable verb listing and the help's screen layout were
filed the same day in a sibling branch. <!-- record-lint: forward-looking -->

abcd's callers are mostly agents, and today an agent learns a verb by reading
its plugin page, which is prose shipped with the plugin version the session
loaded, not with the binary it runs. On 2026-10-30 a session composed an
`ideate` verdict from the published page (schema version 1) and the binary in
use refused it (it records version 2 only); the refusal named version 2 and
the page that carries the shape, so the agent recovered, but only by reading a
second page. The long pages make it worse: the capture page runs to several
thousand words, and an agent must read all of it to call one sub-verb.

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

- We expect agents to get abcd commands right first time more often, because
  they read the running program's own short answer instead of a page that may
  be older than it. Shown wrong if refusals per session do not fall.

## Scope Conditions

- Agents that run abcd's commands and read their output as text. <!-- cond: cond-2610301431075477 -->
- Any host that can run the abcd program, with the plugin or without it. <!-- cond: cond-2610301431077637 -->
- The agent asks the running program before it fills in an input, as each <!-- cond: cond-2610301431077983 -->
  command page's first line tells it to.

## Acceptance Criteria

Each confirmed by the product thinker on 2026-10-30, one at a time.

- **Given** any visible command or sub-command, **when** an agent runs
  `abcd explain <verb> [<sub-verb>]`, **then** it prints the command's one
  sentence, its flags with their types and allowed values, the input it
  accepts with that input's schema version, and one example, all from the
  program's one source of command facts, in at most 40 lines.
- **Given** a command that reads a filled-in input (a `--*-json` flag),
  **when** an agent asks `explain` for that input's shape, **then** it gets the
  shape and an example at the schema version the binary accepts, and a test
  proves that example passes the command's own decoder and validator.
- **Given** the program's one source of command facts, **when** the build
  runs, **then** the per-command JSON files, each command page's generated
  block and `explain`'s output are all produced from it, and `make preflight`
  fails when any committed copy differs.
- **Given** a command page's hand-written prose, **when** it names a flag, a
  command path or an input schema version that the derived facts do not hold,
  **then** a lint fails naming the page and line.
- **Given** any command page, **when** an agent opens it, **then** its first
  instruction tells the agent to ask the running binary (`abcd explain`)
  before it fills in an input, so an older cached page still leads to the
  newer binary's answer.

## Decisions

The interview's decisions as numbered there; 3, 8, 9 and 10 moved with their
parts when the intent was split (2026-10-30).

Settled by the record, so no question was put (2026-10-30):

1. **Asking abcd for the exact input a command accepts stays in this intent.**
   Every scope answer the reviews offered kept it here: it is the part that
   would have prevented the 2026-10-30 refusal.

Ruled by the product thinker on 2026-10-30, in the planning interview:

2. **One command's instructions in a few lines (`explain <verb> <sub-verb>`)
   is in this intent**, chosen over extending the existing `--help` and over
   an intent of its own.
4. **The program's code stays the one source of each command's facts, and
   everything else is derived from it and checked.** The options, their
   allowed values, the input shapes with their version numbers, the examples
   and the one sentence are defined once in the program. From that source the
   build writes a plain JSON file per command, a marked generated block in
   each command's page (options, input shapes, examples), and what `explain`
   prints, and a drift check fails the build when any of them is stale. The
   pages' explanatory prose stays hand-written, and a lint flags a flag name,
   verb path or schema version stated in that prose which the derived JSON
   does not hold. Chosen over a hand-edited JSON file per command as the
   source, after a state-of-the-art pass (2026-10-30): long prose does not
   survive inside JSON (RFC 8259 escapes every line break; even the AWS CLI
   keeps authored examples outside its JSON models), validation rules are
   code a data file cannot hold, clap deprecated its YAML builder in 3.0
   because it deferred errors to run time and cost constant upkeep
   (https://github.com/clap-rs/clap/issues/3087), the release gate's surface
   snapshot would be compared with itself, and every Go-side tool found
   (cobra's doc generator, the Usage project's cobra integration, Fig's cobra
   package) derives from the command tree. The page stays the home of what a
   command does, so adr-2610290238168123 Decision 2 stands and the reversal
   flag lapses unconfirmed.

Settled by the record, so no question was put (2026-10-30):

5. **`explain` is its own verb beside each sub-verb's `--help`** (keep both
   with itd-2609212113220149), following from decision 2.
6. **This intent refines itd-111** rather than bundling with or superseding
   it: it is shipped, so it cannot be bundled into a plan, and superseding
   would retire a capability that stands. It claims only what itd-111 lacks: a
   version on the generated page blocks.
7. **When a page and the running binary disagree, the binary wins**: the
   page's opening line sends the agent to `explain` before it composes an
   input, following from decision 1, since the cached plugin page and the
   provisioned binary move independently by design.

Ruled by the product thinker on 2026-10-30, after the planning interview:

11. **The intent is split three ways**, against the interview's earlier
    choice to keep it whole and as both pre-interview reviews advised: this
    record keeps learning from the running binary and the one source of
    command facts; refusal codes and next steps are itd-2610301428378830,
    which builds on this one because `abcd explain <code>` resolves a code;
    the practice run is itd-2610301428390406.

## Open Questions

_All resolved at the planning interview of 2026-10-30, in `## Decisions`
(scope, the pages' form, the overlaps and the page-against-binary rule); the
refusal-code question moved with itd-2610301428378830._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: agents are abcd's main users, and on 2026-10-30 one built an input from an out-of-date page and was refused; we expect fewer such refusals once agents ask the running program itself, and we are wrong if agents still build inputs from pages.
