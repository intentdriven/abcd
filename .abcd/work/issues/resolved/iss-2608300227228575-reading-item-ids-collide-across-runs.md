---
schema_version: 1
id: "iss-2608300227228575"
slug: "reading-item-ids-collide-across-runs"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-180 adversarial review, 2026-08-30"
found_at: "internal/core/capture/reading.go (refuseExistingRecord loop, findReadingItem)"
resolution: "The ingest mint now probes every readings/rdg-*/ under the ledger lock and redraws on a hit, so two runs in one second cannot mint one rdi-N; findReadingItem's comment no longer claims the uniqueness it does not enforce."
impact: fix
---

IngestReading's on-disk collision probe checks only the current run's directory, so two runs ingested within one UTC second can mint one rdi-N in two run directories (the adr-45 mint is a second plus four digits, and nothing sequences ingests); afterwards that item can be neither dispositioned nor promoted, findReadingItem reports it present in more than one run, and no tree gate refuses it. The probe must be ledger-wide across every readings/rdg-*/ and redraw on a hit, and the findReadingItem comment claiming two runs can never mint one id is false.
