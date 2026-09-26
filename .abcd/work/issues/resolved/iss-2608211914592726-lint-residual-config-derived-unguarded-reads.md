---
schema_version: 1
id: "iss-2608211914592726"
slug: "lint-residual-config-derived-unguarded-reads"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "bughunt-round-6"
found_at: "internal/core/lint/persona.go:26"
resolution: "Every config-derived and walk-leaf read in internal/core/lint (persona roster, surface registry, both context targets, index_drift doc, gate_lockstep runbook and workflow, sub-verb snapshot and surface files, spec-link index, intent tree, issue ledger, forbidden-synonyms walk) goes through readRepoFile/readRepoAbs: lexical containment, resolved-leaf containment, then fsutil.ReadGuarded. TestLintReadsNothingUnguarded keeps a raw read out of the package; TestPersonaRosterSymlinkedOutOfTheRepoIsRefused, TestContextTargetFIFODoesNotHang, TestIssueLedgerLeafSymlinkedOutIsRefused and TestReadRepoFile pin the behaviour."
impact: fix
resolved_by:
  commit: "b48fd584"
---

residual config-derived os.ReadFile sites in internal/core/lint read cloned-repo-controlled paths without the containment/guarded-read stack the roots and glossary walks now use: persona.go registry, lint.go spec-doc and record-store reads, speclinks.go, contextcurrency.go, deliverystate.go changelog, indexdrift.go, subverbs.go snapshots. Triage each by attacker-controllability under a cloned repo and guard those in scope

## Grounds

- pursued: no lint read follows a link out of the checkout, blocks on a FIFO, or reads past its cap; a production read in internal/core/lint that bypasses the helpers and the suite still green would show it wrong
