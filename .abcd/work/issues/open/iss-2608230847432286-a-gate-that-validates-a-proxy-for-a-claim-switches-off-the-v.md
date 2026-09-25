---
schema_version: 1
id: "iss-2608230847432286"
slug: "a-gate-that-validates-a-proxy-for-a-claim-switches-off-the-v"
severity: "major"
category: "process"
source: "user-observation"
found_during: "record-review"
found_at: ".abcd/development/principles/enforcement-claims-are-facts.md"
details: "enforcement-claims-are-facts covers the phantom gate: a check described but not running, whose harm is that readers stop compensating. Three instances from 2026-08-22/23 show the family the principle does not yet name, in which the reassuring signal is real: a gate measuring a proxy for the claim, and a gate measuring the right property over a subject set narrowed by a named exclusion that was defended by a test incapable of failing. A fourth case is recorded as adjacent rather than folded in, because it involves no gate and no enforcement claim. In none of them did anything error, and no instrument surfaced any. Proposed as a paragraph extending that principle, not as a new principle, per one-canonical-primitive."
suggested_fix: "Extend .abcd/development/principles/enforcement-claims-are-facts.md with a paragraph naming the real-signal family and its three worked examples. Do not add a new principle beside it: one-canonical-primitive forbids the third copy, and the Why paragraph of the existing principle already states the mechanism this shares. Decide separately whether the adjacent case below is admitted, because it widens the class from gates that do not gate to assurances nobody issued but everyone read in, and a class without that boundary is harder to apply rather than easier. A maintainer decides adoption; agents agreeing is not the gate."
related_issues: ["iss-2608221457227162", "iss-2608230752354926", "iss-2608221328552172", "iss-2608230817034768", "iss-2608230847432285"]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Which gates do the proxy-gate detectors audit first, and do they warn or refuse?"
---

a gate that validates a proxy for a claim switches off the vigilance an absent gate would have preserved

`enforcement-claims-are-facts` states the rule for a gate that does not exist:
a phantom gate is worse than no gate, because readers who believe a check exists
stop compensating with the vigilance they would otherwise apply.

Three findings from 2026-08-22/23 are that same failure arriving with the
reassuring signal intact, which is what makes them harder to see. All three are
gates. They differ in what the gate got wrong.

**1. No gate at all.** `research/notes/01-harness-interface.md` carried
`Status: Accepted (Phase 0 lock)` and the title `ADR-01` while filed in the
evidence folder. No gate ran on that status claim, because `record_schema` is
scoped to the configured record stores and a file outside them is not a record
it can see. Recorded as iss-2608221457227162, with the detector gap as
iss-2608230752354926.

