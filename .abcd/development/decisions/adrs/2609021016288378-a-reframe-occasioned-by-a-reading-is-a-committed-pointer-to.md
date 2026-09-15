---
id: adr-2609021016288378
slug: a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to
status: accepted
date: 2026-09-02
supersedes: null
superseded_by: null
related_intents: [itd-143, itd-2609020625402518]
related_rfcs: []
related_adrs: [adr-50, adr-55]
---

# ADR-2609021016288378: A reframe occasioned by a reading is a committed pointer to an event whose content stays local, and the frame it fingerprints is the framing as it presently stands, which adr-55 enumerates as three committed surfaces

## Context

The cold-reading design lists where an accepted detection may land. Every
landing exists but one: the frame. The design's landing table keeps two
landings apart, the construal section rewritten by delivery (an Iteration 1
landing) and a frame-level revision record (Iteration 2), "without which a
reframe occasioned by a reading cannot be recorded as a reframe". The closing
run's purpose-durability and convergence readings need to tell a tension
answered by a reframe from one answered by a build.

adr-55 rules that the framing as it presently stands is committed record and
that declined construals, superseded terms and the reasoning that settled a
dispute stay on the local side and are read by nothing automated. It
enumerates that framing as three committed surfaces: The construal section's
statement, the glossary's committed terms, and the committed scope. A frame-level record
therefore cannot carry any prior text, and it cannot fingerprint fewer than
the three surfaces without calling something narrower than the frame "the
frame".

## Decision

**A reframe occasioned by a reading is one record in the working tier**,
beside the other reading families, carrying what occasioned it (a reading
item, a disposition or a surprise), the content fingerprint of each of the
three frame surfaces before and after the change, and the grounds. It carries
no construal text, no term text and no scope text. It is warm, excluded from
every reading by positive inclusion, and written only by a verb that reads
the three surfaces' committed content itself and refuses a before-fingerprint
it cannot find in their history.

**The frame is the framing as it presently stands, which adr-55 enumerates
as three committed surfaces**: The construal section of the framing chapter,
the committed glossary terms, and the committed scope. A
reframe may touch any of the three, and the record shows which changed. A
construal-only rewrite is one instance, not the definition.

**Only a reframe a reading occasioned carries a record.** A researcher may
rewrite any of the three surfaces on their own account, and adr-55's rule
that the construal is rewritten by delivery like any other brief passage
stands for that case. This decision adds a pointer for the reading-occasioned
case and no rule over every edit to the frame.

**The join to the occasion is operator-asserted**, with one check: the
occasion predates the rewrite.

## Alternatives Considered

1. **Fingerprint the construal section alone.** Simpler, and narrower than
   both the design's frame and adr-55's framing: a reading-occasioned change
   to the glossary or the scope would go unrecorded as a reframe. Rejected.
2. **Require a reframe record on every frame edit, with a lint.** Makes
   reframes countable at the cost of a rule on every edit to three brief
   surfaces and a lint that reads history, and the design asks only for the
   reading-occasioned case. Rejected for now; adoptable later without changing
   this record's shape.
3. **Carry the prior text in the record.** Breaches adr-55: the abandoned
   framing is exactly what stays local. Rejected.

## Consequences

- **The three frame surfaces carry no marker and no history**; the join lives
  in the record.
- **The record dispatcher gains the reframe family**, and the scribe's derived
  allow list and the comparative position's derived exclusion rows pick the
  family up from the ledger's directory list without a further change.
- **adr-55 is refined, not superseded**: its rule about what is committed and
  what stays local is untouched, and its enumeration of the framing's
  committed surfaces is the one this record fingerprints.

## Status note

**Accepted by the maintainer on 2026-09-02, at the planning interview for the
Iteration 2 intents**, in the three-surface form because the single-section
form contradicted the design's landing table and adr-55's enumeration of the
framing's surfaces. Drafted
as `proposed` and held there until that interview, because it refines an
adopted decision, which is not an agent's act.
