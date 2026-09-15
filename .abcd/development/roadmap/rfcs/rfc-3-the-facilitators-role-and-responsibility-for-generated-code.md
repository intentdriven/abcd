---
id: rfc-3
slug: the-facilitators-role-and-responsibility-for-generated-code
status: open
discussion_opened: 2026-08-29
discussion_closes: TBD
spawned_from: adr-2609151528057260
spawned_intents: []
related_intents: []
related_adrs: [adr-2609151528057260]
authors: [project]
---

# RFC-3: The facilitator's role, and who answers for code nobody read

## The question

[ADR-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses-and-when-the-loop-st.md)
settles who acts. It deliberately leaves open who answers for the result, and
that question gets harder the better the framework works.

The product thinker is responsible for the software. It is their design and
their promise that it does what they said. They will not read the code, and
under this framework's premise they **should not**: requiring it would reimpose
the constraint the framework exists to remove, and would demand a competence
whose absence defines the reader. The facilitator that reviews the code on their
behalf is, by default, a machine. It may become a person on activation, and it
may switch mid-run.

So the chain that normally discharges responsibility is missing its usual link.
Nobody who is accountable inspected the artefact, and the reviewer that did
inspect it may not be a legal person at all.

## Why this is not settled by ADR-55

ADR-55 could route the work because the work has a natural owner: whoever is
competent to do it. Responsibility does not decompose that way. It does not
follow competence, it does not transfer to a tool, and it cannot be discharged
by a role that cannot hold it.

Three things make the question sharper here than in ordinary delegation.

**The reviewer may be non-human.** Delegating review to a colleague moves
responsibility along a chain of people. Delegating it to a model does not
obviously move it anywhere.

**The mode can change mid-run.** If a human facilitator activates halfway
through a build, it is unclear whether they inherit responsibility for what the
loop did before they arrived, and equally unclear that they should.

**The accountable party cannot audit the delegation.** A product thinker cannot
tell a thorough review from a plausible one, which is the same limitation that
put them in this position. Any answer that rests on them supervising the
reviewer collapses for the same reason.

## Positions to test

**Acceptance is the contract.** The product thinker answers for what they
promised and for accepting delivery against it, not for how it was built. The
acceptance criteria become the object of responsibility, which puts the whole
weight on whether those criteria are good enough to carry it.

**Process evidence substitutes for inspection.** Responsibility is discharged by
being able to show what was done: an independent verifier, an audit trail, a
recorded verdict. This is how several regulated fields work, and the open
question is how much of it survives being scaled down to one or two people with
no institution behind them.

**Responsibility stays with the facilitator when a human occupies it, and
otherwise stays with the product thinker by default.** Simple to state; it makes
activation a transfer of liability, which is a strong thing to hang on a mode
switch.

**The arrangement is not currently defensible and needs a floor.** Worth stating
as a live option rather than a rhetorical one. If it is the honest answer, the
useful output of this RFC is the minimum that would make it defensible.

## What the evidence says

A state-of-the-art sweep ran on 2026-08-29. Its headline is that the arrangement
splits cleanly into a defensible half and an indefensible one.

**Defensible: the product thinker not reading the code.** Every regime examined
is built on the premise that the accountable principal does not read it. That
part is ordinary, not novel.

**Not defensible as stated: an AI review discharging the review obligation.**
What those regimes substitute for principal inspection is not process in place
of review. It is review by a competent party who is independent of the author
and is themselves accountable, plus evidence that this happened, plus a
signature. This framework removes the accountable reviewer and puts an
unaccountable one in its place, and that is the actual gap.

Four findings carry most of the weight.

