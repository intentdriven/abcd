---
id: itd-2611011151475128
slug: a-product-thinker-never-keeps-two-copies
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-102]
severity: minor
refines: [itd-102]
related_adrs: [adr-2611011153058085, adr-2610031751065746, adr-19, adr-53, adr-2610142147242544]
origin: researcher-authored
production_mode: hand-written
---

# Every documentation surface is rendered from its one home

## Press Release

> A product thinker never keeps two copies of what their product is. abcd renders the README and the docs pages from the one home of each claim: positioning, users and roles from the brief, how-to guides and tutorials from the how, principles from the principles, and install, requirements, first run, citation, badges, licence and links from the repository's own facts. Each rendered section names the home it comes from, so a reader who wants to change it knows where. The derived sections render now; the brief-sourced sections become compulsory in every abcd-managed repository once the brief has an editing surface, because from then the owner's own words live in the brief and a render has nothing of theirs to overwrite.

## Why This Matters

_Written by the agent on 2026-11-01 from the product thinker's rulings that
day (dcn-2611011248594647, dcn-2611011551216958), restated; the planning interview confirms or
rewrites it._

A hand-written README is a second home for what the brief already says, and a
second home drifts. abcd's own README shows it: the body beneath its lockup
still describes the product as "a host-agnostic configuration layer for
intent-driven development", the wording the conventions file also carries,
while the brief's identity block now names it differently. Two committed homes
say two things. The principle that every claim has one home
(`every-claim-has-one-home`) already covers `docs/`; nothing yet enforces it
on the README or a docs page, and the principle gains its documentation bound
when this intent ships, not before (the record lands with the act).

The product thinker's rulings on 2026-11-01, restated:

- **No documentation surface holds prose of its own.** What people write
  lives in the design record (the brief, with its principles) or the how
  (itd-2611011239589823: how-to guides, and optionally tutorials, exercises
  and video), and every documentation surface is rendered from its one home,
  section by section as abcd comes to render it. What abcd's README holds by hand
  today is positioning, its users and their roles, all of which belong in the
  brief.
- **The derived sections render now.** Install, requirements and first run
  derive from the release and the repository's facts, the citation from
  `CITATION.cff`, and the badges, licence line, links and contents are the
  parts setup already derives (itd-2610031348087517). None of these waits on
  anything.
- **The brief-sourced sections are compulsory once the brief has an editing
  surface.** From then the owner's own words are written in the brief, so a
  render replaces nothing of theirs. Until then a repository's hand-written
  prose stays where it is.
- **Every documentation surface, not the README alone.** The scope is the
  README and every page under `docs/`: the explanation pages (who the product
  is for, roles, artefacts, process) render from the brief, the how-to and
  tutorial pages from the how, and the reference pages from their sources,
  as the CLI reference already does. Left hand-written, an explanation page
  drifts the way the README did.
- **Not everything goes into the brief.** The brief is the why and the what;
  how-to guides and tutorials belong in the how, and reference facts derive
  from the release and the code, so the brief does not become a user manual.
- **The README's opening is written for a stranger.** Brief prose is written
  for the record's readers; the README is the front door for someone who has
  never heard of the product, and the product thinker writes the text it
  opens with. adr-53 expresses audience by placement, never by a prose
  register, so the agent proposes placing that text as a page of the brief's
  product chapter of its own, rather than writing it in a second register
  inside another chapter.

Proposed by the agent, not ruled: **facts inside an authored page are filled
in, never typed.** A version, a command name or a flag in an explanation or a
guide comes from the release rather than being typed, and is never written as
a version into a committed page (adr-19; itd-2610031348087517's decision 4
applied the same to the README). Where it is filled in is open: the product
thinker ruled on 2026-10-15 that a managed repository's docs are read on the
forge and its site renders no docs (itd-2610091010348892), so there is no site
build to fill anything in for most repositories.

The rule meets two standing constraints, and holds both:

- **The owner's space is theirs** (`the-users-directory-is-theirs`,
  adr-2609091248200336). A render is run by the person, or by a verb the
  person runs, never on a schedule or a session start; it rewrites only the
  sections abcd renders, section by section, holding any section a hand edit
  is still waiting in (adr-2611011153058085); and committing the result stays
  the person's act, as itd-2610031348087517's scope condition already says
  for the setup parts (the space rule now held by adr-2610031751065746).
- **No autonomous rewrite.** itd-102 settled that re-rendering from the
  identity block is always a proposed diff a person adopts. Whether a
  brief-sourced section is rendered in place or held to its home by a
  proposed diff is the open question below; if it settles on rendering in
  place, this intent reverses that settlement of itd-102 and says so.

Links to the record, typed:

- **refines** itd-102: the identity block's one home and drift check, widened
  from the lockup lines to whole documentation surfaces; a candidate reversal
  of its no-autonomous-rewrite settlement, decided by the open question.
- **reverses**, in part, itd-2610031348087517: that intent never touches the
  owner's own words and leaves a hand-edited part as it is. Here the owner's
  words move into the brief or the how, and each section renders from there.
  Flagged for the product thinker to confirm as a reversal; not
  auto-classified.
- The reverse path, catching hand-written text and offering it back to its
  source, is its own intent (itd-2611011151512956), and the rule that a
  render never discards hand-written text unasked is
  adr-2611011153058085 (proposed).

## SOTA

`.abcd/development/research/notes/2026-11-01-documentation-sources-sota.md`
(review 1): explanation authored in the design record and a chosen part
rendered; reference generated from the command tree; how-to and tutorial
pages authored with their steps run as tests; the README composed from stable
authored text and generated regions; every generated region marked; hand
edits caught by regenerating and comparing.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- **Rendered or held.** Whether a brief-sourced section is generated in place
  between markers, as the setup parts are, or held to its home by a check
  that proposes the diff, as the identity block is; the first reverses
  itd-102's no-autonomous-rewrite settlement.
- **The section-to-home map.** Which README and docs section renders from
  which home, recorded where (the docs and identity surface pages of the
  brief), and how a repository with a thinner brief maps the same sections.
- **Where the how lives.** Authored in place under `docs/` with facts filled
  in at build, or kept under the development record and rendered into
  `docs/`; the same question stands in itd-2611011239589823.
- **The front-door text's place.** Whether the text the README opens with is
  its own page of the brief's product chapter (adr-53), or sits beside the
  identity block.
- **Featured principles.** The README shows a few principles, not all; how a
  principle is marked as one the front page shows.
- **The editing surface it waits on.** Which brief-editing work counts as
  "the brief has an editing surface" for the compulsory step: live brief
  sessions, itd-61's reconciliation of a human edit, or itd-90's interview.
- **Whether "render now" keeps the setup question.** itd-2610031348087517
  asks the owner at setup whether abcd keeps parts of the README current;
  whether the derived sections stay behind that question or render without
  it, which would reverse that question too.
- **Where facts are filled in.** For docs a managed repository reads on the
  forge with no site build (itd-2610091010348892): a live badge, a generated
  region committed by a verb the person runs, or no version at all.
- **The terms.** "The how", "documentation surface" and "authored surface"
  have no glossary entry, and the glossary's "rendered surface" means a
  public text held to the identity block; the terms get their entries, or
  the existing one widens, before planning.

## Typed Links

- **built on by and refined by `itd-2611011239589823`** (the how): Its how-to
  and tutorial pages come from that surface.
- **built on by `itd-2611011151512956`** (the way back): Catches a hand edit
  to a page this intent renders and offers it back to its source.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
