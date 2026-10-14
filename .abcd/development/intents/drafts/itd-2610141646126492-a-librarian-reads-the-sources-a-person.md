---
id: itd-2610141646126492
slug: a-librarian-reads-the-sources-a-person
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: []
related_adrs: [adr-2610141646525117]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A librarian turns sources into a quoted reading pack

## Press Release

> **A person hands an agent what they are working on and a few sources, and
> gets back a private reading pack whose every quote matches the text filed for
> its source.** The librarian checks what is already filed, fetches and keeps
> new sources privately with their rights noted, reads each one properly (the
> right page of a scanned magazine, a review in another language, the items of a
> catalogue the person picks), and searches around them along nine broad lenses
> it reads for the subject at hand. A drill-down chases every finding that names
> nothing specific, and what stays vague is listed as an open question instead
> of padded.
>
> "I'm writing a game and gave it the scanned manual of an old one and a German
> review of another," said Alice, a solo founder. "The pack told me what both
> designs were built on, how they were received, which magazine's ten out of
> twelve that was, and that the manual is all rights reserved, so I can study it
> but not reuse a word of it. Each line pointed at the page it came from."

## Why This Matters

Turning a source into ideas is today done by hand, in whatever session happens to
need it, and leaves no trace: ideate has no step for a source at all
(iss-2610141647294219). The research around a source fares no better: abcd's
research agent has no step that files what it finds. The product thinker framed
the answer on 2026-10-14 as a librarian: ideate "is essentially consulting the
librarian first, then processes what the librarian brings (or the librarian does
it depending on the source) and then ideate works with that." The same engine
serves the research command, which turns a question into a shared research note
(the bundle's research intent).

## What's In Scope

- `abcd library pack`, the step an agent runs: it records a reading pack and the
  text the model read, after checking their shape. Its sub-verbs record, show
  and list packs. It is documented on the library command page, in a section for
  agents.
- The pack's shape, its store and its item ids, which ideate and the research
  command cite.
- The host protocol on that page: the steps below, the nine lenses, the form
  checks and the quote normalisation (listed under Design notes).

## What's Out of Scope

- Adding a source to the person's sources corpus: that stays the person's act
  through `library add`, which asks for its class.
- Shared research notes (the research intent) and ideate's use of packs
  (itd-2610141646452213).
- A retrieval index or text-extraction pipeline in abcd.

## What the librarian does

Given a subject and sources (web addresses, local files, or none):

1. **Narrow the subject.** A subject too broad to read the lenses by gets one
   narrowing question; in automatic mode the librarian records the narrowing it
   chose.
2. **Reuse what is filed.** A source whose canonical address or content
   fingerprint is already in the pack store or the person's corpus is reused,
   not fetched again, and the pack says so.
3. **Fetch and keep privately.** Fetched sources are kept in the librarian's own
   owner-only store beside the packs, never in a working tree. A source the
   person hands over as private (a local file, a saved page) is read alone:
   nothing from it is searched for or fetched. A web page the librarian fetched
   from a public address may be searched around, though it is kept privately.
4. **Read properly**, the right part of each document, page images where a text
   layer is garbled, a translation beside every quote not in English, and a
   catalogue shown to the person as a list to pick from.
5. **Search around, by lens**, along the nine lenses, writing down how each lens
   reads for the subject.
6. **Apply the form check** for each kind of document, nested ones included.
7. **Drill down** on every finding that names nothing specific, up to a cap;
   what stays vague becomes an open question.
8. **Hand back the pack**, ranked by usefulness to the subject, with that
   criterion stated.

## Mechanism

We expect each quote in a pack to match its source's filed text because the text
is fingerprinted when it is first filed, before any pack exists, and the binary
refuses a pack whose quote is not in that text or whose text has changed since;
and we expect fewer vague findings than an unaided run because every finding
must declare what it names, and a finding that declares nothing is drilled or
moved to open questions. Shown wrong if a recorded pack holds a quote absent from
its source's filed text, or if, over the test labs' kinds of source, most
findings still end as open questions.

The check proves that each quote matches the text the model filed; it does not
prove that text is faithful to the source. Where a later reader needs that, the
filed fingerprint lets them re-read and compare.

## Scope Conditions

- The host model reads the documents and runs every search and fetch; abcd
  stores what it is handed and checks the shape, the host-delegated default of
  adr-25. An optional retrieval adapter may later take over the reading.
- What the binary enforces: the store's owner-only mode, no writes into a working
  tree, the pack's required fields, the quote check, the fingerprint, and the
  removal of e-mail addresses, handles and the account's own name. What the
  command page instructs and the binary cannot see: that no search or fetch
  receives a private source's content, and which other names in a page to strip.
- One person on one machine; sharing a pack happens only through the research
  command's note.
- Lens 9 reads the repository the librarian runs in, read-only, and says "no
  fit" when the subject is not that project.

## Acceptance Criteria

- **Given** a subject and a web address, **when** an agent records a pack with
  `library pack record`, **then** the source and the text read are kept in the
  librarian's owner-only store with the text's fingerprint, and the working tree
  is unchanged.
- **Given** a source whose canonical address or fingerprint is already filed,
  **when** a pack is recorded, **then** the pack marks it reused and no second
  copy is kept.
- **Given** a pack item whose quote is not in its source's filed text after the
  declared normalisation, **when** the pack is recorded, **then** it is refused,
  naming the item.
- **Given** a source whose filed text has changed since it was fingerprinted,
  **when** a pack citing it is recorded, **then** it is refused.
- **Given** an OCR correction in the filed text, **when** it is recorded, **then**
  it is an overlay that keeps the original span and the page, within the
  declared size limit, and a larger correction is refused.
- **Given** a source whose terms forbid copying, **when** a pack is recorded,
  **then** only the quotes and the fingerprint of the original bytes are kept;
  the text is supplied for the check and discarded.
- **Given** a finding whose `specifics` field is empty, or names a value that is
  not in the finding's text, or names only the finding's own speaker, **when**
  the pack is recorded among the findings, **then** it is refused; ideas and
  principles are exempt.
- **Given** findings marked disputed, **when** the pack is recorded, **then** each
  names the others it disputes and every reference resolves; an item marked
  unsupported carries the citation that failed.
- **Given** a rating with no named source, or with neither a scale nor an
  unscored label, **when** the pack is recorded, **then** it is refused.
- **Given** a page holding e-mail addresses, handles or the account's own name,
  **when** it is kept, **then** they are removed, together with any other name
  spans the agent declared, and the count removed is recorded.
- **Given** the library command page, **when** it is read, **then** it carries the
  rule that a private source is read alone, as an instruction to the agent.
- **Given** a recorded pack, **when** it is read, **then** each of the nine lenses
  carries the librarian's written reading of it for the refined subject.

## Prior Art

- itd-36 (shipped): `library ingest`, the host-distils-and-the-binary-stores
  pattern this follows.
- itd-126 (shipped): the sources corpus and its classes, which the librarian
  leaves as the person's act.
- itd-164 (draft): licence verdicts with an explicit unknown; adr-2610141646525117
  narrows unknown rights to all rights reserved for the librarian only.
- iss-2610100649495706 (open): the request for a research command, answered by
  the bundle's research intent on this engine.

## SOTA

- Deep-research agents in model hosts (multi-step search with cited reports):
  mature as products; their reports are not checked against filed text.
- Retrieval-augmented generation with citation grounding (span-level attribution
  checks): maturing in research; needs an index abcd does not build.

**Path:** 2. A native floor: host-run reading with abcd checking the shape, the
quotes and the fingerprints; a retrieval adapter can take over the reading
later.

## Design notes

These move to the command page and the bundle's spec at planning.

The nine lenses:

   1. **Principles and key features:** what it is built on. For a game: its
      design principles, mechanics and visuals. For a development process: its
      key features and the assumptions behind them.
   2. **Context and timeline:** when and where, what came before and after. For
      history: the timeline and its storylines. For a game: its era, its
      platform's limits, its sequels and remakes.
   3. **Alternatives:** what else does the same job. For a product: its
      alternatives. For a process: other approaches. For history: other
      tellings of the same events.
   4. **Evidence and reception:** does it hold up. For history: corroboration,
      disputed facts, later retellings. For a game: reviews, ratings, rankings.
   5. **Use and best practice:** who it is for, its use cases, today's best
      practice.
   6. **People and sources:** who made it, who is telling it, and why.
   7. **Rights and media:** who owns each part, what may be used, which images
      to collect; rights not found means all rights reserved.
   8. **Open questions and glossary:** what the sources did not answer, and the
      terms they use, explained.
   9. **Fit to my project:** how the findings compare with the person's own
      project, read from the repository the librarian runs in (for example a
      design guide's contrast target against the project's own colours).

The form checks:

   - manual: written to sell and teach; its merit claims are marketing
   - memoir: compared with the author's earlier accounts and the record of the time
   - oral history: who interviewed, when, for what programme, and who is speaking
   - advertisement: its claims checked against reviews and manuals
   - review: the outlet, the reviewer (resolved from a byline or the imprint),
     and its verdict form: a score on a stated scale, an unscored verdict, or none
   - exhibit page and catalogue: who curates it, and what it leaves out
   - post and write-up: the author's stake; a write-up about one's own work is
     "verified as quoted", never "verified as true"
   - paper and report: who funded or published it, and which numbers are its own
   - guideline page: whose platform it serves, and its date
   - encyclopedia article: what it cites, and whether the citation holds
   - transcript: who transcribed it, and from what recording
   - archive record or index page: what it catalogues, and who uploaded it
   A scale stated elsewhere in the same document (a magazine's key to its
   stars) counts as known.

Quote normalisation: whitespace collapsed; typographic quote marks and
apostrophes folded to plain ones; a hyphen at a line end tried both kept and
joined; Markdown inline markup removed only from a page kept as Markdown, never
from plain or scanned text. Case is kept, unlike the memory store's normaliser,
because a quote's case is part of its wording.

## Open Questions

- The drill-down cap and its default, the reading budget for a long document or
  catalogue, the OCR overlay's size limit, and the pack's on-disk shape and item
  ids (technical facilitator, at planning; the bundle's shared spec).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
