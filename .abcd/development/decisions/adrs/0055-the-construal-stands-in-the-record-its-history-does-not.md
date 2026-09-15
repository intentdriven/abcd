---
id: adr-55
slug: the-construal-stands-in-the-record-its-history-does-not
status: accepted
date: 2026-08-28
supersedes: null
superseded_by: null
related_intents: [itd-86, itd-142, itd-143]
related_rfcs: []
related_adrs: [adr-50, adr-51]
---

# ADR-55: The construal stands in the record; its history does not

## Context

adr-50 settled the two-output rule for elicitation: committed products enter
the record, framing traces stay on a local ledger side no automated reviewer
reads. As stated, its bars reach further than its purpose. The purpose was to
protect deliberation — what a human was willing to consider out loud — from
review pressure and re-litigation. But "framing material" covers two things
adr-50 did not separate: the framing *as it presently stands* (the construal
the framing chapter states, the glossary's committed terms, the committed
scope), and the *history of how it came to stand* (declined construals,
superseded terms, the reasoning that settled a terminological dispute). Read
strictly, adr-50 makes the first inadmissible along with the second — which
would leave a widening reading (itd-86's cold reading at the framing
position) nothing legitimate to read against, and would make the framing
chapter (itd-143) an oddity: a committed brief section automated readers must
pretend not to see.

## Decision

**Two separate rules, both unconditional.**

1. **Record admissibility.** The construal as it presently stands is
   committed record: the framing section's statement, the glossary's
   committed terms, committed scope and vocabulary — products the human
   confirmed, with provenance. Declined construals, superseded terms, the
   reasoning that settled a terminological dispute, and every other framing
   trace remain local-ledger content, per-worktree and never committed.
2. **Reader access.** Of framing material, automated readers — the cold
   reading included — may read committed products, and only those; the
   non-framing items a reading sees (shipped intents, disciplines, specs)
   are admitted by their own records, not by this rule. Nothing automated
   reads the local side: not as context, not as evidence, not as
   calibration data, under no flag. A human may read their own traces;
   machinery may not.

adr-50's remaining points carry forward unchanged: the transcript is
discarded once both outputs are written, and there is no third output. The
line moves; its unconditionality does not.

## Alternatives Considered

1. **Keep adr-50's line and drop the framing-position reading.** Rejected:
   the value of a blind reader at the framing surface is exactly coverage of
   what familiarity smooths over (itd-86's case); dropping it protects the
   deliberation by discarding the benefit the protection exists to enable.
2. **Glossary-only admissibility** (readers see committed terms, never the
   framing statement). Rejected: the glossary is the framing's fingerprint,
   not its statement — a reading against the fingerprint alone reconstructs
   the frame by inference, which is less accurate and no less intrusive.
3. **Reader access under a flag.** Rejected again, on adr-50's own ground:
   unconditionality is the value; a flag makes trace-visibility a
   negotiation and the chilling effect returns the moment the flag exists.

## Consequences

- adr-50 stays accepted; this ADR refines it (relation ruled 2026-08-28),
  and brief invariant 14 is amended in the same change to cite both. At
  filing, adr-50's related_adrs gains this ADR's minted id.
- The cold-reading input assembler's include list cites this decision for
  every framing item it admits; its exclusion list cites rule 2 for the
  local side, and the read-block eval asserts the exclusion on fields.
- **The glossary is the residual risk.** A settled term is often the residue
  of a dispute, so admitting committed terms admits that residue in
  compressed form. The discipline falls on whoever writes the term — the
  committed entry carries the meaning, never the argument — and a breach is
  a lapse-log entry, not something this rule can catch mechanically.
- Grounds (recorded here rather than in a side memo, pre-tooling): the
  extension is preferred to dropping the reading because the reading is the
  point; it is preferred to glossary-only because inference from a
  fingerprint is a worse reader than sight of the statement.
