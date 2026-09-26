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

Review 1 of the ahoy lane found the audit's caller match was a regex over raw source, so a comment or a string literal naming ahoy.X, or a .go file under testdata/, counted as a caller. The ahoy audit now parses each file (go/parser) and counts only a selector on the imported ahoy package, with testdata/ excluded; the generalisation this record asks for should reuse that matcher rather than the regex. One looseness remains by design: the ForTest suffix exempts a function as a declared test seam by its name alone, with nothing checking that production never calls it.
