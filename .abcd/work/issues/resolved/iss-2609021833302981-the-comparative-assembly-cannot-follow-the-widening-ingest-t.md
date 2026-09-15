---
schema_version: 1
id: "iss-2609021833302981"
slug: "the-comparative-assembly-cannot-follow-the-widening-ingest-t"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "cold-reading Phase A item 8: the opening-run rehearsal"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reading.md"
resolution: "The derivation now reads the ADR's 'at the target' as reaching an ancestor: a committed widening run qualifies when its recorded target equals the target, or is an ancestor of it and every path changed between the two lies inside the durable readings store and the issue ledger's own record families. A run whose target is not an ancestor is not listed; a run across which anything else changed is listed and refused, naming the first such path with the sentence that the object set changed since the run. The comparative manifest gains candidate_run_target beside target_commit, so a reader has both commits and can diff them. That releases the deadlock: reading ingest leaves the run's records uncommitted, the candidate row reaches them so the next assembly refuses on the dirty gate, and committing them - the act the design sequences between an ingest and the next reading - no longer disqualifies the run. The reading is an interpretation and the maintainer's ruling is owed, captured as iss-2609021857343626."
impact: fix
resolved_by:
  intent: "itd-2609020625407419"
  spec: "spc-2609020626039834"
---

The comparative assembly cannot follow the widening ingest that supplies its candidates: reading ingest writes the widening run's reading records into the committed ledger, where they are uncommitted by construction, and the comparative position's candidate row reaches that store, so the dirty gate refuses the next assembly naming those very records; committing them closes the other end, because the derivation selects a widening run whose own target commit equals the target and the commit has just moved HEAD off it. Both gates are stated in commands/reading.md (Assemble one reading's input: refuses unless HEAD resolves to the target and no included path is uncommitted; The comparative position derives its candidate set from the record: the one committed widening run at the target) and the derivation rule is adr-2609021016272867, so which of the two gives is a design decision the record has not made. Found by rehearsing the opening-run loop on a fixture: the comparative position is reachable only over a widening run whose records were already committed at HEAD, which no run of the loop produces.

## Grounds

- pursued: the comparative position is reachable over a widening run this session ingested and committed, and the rehearsal drives the whole sequence end to end; it would be shown wrong by a comparative reading handed candidates read over an object set the target no longer holds, or by the maintainer ruling that the assembly target should instead be allowed to differ from HEAD
