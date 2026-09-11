---
id: itd-2609111003026787
slug: every-record-family-retires-a-record-the-same-way-so-a-maint
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Every record family retires a record the same way, so a maintainer learns one lifecycle surface instead of four

## Press Release

> _Seeded from a quoted-text intent capture, then grounded against the shipped
> surface. The press-release narrative is confirmed in the planning interview._

A maintainer finishing a piece of work retires its record with the verb they
already know, whatever family the record belongs to. Today they have to remember
four: an issue is `resolved` or `wontfix`, a spec is `closed`, and an intent is
shipped by a hook on somebody else's verb. Two retirements have no verb at all —
an intent reaching `superseded/` and an ADR being superseded are both done by
hand, by moving a file and writing an edge.

The proposal is that retirement reads the same everywhere: the same shape of
invocation, the same vocabulary for what happened, and a verb for each move a
record can actually make.

## Why This Matters

Minting already works this way, which is what makes the asymmetry visible. Every
write-side family holds a `recordid.Minter` and names its family tag, carrying no
allocator of its own (adr-45; `core/capture`, `core/spec`, `core/intent`,
`core/decide`, `core/reading`). One allocator, five families, no coordination
between checkouts. The birth of a record is uniform; its retirement is not.

What the shipped surface does today:

| Family | Terminal move | Verb |
| --- | --- | --- |
| issue | `open/` → `resolved/` | `abcd capture resolve` |
| issue | `open/` → `wontfix/` | `abcd capture wontfix` |
| spec | `open/` → `closed/` | `abcd spec close` |
| intent | `planned/` → `shipped/` | none of its own: a close-hook on `spec close` |
| intent | → `superseded/` | none (7 records moved by hand) |
| ADR | → superseded | none (60 records carry the edge, written by hand) |

Three costs follow, and the third is the one that bites.

**The vocabulary is four words for one idea.** Resolve, wontfix, close, ship. A
maintainer who has learnt one family has learnt that family only.

**Two moves have no verb**, so they are done by hand: a file is moved and an edge
is typed. That is the shape of work where a step gets forgotten, which is exactly
the argument the definition of done already makes about `spec close`.

**The intent's own retirement is reachable only through another record's verb.**
`spec close` ships the linked intent as a hook, so an intent whose spec was never
opened, or whose spec is closed already, has no path at all. The release cut does
not refuse such an intent; it simply does not see it, and it ships with no
changelog line. That is not hypothetical: `spc-26` sat open while `itd-121`'s
behaviour had been live for some time, and the gap was found by reading rather
than by any gate.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline). The bullets below are UNCONFIRMED PROPOSALS
> written from the evidence above, not approvals: the planning interview walks
> every one, and the maintainer accepts, edits or strikes each. The draft cannot
> be planned until they are theirs._

- **Given** a planned intent whose work has landed, **when** the maintainer
  retires it, **then** a verb of the intent's own makes the move, and it does not
  depend on a spec existing or being open.
- **Given** an intent being superseded by another, **when** the maintainer
  retires it, **then** a verb writes the typed edge on both records and moves the
  superseded one, with no file moved by hand.
- **Given** an ADR being superseded, **when** the maintainer records that,
  **then** a verb writes the edge both ways, so the standing rule that an ADR is
  never amended is enforced by the surface rather than by memory.
- **Given** any record family, **when** the maintainer retires a record, **then**
  the invocation has the same shape and the outcome is named from one closed
  vocabulary shared across families.
- **Given** the existing verbs, **when** this ships, **then** invocations that
  work today still work: the vocabulary may gain a shared form, and it does not
  break the spelling a reader has already learnt.

## Open Questions

- **Is one verb the goal, or one shape?** A single `abcd retire <id>` dispatching
  on the id's family is the strongest reading and the most disruptive; a shared
  shape per family, with the four words preserved, is the weaker one. The
  record-id dispatch form (`abcd itd-20`) shows the machinery for the strong
  reading already exists.
- **Does `resolved` versus `wontfix` survive?** Those two carry a real
  distinction — fixed, versus decided against — and a uniform vocabulary must
  keep it rather than flatten it. `closed` carries neither meaning today.
- **What happens to `spec close`'s hook?** If an intent gains its own retirement
  verb, the hook becomes a second path to the same move. Two paths is how records
  drift; one of them should probably call the other.
- **Is the ADR supersede edge a lifecycle move at all?** An ADR has no `superseded/`
  directory: it is superseded in place by a header. That may make it a different
  act wearing the same word, which is the argument for splitting it out.
- **Does this reach the reading families?** Admissions, dispositions and reading
  runs have their own fates. Whether they are in scope decides how much of the
  ledger this touches.
