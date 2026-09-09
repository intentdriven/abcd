# Prompt Quality Infrastructure

An agent prompt is code nobody compiles. It rots as models evolve and prompting
practice shifts, and it rots silently: a prompt that has quietly stopped working
returns plausible output, which is the failure mode hardest to notice. The
machinery here exists so that rot leaves a trace.

What ships today makes four things impossible to do by accident: shipping a prompt
with no version, changing one without a changelog entry, declaring that a prompt
reads untrusted input without carrying a canary for it, and leaving the
untrusted-input question unanswered. What ships does **not** run any prompt or
judge any output — checking that a prompt still works needs a test harness, and
that harness is a design target.

The design is three layers plus the itd-5 additions. **Layer C ships and is armed
as a blocker; layers B and D are staged**, and each claim below says which it is.
[`01-agents.md`](01-agents.md) is the field-by-field register of what a prompt
declares.

## C — the prompt linter (ships)

`agent_contract`, in `internal/core/lint`, is record-lint's dedicated rule for the
agent-prompt tree. `agents/` sits outside both the record-lint and docs-lint roots,
so the per-file rules do not reach it; this rule walks the tree directly from the
`agents_dir` key in the record-lint configuration, skipping directories,
non-markdown files, and the README and changelog stems. It is configured as a
blocker, so it runs on every `make record-lint`, every `make preflight` and the CI
record gate. The operator-facing statement of the same contract is
[`agents/README.md`](../../../../agents/README.md).

What it enforces on every invocation:

- A `reads_untrusted_input` declaration on every prompt, whatever its value. The
  declaration is required of all of them, not only of the ones that admit `true`: a
  rule that fires only on `true` is one a prompt opts out of by deleting a line,
  and silence is not `false`, it is undeclared.
- `prompt_version` present and a valid semver, on every prompt. It is what the
  changelog entry is keyed on, whatever the prompt reads.
- On a prompt declaring `reads_untrusted_input: true`: both `capability_scope`
  fields, present and non-empty.
- On the same prompt: `agents/<name>/fixtures/injection-canary.json`, present, a
  regular file and non-empty. An empty file or a symlink is refused, because a
  canary that asserts nothing reports the contract met without testing it.
- A `### <agent> <version>` entry in `agents/CHANGELOG.md` for every prompt's
  current version. This half needs no git, so a new prompt with no entry and a
  bumped version with no entry both fail.

One further check runs only over an armed diff range: a prompt whose body changed
in the range without its version changing. That is the one thing the tree cannot
say on its own, and it is an edit that can never acquire a changelog entry, because
the entry is keyed on the version. The range comes from the CI caller, never from
the in-tree config, on the same reasoning as the receipt gate: a gate a committer
can point at an empty range is a gate a committer can disarm.

**Staged on top of what ships:** set-membership of `capability_scope.task_classes`
against a closed enum. The shipped check requires the field to be non-empty and
reads no enumeration. The enum's source of truth today is the reserved-vocabulary
table in [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md), prose
the binary does not read, so the membership check has no artefact to measure
against until the enum acquires a machine-readable home (iss-265).

## B — golden-test fixtures (staged)

The design target: each agent ships two or three fixture inputs with an expected
output structure, validated by schema and judged by an oracle for whether the
output is good enough, run in CI by a generic harness.

Neither half exists. There is no `internal/core/prompttest` package, and every
shipped agent's `fixtures/` directory holds exactly one file, the injection canary,
which is layer C's presence check rather than a golden test. The harness lands with
the first Pass-A agent spec ([Phase 6](../../roadmap/phases/phase-6-lifeboat.md)),
the point at which a second agent exists to generalise the runner over.

This is the layer that would catch a regression when a model changes, and it is
also what would **run** an injection canary rather than merely require one.

**Deferred to the prompt-test spec with it:** the broader structural checks —
missing role definitions, missing output schemas, vague instructions, missing
example input and output, prompt-length outliers, and the research and audit
footers. The linter catches structural issues; it does not catch semantic quality.

## D — periodic SOTA audit (staged, research-gated)

The design target: an audit prompt run once per minor release through the oracle
adapter over all agent prompts, checking each against its own research file rather
than against general knowledge — where does this prompt diverge from what the
research recommends, and is the divergence justified? Findings would land in the
local ephemeral tier and be treated as RFC input, never auto-applied.

Nothing of this ships: no template, no adapter entry point, no invocation. It is
gated on the research files besides, which do not exist for any shipped agent.

## The itd-5 additions

- **`prompt_version` frontmatter (ships).** Every prompt carries a semver, initial
  value `0.1.0`, and `agents/CHANGELOG.md` records each bump with a one-line
  rationale. Bump rules, semver-adapted: MAJOR for a behaviour-breaking output
  schema change, MINOR for a behaviour change preserving the schema, PATCH for a
  typo or non-behavioural edit.
- **The `0.x` calibration band (ships).** itd-81 amends itd-5 and governs over this
  brief's earlier expectation of a measured delta per bump. `0.x` means shipped and
  wired, honestly unmeasured; `1.0.0` means measured against a corpus and locked,
  and the lock must be earned. So the self-improvement outcome and the
  calibration delta are recorded at lock, not at each bump. Every shipped prompt
  sits in the `0.x` band, and every changelog entry today records the delta as
  unmeasured.
- **One-shot oracle self-improvement pre-flight (staged).** Before a prompt locks
  at `1.0.0`, the author submits it to a reviewer with a rewrite-for-clarity
  directive, runs the goldens against both variants, and accepts the reviewer's
  variant only if it scores at least as well and is more than 10% shorter. No agent
  has reached `1.0.0`, and the goldens the comparison needs are layer B, so the
  gate has not fired for any shipped prompt: each records exactly that in its first
  changelog entry.
- **Injection-canary fixtures (ships, as a presence check).** Every agent reading
  untrusted input carries at least one fixture with a prompt-injection payload, and
  `agent_contract` refuses a prompt that declares `true` without a regular,
  non-empty one. Every shipped prompt declares `true` and carries one. *Running*
  the canary and judging that the hostile text is quoted as data rather than obeyed
  needs the layer-B harness, so "failing the canary blocks the agent's spec from
  closing" is staged with it.

## Research-driven prompts (staged)

The design target: every agent has a research file under
`.abcd/development/research/prompting/agents/<name>.md`, holding current practice
for that agent's role. Research is a **gate** rather than a **source**: the author
writes the prompt informed by it, and the audit checks alignment after the fact, so
the author keeps their freedom and the auditor gets ammunition.

Three research files exist today, alongside a template and the directory's own
README; none names a shipped agent, and no shipped prompt has one. The one general
baseline that does ship is
[`01-general-best-practices.md`](../../research/prompting/01-general-best-practices.md),
covering cross-cutting prompting practice.

## Per-agent spec acceptance

Two of the six acceptance lines hold today, both enforced by `agent_contract`. The
rest are staged, and no shipped prompt satisfies any of them.

| Acceptance line | State |
|---|---|
| Prompt carries `prompt_version` frontmatter | ships (`agent_contract`) |
| Injection-canary fixture present for an agent reading untrusted input | ships (`agent_contract`, presence only) |
| Research file exists and reads as non-trivial under review | staged |
| Prompt cites its research file in a footer | staged: no shipped prompt carries one |
| At least two golden-test fixtures pass | staged with the layer-B harness |
| Prompt carries a last-audited footer line | staged: no shipped prompt carries one |

**In a later phase, recorded as intents:** itd-14, a prompt registry with a
full diff-on-update workflow treated like code; and itd-15, running the prompt
audit as part of abcd's own disembark of a reference implementation.
