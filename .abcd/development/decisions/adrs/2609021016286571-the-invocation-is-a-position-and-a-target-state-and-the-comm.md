---
id: adr-2609021016286571
slug: the-invocation-is-a-position-and-a-target-state-and-the-comm
status: accepted
date: 2026-09-02
supersedes: adr-58
superseded_by: null
related_intents: [itd-183, itd-184, itd-199, itd-2609021003095168]
related_rfcs: []
related_adrs: [adr-56, adr-58, adr-2609021016272867]
---

# ADR-2609021016286571: The invocation is a position and a target state, and the committed preset for the position supplies the rest

## Context

The design fixes the cold-reading invocation at two operands: the operator
supplies a position and a target state, and the reading's object and question
come from the definition (framework section 8.2 and ruling M8; companion
section 4.1). adr-58 added a third operand, the scope, because the assembled
input could not be handed to any reader and the instrument had no way to be
pointed at anything. The operand kept the property the design protects, that
no operand carries prose, and it broke the design's letter.

The maintainer's instruction on 2026-09-02 is that Iteration 2 runs exactly
as the design specifies. The scope operand therefore goes, and what it did is
kept by other means that the design already admits: the record.

## Decision

**The invocation is `--position` and `--target`, and nothing else.**

**A committed preset per position supplies what the reading is handed.** The
preset file already maps each position to the kinds, records and paths it
receives (itd-199). The assembler applies the preset for the position it was
invoked at, with no operand naming it. Changing what a position reads is a
commit to the preset file, reviewed and inside the dirty gate, never a flag.
There is no override at invocation and no override stamp; the manifest
records the preset applied and its hash, so a run is reproducible from the
commit it names.

**One preset file, one entry per position.** The `cold` and `warm` names
retire as invocation tokens; a repository that wants a wider reading commits
a wider entry for that position and the manifest shows it. The size report
goes on stating what a run costs and the over-target line goes on naming the
figure against the target.

**No repository path is accepted at the invocation**, as before, and the
presets remain the only place a path may be named.

## Alternatives Considered

1. **Keep the scope operand and amend the design.** Convenient for
   calibration, and the reason adr-58 was taken. Rejected: the design is to
   be run as written, and the operand is not needed to point the instrument,
   since the committed preset points it.
2. **Keep the operand but default it.** Leaves the operator a channel the
   design closes. Rejected.

## Consequences

- **adr-58 is superseded.** Its preset file, dirty-gate membership and
  no-path rule survive; its operand does not.
- **Brief invariant 15's operand enumeration returns to two.**
- **itd-199's shipped verb changes**: `--scope` is removed, the preset for
  the position is applied, and the override stamp leaves the manifest. One
  intent carries that change and lands with itd-194.
- **itd-184's operand pin** is updated to the two operands, and fails closed
  on any addition as it was designed to.
- **The comparative position** derives its widening run from the record
  (adr-2609021016272867) and needs no operand either.

## Status note

**Accepted by the maintainer on 2026-09-02** by the instruction that
Iteration 2 runs exactly as the design specifies. Drafted and adopted in the
same change as adr-2609021016272867's correction.
