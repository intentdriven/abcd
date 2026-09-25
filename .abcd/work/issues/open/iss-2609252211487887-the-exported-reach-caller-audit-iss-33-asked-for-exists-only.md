---
schema_version: 1
id: "iss-2609252211487887"
slug: "the-exported-reach-caller-audit-iss-33-asked-for-exists-only"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/exported_reach_test.go"
---

The exported-reach caller audit iss-33 asked for exists only for internal/core/ahoy (TestEveryExportedAhoyFunctionHasAFrontDoor). A crude survey of the other internal/core packages (an exported top-level function with no 'pkg.Name' selector in non-test Go outside its package) lists about 140 names; many are reached only inside their own package, which is over-export rather than dead code, but some have no production caller anywhere, e.g. launch.Ship (grep for '.Ship(' outside tests finds none). Each hit needs sorting into dead scaffolding (delete or wire), in-package-only (unexport), or a declared test seam (name it ...ForTest), and the audit then generalised to every core package.
