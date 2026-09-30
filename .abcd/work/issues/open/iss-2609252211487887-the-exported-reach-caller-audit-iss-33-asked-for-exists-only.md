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
deferral_reason: "audit + ratchet built (run A 2026-09-29, lane reachAudit); 136 names remain in internal/reachaudit/testdata/core-unreached.txt after the 2026-09-30 sort (lane drainReach) and the removals of the lanes landed beside it (integration 24b-3). The generalised audit is TestEveryExportedCoreFunctionIsReachedOrBaselined over every internal/core package, on the one parsed matcher the ahoy audit also uses, so the count can only fall; what remains is the sort of each baselined name into delete, wire, unexport or ...ForTest, a separate judgement per name in packages other lanes are editing."
---

The exported-reach caller audit iss-33 asked for exists only for internal/core/ahoy (TestEveryExportedAhoyFunctionHasAFrontDoor). A crude survey of the other internal/core packages (an exported top-level function with no 'pkg.Name' selector in non-test Go outside its package) lists about 140 names; many are reached only inside their own package, which is over-export rather than dead code, but some have no production caller anywhere, e.g. launch.Ship (grep for '.Ship(' outside tests finds none). Each hit needs sorting into dead scaffolding (delete or wire), in-package-only (unexport), or a declared test seam (name it ...ForTest), and the audit then generalised to every core package.

Review 1 of the ahoy lane found the audit's caller match was a regex over raw source, so a comment or a string literal naming ahoy.X, or a .go file under testdata/, counted as a caller. The ahoy audit now parses each file (go/parser) and counts only a selector on the imported ahoy package, with testdata/ excluded; the generalisation this record asks for should reuse that matcher rather than the regex. One looseness remains by design: the ForTest suffix exempts a function as a declared test seam by its name alone, with nothing checking that production never calls it.

Review 2 of the ahoy lane found two more loosenesses in the parsed matcher, both in the direction of counting a caller that is not one; neither reaches a real file today, and the generalisation should close both. Build constraints are ignored: parser.ParseFile reads a non-test file under `//go:build ignore` or an eval-only tag like any other, so a selector there counts as a front door although no production build compiles it. And the parse runs with SkipObjectResolution, so a local variable named `ahoy` in a file that also imports the package makes `ahoy.X` on the variable read as a call into the package. A dot import returns no selector and a blank import none either, so those two fail loud, the safe direction.

The generalisation is built (run A 2026-09-29, lane reachAudit). internal/reachaudit holds the one matcher, and both TestEveryExportedAhoyFunctionHasAFrontDoor and the new TestEveryExportedCoreFunctionIsReachedOrBaselined use it. It closes both review-2 loosenesses: a file counts only when a release target (the Makefile's TARGETS, no build tags, cgo off) compiles it, so `//go:build ignore` and an eval-only tag name no caller; and the parse keeps object resolution, so a selector whose left side resolves to a local declaration shadowing the import's name is not a caller. It also skips a nested module or worktree, and binds an unaliased import to the imported package's declared name rather than its directory's. Over the whole core it finds 196 unreached names, against the crude text survey's 140; the parsed count is the one the gate holds. Those 196 are the committed baseline; the test fails on an unreached name the baseline lacks and on a baseline line that is reached or gone. The sort of each name is what this record still asks for. Two names have no reference anywhere, tests included: lint.PrincipleEvidence and oracle.BundledDenylist. Both stay in the baseline, because open lanes are editing their packages.

## Progress 2026-09-30

Run A, lane drainReach, sorted the baselined names in memory (outside writer.go), reading, readingitem, statusline, lifeboat, glossary, changelog, reviews, mode, issuerecord, identity, cite, source, mdrecord, machineload, grounds, scribe and spec (outside store.go). The baseline falls from 195 names to 140.

Unexported, because only their own package (its tests included) uses them, 54 names: changelog.LatestVersionIn, MaxImpact; cite.NewHTTPChecker (now newShippedHTTPChecker), ParseReceipt; glossary.RenderIndex, RenderLayout; grounds.ParseToken; identity.EffectiveCommitter; lifeboat.RecordManifestSHA256, Render (now renderMapping), Tiers (now allTiers), VerifyManifest; machineload.ParseLoadavgSysctl; memory.BuildCitation, CountSourceTokens, CoverageIndexPath, DetectLicence, NormaliseSourceText, QueryPages, RenderCitedMatches, RenderNoMatches, ResolveDistilledPages, SourceContentHash, ValidateDistilledPage; reading.Admits, AssemblingPositions, DefinitionPath, DeriveCandidateRun, EncodeBundle, EncodeManifest, ExclusionsFor, Kinds (now allKinds), LoadDefinitions, ManifestHash, PresetFor, PresetWindow, Render (now renderCharter), Scans (now allScans), WideningRuns; readingitem.LocateDisposition, LocateSurprise; reviews.Pin (now summaryPin), Read (now readEntries); scribe.DecodeManifest, Exclusions; source.Load (now loadCorpus); spec.RenderSteps, SortByNumber; statusline.Contrast, ContrastRGB, FormatRatio, ParseColor, Render (now renderRow), Validate (now validateSettings).

Renamed to a declared test seam, 1 name: mode.QuestionOpen became QuestionOpenForTest; only the external mode tests read the question marker through it. It is also the one candidate for a later WIRE: if the status line comes to show that a question is open, this is its read side.

Deleted: none. Every name in scope has a caller in its own package or its tests, and none is dead.

Left in the baseline, 28 names in these packages, each for a reason outside this sort:
- called from, or named by, a file reserved to another lane: memory.Dir, IsMemoryPageName, LoadRegistry, ParsePageFilename, RenderContradictions, RenderIndex, SerializeRegistry, SourceClasses, SourceHashes, SourcesIndexPath and MergeIngest (memory/writer.go calls or names them), memory.WithStoreLock and WritePages (defined in writer.go), spec.Validate (spec/store.go calls it) and spec.WithStoreLock (defined in store.go);
- called by tests in another package, so unexporting needs those tests rewritten or a ...ForTest seam: identity.LoadPin and statusline.LoadFrom (ahoy tests), issuerecord.ParseBlock and ParseScalarOrList (capture tests), lifeboat.ManifestSHA256, mode.SetAt, reading.AssemblerVersion, DecodeManifest and LoadDefinition (cli tests), mdrecord.IsHeading (an intent test), scribe.AllowList (a lint test);
- cited by qualified name in a comment in a package being edited elsewhere: changelog.DeriveNext (launch/semver.go) and glossary.Scan (site/glossary.go).

The record stays open: the rest of the baseline lies in packages other lanes are editing.
