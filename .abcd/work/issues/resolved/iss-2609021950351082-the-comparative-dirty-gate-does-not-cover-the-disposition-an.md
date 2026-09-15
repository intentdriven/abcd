---
schema_version: 1
id: "iss-2609021950351082"
slug: "the-comparative-dirty-gate-does-not-cover-the-disposition-an"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "cold-reading Phase A, pre-PR review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/assemble.go"
resolution: "refuseDirtyIncludedPaths now treats any dirty path under the ledger's dispositions or admissions family as included at the comparative position, deriving both roots from capture.LedgerRelPath and issueschema.DispositionsDir/AdmissionsDir rather than naming them literally. Three cases in candidates_test.go hold it: a committed disposition deleted in the working tree alone now refuses naming the record; an uncommitted admission that disqualifies one of two otherwise-ambiguous runs now refuses naming it; and a clean tree carrying committed fate records still derives."
impact: fix
---

The comparative dirty gate does not cover the disposition and admission families, so an uncommitted fate silently selects a different candidate run. refuseDirtyIncludedPaths in internal/core/reading/assemble.go names LintConfigPath and PresetConfigPath by hand but not the two ledger families the comparative derivation reads from the filesystem: capture.ItemFate walks the dispositions and admissions directories to decide whether a widening run is still pre-admission. Commit a disposition on a candidate, delete it in the working tree alone, and the assembly admits the run and mints a manifest whose named commit still carries the disposition — a re-run at that commit would derive a different candidate set. Expected: at the comparative position any dirty path under either family is included, so the gate refuses and names it.

## Grounds

- pursued: the comparative assembly refuses whenever the fate records the derivation reads differ between the working tree and the commit the manifest names, so a manifest never promises a re-run that would derive a different candidate set. What would show it wrong: an assembly that succeeds while a dispositions/ or admissions/ path is dirty, or a refusal over a clean tree whose fate records are merely present.
