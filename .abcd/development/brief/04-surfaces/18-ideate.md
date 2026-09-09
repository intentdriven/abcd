# `/abcd:ideate` — Gate an Idea Before It Becomes a Record Entry

A big, unproven idea is cheap to propose and expensive to re-litigate. Six months
later nobody remembers whether it was rejected on evidence or never really
examined, so it comes back. `/abcd:ideate` puts one idea through a three-leg
admission gauntlet and leaves a dated record of the verdict — for a survivor and
for a casualty alike, with the alternatives that were weighed.

What it costs: three host-run judgement legs, which is real time. What it buys:
a survivor arrives at `drafts/` already checked, and a casualty leaves a reason a
later session can find before proposing the same thing again.

It is **optional and never a gate**. No other verb requires it, nothing warns when
it is skipped, and capture friction stays at one line. The routing help in the
`intent` and `capture` surfaces names it for big, unproven ideas; that is a
pointer, not a precondition. Ideate mints no intent of its own — a survivor is
promoted by hand.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `record` | — | shipped |

## Who does what

Ideate follows the host-delegated pattern the `disembark` synthesis family
proved. The **host** runs the three judgement legs as agents; the **binary** is
the deterministic frame that validates their output, proves every citation, and
writes the record. The binary never fetches a source, never calls a model, and
never decides whether an idea is any good.

## The three legs, in order

| # | Leg | What it produces |
|---|---|---|
| 1 | Primary-source research | A claims table: each load-bearing claim, the **primary** source it was checked against, and the finding (`verified` / `falsified` / `unverifiable`) |
| 2 | Record grill | Hits on the existing record, each cited by a record id the binary proves resolves, carrying a relation (`covered` / `contradicted` / `superseded`) |
| 3 | Adversarial review | Kill attempts, each with an outcome (`survived` / `partial` / `fatal`) |

The order is validated rather than assumed: the legs travel as an ordered array,
and a payload whose legs are missing, reordered, or duplicated is refused. Each
leg changes what the next one is looking at, so running them out of order
produces a different and weaker result.

Only records carry citable ids, and leg 2's citations are held to that: the
brief, the principles, the research notes and the decision log have no id to
cite, so a hit on one of those rides in the `note` field of the nearest citable
record.

Leg 3 is defined by its **conduct**: fresh context, off policy, unknown
authorship. The evaluator did not conduct the research and receives the idea as
an artefact whose provenance it does not know. That is the
evaluator-outside-the-loop principle applied to ideas, and it is the one measured
debiasing effect the protocol rests on. The binary cannot observe how an agent was
run, so this obligation lives in the command page's orchestration script, which
instructs the host to strip authorship and prior-leg framing before the hand-off.

## Behaviour

```bash
abcd ideate record <idea-slug> --verdict-json <file|-> --json
```

`--verdict-json` is **required**. There is no evidence-only fallback, as there is
for three of `disembark`'s four synthesis verbs, because there is no evidence-only
verdict an idea could have: a binary that invented one would be doing the judging.

The verb writes the verdict record as a dated research note under
`.abcd/development/research/notes/`, and one dated pointer line in
`.abcd/work/DECISIONS.md`. No new record family: a killed idea is a research
outcome, and the research directory is where a session looks before re-proposing
one.

Exit codes are the release-cut shape without the middle state: `0` recorded, `2`
refused with nothing written. There is no exit 1, because a verdict is either
recordable or it is not.

## Both writes are committed, so both are redacted first

The verb takes intent's fail-closed posture rather than capture's
redact-and-report one: a repository whose scanner configuration cannot be read
refuses the run outright, because recording under a detector the verb cannot
trust is worse than not recording.

The gate is two-stage. Stage one redacts the free-text fields before the renderer
sees them, so the redaction applies to the inputs and the markdown escape stays
the last transformation. Stage two re-scans the rendered record and its pointer
line, and a span that survives is a whole refusal with nothing written, naming
the kinds it found and never the text. The redaction count comes back on the
result, and a non-zero count is reported to the caller: a record that no longer
says what its author wrote is a fact the author needs, and redacting in silence
is how a redactor is discovered by its damage.

## What the binary refuses

Refusals are **whole-document**, never cite-or-be-dropped. A verdict record with
a quietly-dropped falsified claim or grill hit is worse than no record, because a
later session trusts it.

The refusals a caller actually meets fall into four groups, each protecting
something a later reader depends on:

- **The payload cannot be tied to a definition**: no `schema_version`, one this
  build does not support, or no semver `prompt_version`. A verdict is the output
  of a named prompt at a named version, and an unstamped payload cannot be traced
  back to what produced it.
- **The evidence does not hold up**: a cited record id that does not resolve, a
  cited value that is not a record id at all, a claim naming no primary source, an
  out-of-enum verdict or outcome. Every closed set is closed, and an unregistered
  value refuses the document rather than being coerced.
- **The record would mislead a later reader**: an empty `rejected_alternatives`
  list with no explicit marker saying nothing was weighed, because silence and
  "nothing was weighed" read identically; or a verdict record that already exists
  for this slug and date, because overwriting erases a recorded reason.
- **The write is not safely containable**: a slug that is not lower-case
  kebab-case, a symlinked component under the research directory, or a repository
  with no `.abcd/work/DECISIONS.md` for the pointer line to land in.

The research directory itself is created when absent: nothing else in abcd
establishes it, so refusing would fail the first run in every repository, after
the three host legs have already been paid for. The create is exclusive, which is
the no-overwrite guarantee, and the decision log's read-modify-write runs under
the same advisory lock the ledger allocators use.

## References

- Plugin command: [`commands/ideate.md`](../../../../commands/ideate.md)
- Intent: [`itd-104`](../../intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes-a-record-entry-resea.md)
- Spec: [`spc-18`](../../specs/closed/spc-18-abcd-gates-a-new-idea-before-it-becomes-a-record-entry-resea.md)
- Routing-help neighbours: [`05-intent.md`](05-intent.md), [`06-capture.md`](06-capture.md)
