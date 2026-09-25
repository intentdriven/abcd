---
schema_version: 1
id: "iss-2609252045148890"
slug: "narration-hard-tier-refuses-a-subordinate-clause-no-longer"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
resolution: "A 'no longer' in a clause a subordinator opens (until, once, when, whenever, if, unless, after, before, while, whose) passes the narration hard tier; the subordinators end a clause for the 'no longer' reading alone, so the 'used to' reading still reads through 'until v0.6'. The seven sentences are in narrationSpecification's must-pass set, and every earlier row stays green."
impact: fix
resolved_by:
  commit: "dcd5e124"
---

The launch gate suite's change-narration hard tier refuses a 'no longer' inside a subordinate clause, the commonest present-state shape in reference prose: 'Retry until the error no longer appears.', 'Stop when the gate no longer reports a finding.', 'If the path no longer exists, the loader skips it.', 'Delete the copy once it is no longer needed.', 'Cached entries are dropped once they are no longer referenced.', 'Once a record is resolved it is no longer open.' and 'Branches whose upstream no longer exists are pruned.' all hard-fail, because noLongerNarrates exempts only 'no longer than' and a relative pronoun over is/are. The words can tell these apart: a subordinator (until, once, when, whenever, if, unless, after, before, while, whose) opens the clause. Remedy: exempt a 'no longer' whose clause a subordinator opens, and add the seven sentences to narrationSpecification's must-pass set.

## Grounds

- pursued: the seven subordinate-clause sentences pass TestNarrationGatePassesPresentTense, every refusal row still hard-fails in TestNarrationGateHardFailsOnAChangeConstruct, and the shipped doc bodies carry 0 findings; a refusal row passing (such as 'Until v0.6 the dry-run used to skip the tags.'), a pass row refused, or a shipped doc body finding would show it wrong
