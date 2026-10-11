---
id: itd-2610100627454580
slug: an-agent-in-an-abcd-managed-repository
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609201916056194]
---

# An agent's text comes back from abcd's rewriter written for its reader and within its limits

## Press Release

> An agent in an abcd-managed repository hands text meant for a reader to abcd's rewriter and gets back text that fits. A question for the product thinker comes back in plain words within the question box, a docs entry within the writing style, a record title within the filename budget; abcd's own counter judges every limit, and the rewriter keeps going until the text passes or says it cannot. The person's own words are never rewritten, and the model it runs on is the routing table's choice, so a team can pick a cheaper or a stronger one.

## Why This Matters

_Proposed by the facilitator's review of 2026-10-10; not yet confirmed by the product thinker._

An agent writes for three readers at once and for limits it cannot count: a question must fit the question box (twenty-four rows at eighty columns), a product thinker's summary must use plain words with no internals, a docs entry must follow the writing style, and a record title must keep its file within the Windows path limit (iss-2610100626320367). Today one case is covered: the `question-drafter` agent drafts abcd's questions to the asking rules, and only when the asking agent remembers to use it. Every other text is written once by the agent doing the work, judged afterwards by a gate or by the person, and rewritten by hand when it fails.


## Mechanism

_Proposed by the facilitator's review of 2026-10-10; not yet confirmed by the product thinker._

We expect text to reach its reader fitting on the first showing, because the rewriter is given the reader's profile and abcd's own counter, and stops only when the counter passes, rather than an agent estimating a count it cannot see. Shown wrong if a lab finds the rewritten text losing meaning the original carried (a dropped qualifier, an invented claim) as often as it fixes a limit, or if the call's cost and delay make agents skip it as they skip the drafter today.


## Scope Conditions

_Proposed by the facilitator's review of 2026-10-10; not yet confirmed by the product thinker._

- Holds for a host that can start a sub-agent from an agent definition with its own model; abcd itself cannot start one, so the rewriter is reached through the asking agent, prompted by the rules and by a gate's refusal.
- Holds while abcd's counters (the question check's rows, docs-lint, the slug cap) are the judge of every limit; the rewriter never judges its own fit.


## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Which model is the default? The lab iss-2610090816296964 already plans compares the quick tier (Haiku 4.5) with the default tier on the drafter's material for fit, lost meaning and cost; its result sets the default, and the routing table (itd-2609201916056194) lets a team choose another.

## Prior Art

- The `question-drafter` agent (agents/question-drafter.md, first shipped in v0.13.3): drafts abcd's questions to the asking rules and counts rows as the question check does. This intent grows it into the general rewriter rather than adding a second agent (principle: one canonical primitive).
- iss-2610090816296964: the no-repetition rule for question text and the lab comparing model tiers for the drafter.
- iss-2610080618506115: the model-tier idea the lab feeds.
- itd-2609201916056194 (shipped): a delegated agent run through a chosen model, the routing this rewriter's model choice uses.
- iss-2610100626320367: the record filename cap, which stays the guarantee; the rewriter can only make a shortened title read well.

## Decisions

- 2026-10-10: the product thinker proposed a dedicated rewrite agent (Haiku by default, the model configurable) that rewrites any text for the product thinker, the facilitator or the user-facing docs until it meets given limits, run automatically by agents for every writing task, and asked for an adversarial review: "Critically review my proposal, I'm not invested in it", adding that it "becomes a general rewrite agent for abcd-managed repos". The review found most limits are counts that abcd's code measures and the rewriter can only be asked to meet; that abcd cannot start an agent itself, so "automatically" means a rule and a gate, as the drafter has today; that running it for every text costs a sub-agent call each time; that meaning drift is the main risk; and that the drafter and the routing table already exist. Offered (A) growing the drafter into the general rewriter, (B) a new agent for every writing task, or (C) code fixes only, the product thinker chose "C first, then A": the code fixes are iss-2610100626211810 (the mode set from the question's chip), iss-2610100626327722 (the rows line in a refusal) and iss-2610100626320367 (the filename cap), and this draft is A.
- 2026-10-10: the product thinker settled three of the draft's open questions. Which texts first: "All person-facing", so questions, record titles, product-thinker summaries and docs entries go through the rewriter, and chat replies and commit messages do not. Who checks that a rewrite kept the meaning: "A separate checker", a second agent independent of the rewriter, at the cost of a second call per text. How its use is made to stick: "Stamped output", so the rewriter stamps what it returns and a gate checks the stamp on the texts in scope, accepting that a stamp can be forged by an agent that means to skip it. The default model stays with the lab.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
