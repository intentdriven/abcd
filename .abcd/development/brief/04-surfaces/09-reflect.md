# `/abcd:reflect` — Phase Retrospective

> **Delivery state**: `/abcd:reflect` is a design-target surface — no `reflect`
> verb exists on the shipped binary, no `commands/reflect.md` plugin command
> exists, and nothing this page describes is shipped: there is no `reflect-writer`
> capability or `--allow-empty`/`--overwrite` flag in the binary, no
> `reflection-composer` agent (the `agents/` catalog ships, but has no
`reflection-composer.md`), and no
> `.abcd/retrospectives/` output tree in the working tree. `.abcd/logbook/` is a
> different case: it is not an unbuilt design target but a **retired** location
> (see § Where the receipt lives, below). The backing intent sits in
> [`intents/planned/`](../../intents/planned/itd-24-reflect-command.md) (itd-24);
> delivery state is the intent lifecycle's, not this page's (see the [brief
> README's provenance note](../README.md)). The prose below records the design
> contract in present tense as the brief's intents do.

`/abcd:reflect <phase-id>` composes a structured retrospective for a **completed
phase** of the voyage (itd-24). It is **phase-only grain**: the per-intent form
was dropped (per-intent reflection is the `intent-auditor`'s Role 1).
The command markdown performs ZERO writes — every write goes through the
deterministic reflect-writer capability, which renders
`.abcd/retrospectives/<phase-id>/README.md`.

This surface doc records the design contract; the runtime behaviour (contract
verification, README write, consumed-receipt-path + phase/audit/member-spec
links) is owned by the predecessor store's task
`spc-83-operator-surfaces-manifest-lockstep.3` and the
command file `commands/reflect.md`.

## Argument

The command takes exactly one positional argument: a **phase id**, which is a
filename stem in [`roadmap/phases/`](../../roadmap/phases/) (e.g.
`phase-0-substrate`, `phase-5-run-seam`). It is NOT `itd-N` and NOT a
milestone/`spc-N` id. `/abcd:reflect <itd-N>` is refused — reflection is
phase-grained only.

Bare `/abcd:reflect` (no argument) renders help/state and writes nothing.

## What it does

1. Selects the **latest** spc-66 (predecessor store) phase-audit receipt whose
   `phase_id` matches the argument (newest `timestamp` wins), read from the
   local-ephemeral logs tier (see § Where the receipt lives).
2. Runs the `reflection-composer` agent as a seeded single-pass interview: five
   seeded questions drawn from the receipt's per-bullet acceptance verdicts.
   Thin answers trigger one clarifying question; a deliberately-empty section
   renders an explicit "none recorded" line.
3. Shells the deterministic writer with the collected answers as JSON. The
   writer renders the five-section README, records the consumed audit-receipt
   path, and links to the phase doc + audit report + member specs ONLY.

## The five-section template (enforced)

The retrospective README always carries these five sections, in this order:

| # | Section | Content |
|---|---------|---------|
| 1 | Went well | Successes and strengths, with specific examples |
| 2 | Could improve | Issues and gaps |
| 3 | Lessons learned | Transferable insights framed for future-you |
| 4 | Decisions made | Architectural / design choices crystallised in the phase |
| 5 | Metrics | Qualitative + simple counts (no DORA/velocity telemetry) |

An empty section triggers a clarifying question in the interview; a
deliberately-empty section renders an explicit "none recorded" line rather than
being omitted.

## Refusals

The writer refuses (each with a message naming the phase-audit prerequisite):

| Refusal | Condition |
|---------|-----------|
| Non-phase argument | `itd-N` or free text — phase-only grain |
| No reflection answers | A bare `{}` on stdin (hollow all-"none recorded") — refused unless `--allow-empty` |
| No spc-66 (predecessor store) audit receipt | No phase-audit receipt exists for the named phase |
| Empty-audited latest receipt | `member_specs` empty OR `done_total.total == 0` — nothing shipped to reflect on |
| Re-run without `--overwrite` | A retrospective already exists for the phase |

The writer also enforces write-site containment (defense-in-depth): the resolved
target must be inside `.abcd/retrospectives/`, and receipt-supplied
`member_specs[].spec_id` values are validated against the `spc-NN-slug` shape
before they are rendered into link text.

## Invocation model

The HOST session runs the reflection-composer interview per
`agents/reflection-composer.md`, collects the structured answers as a JSON object
(one key per section: `went_well`, `could_improve`, `lessons_learned`,
`decisions_made`, `metrics`), and pipes that JSON to the writer. The writer is
fully testable WITHOUT the agent — JSON answers in, README out. The writer is the
SINGLE dispatch target and the ONLY writer.

## Where the receipt lives

The receipt shape this surface consumes is the predecessor store's spc-66
phase-audit report, and the predecessor store wrote it to
`.abcd/logbook/audit/phase-<ts>/report.json`. **That location is retired here.**
A 2026-07-12 maintainer adjudication (iss-36 and iss-56, resolved as iss-73)
placed runtime artefacts in the gitignored `.abcd/.work.local/logs/` tier
instead, and a detector holds it: `TestNoRetiredLogbookLocationInSource` fails
the build if any non-test Go source under `internal/` so much as names
`logbook`. A delivered `reflect` therefore reads its receipt from
`.abcd/.work.local/logs/`, the tier the neighbouring `review-collator` row in
[`../05-internals/01-agents.md`](../05-internals/01-agents.md) already names;
the retired path survives in this record only as the predecessor store's, never
as a path to implement against.

## Output path and single-source-of-truth

Output is fixed at `.abcd/retrospectives/<phase-id>/README.md`, committed as
part of the phase's permanent record. That path is a peer of `.abcd/work/` and
`.abcd/development/`, not of `.abcd/development/intents/`, and it is **not one
of the three tiers** `AGENTS.md` fixes: delivering itd-24 has to place the tree
in an existing tier or record a decision admitting a fourth, and until then the
output path is a design target's proposal rather than a settled location. The
README LINKS to the phase doc, the audit report (its
receipt path recorded in the README), and each member spec — it never copies
their bodies. v1 links are limited to those three: the spc-66 (predecessor store) receipt carries no
intent ids, so intent links are a recorded future extension.

Canonical glossary terms (`voyage`, `persona`) are used in body prose; the README
records `glossary_terms_used: core/voyage, core/persona`.

## Lifeboat forward requirement (grill Q6)

The lifeboat must pack EVERY phase retrospective a voyage produced, so the full
reflection arc travels between voyages. Because `/abcd:reflect` is a design
target and produces no retrospectives yet, this is a **documented forward
requirement on the disembark pack** — NOT a behaviour this surface implements.
It is recorded here and in the
itd-24 intent acceptance so a later reader treats it as a requirement, not a
shipped capability.

## v1 scope

- Single seeded interview pass (per-bullet verdicts → five questions); multi-turn
  depth is a recorded future extension.
- No auto-triggering after phase close; reflection is deliberately on-demand.
- The draft's "offer to run the phase-fidelity-reviewer inline" on a missing
  audit is deferred — v1 refuses instead (recorded in the itd-24 intent).

## Related documentation

- Command file: `commands/reflect.md`
- Agent: `agents/reflection-composer.md` (the 16th catalog agent — see
  [`../05-internals/01-agents.md`](../05-internals/01-agents.md))
- Intent: `itd-24` (`../../intents/planned/itd-24-reflect-command.md`)
- The predecessor store's spc-66 phase-audit contract reflect is designed to
  consume: its phase-audit report schema is a design target for the Go binary
  (`internal/core/...`), not yet shipped
- Naming / VR001 registration: [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)
