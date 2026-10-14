---
id: adr-2610141646525117
slug: the-librarian-treats-unknown-rights-as
status: accepted
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610141646126492, itd-2610141646452213]
related_rfcs: []
related_adrs: []
---

# ADR-2610141646525117: The librarian treats unknown rights as all rights reserved, files privately first, and admits only named ratings

## Context

The librarian (itd-2610141646126492) fetches, files and reads other people's material and
gathers ratings about it. Three trust questions were open: what may be done with
a source whose rights cannot be established, where its material is kept, and
which ratings may count. The product thinker answered each in the planning
interview of 2026-10-14 (session abcd-66), after testing the plan on example
sources: a scanned 1986 game manual whose archive page carries no rights
statement, a memoir under a non-commercial Creative Commons licence carrying a
reader rating, and a museum page that is all rights reserved.

## Decision

- **Unknown rights are all rights reserved.** A source whose rights the librarian
  cannot establish is recorded as all rights reserved: studied and quoted briefly,
  never reused. The product thinker, verbatim: "if we don't find any info, we
  must assume no rights".
- **Private first.** Everything the librarian fetches is filed in the sources
  corpus as confidential, and its reading packs are kept owner-only under
  `~/.abcd.noindex/library/<root-sha>/packs/`. Nothing it writes is committed.
- **Named ratings only.** A rating or ranking enters a reading pack only with a
  named source and a known scale; reader and anonymous ratings are left out.

Brief invariant: the librarian never records a source as open unless a rights
statement says so, never writes into a working tree, and never admits a rating
without a named source and a scale.

## Alternatives Considered

- **Unknown rights treated as unknown, decided per use:** leaves the decision to
  whoever later quotes it, which is where reuse slips through. Rejected.
- **Files shared with the project by default:** matches the original ingest
  default, but commits confidential material the moment it is read. Rejected.
- **All ratings, labelled by trust:** shows popular opinion, at the price of
  admitting easily gamed numbers and of labels that are the librarian's own
  judgement. Rejected by the product thinker in favour of named ratings only.
- **Ratings reported but never used as evidence:** keeps scores out of verdicts
  entirely, so strong reception would count for nothing. Rejected.

## Consequences

- A reading pack can say less than the web does: popular opinion is missing, and
  an openly licensed source without a findable statement is treated as closed.
- Any later sharing of a pack, or adoption of an idea built on one, is a
  person's deliberate act, under the same rules ideate's private verdicts follow.
