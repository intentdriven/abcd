---
schema_version: 1
id: "iss-2608311621412224"
slug: "the-reading-assembler-s-staged-runs-line"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/status.go"
resolution: "staged_runs lists only the parked runs no ingest has given an outcome: Describe probes each parked run id for a commit marker or a refusal record, through runOutcome, the probe the rerun refusal already makes, so an ingested or refused run no longer renders as outstanding. Ingest still leaves its parking directory as the run's local evidence."
impact: fix
resolved_by:
  commit: "ef5f1abd9493eefd20539aff9af5d2a5db03714a"
---

The reading assembler's staged_runs line cannot tell a parked run from an ingested one. Describe lists every rdg-* directory under the assembly parking area, and nothing removes that directory after its run is ingested, so a committed run and one no reading has ever been given render identically. The Status doc comment claimed the filter existed until it was corrected; the behaviour did not. An operator reading the bare verb to find out what is outstanding is given a list that grows monotonically and answers a different question. Found while wiring orphaned_ingests, which is a separate field precisely because this one cannot be trusted to mean outstanding.

## Grounds

- pursued: a parked run leaves staged_runs exactly when its id has run.json or refusal.json in the readings record, and a run awaiting ingest stays listed; an ingested or refused run still listed, or a run awaiting ingest missing from the list, would show it wrong
