---
schema_version: 1
id: "iss-2609252145018018"
slug: "internal-core-lint-still-reads-repository-content-through"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "Covered by lintA's b48fd584 (iss-2609012037127981): every one of the fifteen sites named here reads through readRepoFile/readRepoAbs (containment, then fsutil.ReadGuarded) or fsutil.ReadGuarded directly, and TestLintReadsNothingUnguarded, parsed on go/ast since fix2-lintA, refuses any os.ReadFile/Open/OpenFile, fs.ReadFile or Root.Open in non-test lint code."
impact: fix
resolved_by:
  commit: "b48fd584"
---

internal/core/lint still reads repository content through unbounded os.ReadFile at about fifteen sites (contextcurrency.go, indexdrift.go, persona.go, schema.go's bucket read, speclinks.go, subverbs.go, and lint.go's intent-tree, spec-store, registry and surface reads), where the per-root markdown walk and, since iss-131, receipt_gate read through fsutil.ReadGuarded after containment. A committed symlink to /dev/zero or an oversize file at one of those paths is followed and read unbounded. The sweep is the unhardened-sibling class iss-131 named; it changes symlink handling at each site, so each needs its own containment decision and test.

## Grounds

- pursued: no read of repository content in internal/core/lint follows a symlink out of the tree or reads unbounded; a non-test os.ReadFile or os.Open back in internal/core/lint, or TestLintReadsNothingUnguarded going green over one, would show it wrong
