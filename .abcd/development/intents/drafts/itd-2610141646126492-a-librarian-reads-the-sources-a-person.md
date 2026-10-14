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
> nine broad lenses that it reads for the subject at hand. A drill-down pass
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

1. **Narrow the subject.** A subject too broad to read the lenses by ("a bit of
   computing history") gets one narrowing question; in auto mode the librarian
   writes down the narrowing it chose. The refined subject is kept in the pack.
2. **Shelf first.** Sources already filed in the person's sources corpus on this
   subject are reused, never fetched twice.
3. **Fetch and file privately.** New sources are filed in the corpus as
   confidential by default. People's names in a saved page that are not the
   source (the saver's own name, commenters' names, e-mail addresses and
   handles in its metadata) are stripped before filing. When a source's own
   terms forbid copying (a manual's "no part may be copied to any electronic
   medium"; a site's ban on downloading), only the quotes and a fingerprint of
   the text are kept, and a quote is re-checked by reading the source again. A
   private copy of a public document records the public address it names,
   without fetching it.
4. **Read properly.** The model reads each document, scans included, and finds
   the part that matters (one review on page 52 of a 116-page magazine; one
   advertisement on one page of an issue). Where a text layer is garbled (an
   angled scan, interleaved columns) the model reads the page images instead.
   The text it read, in reading order, is stored with the pack, and the pack
   says what was not read (pages past the reading budget, images it could not
   open). A source in another language is quoted in its own words with a
   translation beside it, kept in the private pack even where the publisher
   reserves translation. A catalogue or list page is shown as a list and the
   person picks the items to read; auto mode picks the most relevant few and
   records why.
5. **Search around, by lens.** Nine broad lenses apply to every subject, each
   read for the context, and the librarian writes down how it read each lens so
   the person can correct it:

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

   Searches go around public sources only: a private source is read alone, and
   nothing from it (text, headings, claims) is sent to any search or fetch
   service (adr-2610141646525117).
6. **Apply the form check.** Every kind of document has a check of its own, and
   a document inside another (an advertisement shown on a museum page; an
   interview in a catalogue) gets its own as well:
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
7. **Drill down.** A finding is specific when its content names something: a
   named person, organisation, product or place the finding is about, a date or
   a number. A quote alone, or the name of whoever said it, does not make a
   finding specific. Ideas and principles are proposals, not facts, and are not
   held to this test. A
   finding that is not specific is searched again on exactly that gap,
   automatically and up to a capped number of rounds; the person can also ask
   for a deeper search on any lens. What a drill-down turns up is linked to the
   finding it came from. What stays vague after the cap is listed as an open
   question, never padded. Two specific findings that disagree (three named
   sales figures for one machine), or a source that contradicts itself, are
   kept side by side as disputed. A specific claim whose cited source does not
   say it is marked unsupported. A drill-down may be settled inside the same
   source (another page) as well as by a search.
8. **Hand back a reading pack:** ideas, principles, claims and references, each
   with its source, its locator (document page, printed page, column or
   section), its quote and the quote's provenance (seen at the source, or
   reported via a reprint or summary; a search summary is never a quote), its
   lens, its form check, its rights, and the date it was found; ranked by
   usefulness to the refined subject, with that criterion stated. The glossary
   of lens 8 is kept apart from the open questions, and a pack names the other
   packs it connects to (two games from the same studio).

**Quotes and the stored text.** A quote is checked against the text stored for
its source after one declared normalisation: whitespace collapsed; typographic
quote marks and apostrophes folded to plain ones; a hyphen at a line end tried
both kept and joined; markup removed only from a markup source (a web page),
never from plain or scanned text. A quote is faithful to the page; where the
stored text is a scan's OCR and differs, the correction is marked in the stored
text and the quote names it. Pages found while searching that the pack quotes
are kept the same way as the source, under the same terms rule. A person whose
name changed is named by their current name, with the credited name quoted.

Rights are recorded per part, not per source: a page's text, its images, its
comments and the artefacts it links to can each carry their own terms, and so
does every page found while searching around it that the pack quotes. A page
readable by people but barred for use by AI is recorded as such. Rights that
cannot be established are recorded as all rights reserved (adr-2610141646525117).

A rating enters the pack only with a named source: a score with a known scale,
or a named critic's verdict without a scale, labelled unscored. Reader and
anonymous ratings, and ratings known only second-hand, are left out (adr-2610141646525117).

## Mechanism

We expect the pack to be trustworthy because every item is anchored to a filed
source and to a quote the binary can find in the text the model stored for that
source, with the quote's provenance said; and we expect it to be specific
rather than vague because specificity is judged by content, and the drill-down
turns every vague finding into a specific one or an open question. The broad
lenses work across subjects because the librarian records its reading of each
lens for the subject, which the person can see and correct.

The test labs of 2026-10-14 (lab-261014165333-8d28f83: 14 sources of nine
kinds, from a scanned 1986 game manual to private research reports; and its
second round, lab-261014193413-981af0b, on five of the hardest) shaped this
text: its findings are why specificity is judged by content, why the
stored text is what the model read, why every form has a check, and why
rights are per part.

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
  owner-only; nothing the librarian writes is committed. The privacy promise
  covers what abcd writes; the host's own stores (its session transcripts, its
  fetch caches) hold whatever the model read, and the command page says so.
