---
schema_version: 1
id: "iss-2608311621412224"
slug: "the-reading-assembler-s-staged-runs-line-cannot-tell-a-parke"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/status.go"
---

The reading assembler's staged_runs line cannot tell a parked run from an ingested one. Describe lists every rdg-* directory under the assembly parking area, and nothing removes that directory after its run is ingested, so a committed run and one no reading has ever been given render identically. The Status doc comment claimed the filter existed until it was corrected; the behaviour did not. An operator reading the bare verb to find out what is outstanding is given a list that grows monotonically and answers a different question. Found while wiring orphaned_ingests, which is a separate field precisely because this one cannot be trusted to mean outstanding.
