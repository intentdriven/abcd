# `/abcd:ideate` — Gate an Idea Before It Becomes a Record Entry

`/abcd:ideate` puts a big, unproven idea through a three-leg admission gauntlet
and records the verdict — whether the idea survives or dies. Only a survivor
graduates to a draft intent, and ideate mints no intent itself.

It is **optional and never a gate**. No other verb requires it, nothing warns
when it is skipped, and capture friction stays at one line. The routing help in
the `intent` and `capture` surfaces names it for big, unproven ideas; that is a
pointer, not a precondition.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `record` | — | shipped |


## Division of labour

Ideate follows the host-delegated pattern the `disembark` synthesis family
proved. The **host** runs the three judgement legs as agents; the **binary** is
the deterministic frame that validates their output, proves every citation, and
writes the record. The binary never fetches a source, never calls a model, and
never decides whether an idea is any good.

## The three legs, in order

| # | Leg | What it produces |
|---|---|---|
| 1 | Primary-source research | A claims table: each load-bearing claim, the **primary** source it was checked against, and the finding (`verified` / `falsified` / `unverifiable`) |
| 2 | Record grill | Hits on the existing record, each cited by a record id the binary proves resolves, carrying a relation (`covered` / `contradicted` / `superseded`). The citation grammar admits four families and no more: `adr-N`, `itd-N`, `iss-N`, `spc-N`. The brief, the principles, the research notes and the decision log carry no citable id, so a hit on one of those rides in the `note` field of the nearest citable record |
| 3 | Adversarial review | Kill attempts, each with an outcome (`survived` / `partial` / `fatal`) |

The order is validated, not assumed: the legs travel as an ordered array and a
payload whose legs are missing, reordered, or duplicated is refused. Each leg
changes what the next one is looking at, so running them out of order produces a
different — and weaker — result.

Leg 3 is defined by its **conduct**: fresh-context, off-policy, unknown
authorship. The evaluator did not conduct the research and receives the idea as
an artefact whose provenance it does not know. That is the
evaluator-outside-the-loop principle applied to ideas, and it is the one measured
debiasing effect the protocol rests on. The binary cannot observe how an agent
was run, so this obligation lives in the command page's orchestration script,
which instructs the host to strip authorship and prior-leg framing before the
hand-off.

## Behaviour

```bash
abcd ideate record <idea-slug> --verdict-json <file|-> --json
```

`--verdict-json` is **required**, exactly as `disembark graveyard
--lessons-json` is. Three of `disembark`'s four synthesis verbs do carry a
deterministic fallback: `press-release`, `principles` and `review` each run
evidence-only when their `--*-json` is absent, because a packed lifeboat's own
files carry the evidence they need. `graveyard` has none, and neither has this
verb: there is no evidence-only verdict an idea could have, and a binary that
invented one would be doing the judging.

The verb writes two things:

- `.abcd/development/research/notes/YYYY-MM-DD-ideate-<idea-slug>.md` — the verdict
  record: the idea as captured, the three legs, the verdict, and the rejected
  alternatives, rendered for a human.
- one dated pointer line in `.abcd/work/DECISIONS.md`.

No new record family: a killed idea is a research outcome, and the research
directory is where a session looks before re-proposing one.

**Both writes are committed, so both are redacted before either is made.** The
verb takes intent's fail-closed posture rather than capture's redact-and-report
one: a repository whose scanner configuration cannot be read refuses the run
outright, because recording under a detector the verb cannot trust is worse than
not recording. The gate is two-stage. Stage one redacts the free-text FIELDS —
the idea, every claim, note, kill attempt and rejected alternative — before the
renderer sees them, so the redaction is applied to the inputs and the markdown
escape stays the last transformation. Stage two re-scans the rendered record and
its `DECISIONS.md` pointer line, and a span that survives is a whole refusal with
nothing written, naming the kinds it found and never the text. A literal `$HOME`
sweep runs after the scanner as defence in depth. The count comes back on the
result as `redactions`, and a non-zero count is reported to the caller: a record
that no longer says what its author wrote is a fact the author needs, and
redacting in silence is how a redactor is discovered by its damage.

Exit codes are the release-cut shape without the middle state: `0` recorded, `2`
refused with nothing written. There is no exit 1, because a verdict is either
recordable or it is not.

## What the binary refuses

| Refusal | Why |
|---|---|
| A payload with no `schema_version`, or one this build does not support | The gate is three-branched: absent, too new for this build, and otherwise unsupported are each their own refusal, so a reader is told which of the three happened |
| A payload with no semver `prompt_version` | The verdict is the output of a named prompt at a named version; an unstamped payload cannot be tied back to the definition that produced it, and `commands/ideate.md` carries both fields in its payload block |
| A cited record id that does not resolve in the repository | A grill hit on a record that does not exist is a hit on nothing. The refusal names every offending id |
| A cited value that is not a record id at all | Bounded before it is matched or echoed |
| Legs missing, reordered, or duplicated; a leg carrying another leg's evidence | The gauntlet is exactly three legs, in order |
| An out-of-enum verdict, claim status, grill relation, or kill outcome | Each set is closed; an unregistered value is a whole-document refusal, never a coercion |
| A claim naming no primary source | A claim checked against nothing is an assertion |
| An empty `rejected_alternatives` list with no explicit `no_rejected_alternatives` marker | Silence and "nothing was weighed" are indistinguishable to a later reader, and the record exists to stop the idea being re-litigated |
| A slug that is not lower-case kebab-case | The slug becomes a filename; the grammar is the lexical half of the write containment, `os.Root` the other |
| A verdict record that already exists for this slug and date | Overwriting would erase a recorded reason, which is the one thing the verb exists to preserve |
| A repository with no `.abcd/work/DECISIONS.md` | A record nothing points at is a record nobody finds — refused before anything is written |
| A symlinked component anywhere in `.abcd/development/research/notes/` | The write goes through one `os.Root` opened at the repository root, which refuses symlink traversal at every level, not just the leaf |

The research directory itself is **created when absent**: nothing else in abcd
establishes it and no convention check requires it, so refusing would fail the
first run in every repository — after the three host legs have already been paid
for, and with the verdict unrecoverable when it arrived on stdin. The exclusive
create is the no-overwrite guarantee, and the decision log's read-modify-write
runs under the same advisory lock the ledger allocators use.

Refusals are **whole-document**, never cite-or-be-dropped. A verdict record with
a quietly-dropped falsified claim or grill hit is worse than no record, because a
later session trusts it.

## References

- Plugin command: [`commands/ideate.md`](../../../../commands/ideate.md)
- Intent: [`itd-104`](../../intents/shipped/itd-104-abcd-gates-a-new-idea-before-it-becomes-a-record-entry-resea.md)
- Spec: [`spc-18`](../../specs/closed/spc-18-abcd-gates-a-new-idea-before-it-becomes-a-record-entry-resea.md)
- Routing-help neighbours: [`05-intent.md`](05-intent.md), [`06-capture.md`](06-capture.md)
