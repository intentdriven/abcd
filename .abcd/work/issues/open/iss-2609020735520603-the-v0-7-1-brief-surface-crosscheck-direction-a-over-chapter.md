---
schema_version: 1
id: "iss-2609020735520603"
slug: "the-v0-7-1-brief-surface-crosscheck-direction-a-over-chapter"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "release-v0.7.1-crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
---

The v0.7.1 brief-surface crosscheck (Direction A over chapters 17 to 31 and Direction B over the five surfaces, tier full) found thirty-four discrepancies, every one in prose no lint rule reads: 02-constraints/04-naming.md (4), 05-internals/01-agents.md (1), 03-configuration.md (5), 05-prompt-quality.md (4), 08-skills.md (1), 06-delivery/01-build-sequence.md (5), 02-verification-matrix.md (7), 04-surfaces/23-reading.md (2), plus four against the CLI verb tree; the eight assigned 04-surfaces chapters gated by surface_coverage were clean but for reading. The consequential ones: 23-reading.md line 128 claims every regime signature ships enforced when all four are observed (ingest_regime.go and its test record the change; the brief sentence was never updated), an overstated safety property on a shipped release; 06-delivery/02-verification-matrix.md line 55 says never a capture promote CLI sub-verb while the verb ships and 04-surfaces/README.md documents it; the disembark invocation is written as 'disembark <source-repo> to <dest>' in seven places across four chapters where the real shape is 'disembark pack <repo> <dest>' with no 'to'; an internal/adapter/{oracle,history,spec,run,scanner} plus internal/registry layout is asserted in two chapters and does not exist; 06-delivery/01-build-sequence.md line 38 attributes abcd init and abcd config get|set to the shipped install milestone and neither exists; the operator-internal verb count disagrees between 08-skills.md (five) and 04-surfaces/README.md (three, correct). The full list with claim and reality per item is the crosscheck output kept with the v0.7.1 receipts. Worth deciding whether the un-gated chapters join the surface_coverage rule's scope rather than being corrected by hand again.