**2. A real gate measuring the wrong property.** The
`cli-verb-taxonomy-restructure` ideate verdict marked a leg 1 claim `verified`
that was wrong in its denominator. Its own errata states the lesson exactly:
"`ideate record` proves every cited id resolves, and proves nothing about
whether a claim is true. A `verified` mark is a claim about the author's
diligence, not an assertion the binary checked." See
[the verdict's errata](../../../development/research/notes/2026-08-22-ideate-cli-verb-taxonomy-restructure.md).

**3. A real gate measuring the right property over the wrong subjects.**
`TestParentsRefuseAnUnknownSubverbAtExitTwo` asserts that every parent command
refuses an unknown sub-verb at exit 2, and derives its subjects from the live
command tree so that a parent added later is exercised without anyone
remembering to add it. That is good design and it worked. It was green while
`abcd capture nosuchthing` minted a durable issue at exit 0, because the sweep
derived from the live tree **minus** `freeTextParents = ["capture", "intent"]`.

Three details make this the sharpest of the three. The exclusion carried a
written justification that read as a design choice, so a reader met a reason
rather than an omission. The two exempted verbs were the two where an unwanted
write costs most, so the exclusion selected for blast radius in the wrong
direction. And the exemption had its own guard, `TestFreeTextParentsAreStillExempt`,
whose stated purpose was that the exemption must keep earning itself.

**That guard could not fail.** It called `cmd.Args(cmd, []string{"some prose"})`
on a `cobra.ArbitraryArgs` validator, which accepts everything by definition.
The second-order gate is the mechanism by which the first-order hole became
invisible rather than merely undocumented: a reader who wondered whether the
exemption was still earned would find a test asserting exactly that, and it was
incapable of reporting otherwise. Recorded as iss-2608221328552172, fixed by
deleting the exclusion rather than adding entries to it, and by replacing the
vacuous guard with one that asserts behaviourally that prose still files.

**Why 2 and 3 are worse than 1.** An absent gate leaves a reader with nothing to
trust, so some vigilance survives. A gate that runs and reports green actively
withdraws that vigilance and supplies false assurance in its place. The existing
principle's own reasoning, that the harm is the withdrawal of compensating
attention, applies with more force here than to the phantom case it was written
for.

**2 and 3 are orthogonal, and the distinction is operational.** Shape 2 is right
subjects, wrong property. Shape 3 is right property, wrong subjects. A search
that finds one will not find the other: a reader holding only the proxy
formulation would grep for proxy measurement and walk straight past an exclusion
list. The rule shape 3 yields: **an exemption from a gate is an enforcement
claim in negative form, and carries the same burden of proof as the gate.** If a
name is excluded from a check, something must be able to fail when the exclusion
stops being true, and a test asserting that an exemption is still earned must
itself be capable of failing.

**The class, stated to cover the three:** a gate is evidence only for the
property it measures over the subjects it measures, and a reader will credit it
with the property they care about across the subjects they assume. A status
field is a claim rather than evidence for it; a gate that checks a claim's form
has not checked its truth; and an exclusion from a gate is a claim carrying the
gate's own burden of proof.

## An adjacent case, deliberately not folded in

A session-end capture finding reported a clean monotone threshold from 11 of 24
transcripts. The full sweep breaks it, and the way it breaks is the point. The
eleven omitted **every** confounded row in the set: the counterexample just below
the claimed boundary, three sessions still running, and two predating the store.
Six rows, all six of them the ones that would have broken the claim, none
excluded deliberately. The sample was drawn in a way correlated with the very
property being measured, which is structure rather than luck. Recorded as
iss-2608230817034768, corrected by the session that filed it once a peer pushed
on the width of the interval.

Stated best by the session that found shape 3: **a sample that excludes its own
counterexamples is an exemption list with no list.** That is also why it is the
hardest of these to catch. An exemption list is a written artefact somebody can
read and challenge, as shape 3 proves; a sample's exclusions are invisible
unless somebody widens it.

**It is recorded here as adjacent rather than as a fourth shape, and the
difference is named so a later reader can split it out cheaply.** Shapes 1 to 3
are all gates, and all involve a claim somebody made about enforcement that a
reader believed. Here there was no gate and no such claim: nobody asserted a
check, and the assurance was inferred from the presentation of the evidence
itself. `enforcement-claims-are-facts` is specifically about claims made in
documents about enforcement, and its stated Why is readers who believe a check
exists. Admitting this case moves the class from *gates that do not gate* to
*assurances nobody issued but everyone read in*, which removes its boundary and
makes it harder to apply rather than easier. Whether to admit it is a maintainer
call, and it is deliberately not taken here.

## What did not catch any of them

**No instrument surfaced any of these.** No lint, test or gate reported a
problem, and the one gate that ran on shape 2 reported success. That is the
defensible statement, and it says where to look: not at the failures, at the
greens.

What did catch them is deliberately not claimed here. The tempting stronger
version, that a peer caught every one, does not survive its own standard: it
reads as more assurance than the evidence supports, which is the failure this
record is about. It is recorded separately as iss-2608230847432287, with its
qualifications about thin independence, catches that were record recall rather
than review, and at least one that was luck. Whatever is catching these, it is not yet a property of the
system, and this record claims only that no instrument is.

Routing is left open deliberately. The paragraph is the cheapest rung, but
whether the class also warrants detectors is a maintainer call, and so is
whether the adjacent case is admitted. Agents agreeing that a principle should
change is not the gate that changes it.
