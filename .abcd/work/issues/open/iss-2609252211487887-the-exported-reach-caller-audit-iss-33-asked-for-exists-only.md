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
deferred_after: "v0.11.1"
deferral_reason: "audit + ratchet built; 196 names remain in internal/reachaudit/testdata/core-unreached.txt (run A 2026-09-29, lane reachAudit). The generalised audit is TestEveryExportedCoreFunctionIsReachedOrBaselined over every internal/core package, on the one parsed matcher the ahoy audit also uses, so the count can only fall; what remains is the sort of each baselined name into delete, wire, unexport or ...ForTest, a separate judgement per name in packages other lanes are editing."
---

The exported-reach caller audit iss-33 asked for exists only for internal/core/ahoy (TestEveryExportedAhoyFunctionHasAFrontDoor). A crude survey of the other internal/core packages (an exported top-level function with no 'pkg.Name' selector in non-test Go outside its package) lists about 140 names; many are reached only inside their own package, which is over-export rather than dead code, but some have no production caller anywhere, e.g. launch.Ship (grep for '.Ship(' outside tests finds none). Each hit needs sorting into dead scaffolding (delete or wire), in-package-only (unexport), or a declared test seam (name it ...ForTest), and the audit then generalised to every core package.

Review 1 of the ahoy lane found the audit's caller match was a regex over raw source, so a comment or a string literal naming ahoy.X, or a .go file under testdata/, counted as a caller. The ahoy audit now parses each file (go/parser) and counts only a selector on the imported ahoy package, with testdata/ excluded; the generalisation this record asks for should reuse that matcher rather than the regex. One looseness remains by design: the ForTest suffix exempts a function as a declared test seam by its name alone, with nothing checking that production never calls it.

Review 2 of the ahoy lane found two more loosenesses in the parsed matcher, both in the direction of counting a caller that is not one; neither reaches a real file today, and the generalisation should close both. Build constraints are ignored: parser.ParseFile reads a non-test file under `//go:build ignore` or an eval-only tag like any other, so a selector there counts as a front door although no production build compiles it. And the parse runs with SkipObjectResolution, so a local variable named `ahoy` in a file that also imports the package makes `ahoy.X` on the variable read as a call into the package. A dot import returns no selector and a blank import none either, so those two fail loud, the safe direction.

The generalisation is built (run A 2026-09-29, lane reachAudit). internal/reachaudit holds the one matcher, and both TestEveryExportedAhoyFunctionHasAFrontDoor and the new TestEveryExportedCoreFunctionIsReachedOrBaselined use it. It closes both review-2 loosenesses: a file counts only when a release target (the Makefile's TARGETS, no build tags, cgo off) compiles it, so `//go:build ignore` and an eval-only tag name no caller; and the parse keeps object resolution, so a selector whose left side resolves to a local declaration shadowing the import's name is not a caller. It also skips a nested module or worktree, and binds an unaliased import to the imported package's declared name rather than its directory's. Over the whole core it finds 196 unreached names, against the crude text survey's 140; the parsed count is the one the gate holds. Those 196 are the committed baseline; the test fails on an unreached name the baseline lacks and on a baseline line that is reached or gone. The sort of each name is what this record still asks for. Two names have no reference anywhere, tests included: lint.PrincipleEvidence and oracle.BundledDenylist. Both stay in the baseline, because open lanes are editing their packages.
