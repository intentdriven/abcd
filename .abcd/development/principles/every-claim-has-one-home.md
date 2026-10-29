---
id: prn-every-claim-has-one-home
claim_type: criterion
reference: "the brief"
comparison: "The brief's passages that restated a command page, an ADR or another chapter, compared against the home they restated, in the consistency pass of 2026-10-26."
evidence: [dcn-2610271112331508]
---
# Every claim has one home

**The rule.** Each fact, rule, behaviour and list the record holds is written
in exactly one place. Every other place that needs it names that place and
stops, rather than saying it again in its own words.

**Why.** A restated claim is a second claim. Nothing ties the two copies
together, so the first edit to either one leaves them disagreeing, and a
reader has no way to tell which one the code, the gates or an agent
follows. The consistency pass of 2026-10-26
(`.abcd/work/reviews/2026-10-26-consistency/`) filed 1,301 records, and
most of them were exactly this: a passage in the brief that paraphrased a
command page, an ADR or another chapter, and had drifted from it. A pointer
cannot drift that way. It is either right about where the claim lives or
plainly broken, and a link that no longer resolves fails the record and
docs lints' link check rather than reading as a quiet contradiction.

**Bounds.**

- A pointer may say in one line what it points to, enough for a reader to
  decide whether to follow it: an invariant written as one line naming its
  rule and the ADR that holds it, or a short form in `AGENTS.md` that names
  the rules domain carrying the whole rule. It holds nothing the home does
  not, and it is not where the claim is changed.
- A generated copy is not a second home. A file built from its source by
  one writer, with a check that the two still match (the CLI reference, a
  command chapter's generated appendix, the `SHELL` domain built from the
  hazard registry), has its home in that source.
- History is not a copy. A superseded design is deleted from the brief, not
  kept beside the design that replaced it; git and the ADRs keep what it
  said and why it changed.
- Terms have one home too: the glossary. A chapter uses a term and links it,
  never defines it again.
- Code is out of scope. A primitive in code has one home under
  [one canonical primitive](one-canonical-primitive.md); this rule covers
  the written record: the brief, the command pages, the principles, the
  rules domains and the conventions file.

**Where it was ruled.** The product thinker made it a standing rule on
2026-10-27 (dcn-2610271112331508), settling the brief-size study's second
question.
