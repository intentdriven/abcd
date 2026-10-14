---
id: itd-2610141646126492
slug: a-librarian-reads-the-sources-a-person
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A librarian turns sources into a quoted reading pack

## Press Release

> **A person hands abcd's librarian what they are working on and a few sources,
> and gets back a private reading pack they can trust line by line.** The
> librarian checks the shelf first for sources already filed on the subject,
> fetches and files new ones privately with their rights noted, reads each one
> properly (the right page of a scanned magazine, a review in another language,
> the items of a catalogue the person picks), and searches around them along
> eight broad lenses that it reads for the subject at hand. A drill-down pass
> chases every finding that names no source or nothing specific, and what stays
> vague is listed as an open question instead of padded. Every item in the pack
> carries the quote it rests on.
>
> "I'm writing a game and gave it the scanned manual of an old one and a German
> review of another," said Alice, a solo founder. "The pack told me what both
> designs were built on, how they were received, which magazine's ten out of
> twelve that was, and that the manual is all rights reserved, so I can study it
> but not reuse a word of it. Each line pointed at the page it came from."

## Why This Matters

Turning a source into ideas is today done by hand, in whatever session happens to
need it, and leaves no trace: the lab's first finding (iss-2610141647294219) is that
ideate has no step for a source at all, so the runner's choice of which ideas
to take and how to word them shaped every later leg and was recorded nowhere.
The research around a source fares no better. abcd's `sota-researcher` agent
finds things and keeps nothing, and another managed repository has already asked
for research that is kept (iss-2610100649495706).

The librarian is the product thinker's own reframe: ideate "is essentially
consulting the librarian first, then processes what the librarian brings (or the
librarian does it depending on the source) and then ideate works with that." It
is usable on its own as well as from ideate, because the reading pack is worth
having whether or not an idea is being judged.

Decided in the product thinker's planning interview of 2026-10-14 (session abcd-66), which followed lab lab-261014142726-9c4d759, a hand-run of /abcd:ideate on one external write-up.

## What the librarian does

Given a subject (what the person is working on) and sources (web addresses,
local files, or none):

1. **Shelf first.** Sources already filed in the person's sources corpus on this
   subject are reused, never fetched twice.
2. **Fetch and file privately.** New sources are filed in the corpus as
   confidential by default, with their rights noted. Rights that cannot be
   established are recorded as all rights reserved (adr-2610141646525117).
3. **Read properly.** The model reads each document, scans included, and finds
   the part that matters (one review on page 52 of a 116-page magazine; one
   advertisement on one page of an issue). A source in another language is
   quoted in its own words with a translation beside it. A catalogue or list
   page is shown as a list and the person picks the items to read; auto mode
   picks the most relevant few.
4. **Search around, by lens.** Eight broad lenses apply to every subject, each
   read for the context, and the librarian writes down how it read each lens so
   the person can correct it:

1. **Principles and key features:** what it is built on. For a game: its design
   principles, mechanics and visuals. For a development process: its key features
   and the assumptions behind them.
2. **Context and timeline:** when and where, what came before and after. For
   history: the timeline and its storylines. For a game: its era, its platform's
   limits, its sequels and remakes.
3. **Alternatives:** what else does the same job. For a product: its
   alternatives. For a process: other approaches.
4. **Evidence and reception:** does it hold up. For history: corroboration,
   disputed facts, later retellings. For a game: reviews, ratings, rankings.
5. **Use and best practice:** who it is for, its use cases, today's best practice.
6. **People and sources:** who made it, who is telling it, and why.
7. **Rights and media:** who owns it, what may be used, which images to collect;
   rights not found means all rights reserved.
8. **Open questions and glossary:** what the sources did not answer, and the
   terms they use, explained.

   The kind of document adds one check of its own: a review's outlet and scoring
   scale; an advertisement's claims checked against reviews and manuals, because
   advertisements exaggerate; an oral history's occasion (who interviewed, when,
   for what programme); a memoir's distance from the events.
5. **Drill down.** A finding that names no source or nothing specific (a name, a
   date, a number or a quote) is searched again on exactly that gap,
   automatically and up to a capped number of rounds; the person can also ask
   for a deeper search on any lens. What stays vague after the cap is listed
   under lens 8, never padded.
6. **Hand back a reading pack:** ideas, principles, claims and references, each
   with its source, its quote, its lens and its form check, ranked.

Ratings and rankings enter the pack only with a named source and a known scale;
reader and anonymous ratings are left out (adr-2610141646525117).

## Mechanism

We expect the pack to be trustworthy because every item is anchored to a filed
source and a verbatim quote that the binary can find in the stored text, so a
fabricated or misremembered item cannot be written; and we expect it to be
specific rather than vague because the drill-down turns the vague findings into
either a specific one or an open question, and the binary refuses a finding
with neither a source nor a specific. The broad lenses work across subjects
because the librarian records its reading of each lens for the subject, which
the person can see and correct.

## Scope Conditions

- The host model reads the documents. abcd builds no text-extraction or retrieval
  pipeline (no PDF tool, no OCR, no search index): it stores what the model hands
  it and checks the shape (T8 of the interview, under the principle "basics built
  in, an external dependency brings full power"). An optional retrieval adapter
  may later take over the reading.
- The librarian is a sub-verb family of `abcd library` with a `block: agents`
  command page: the host runs the work, the binary validates and stores, as
  `ideate record` does. It adds no top-level verb.
- Reading packs are private, under `~/.abcd.noindex/library/<root-sha>/packs/`,
  owner-only; nothing the librarian writes is committed.
- One person on one machine; sharing a pack is out of scope.

## Acceptance Criteria

- **Given** a subject and a web address, **when** the librarian runs, **then** the
  source is filed in the sources corpus as confidential with its rights noted, and
  a source with no rights statement is recorded as all rights reserved.
- **Given** a source already filed on the subject, **when** the librarian runs
  again, **then** it is reused from the shelf and not fetched again.
- **Given** a reading pack whose item quotes a sentence that is not in the stored
  text of its source, **when** the pack is recorded, **then** the binary refuses
  it and names the item.
- **Given** a pack finding with no source or nothing specific (no name, date,
  number or quote), **when** it is recorded, **then** the binary refuses it.
- **Given** a rating with no named source or no scale, **when** the pack is
  recorded, **then** the binary refuses it.
- **Given** a source not in English, **when** an item quotes it, **then** the item
  carries the original words and a translation.
- **Given** a catalogue page, **when** the librarian reads it outside auto mode,
  **then** the person is shown the items and picks which to read.
- **Given** a finding left vague after the drill-down cap, **when** the pack is
  written, **then** it appears under open questions, not among the findings.
- **Given** a recorded pack, **when** its files are listed, **then** each lies
  under `~/.abcd.noindex/library/<root-sha>/packs/` with owner-only permissions,
  and the working tree is unchanged.
- **Given** a recorded pack, **when** it is read, **then** each of the eight lenses
  carries the librarian's written reading of it for the subject.

## Open Questions

- The cap on drill-down rounds and its default (technical facilitator, at
  planning).
- The pack's on-disk shape and its id, so ideate can cite an item
  (technical facilitator, at planning; the bundle's shared spec).
- Whether the open research-verb request (iss-2610100649495706) is answered by
  the librarian run on its own, or stays a separate verb (product thinker).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
