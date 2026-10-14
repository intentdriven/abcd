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

- **Unknown rights are all rights reserved.** A part of a source whose rights the
  librarian cannot establish is recorded as all rights reserved: studied and
  quoted briefly, never reused. The product thinker, verbatim: "if we don't find
  any info, we must assume no rights". Rights are recorded per part (a page's
  text, its images, its comments, the artefacts it links to) and for every page
  found while searching that a pack quotes; a part readable by people but barred
  for use by AI is recorded as such.
- **Private first.** Everything the librarian fetches is filed in the sources
  corpus as confidential, and its reading packs are kept owner-only under
  `~/.abcd.noindex/library/<root-sha>/packs/`. Nothing it writes is committed.
  People's names in a saved page that are not the source (the saver's own,
  commenters') are stripped before filing.
- **A private source is read alone.** No search or fetch service receives any of
  a private source's text, headings or claims; its claims stay unchecked unless
  the person supplies a public source for them.
- **Named ratings only.** A rating enters a reading pack only with a named
  source: a score with a known scale, or a named critic's verdict without a
  scale, labelled unscored. Reader and anonymous ratings, and ratings known only
  second-hand, are left out.

Brief invariant: the librarian never records a part of a source as open unless
a rights statement says so, never writes into a working tree, never sends a
private source's content to a service, and never admits a rating without a
named source.

The test lab of 2026-10-14 (lab-261014165333-8d28f83) led to the per-part
rights, the reading-alone rule, the name stripping and the unscored-verdict
label, each ruled by the product thinker the same evening: one test run sent a
private page's headings to a web tool, two saved pages held other people's
names, and the strict ratings rule left two 1980s games with no reception at
all.

## Alternatives Considered

- **Unknown rights treated as unknown, decided per use:** leaves the decision to
  whoever later quotes it, which is where reuse slips through. Rejected.
- **Files shared with the project by default:** matches the original ingest
  default, but commits confidential material the moment it is read. Rejected.
- **Ratings known only second-hand, labelled as such:** fills gaps for older items with weaker evidence. Rejected (2026-10-14).
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
