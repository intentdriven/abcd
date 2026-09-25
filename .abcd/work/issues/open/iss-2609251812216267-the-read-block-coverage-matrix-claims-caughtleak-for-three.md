---
schema_version: 1
id: "iss-2609251812216267"
slug: "the-read-block-coverage-matrix-claims-caughtleak-for-three"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

The read-block coverage matrix claims caughtLeak for three comparative rows — dispositions, admissions and surprises never reach the comparative reading, falsifier 'delete the derived row and add an include row for it' — but the mutation leaks nothing on the eval corpus: the comparative preset selects only the discipline kind and the candidate set, so an include row for the family at comparative emits no item. Watched on a scratch copy for surprises (an include row plus the derived row removed: TestReadBlockBaselineIsClean stays green). The removed derived row IS caught, by TestManifestNamesEveryExcludedFamily, so the rows' Caught should be caughtFamily with no class, or the corpus needs a comparative-reachable plant. Found while giving the reframe family its comparative coverage row (spc-2609020626048705), which is declared caughtFamily for this reason.
