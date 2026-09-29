---
schema_version: 1
id: "iss-46"
slug: "lint-scope-holes"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "Makefile"
related_intents: [itd-2609291924469783]
deferred_after: "v0.11.1"
deferral_reason: "Re-worded by autonomous run A (lane recRulings, 2026-09-29) after the product thinker's ruling BT3 of 2026-09-29: four of the five holes are closed (re-checked at 1ded82939), and the remaining general warn baseline is ruled as ONE shared baseline file for every warn rule, shrink-only, in which a new warning fails, drafted as itd-2609291924469783. Owed: that draft's planning interview with the product thinker. The scope matrix beyond links stays with this record. capture defer refuses a minor record, so these two fields were set by hand."
---

lint scope holes and gate parity: link-lint does not cover all committed markdown; the persona rule is absent from docs-lint; record-lint blocking semantics are inconsistent between local and CI and there is no warn-baseline ratchet; gofmt is missing from make preflight though attributed to it; repo-local hooks activation and its provisioning dependency are undocumented. Detector (per ratchet-not-big-bang): a lint scope matrix (which rule covers which tree, checked into the record) plus baseline-ratchet support in record-lint so new rules arm immediately against frozen violations. Acceptance corpus: the five holes above.