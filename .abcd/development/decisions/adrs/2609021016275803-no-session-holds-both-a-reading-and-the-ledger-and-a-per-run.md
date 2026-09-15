---
id: adr-2609021016275803
slug: no-session-holds-both-a-reading-and-the-ledger-and-a-per-run
status: accepted
date: 2026-09-02
supersedes: null
superseded_by: null
related_intents: [itd-183, itd-188, itd-2609020625402599]
related_rfcs: []
related_adrs: [adr-55, adr-56]
---

# ADR-2609021016275803: No session holds both a reading and the ledger, and a per-run context stamp is how the record shows it

## Context

Brief invariant 15 states that no session holds both a reading and the ledger.
The scribe definition (itd-188) states the same as the inverse of the read
block: a reading receives a positively included slice of the shipped
repository and no ledger; the scribe receives ledger content and no shipped
tree. The invariant has a sentence and no mechanism.

Its fidelity verdict said so. Both of itd-188's criteria were found met by
declaration: no assembler runs for the scribe, and session retention cannot
show that no session held both. The read block, by contrast, is held by an
assembler, a manifest and an eval. One half of the wall is built and the
other is asserted.

A first attempt at a mechanism, a fixed stamp token per session kind, fails
invariant 16: the tokens would be committed in the documentation and the code,
so any session that read them would carry both and be reported in breach,
while a session that reached the ledger by any route other than the scribe
verb would carry one stamp and be reported clean. The attestation would say
more than the examination establishes.

## Decision

**Every assembly carries a per-run context stamp**: the kind of session it is
for (reading or scribe), the run identifier, and a digest of the assembled
context. The stamp is matched exactly. A session that reads the documentation
carries none; a session that was handed a bundle or a scribe context carries
the one for that run.

**The scribe's context is assembled by a verb with an allow list derived from
the ledger's own directory list**, so a record family added later is inside
the scribe's world and outside the reading's by the same declaration, and the
scribe's manifest is parked in the local tier at assembly and promoted beside
the run at ingest, inside the read block.

**The session store gains a separation check** that reports a retained
transcript carrying both stamps of one run, by name; reports the runs it saw
when none does; and reports the property as unobserved, never as clean, when
no retained transcript carries any stamp. The check reads transcript metadata
only, which keeps it inside the consumer list invariant 15 enumerates.

**What the check does not claim** is stated with it: a host that assembles a
session's context before anything is retained is outside its reach, and there
the scribe definition's protocol remains the gate.

## Alternatives Considered

1. **Leave the property to the protocol.** The reading and scribe sessions are
   run separately by instruction. Rejected: the design's whole argument is that
   a boundary is checkable rather than assertable, and this is the one boundary
   that stayed assertable.
2. **A fixed stamp per session kind.** Rejected for the reason in the context:
   it produces false violations and false clean passes, and an attestation may
   not outrun its examination.
3. **Scan transcript bodies for ledger paths and bundle content.** Reaches the
   bodies, which invariant 15 reserves to the custodian's own reads and to
   nothing else. Rejected.

## Consequences

- **Brief invariant 15 is amended** to name the stamp and the check as the
  mechanism behind its sentence, and to state the unobserved case.
- **The reading bundle's shape changes** by one field, which moves the
  assembler version on itd-199's precedent.
- **The scribe becomes a verb of its own** (`scribe assemble`, `scribe
  ingest`), never a sub-verb of `reading`, because the two contexts must never
  share a front door; its ingest writes through the capture verbs' own
  validators and inherits the ordering gate adr-2609021016272867 states.
- **The scribe transcribes dispositions and reads the reading records from the store.** The framework's scribe section has the scribe transcribe reading outputs as well, and its output-contract section has the ingest verb write the reading records; under the record the verb does that first, so the scribe is handed the run's committed reading records rather than a raw output a second time. That is the framework agreeing with itself through the verb, and it is stated here so nobody reads it as a departure.
- **The transcript store's metadata gains one key** for the stamps, written by
  the custodian at capture and read by the check; bodies are untouched.

## Status note

**Accepted by the maintainer on 2026-09-02, at the planning interview for the Iteration 2 intents**, after each decision was checked against the design framework v4 and its readings companion v4 and found not to contradict them. Drafted as `proposed` and held there until that interview, because it amends a brief invariant and adds a consumer to the transcript store's enumerated list, neither an agent's act. The invariant amendment is adopted with it.
