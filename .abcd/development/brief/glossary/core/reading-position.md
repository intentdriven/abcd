---
term: reading-position
bounded_context: core
definition: One of the four questions a cold reading can be commissioned to answer — widening, entailment, comparative or detection. The position fixes the reading's object, its question and the supply regime its output is validated against; `abcd reading assemble --position` names it.
aliases: ["reading position", "position"]
forbidden_synonyms: ["role", "perspective", "viewpoint", "lens"]
status: stable
introduced_in: itd-184
starts_when: null
ends_when: null
not_to_be_confused_with: core/oracle
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# reading-position

A **reading position** is one of four questions a cold reading may be asked, each with its own
object, its own output shape, and its own supply regime — the kind of claim its answers are
allowed to be. The position is a closed operand of
[`abcd reading assemble`](../../04-surfaces/23-reading.md), and it decides what the assembler
passes: blindness is a property of the input, so a position sees the intersection of its
declared includes with the scope it was commissioned about.

| Position | The question | Supply regime |
|---|---|---|
| widening | Given the situation as this design construes it, what configurations does the construal admit that are not present in what has been committed to? | generative |
| entailment | What does this design commit to, by being the kind of thing it is, that its articulation does not state? | explicative |
| comparative | For each candidate and each declared criterion, how do options of this shape ordinarily behave? | evaluative |
| detection | Where is the shipped tree in tension with the claim record? | registrative |

## The spelling

The record writes the full phrase on first use and drops to bare "position" inside the reading
chapter and the readings charter, where the include table is described as positive at every
grain — "a position, a source directory, a file match". The glossary fixes **reading position**
as the form to use outside those two documents (iss-2609012245352480): "position" alone is a
common English word, and the surface's own `--position` flag is the only other place it is
unambiguous.

**A position is not an agent, and not a surface.** The four definitions live under `agents/`
because that is where the harness looks, but the position is the question; the agent is one
host's way of answering it. The [surface](surface.md) is `/abcd:reading`, which assembles the
input and validates the output — and never runs a reading.

## When to use

Use "reading position" whenever the subject is which question a reading was commissioned to
answer, or which supply regime its output is checked against.

## When NOT to use

Do not use it for a review [oracle](oracle.md) or for one of the intent auditor's roles: those
judge a specific artefact against its own claims. Do not use bare "position" in a document that
also discusses phases, scope or surfaces.

## Examples

- "`comparative` does not assemble and refuses: its object has no channel."
- "`ingest --reading-json` validates the output against the regime the position's definition licenses."

## Related terms

- [construal](construal.md) — what the widening position reads against
- [ledger](ledger.md) — ledger content is what a reading must not see
- [oracle](oracle.md) — a model invoked to review; a position is a question, not a reviewer
- [surface](surface.md) — `/abcd:reading`, the front door that assembles and validates
