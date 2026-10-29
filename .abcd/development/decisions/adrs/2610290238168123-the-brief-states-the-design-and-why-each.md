---
id: adr-2610290238168123
slug: the-brief-states-the-design-and-why-each
status: accepted
date: 2026-10-29
supersedes: adr-2609231028044006
superseded_by: null
related_intents: [itd-147, itd-60, itd-2610290239150589]
related_rfcs: []
related_adrs: [adr-2609231028044006, adr-5]
---

# ADR-2610290238168123: The brief states the design and why; each command's page states what abcd does, once, and is the copy checked against the code

## Context

Each abcd command's behaviour (what it refuses, what it prints, the steps it takes,
its exit codes) was written out twice by hand: in the brief's chapter for the
command under `04-surfaces/`, and in the command page under `commands/` that an
agent loads and follows when it runs the command. adr-2609231028044006 settled
the shape half of a chapter (flags and sub-verbs come from a generated appendix,
and the prose states none) and left behaviour "hand-maintained", as "a
review-grain claim", in the chapter. The documentation-fidelity gate of itd-60
checks the brief against the code; nothing checks the command pages. itd-147 and
itd-60 made the brief the shipped state.

The brief-size study of 2026-10-25
(`.abcd/development/research/notes/2026-10-25-brief-size-study.md`) measured what
that costs. The brief held about 247,000 words; readers put about
69% of it as restating another file, and the command chapters alone at 76%
(about 121,000 words). Only about 15% repeated another file word for word, so
the duplication was paraphrase, not copy-paste, and the two copies had drifted
apart in both directions: some command pages were wrong where the chapter was
right, and some chapters wrong where the page was right. The consistency pass
of 2026-10-26 filed 1,301 records over the brief and the intents, most of them
from that second copy.

## Decision

The product thinker ruled on 2026-10-27 (dcn-2610271112331508,
dcn-2610271112339516, dcn-2610271112330235):

1. **Always a single source of truth.** Every fact, rule, behaviour and list has
   one home; every other place points to it.
2. **A command's behaviour lives in its command page.** The page under
   `commands/` is the one home of what the command does: its refusals, its
   output, its steps and its exit codes. The documentation-fidelity check moves
   to the command pages: a false sentence there holds a spec close and a release
   cut, as a false sentence in the brief does today.
3. **The brief states the design and why.** Each chapter under `04-surfaces/`
   keeps why the command exists, the trade-offs taken and why it refuses what it
   refuses, its `## Sub-verbs` table and its generated appendix, and points to
   the command page for behaviour. The brief as a whole states the target design
   in the present tense; a part that is built carries a status tag saying so,
   rather than an unbuilt part being marked as staged.

The shape rule of adr-2609231028044006 is carried forward unchanged: every flag
and sub-verb a chapter states comes from its generated appendix, and the prose
above it states none. `TestSurfaceAppendicesMatchCommandTree` and
`TestSurfaceChapterProseStatesNoShape` keep holding it.

## Alternatives Considered

- **The chapter as the home, with the command page keeping only the agent's
  steps.** Keeps the design record complete in one place, but the copy agents
  follow loses what they need to act, and the copy that does harm when wrong
  stays unchecked. Rejected by the ruling.
- **Both copies, and a check that reads each pair against the other.** Moves
  nothing, but every change is still written twice and the measured upkeep
  stays. Rejected: "always SSOT".
- **The brief as the shipped state, unbuilt parts marked (itd-147, itd-60).**
  Kept checkable against the code, but once behaviour moves to the command pages
  the brief has nothing left to check against the code, and the person wants it
  to read as the product they are building. Superseded.

## Consequences

- The behaviour passages of every `04-surfaces/` chapter move to their command
  page, or are cut where the page already says it; this is the brief's
  single-home rewrite the 45 deferred consistency majors wait on.
- The documentation-fidelity gate, its review instructions and its receipts read
  the command pages, and no longer the brief, for claims about behaviour.
- itd-147's and itd-60's statement that the brief is the shipped state, the
  phase document that says so (`roadmap/phases/phase-8-brief-currency.md`), and
  invariant 18's wording are amended by the rewrite, pointing here.
- adr-5 ("the brief is the project's current state") still holds: the target
  design is the decided state. The truth rule in `00-meta.md` changes its one
  variance from "shipped vs staged" to "built or not yet", tagged on the built.
- A command page now carries the claims the gate checks, so its prose is held to
  the same review grain as a chapter's was.
