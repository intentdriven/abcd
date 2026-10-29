---
id: prn-every-claim-has-one-home
claim_type: criterion
reference: "the brief"
comparison: "Passages of the brief and the intents, each read against the rest for contradiction, in the consistency pass of 2026-10-26."
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
(`.abcd/work/reviews/2026-10-26-consistency/`) read the brief and every
current intent against each other and filed 1,301 records from 1,307
findings. 1,080 of those findings set a passage of the brief against
another passage of the brief or against an intent, and 800 of the whole
were premise contradictions: two places that state the same premise and
disagree on it. A pointer cannot disagree with its home the same way. A
markdown link whose target is gone fails the record and docs lints' link
check; a bare path in a rules domain is checked by nothing yet, so a
renamed home leaves it pointing at nothing, silently.

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
  said and why it changed. A dated record (an ADR, a decision, a shipped
  intent) says what was settled on its date, and a later record that
  changes it supersedes it rather than standing beside it as a second
  current answer.
- Terms have one home too: the glossary. A chapter uses a term and links it,
  never defines it again.
- The rule covers the whole written record: the brief, the intents, the
  ADRs and decisions, the command pages, the principles, the rules domains,
  the conventions file and `docs/`. Code is out of scope: a primitive in
  code has one home under
  [one canonical primitive](one-canonical-primitive.md).

**Promotion.** The rule is unenforced. `abcd intent consistency` finds two
places that disagree only after they have drifted, and nothing flags a
restatement as it is written, nor a bare path in a rules domain whose
target is gone. A lint that did either would promote this principle to a
discipline.

**Where it was ruled.** The product thinker made it a standing rule on
2026-10-27 (dcn-2610271112331508), in settling how the brief's restated
sections are cut back.
