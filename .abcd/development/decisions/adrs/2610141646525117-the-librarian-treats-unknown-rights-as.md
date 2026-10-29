---
id: adr-2610141646525117
slug: the-librarian-treats-unknown-rights-as
status: accepted
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610141646126492, itd-2610141646452213, itd-2610142116477426, itd-164]
related_rfcs: []
related_adrs: [adr-41, adr-25, adr-2610091918443997, adr-2610141646523373, adr-2610142116480836]
---

# ADR-2610141646525117: The librarian treats unknown rights as all rights reserved, files privately first, and admits only named ratings

## Context

The librarian (itd-2610141646126492) fetches, keeps and reads other people's
material, and gathers ratings about it. Three trust questions were open: what may
be done with a source whose rights cannot be established, where its material is
kept, and which ratings may count. The product thinker answered each in the
planning interview of 2026-10-14, after two hand-run test labs on 14 example
sources (a scanned 1986 game manual whose archive page carries no rights
statement, a memoir under a non-commercial Creative Commons licence carrying a
reader rating, private research reports, a museum page that is all rights
reserved). The labs live in the machine-local lab store; their load-bearing
findings are carried by the issues the bundle's intents cite.

Two terms are kept apart. A **private source** is material the person hands over
as private (a local file, a saved page). A source **kept privately** is anything
the librarian holds, in its own owner-only store; a web page it fetched from a
public address is kept privately but is not a private source.

## Decision

- **Unknown rights are all rights reserved.** A part of a source whose rights the
  librarian cannot establish is recorded as all rights reserved: studied and
  quoted briefly, never reused. The product thinker, verbatim: "if we don't find
  any info, we must assume no rights". Rights are recorded per part (a page's
  text, its images, its comments, the artefacts it links to) and for every page
  found while searching that a pack quotes; a part readable by people but barred
  for use by AI is recorded as such. This holds for the librarian's packs only;
  the licence verdicts of itd-164 keep their own explicit unknown.
- **Kept privately, outside the corpus.** The librarian keeps what it fetches in
  its own owner-only store beside the packs, never in a working tree. Adding a
  source to the person's sources corpus stays the person's act through
  `library add`, which asks for its class and never guesses it (adr-41).
  E-mail addresses, handles and the account's own name are removed before
  anything is kept, with any other name spans the agent declares.
- **A private source is read alone.** No search or fetch receives a private
  source's text, headings or claims, even when it is a copy of a public page; its
  claims stay unchecked unless the person supplies a public source for them.
- **Terms that forbid copying are honoured.** When a source's own terms forbid
  copying or downloading, only its quotes and a fingerprint of its original bytes
  are kept. Translations of quoted text are kept in the private pack even where
  the publisher reserves translation, because the pack is one person's private
  study copy and is never shared.
- **Named ratings only.** A rating enters a reading pack only with a named
  source: a score with a known scale, or a named critic's verdict without a
  scale, labelled unscored. Reader and anonymous ratings, and ratings known only
  second-hand, are left out.

What the binary enforces: the store's owner-only mode, no writes into a working
tree, the removal of e-mail addresses, handles and the account's own name, the
fingerprint, and the required fields of a rights record and of a rating. What it
cannot see, and the library command page instructs instead: that no search or
fetch receives a private source's content, and which other names to strip.
Rights and rating provenance are recorded as the agent declares them, not
verified.

## Alternatives Considered

- **Unknown rights treated as unknown, decided per use:** leaves the decision to
  whoever later quotes it, which is where reuse slips through. Rejected.
- **Files shared with the project by default:** matches the original ingest
  default, but commits confidential material the moment it is read. Rejected.
- **Filing fetched sources in the corpus as confidential:** reuses the corpus,
  but guesses a class the corpus leaves to the person, and fills the banned-names
  list with ordinary titles. Rejected.
- **Ratings known only second-hand, labelled as such:** fills gaps for older
  items, but a second-hand score cannot be checked against its named source.
  Rejected by the product thinker.
- **All ratings, labelled by trust:** shows popular opinion, at the price of
  admitting easily gamed numbers and labels that are the librarian's own
  judgement. Rejected by the product thinker.
- **No translation where a publisher reserves it, or an English summary:**
  honours the reservation in a private copy no one else reads, at the cost of
  quotes the person cannot read. Rejected by the product thinker.

## Consequences

- A reading pack can say less than the web does: popular opinion is missing, and
  an openly licensed source without a findable statement is treated as closed.
- The privacy of a private source rests on the agent following the command page;
  the binary cannot observe the host's searches.
- Any later sharing of a pack happens only through the research command's note,
  which re-applies these rules (itd-2610142116477426).
- The rule is to be added to the brief's invariants when the librarian ships,
  carrying only what the binary enforces.
