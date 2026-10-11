---
schema_version: 1
id: "iss-2609020735520603"
slug: "the-v0-7-1-brief-surface-crosscheck"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "release-v0.7.1-crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
resolution: "Every item the record names and every one of the 28 failing entries of the crosscheck output (.abcd/work/reviews/31e9534f76e48b0fdb25110afe68dfc84d77d7d1/iss35-brief-surface-crosscheck.json) was re-checked at 8322cdf65. Already corrected at that base by earlier passes: 23-reading.md's regime-signature claim (985fddf93 removed the signature registry, and the chapter says so), the verification matrix's capture promote row and every other matrix row the output names (b-20 to b-28), 01-build-sequence.md's abcd init and config get|set attribution and its adapter scaffold and dry-run claims (b-16 to b-19), the operator-internal verb list (08-skills.md now points at 04-surfaces/README.md's list rather than counting), 03-configuration.md's dev-sync, --archive, embark scan and config/ claims (b-9 to b-13), 05-prompt-quality.md's linter paragraph (b-14), and b-1 to b-7 in 07-memory.md, 04-naming.md and 01-agents.md. Corrected here (d2f7a968e): the disembark invocation, spelt with the retired 'to' operand at sixteen sites in eight brief and glossary files, now reads 'disembark pack <source-repo> <dest>', and 02-adapters.md and 04-universal-patterns.md say that scanner is the one capability seam under internal/adapter/ while oracle, history, spec, run and internal/registry are design targets, with the source-reader table naming internal/core/history and internal/core/spec. The record's closing suggestion, that the un-gated chapters join surface_coverage's scope, is a proposal rather than a defect and is not taken up here."
impact: internal
resolved_by:
  commit: "d2f7a968e"
---

The v0.7.1 brief-surface crosscheck (Direction A over chapters 17 to 31 and Direction B over the five surfaces, tier full) found thirty-four discrepancies, every one in prose no lint rule reads: 02-constraints/04-naming.md (4), 05-internals/01-agents.md (1), 03-configuration.md (5), 05-prompt-quality.md (4), 08-skills.md (1), 06-delivery/01-build-sequence.md (5), 02-verification-matrix.md (7), 04-surfaces/23-reading.md (2), plus four against the CLI verb tree; the eight assigned 04-surfaces chapters gated by surface_coverage were clean but for reading. The consequential ones: 23-reading.md line 128 claims every regime signature ships enforced when all four are observed (ingest_regime.go and its test record the change; the brief sentence was never updated), an overstated safety property on a shipped release; 06-delivery/02-verification-matrix.md line 55 says never a capture promote CLI sub-verb while the verb ships and 04-surfaces/README.md documents it; the disembark invocation is written as 'disembark <source-repo> to <dest>' in seven places across four chapters where the real shape is 'disembark pack <repo> <dest>' with no 'to'; an internal/adapter/{oracle,history,spec,run,scanner} plus internal/registry layout is asserted in two chapters and does not exist; 06-delivery/01-build-sequence.md line 38 attributes abcd init and abcd config get|set to the shipped install milestone and neither exists; the operator-internal verb count disagrees between 08-skills.md (five) and 04-surfaces/README.md (three, correct). The full list with claim and reality per item is the crosscheck output kept with the v0.7.1 receipts. Worth deciding whether the un-gated chapters join the surface_coverage rule's scope rather than being corrected by hand again.

## Grounds

- pursued: no brief chapter spells a disembark invocation with a 'to' operand or presents internal/adapter/{oracle,history,spec,run} or internal/registry as present; a grep of the brief for either printing a line outside an explicit design-target statement would show it wrong
