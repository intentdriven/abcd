---
id: adr-56
slug: an-exclusion-control-asserts-only-what-it-can-prove
status: accepted
date: 2026-08-30
supersedes: null
superseded_by: null
related_intents: [itd-183, itd-194]
related_rfcs: []
related_adrs: []
---

# ADR-56: An exclusion control asserts only what it can prove

## Context

itd-183 shipped the cold-reading assembler with an exclusion floor and a
manifest that states which constructs the floor refused. Ten records stand
against that floor. Read one at a time they are ten spellings; read together
they are one mechanism with one consequence.

The mechanism is that the floor recognises a construct by the shape of the
container it sits in rather than by what the container holds. A file whose
extension is not `.md` is not scanned. A region the fence mask wrongly covers
is not scanned. A frontmatter block that does not open at line 0 is not
frontmatter. In each case the floor does not fail; it declines, silently.

The consequence is the part that makes this an architectural decision rather
than a bug list. The manifest goes on asserting the exclusion regardless. A
third party holding the bundle and the manifest cannot tell the difference
between a scan that ran and found nothing and a scan that never ran, because
the two produce byte-identical attestations. That is not a leak with a
manifest attached; it is an attestation that does not attest. The itd-183
fidelity audit found it live on this repository's own corpus, and the auditor
recorded the finding as manifest and bundle disagreeing with the manifest as
the side that is wrong (iss-2608301450065320).

Nine rounds of review closed nine spellings of that mistake by adding
patterns. The tenth was found not by review but by running the instrument.
The records name the pattern themselves: a spelling arms race needs a design,
not more patterns.

## Decision

**An exclusion control asserts only what it can prove.**

1. **A control that declines to run does not report a clean pass.** Where a
   control cannot examine an input — the type is one it cannot parse, the
   construct is one it cannot resolve, the region is one it cannot bound — the
   answer is a refusal naming the input, never silent admission. This is the
   `loud-staging` principle applied to a control rather than to a stage, and it
   is already the shape `unresolvableFrontmatterShape` holds for one class.

2. **An attestation may not outrun the examination behind it.** A manifest,
   receipt, or verdict that states a construct was excluded is making a claim
   about a scan that ran over the bytes that shipped. Where the two can
   diverge, the artefact states the narrower truth, or the divergence is closed
   before the artefact is emitted. An assertion whose truth conditions are not
   established by the code that emits it is a false statement waiting for a
   reader.

3. **Admission and comprehension describe one set.** Where a system admits
   inputs through one surface and examines them through another, the two
   surfaces are held to the same set. A gap between them is not an
   implementation detail: it is the space in which point 2's divergence lives.
   Whichever surface moves, they meet.

The rule binds the cold-reading assembler first, and every later control whose
output is an attestation a third party relies on.

## Alternatives Considered

**Widen the floor to read whatever the include table admits.** The floor gains
a real parse applied to any admitted document, so a record-shaped construct is
recognised inside a Go fixture, a JSON blob or a doc comment. This closes the
class from the comprehension side and keeps the corpus a reading sees at full
size. Rejected for now on cost and provability: it grows a parser the floor
does not have, and the resulting claim is empirical (the parse caught
everything) where the chosen branch's claim is structural (nothing unparseable
was admitted). Recorded rather than dismissed — it is the branch to revisit if
the narrowing proves to cost more corpus than a reading can spare, which
itd-194 leaves as an open question with the measurement unowned.

**Keep adding signals.** Rejected on evidence rather than on principle: nine
rounds did exactly this, closed nine spellings, and left the tenth live on the
corpus. The records are unanimous that fixing them one at a time is how the
floor spent those rounds.

**Restate the manifest's claim honestly and admit the gap.** The manifest
would say which documents were scanned rather than which constructs were
excluded, leaving the reader to judge. Rejected as insufficient alone: it makes
the attestation true, which is point 2, but leaves the leak, and a reader who
must reconstruct the control's coverage from a document list has been handed
the control's job.

## Consequences

- itd-194 implements point 3 by moving admission: the include table refuses
  what the floor cannot parse. Point 2 then holds by construction rather than
  by scan.
- The corpus a reading sees gets smaller, by an amount nobody has measured.
  itd-194 carries the measurement as an open question; this ADR is the reason
  it must be taken before a reading runs rather than after.
- The rule reaches past the assembler. Any later verb emitting a receipt,
  verdict or manifest inherits points 1 and 2, and the brief carries it as an
  invariant so a new control cannot be written without meeting it.
- The rejected branch stays live as a revisit path, so a later widening is an
  amendment to this ADR rather than a rediscovery of the argument.

## Refinement (2026-09-02)

The third rule above, that admission and comprehension describe one set, is
refined by the maintainer's ruling that source and tests are opt-in to a
reading and that an opted-in item travels whole and marked unscanned in the
manifest. Where a control admits an input it cannot examine, it may pass the
input only if its attestation says, per item, that the examination did not
run; the second rule then carries what the third rule carried, and the two
surfaces are made to agree by disclosure rather than by refusal. Refusal
remains the rule for an input the control claims to have examined and could
not. Recorded in the decision log on 2026-09-02; implemented by itd-194.

