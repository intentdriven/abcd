---
id: adr-2609021016272867
slug: the-comparative-reading-receives-one-widening-run-s-candidat
status: accepted
date: 2026-09-02
supersedes: null
superseded_by: null
related_intents: [itd-183, itd-186, itd-199, itd-2609020625407419]
related_rfcs: []
related_adrs: [adr-55, adr-56, adr-58]
---

# ADR-2609021016272867: The comparative reading receives one widening run's candidates, a positional exception to the prior-run exhaust, with the run derived from the record and no operand added

## Context

Two rulings the cold-reading workstream shipped cannot both hold as written.

The first is the prior-run exhaust. The include table's exclusions state that
the instrument's own output is never its input, itd-186's fourth criterion
pins it (prior manifests and reading records never reach a reading), and the
readings companion states the general rule: no reading sees another's output,
because a returned item is revision history on the next run.

The second is the comparative reading's object. The 2026-08-28 rulings fix it
as the widening reading's returned configurations before admission, and admit
no other source. Those configurations are reading records in the readings
store, which the first ruling excludes.

itd-199 resolved the collision by refusing: the comparative position does not
assemble, because its object "is not repository material and has no channel
today". That was the right move over serving the position a corpus that is
not its object, and it leaves one of the four positions structurally
unavailable. Iteration 2 needs all four.

The design fixes the invocation at a position and a target state (ruling M8,
framework section 8.2, companion section 4.1). The widening run whose
candidates a comparative reading characterises must therefore come from the
record, not from an operand.

## Decision

**At the comparative position, and at no other, the assembler admits two body
fields of one derived widening run's items: the configuration and what admits
it, keyed by the item's identifier.** Everything else in the readings store
stays excluded at that position as at every other: the run's manifest, the
items' envelopes and patterns, every disposition, admission and surprise, and
every other run. The exception is a row of the include table, so the rendered
charter and the assembler version carry it, and the manifest asserts the
exclusion of the rest of the store by rows derived from the ledger's own
directory list.

**The ground is that the candidate text is cold and its fate is warm.** The
widening reading produced the configurations without ledger access, under the
same blindness the comparative reading holds; what has happened to them since
(dispositions, admissions, surprises) is the researcher's judgement, which is
exactly what a reading must not see. The store holds both halves, and they
separate at field and directory grain, which is the separation the assembler
already performs on a shipped intent.

**The candidate run is derived from the record, and the invocation gains
nothing.** At the comparative position the assembler selects the committed
widening run at the target whose items all carry no disposition and no
admission. Exactly one must qualify: with none it refuses and says so; with
more than one it refuses and names them, and the operator resolves that by
dispositioning one run's items, which is the act the design places after the
comparative reading anyway. The invocation stays a position and a target
state, as the design specifies.

**The ruled order is a gate.** Admission follows characterisation, so at the
widening position no disposition of any state can be written until a committed
comparative run names the widening run, and a comparative assembly refuses if
any candidate already carries a disposition or an admission. A widening run
with fewer than two candidates yields a comparative run with an empty item
set, committed through the ingest verb, so the not-exercised outcome is a run
record and never a mutable file.

**The comparative preset refusal is withdrawn.** itd-199's tenth criterion
refused a preset naming a comparative scope. The committed presets gain a
comparative entry naming the criteria discipline, and every other include row
excludes the comparative position, so no scope can hand it the tree, the brief
or the intents.

## Alternatives Considered

1. **Keep the refusal.** Leaves the position unavailable and Iteration 2
   unable to run Step 4. Rejected: the design names four positions and the
   companion fixes this one's object.
2. **Name the widening run with an operand.** A closed shape that keeps the
   no-prose property, and the draft of this decision took it. Rejected because
   the design fixes the invocation at a position and a target state and an
   added operand contradicts that letter; the record already holds enough to
   derive the run, and the one ambiguous case (two undispositioned widening
   runs) is resolved by the very act the design sequences next.
3. **Admit the whole run.** Simpler, and wrong: the run's manifest, patterns
   and envelopes are the instrument's account of itself, which is revision
   history. Rejected on the same ground the exception rests on.
4. **A separate candidates file outside the readings store.** Would duplicate
   the reading records and give the assembler a second source of truth for
   one item. Rejected under the one-canonical-primitive rule.

## Consequences

- **Brief invariant 15's read-block clause gains the one positional
  exception** with its limit (one run, two fields, one position). Its operand
  enumeration does not move.
- **itd-186's fourth criterion gains the same exception**, and the read-block
  eval gains a comparative case: with dispositions, admissions and a second run
  planted, the comparative bundle carries the named run's two candidate fields
  and nothing else from the store.
- **itd-199's surviving condition on the comparative refusal is
  re-dispositioned** through the condition verb, with this decision's
  delivering intent as the occasion.
- **The operator is told what the assembler selected and why.** The result and the manifest name the widening run derived, and a refusal for none or more than one lists the widening runs at the target with each run's item count and its disposition state, so the operator sees what to disposition to make the selection unambiguous.
- **The closing run at Step 4 is repeatable**, because the gate makes a
  disposition impossible before the comparative run, and a second comparative
  run over the same widening run is a second run; recurrence stays warm work.

## Status note

**Accepted by the maintainer on 2026-09-02, at the planning interview for the
Iteration 2 intents**, and corrected the same day to the derived form: the
draft's named operand contradicted the design's invocation of a position and
a target state, and the maintainer's instruction is that Iteration 2 runs
exactly as the design specifies. The exception to the exhaust and the
ordering gate are the companion's own positions and are unchanged.