**An automated verifier can earn review credit, but only as a qualified tool.**
[DO-330](https://www.rapitasystems.com/do-330) exists for exactly this question,
and [ISO 26262-8](https://www.itemis.com/en/glossary/tool-qualification/) offers
the more useful dial: tool confidence derives from impact and from *error
detection*, the confidence that a tool's mistake is caught by other means.
Qualification assumes a tool returns the same output for the same input, which
model non-determinism defeats at the root. The design move available here is
therefore to raise error detection rather than to claim qualification: a miss by
the AI reviewer must be caught downstream by something that is not a model.

**Measured performance does not support the claim.** On 1,000 verified pull
requests, the best configuration reached 16.65% precision and 19.38% F1, and the
authors conclude current tools do not perform sufficiently well
([arXiv:2509.01494](https://arxiv.org/html/2509.01494v1)). Aggregating multiple
reviews raised F1 to 43.67%, which is the argument for more than one reviewer
and against a single one. Independence has one evidenced condition: a different
vendor or model family, because self-consistent errors do not shrink with scale
and often differ across models
([arXiv:2505.17656](https://arxiv.org/abs/2505.17656)). A different prompt alone
buys almost nothing, and same-model self-correction is contraindicated
([arXiv:2310.01798](https://arxiv.org/pdf/2310.01798)).

**The largest project to deliberate this ruled against the premise.** The Linux
kernel's coding-assistant policy requires that a human reviews all AI-generated
code, forbids an agent from signing off because only a human can certify the
Developer Certificate of Origin, and makes no provision for an agent reviewing a
patch ([docs.kernel.org](https://docs.kernel.org/process/coding-assistants.html)).
Its `Assisted-by:` trailer is the convention this repository already follows, so
citing it for the trailer while omitting the review clause would be selective.

**The law does not require inspection; it requires outcomes.** The revised
Product Liability Directive makes software a product, presumes defectiveness
where technical complexity makes proof excessively difficult, and removes the
development-risk defence for defects traceable to updates
([Directive (EU) 2024/2853](https://eur-lex.europa.eu/eli/dir/2024/2853/oj/eng)).
California removed the defence that the AI acted autonomously
([AB 316](https://leginfo.legislature.ca.gov/faces/billTextClient.xhtml?bill_id=202520260AB316)).
Separately, the AI Act's human-oversight article speaks throughout of a *natural
person*, which is structurally what an AI-by-default facilitator is not
([Article 14](https://artificialintelligenceact.eu/article/14/)).

## The charge this RFC has to answer

The responsibility gap is managed rather than resolved. It is structural, and no
arrangement here dissolves it.

The moral crumple zone is a different thing and it is avoided rather than
managed. Elish's diagnosis names a person held answerable for what they could
not control, the component that absorbs the penalty when the system fails
([Engaging Science, Technology, and Society, 2019](https://estsjournal.org/index.php/ests/article/view/260)).
Scoping the product thinker's answerability to the promise they wrote and the
delivery they accepted attributes responsibility to something they do control,
which is the exit rather than a mitigation. What remains is not a crumple zone
but its precondition: an acceptance is only as meaningful as the signal it rests
on. Closing that residual is what verification escalation exists for, and it is
why the strength of the acceptance signal is a setting rather than a constant.

The zone forms under three conditions, and stating them turns the diagnosis into
a test this design can be held to: no transparency into what the automation did,
no authority to change anything, and a drive for efficiency
([Bangma, 2026](https://www.linkedin.com/posts/sylvain-bangma_ai-puts-people-in-a-moral-crumple-zone-share-7498728574032494592-CnjG)).

Measured against them, the framework answers the second well and the first
partly. The product thinker sets the acceptance criteria, owns the register they
are addressed in, chooses the verification rung, and configures the facilitator,
which is authority over what determines the outcome rather than a veto at the
end. Transparency into what was decided is strong; transparency into how good
the check that judged it was is weak, and a verdict written in file-and-line
citations by a reviewer of low measured precision is opaque in exactly the way
that matters.

The third is unanswered. Climbing the ladder costs money and time, nothing in
the design resists the pull towards its cheapest rung, and for a solo builder
the organisation applying that pressure is the builder, which makes it harder to
notice rather than weaker (`iss-2608290944122400`). A ladder nobody climbs is a
rubber stamp with more steps. **Whether the chosen rung must be visible in what
abcd produces, so that shipping on the cheapest one is a stated position rather
than a silent default, is the open question this RFC most needs answered.**

The most defensible available theory is the **tracing condition** from the
meaningful-human-control literature: system behaviour must trace to a proper
understanding held by some human who grasps the system's capabilities and
failure modes
([Frontiers in Robotics and AI 5:15](https://www.frontiersin.org/articles/10.3389/frobt.2018.00015/full)).
It does not require reading code. It does require the product thinker to hold
accurate expertise about how this generation-and-review machinery fails, which
means the claim "they lack the expertise to verify" overshoots and should be
narrowed to "they lack code-reading expertise, and are not required to acquire
it".

## The minimum that would make this defensible

1. Relocate the object of responsibility from the code to the **behavioural
   contract**: the product thinker authors and owns the acceptance criteria and
   a list of ways the thing could be wrong, and answers for those.
2. Demote the AI review from *the* review to **a filter with measured, low
   precision**, and carry the assurance on a detection layer that is not a
   model: tests, metamorphic relations, staged rollout, fast rollback.
3. State the independence condition concretely, a different vendor and a fresh
   context, and record the residual rather than implying it is eliminated.
4. Bound and state the blast radius. The arrangement is defensible where a
   failure hurts the person shipping it and progressively less so as it reaches
   third parties.

The cheapest artefact that serves 1 and satisfies the tracing condition is a
**defeater log**: a written list of how this could be wrong, what would show it,
what was done, and what residual doubt is knowingly accepted
([Assurance 2.0, arXiv:2205.04522](https://arxiv.org/abs/2205.04522)). It is the
only item in the sweep that a non-coder can author, verify, and be genuinely
answerable for.

## Open questions

1. Is the object of the product thinker's responsibility the software, or the
   promise they wrote and the delivery they accepted?
2. Does an automated verifier discharge a review obligation, and under what
   independence conditions (a different model, a different vendor, an
   adversarial framing)?
3. What does activating a human facilitator mid-run do to responsibility for
   what preceded it?
4. What is the smallest set of evidence a one-person team can produce that a
   sceptical outsider would accept?
5. What does the repository's own attribution convention mean once the
   facilitator is a machine? `AGENTS.md` states that the human is the author of
   record and responsible for all AI-assisted output. Under ADR-55 the only
   human left may be the product thinker, who becomes author of record for code
   they cannot judge.