- Lens 9 reads the repository the librarian runs in, read-only, and says "no
  fit" when the subject is not that project (writing a game, from inside abcd).
- One person on one machine; sharing a pack is out of scope.

## Acceptance Criteria

- **Given** a source whose terms forbid copying, **when** it is filed, **then**
  only its quotes and a fingerprint of its text are stored.
- **Given** a scanned source whose OCR differs from the page, **when** an item
  quotes the page, **then** the correction is marked in the stored text and the
  quote is accepted.
- **Given** a quote with typographic quote marks, or a word hyphenated at a
  line end, **when** the pack is recorded, **then** the quote check accepts it
  under the declared normalisation.
- **Given** a specific claim whose cited source does not say it, **when** the
  pack is written, **then** the claim is marked unsupported.

- **Given** a subject and a web address, **when** the librarian runs, **then** the
  source is filed in the sources corpus as confidential with its rights recorded
  per part, and a part with no rights statement is recorded as all rights reserved.
- **Given** a source already filed on the subject, **when** the librarian runs
  again, **then** it is reused from the shelf and not fetched again.
- **Given** a saved page holding the saver's or commenters' names, **when** it is
  filed, **then** those names are stripped from the stored copy.
- **Given** a reading pack whose item quotes a sentence that is not in the text
  stored for its source (after the declared normalisation of whitespace,
  hyphenation and markup), **when** the pack is recorded, **then** the binary
  refuses it and names the item.
- **Given** an item whose quote came from a search summary, **when** the pack is
  recorded, **then** the binary refuses it unless the quote's provenance says it
  was reported via that summary.
- **Given** a finding that names no name, date or number, quoted or not, **when**
  it is recorded among the findings, **then** the binary refuses it.
- **Given** two findings on one fact with different named values, **when** the
  pack is written, **then** both are kept and marked disputed.
- **Given** a rating with no named source, or with neither a scale nor an
  unscored-verdict label, **when** the pack is recorded, **then** the binary
  refuses it.
- **Given** a private source, **when** the librarian runs, **then** no search or
  fetch service receives any of its text, headings or claims.
- **Given** a source not in English, **when** an item quotes it, **then** the item
  carries the original words and a translation.
- **Given** a catalogue page, **when** the librarian reads it outside auto mode,
  **then** the person is shown the items and picks which to read; in auto mode
  the pick and its reason are recorded in the pack.
- **Given** a document of any listed form, or one inside another, **when** it is
  read, **then** the pack carries that form's check and its result.
- **Given** a finding left vague after the drill-down cap, **when** the pack is
  written, **then** it appears under open questions, not among the findings.
- **Given** a recorded pack, **when** its files are listed, **then** each lies
  under `~/.abcd.noindex/library/<root-sha>/packs/` with owner-only permissions,
  and the working tree is unchanged.
- **Given** a recorded pack, **when** it is read, **then** each of the nine lenses
  carries the librarian's written reading of it for the refined subject, and
  lens 9 compares the findings with the repository it ran in.

## Open Questions

- The cap on drill-down rounds, its default, and whether it counts per finding
  or per pack (technical facilitator, at planning).
- The reading budget for a long document or a catalogue (technical facilitator,
  at planning).
- The pack's on-disk shape and its id, so ideate can cite an item (technical
  facilitator, at planning; the bundle's shared spec).
- Whether the open research-verb request (iss-2610100649495706) is answered by
  the librarian run on its own, or stays a separate verb (product thinker).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
