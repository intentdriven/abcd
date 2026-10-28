---
id: itd-2610030720038073
slug: abcd-keeps-its-home-folder-out-of
spec_id: spc-2610031309233367
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2609091014076309]
related_adrs: [adr-2610030720195401]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: breaking
---

# abcd keeps its home folder out of desktop indexing by default

## Press Release

> On a Mac, abcd's own folder no longer gives the desktop search indexer work to do. The folder is named `~/.abcd.noindex`, a name the indexer passes over, so opening a working copy, writing a run log, or keeping a transcript there sets off no indexing burst, and the computer stays responsive while abcd works. A new install starts with that name. An existing install stops at its next run and asks the person to rename ~/.abcd to ~/.abcd.noindex, naming the one command; after that, abcd runs as before. A project abcd manages keeps pointing at the old folder until abcd's setup runs in that project again, which brings it up to date; a script a person wrote themselves is theirs to update. What search shows does not change, because the folder's contents were never in search results, and abcd never touches the computer's own search settings.

_Proposed by the facilitator from decisions 1 to 4; confirmed as written by the product thinker at the planning interview, 2026-10-03 (decision 5)._

Previous wording (superseded at the interview):

> abcd keeps its home folder out of desktop search by default: on a Mac, Spotlight skips abcd's home (worktrees, run logs, transcripts, caches), so opening a lane's working copy no longer sets off an indexing burst, and a person who wants it searchable turns indexing back on with one setting

## Why This Matters

_Written by the facilitator's agent at filing from the 2026-10-02 run log; corrected to that log at review, 2026-10-03._

On 2026-10-02 an autonomous run opened eight lane worktrees in abcd's machine-scoped store under the person's home folder between 07:14:30Z and 07:18:16Z. At 07:32:31Z the run log records the machine's one-minute load at 26 to 36 while every lane waited on its load gate, which it attributes to the lanes' test phases coinciding. At 07:32:53Z it records load 110 to 148 with no test running and names the desktop indexer's processes indexing the eight new worktrees: corespotlightd (about 114% CPU), mds_stores (about 70%), and mdworkers. Every lane stayed load-gated until the burst passed (run log `~/.abcd/runs/488a0aa9/2026-10-02.jsonl`, the `stop` line at 07:32:53Z); the log does not measure the time lost. Each worktree is a full checkout that lives for hours and is then removed, so indexing it buys nothing a person searches for, and the dot-prefixed home is already left out of search results: the cost is the indexer's scan, not what search shows (decision 1). The store grows as the worktree store intent (itd-2609091014076309) ships, and the whole home held about 12 GB on the product thinker's machine.

Typed links: `refines` itd-2609091014076309 (the machine-scoped worktree store whose creation set off the burst); `related_adrs` adr-2610030720195401 (abcd achieves this only by changing its own folder, never the computer's search settings; routed to its own decision record by the itd-84 decomposition, a routing the product thinker confirmed on 2026-10-03, recorded in the decomposition-calibration research note). The principle `the-users-directory-is-theirs` is the stance the home rests on (the home is abcd's declared space); a principle carries no id, so it stays in prose.

## Mechanism

We expect creating folders under abcd's home to cost the desktop indexer no work because macOS's indexer honours a folder name ending in `.noindex` when it scans, not only when it shows results, for that folder and everything beneath it, as Xcode's DerivedData folders rely on; indexer CPU (corespotlightd, mds_stores, mdworker) unchanged on the 2026-10-02 run shape, eight working copies opened within four minutes, shows the claim wrong.

_Proposed by the facilitator from decisions 1 to 4; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- macOS versions on which a dated receipt shows the `.noindex` name honoured at scan time, macOS 27.0 first; no Apple page documents the name, so each later macOS version holds the claim only once its own receipt is dated. <!-- cond: cond-2610031309233205 -->
- The macOS desktop indexer (corespotlightd, mds_stores, mdworker); any other program on a Mac that scans the home folder is outside the claim. <!-- cond: cond-2610031309237054 -->
- Windows is a later change, with the per-folder attribute `FILE_ATTRIBUTE_NOT_CONTENT_INDEXED` named for it (decision 3). <!-- cond: cond-2610031309234392 -->
- Linux needs no change from abcd: Tracker skips a directory holding `.git` by default, and Baloo's exclusions are a user setting abcd never edits (decision 3). <!-- cond: cond-2610031309236184 -->

_Proposed by the facilitator from decisions 1 to 4; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Proposed by the facilitator from decisions 1 to 4 and the design review's criteria that survive them; each is unconfirmed until walked with its addressee. No criterion is phrased as "search does not find X": the dot-prefixed home already passes that today, so it would test nothing._

- D1 (product thinker; CONFIRMED 2026-10-03) Given an account with no abcd home, when abcd first creates its home, then `~/.abcd.noindex` exists and no `~/.abcd` is created (example: a first install on a clean account, followed by a first run that writes a run log, leaves only `~/.abcd.noindex`); a go test against a temporary home folder asserts it.
- D2 (product thinker; CONFIRMED 2026-10-03, reworded by decision 6) Given an existing `~/.abcd` and no `~/.abcd.noindex`, when any abcd command or hook runs, then it stops before writing anything, names the folder and the one rename command, and creates no new folder; with both folders present it stops and names both (example: the person updates abcd, runs any command, reads the one line, renames the folder, and runs again); a go test against a temporary home folder asserts each case.
- D3 (technical facilitator; CONFIRMED 2026-10-03) Given the Go source tree, when an AST literal test walks every non-test file, then only the one home resolver's package spells the home's folder name, `.abcd.noindex` or `.abcd`, in the shape of TestOnlyTheHistoryPackageNamesTheStorePath (example: a new reader that joins the home folder with `.abcd` itself fails the test, naming its file and line); a go test asserts it.
- D4 (product thinker; CONFIRMED 2026-10-03) Given a managed project whose conventions block names `~/.abcd/rules.json` and `~/.abcd/trusted-roots`, when abcd's setup runs in that project again, then the block names `~/.abcd.noindex/rules.json` and `~/.abcd.noindex/trusted-roots` and every line outside the block is byte-identical, while before setup runs there the block is untouched (example: a project set up before the change keeps its old block until setup runs, then carries the new paths); a go test asserts both states.
- D5 (technical facilitator; CONFIRMED 2026-10-03) Given every abcd code path, when an AST literal test walks the non-test source, then none names `mdutil`, `.Spotlight-V100`, or `VolumeConfiguration.plist` (the Spotlight privacy list) (example: a change that adds a call running `mdutil` fails the test, naming its file and line); a go test asserts it.
- D6 (technical facilitator; CONFIRMED 2026-10-03) Given a Mac with indexing on, macOS 27.0 first, when eight working copies are created under the home within four minutes, the 2026-10-02 run shape, then corespotlightd, mds_stores, and mdworker CPU, sampled every five seconds for a minute, stays under 20% combined, and no run log carries a `stop` naming indexing (example: on 2026-10-02 corespotlightd sat near 114% and mds_stores near 70%; the receipt records samples from the same shape before and after the rename); a dated receipt, naming the macOS version, records it.

## Review findings (design and record discipline, 2026-10-03)

_Written by the facilitator's agent from the two reviews in the local tier (reports/review-desktopsearch-design.md and reports/review-desktopsearch-records.md), after the interview's decisions 1 to 4._

Applied:

- The goal is the indexer's work, not what search shows (design 1): the title says indexing, the Mechanism is the scan-time claim, and no criterion is phrased as search not finding something.
- The press release's "one setting" clause is dropped (design 5, records 10): no such setting exists under decisions 1 and 2. The decomposition-calibration research note's routing row repeats the phrase; it is outside this change.
- Typed links are in the front matter (records 3): `refines: [itd-2609091014076309]` and `related_adrs: [adr-2610030720195401]`; the principle stays in prose, having no id.
- Why This Matters quotes the run log (records 8): load 26 to 36 at 07:32:31Z and 110 to 148 at 07:32:53Z, the indexer processes the log names, no PDF importer, and no time-lost figure, which the log does not carry.
- The decision record grounds its rule on ownership (design 7, records 7 and 9): the setting is the person's and reaches every program, abcd could write it only through a root-only path, and the principle `the-users-directory-is-theirs` is cited as the stance the rule extends. Its `related_adrs` gains adr-2609091248200336, the principle's ruling (records 4); "through indexing alone" is corrected; and its "confirmed" names the routing and where that confirmation is recorded.
- The decision record's "says so loudly when no per-folder method works" is replaced by a dated receipt per macOS version (design 9): no interface reports whether a folder is scanned.
- The marker-file branch is dropped (design 2), as decision 2 also rules.
- Facilitator- and agent-written sections are marked (records 11). The front matter's `origin` and `production_mode` keep their values; the filing commit's `Assisted-by:` trailer is the disclosure.
- Hygiene held (records 12): `~/.abcd`-relative paths only, no private repository names, roles not persons.

Overtaken by the decisions:

- Narrowing the record to the worktree store, with other stores as a measured scope condition (design 3, records 1): overtaken by decision 1, the whole home.
- Renaming one store, `~/.abcd/worktrees.noindex/` (design 3), and its criteria on that path and on a symlinked alias: overtaken by decision 2, the home itself renamed.
- Folding this draft into itd-2609091014076309 and its open spec, or standing it as `blocked_by` that intent behind a marker laid by the store's creation seam (records 2): overtaken by decision 2; the rename is the home's, not the store's, so the draft stands on its own with no dependency.
- A marker file (open question 1's marker branch, records 2's marker route, the decision record's "undone by deleting the marker"): overtaken by decision 2, which records the marker as no longer honoured on current macOS.
- Closing the whole-home question as moot for search (design 4): overtaken by decision 1, which answers it.
- Leaving an old home in place, listed as outside and never moved (design 6, records Q7): overtaken by decision 2, every install moves once, and decision 4, which admits a move abcd can prove it made.
- The reversals of adr-2609091248200336's location clause and of invariant 15's single spelling (records 5 and 6): taken up by decision 4, each by its own superseding record at planning, never an amended record.
- Windows in the same resolver now, and a Tracker receipt (design 8 and its criteria): overtaken by decision 3, Windows later and Linux settled.

Not applied here: appending the routing ruling to the shared decision log (records 4) is outside this change's files; the routing's confirmation is cited from the research note instead.

## Decisions

1. 2026-10-03, the product thinker at the planning interview, asked which folders the change covers (the working-copy folder only, which the 2026-10-02 log shows caused the burst; the whole home): the whole home folder. On the product thinker's machine the home held about 12 GB (lab 7.4 GB, transcripts 3.2 GB, sources 1.1 GB, worktrees 743 MB). The reviews found the home already left out of search results, its name starting with a dot; the change is about the indexer's scanning, not about what search shows.
2. 2026-10-03, the product thinker, asked the method after the reviews found the planned marker file no longer honoured on current macOS (each inner folder renamed to end in `.noindex`; the home itself renamed; the person asked to add the folder in System Settings), and after asking how existing projects would be updated (answered from the code: each managed project's block names the home's top-level `rules.json` and `trusted-roots`, and running setup there again refreshes the block): rename the home itself, `~/.abcd` to `~/.abcd.noindex`. Every install moves once; a managed project names the old folder until setup runs there again; scripts a person wrote are never updated. The product thinker first leaned to renaming each inner folder and asked for an example, then weighed the routes again with the effect on existing projects in view.
3. 2026-10-03, decided without a question (no Windows user on the record): Windows is an explicit later, the per-folder attribute `FILE_ATTRIBUTE_NOT_CONTENT_INDEXED` named for it. Linux is settled by its indexers: Tracker skips a directory holding `.git` by default, and Baloo's exclusions are a user setting abcd never edits.
4. 2026-10-03, flagged for planning (reversals, each needing its own superseding record, never an amended ADR): decision 2 changes the location adr-2609091248200336 binds (`~/.abcd/worktrees/<root-sha>/<name>/`), the transcript store's single spelling under brief invariant 15, every `~/.abcd` path in AGENTS.md, the docs, the brief and the managed block, and the trust paths that read `~/.abcd` (path-entry, trusted-roots, rules.json, the plugin hook's shell guard). How an existing home moves (abcd created it, so the principle admits a move abcd can prove it made), and how both names are read during the move, is the spec's.
5. 2026-10-03, the product thinker: the revised press release confirmed as written.
6. 2026-10-03, the product thinker, proposing it themselves: abcd moves nothing. An existing `~/.abcd` stops every abcd command and hook before it writes anything, naming the one rename command, until the person renames it; with both folders present abcd stops and names both. The automatic move (decision 2's "every install moves once") and its proof that abcd made the folder are dropped, so no move code is carried towards v1.0.0. The press release's sentence and criterion D2 are reworded to match and confirmed in that wording. The facilitator's caveat, recorded with it: the check runs before any write on every entry point, the hooks and the status line included, or a hook could create a fresh `~/.abcd.noindex` beside the old folder.
7. 2026-10-03, the product thinker, asked what abcd's safety check does while the old folder stands: block every command except the rename, until the folder is renamed (spec open question 3).
8. 2026-10-04, the product thinker, asked whether today's release includes the rename and the stop (in today's release; next release; decide later), with the cost shown (every abcd session pauses on the day; they update, abcd renames the folder, and they paste one line to re-link the work folders): in today's release. The release waits until step 3 is built and reviewed, and lands it last, with every session paused.
9. 2026-10-04, the product thinker, asked through the coordinating session whether abcd may make eight working copies and measure desktop search's load before and after the rename before the release (the D6 receipt, a load experiment on their machine): "Yes, run it". Taken 2026-10-04 before the rename on a .noindex probe folder, PASS by the threshold: 2.8% combined against 4.4% for the old shape, which did not burst, so inconclusive on the suffix's effect (research note 2026-10-04-noindex-scan-receipt).

## Open Questions

The four questions filed with the draft are answered: the method and the whole home by decisions 1 and 2, existing installs by decisions 2 and 4, other platforms by decision 3. Nothing remains for the product thinker beyond confirming the press release, the Mechanism, the Scope Conditions, and the acceptance criteria. Owed to the spec, for the technical facilitator:

- Where the stop is checked (decision 6): the one home resolver refuses before any write, on every entry point, the plugin hooks' shell wrapper and the status line included; and how the trust paths (path-entry, trusted-roots, rules.json) read the new home without admitting a symlinked alias the home-scope readers refuse today.
- The shell guard's path: how the plugin hook's guard finds the home when it runs ahead of the binary, and what it says while the old folder stands.
- The superseding records decision 4 lists, one each for adr-2609091248200336's location and for invariant 15's transcript store spelling, and the line each changes in AGENTS.md, the docs, the brief, and the managed block.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-d5bf5bbbdcef -->
Fidelity review — receipt rcp-d5bf5bbbdcef (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:f2447648681cd37d141d6545cc40a33d85d1112b8956b2c154462c98a2de8ab2
Input attestations: diff:b3f68597f0239a89443ab8cba84ec454606b5367..a6178234f9260505df1666ab9a06986bc35f18fa (PR #795, spec step 1: the two superseding records proposed; the request named no range, so each was read from its merge commit's parents; digest over git diff --no-color --no-ext-diff)@sha256:3e99536b814e956d0f7464ef784fae71702746ce144ab7ca1073d788a9bc7ecd; diff:5e40bec752bd16e10946862d5e197134dfe7f43e..e218cff1351e54aec19e751bbb7f89562528b8fa (PR #799, spec step 2: internal/abcdhome and the D3 and D5 boundary tests)@sha256:6d5a90d49241ec1876692182a584aedfa1fb334109531756ef83b8b676086af1; diff:a17dadf0dfc0de664dfeb0c5f4746758a8983fc1..d23a1abc30a4c414c751432e4b5f95e928ad658b (PR #815, spec steps 3 and 4: the rename, the stop and the texts)@sha256:b8b2edef0ff587b478d74cffe235500d86a004379300435c2d4691d5938b5ae9; diff:22c8bf0d53ecc776f7f48b07797f34e98b559b31..1117708bcd5f74d8c2d953da018a1a5004a5d129 (PR #825, spec step 5: the D6 receipt 6538ab4a6 and the spec close 44a73e6f1)@sha256:efa275c5b31f9d2dc5ee0ed0a91c18ea7611ccb3f02e999fbb0eb5d30f3cb6a8; repository:40320db01e6277d61e58fbc8cc2988346c444e52 (origin/main 2e03c61e4 plus committed audit notes, where every cited file:line was read; D3 and D5 greps repeated at the close commit 44a73e6f1)@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 2 · NOT_MET 1 · INCONCLUSIVE 0

Audit owed (receipt rcp-d5bf5bbbdcef): ac-6 NOT_MET. Remedy: fix, then re-run the audit. Carried by iss-2610241551259104.

Per-criterion verdicts:
- ac-1 — MET: TestFirstRunCreatesOnlyTheNoindexHome sets HOME to an empty temporary folder, runs ahoy install --yes then implement join and implement log through the front door, finds the run log under abcdhome.Path(home, "runs", sha) and fails on a ~/.abcd or any other .abcd* entry in the home; the resolver's one name constant is .abcd.noindex, and hooks/bootstrap.sh, the other door that can create the home first, makes only ~/.abcd.noindex
  evidence: internal/surface/cli/firstrun_home_test.go:65 — "logs, err := filepath.Glob(abcdhome.Path(home, "runs", sha, "*.jsonl"))"
  evidence: internal/surface/cli/firstrun_home_test.go:72 — "if _, err := os.Lstat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {"
  evidence: internal/abcdhome/abcdhome.go:38 — "const name = ".abcd.noindex""
  evidence: hooks/bootstrap.sh:607 — "_rm_dir="$home_dir/.abcd.noindex""
- ac-2 — MET_WITH_CONCERNS: cli.run checks abcdhome.Check before root.Execute and answers in each command's form; TestOldHomeStopsEveryVerb, TestOldHomeStopsEveryHook and TestBothHomesStopAndNameBoth assert the line naming ~/.abcd and the rename command (or both folders), the exit status, an unchanged home tree and no ~/.abcd.noindex for ordinary verbs plain and --json, every hook verb, guard hook and statusline, and TestHookWrapperStopsBeforeProvisioning and TestBootstrapWritesNothingBesideTheOldHome hold the two shell doors to the same. Concern, a signed-off divergence (spec open question 6, 2026-10-04): the line also prints a worktree-repair command, because after the rename every store worktree is listed prunable until git worktree repair runs in it, so the person's path is rename, repair, run again rather than the one command the criterion and its example describe
  evidence: internal/surface/cli/cli.go:6816 — "if stop := homeStop(); stop != nil {"
  evidence: internal/surface/cli/homestop_test.go:156 — "if _, err := os.Lstat(abcdhome.Path(home)); !errors.Is(err, os.ErrNotExist) {"
  evidence: internal/surface/cli/homestop_test.go:287 — "func TestBothHomesStopAndNameBoth(t *testing.T) {"
  evidence: internal/surface/cli/hooks_homestop_test.go:73 — "func TestHookWrapperStopsBeforeProvisioning(t *testing.T) {"
  evidence: internal/surface/cli/hooks_homestop_test.go:133 — "func TestBootstrapWritesNothingBesideTheOldHome(t *testing.T) {"
  evidence: internal/abcdhome/abcdhome.go:89 — "reconnect the working copies kept there with \`" + RepairCommand"
- ac-3 — MET: TestOnlyTheHomeResolverNamesTheHome parses every non-test Go file under internal/ and cmd/ (testdata skipped, failing below 100 files) and reports file, line and literal for any .abcd.noindex or ~/.abcd literal and any .abcd-led string beside a home value outside internal/abcdhome; TestHomeNameScannerIsArmed proves the criterion's example, filepath.Join(home, ".abcd"), is reported; a grep of the non-test tree at the close (44a73e6f1) and at HEAD finds no such spelling outside the resolver, every remaining .abcd literal being the repository tier's
  evidence: internal/abcdhome/boundary_test.go:589 — "func TestOnlyTheHomeResolverNamesTheHome(t *testing.T) {"
  evidence: internal/abcdhome/boundary_test.go:604 — "lines = append(lines, fmt.Sprintf("%s:%d: %q", rel, h.line, h.text))"
  evidence: internal/abcdhome/boundary_test.go:715 — "`func f(home string) string { return filepath.Join(home, ".abcd") }`"
  evidence: internal/abcdhome/boundary_test.go:580 — "if walked < 100 {"
- ac-4 — MET: TestSetupRefreshesTheBlocksHomePaths writes the block as planted before the rename (a testdata fixture naming ~/.abcd/rules.json and ~/.abcd/trusted-roots) between the owner's own text, asserts Detect reports marker.outdated and leaves AGENTS.md byte-identical, then after Install asserts the block names ~/.abcd.noindex/rules.json and ~/.abcd.noindex/trusted-roots, holds no ~/.abcd/, and every byte outside the block is identical; only install and embark call installMarkerFile, so nothing else rewrites the block before setup runs
  evidence: internal/core/ahoy/noindex_block_test.go:46 — "if !hasGap(det.Gaps, "marker.outdated") {"
  evidence: internal/core/ahoy/noindex_block_test.go:49 — "if got, _ := os.ReadFile(agents); !bytes.Equal(got, planted) {"
  evidence: internal/core/ahoy/noindex_block_test.go:65 — "for _, want := range []string{"~/.abcd.noindex/rules.json", "~/.abcd.noindex/trusted-roots"} {"
  evidence: internal/core/ahoy/noindex_block_test.go:75 — "if !bytes.Equal(got[:gotLoc[0]], planted[:plantedLoc[0]]) || !bytes.Equal(got[gotLoc[1]:], planted[plantedLoc[1]:]) {"
  evidence: internal/core/ahoy/apply.go:1421 — "wrote, err := installMarkerFile(path)"
- ac-5 — MET_WITH_CONCERNS: TestNoCodeNamesTheSearchSettings walks every non-test Go file under internal/ and cmd/ and fails naming file, line and literal on mdutil, .Spotlight-V100 or VolumeConfiguration.plist, and TestSearchSettingsScannerIsArmed proves each needle is caught in a literal and ignored in a comment; no shipped literal names them. Concern, a narrower scope than promised: the criterion is given every abcd code path, but the walk reads .go files only, so abcd's shell code paths (the hooks/*.sh wrappers run on every hook event, hooks/bootstrap.sh, the git hooks ahoy plants from internal/core/ahoy/defaults) are outside the test; a grep finds none of them naming the settings today, and nothing would fail if one began to
  evidence: internal/abcdhome/boundary_test.go:849 — "func TestNoCodeNamesTheSearchSettings(t *testing.T) {"
  evidence: internal/abcdhome/boundary_test.go:822 — "var searchSettingsNeedles = []string{"mdutil", ".Spotlight-V100", "VolumeConfiguration.plist"}"
  evidence: internal/abcdhome/boundary_test.go:534 — "for _, tree := range []string{"internal", "cmd"} {"
  evidence: internal/abcdhome/boundary_test.go:547 — "if !strings.HasSuffix(d.Name(), ".go") || (isTest && !tests) {"
- ac-6 — NOT_MET: Promised: a dated receipt naming the macOS version that records, for eight working copies created under abcd's home in the 2026-10-02 shape, indexer CPU sampled every five seconds for a minute under 20% combined, with samples from the same shape before and after the rename, and no run log carrying a stop naming indexing. Exists: one receipt (2026-10-04, macOS 27.0 build 26A428) that measured eight copies in a probe folder ~/.d6-probe.noindex outside the home, before the rename, at 2.8% mean (peak 6.1%) against a proxy old shape ~/.d6-index-probe that did not burst, so by its own words inconclusive on the suffix's effect, and that records no run-log check at all, deferring it to this audit; run logs are never in the repository and this machine's home holds no runs/ folder, so the run-log half stays unrecorded and the threshold pass rests on a proxy rather than on the home
  evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:5 — "eight working copies in a `.noindex`-suffixed probe folder shaped as the renamed store, `~/.d6-probe.noindex/worktrees/<root-sha>/`, cost the desktop search indexer **2.8%** CPU combined on average over the minute after (peak 6.1%)"
  evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:6 — "So the receipt passes D6's threshold, and the comparison is inconclusive about the suffix's effect, because the old shape did not burst."
  evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:28 — "The run-log half of D6 cannot be read before the rename: no run has opened its lanes in the renamed store. The audit reads the next run's log against this criterion."
  evidence: .abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:102 — "Taken 2026-10-04 before the rename on a .noindex probe folder, PASS by the threshold: 2.8% combined against 4.4% for the old shape, which did not burst, so inconclusive on the suffix's effect"

Gap audit:
- honoured:
  - A new install starts with the name ~/.abcd.noindex, and no ~/.abcd is created
    evidence: internal/abcdhome/abcdhome.go:38 — "const name = ".abcd.noindex""
    evidence: internal/surface/cli/firstrun_home_test.go:72 — "if _, err := os.Lstat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {"
  - An existing install stops at its next run, before writing anything, and asks the person to rename ~/.abcd to ~/.abcd.noindex, naming the rename command
    evidence: internal/surface/cli/cli.go:6816 — "if stop := homeStop(); stop != nil {"
    evidence: internal/abcdhome/abcdhome.go:48 — "const RenameCommand = "mv ~/" + oldName + " ~/" + name"
    evidence: hooks/bootstrap.sh:144 — "if [ -e "$HOME/.abcd" ] || [ -L "$HOME/.abcd" ]; then"
  - A project abcd manages keeps pointing at the old folder until abcd's setup runs in that project again, which brings it up to date
    evidence: internal/core/ahoy/noindex_block_test.go:49 — "if got, _ := os.ReadFile(agents); !bytes.Equal(got, planted) {"
    evidence: internal/core/ahoy/noindex_block_test.go:65 — "for _, want := range []string{"~/.abcd.noindex/rules.json", "~/.abcd.noindex/trusted-roots"} {"
  - abcd never touches the computer's own search settings
    evidence: internal/abcdhome/boundary_test.go:822 — "var searchSettingsNeedles = []string{"mdutil", ".Spotlight-V100", "VolumeConfiguration.plist"}"
    evidence: .abcd/development/decisions/adrs/2610030720195401-abcd-keeps-its-own-folders-out-of.md:4 — "status: accepted"
- diverged:
  - Naming the one command; after that, abcd runs as before. Delivered as two commands: the stop line also prints a worktree repair, because after the rename every store worktree is listed prunable by its repository until git worktree repair runs in it, a step the person owes before the store's worktrees are linked again (spec open question 6, signed off 2026-10-04)
    evidence: internal/abcdhome/abcdhome.go:67 — "const RepairCommand = `find ~/` + name + \`/worktrees -type d"
    evidence: internal/abcdhome/abcdhome.go:89 — "reconnect the working copies kept there with \`" + RepairCommand"
    evidence: .abcd/development/specs/closed/spc-2610031309233367-abcd-keeps-its-home-folder-out-of.md:435 — "Decided: the technical facilitator, 2026-10-04, asked who repairs them (the person, one printed command; abcd, on its first run; decide later): the person, one printed command."
  - abcd's own folder no longer gives the desktop search indexer work to do, its name being one the indexer passes over. On macOS 27.0.1 the name keeps file contents out of the index but every file under ~/.abcd.noindex is still catalogued by name (589,828 entries on 2026-10-16), each new file costing the catalogue a little work and a lab or worktree copy creating tens of thousands at once
    evidence: .abcd/work/issues/open/iss-2610161806164689-on-macos-27-0-1-spotlight-indexes-abcd-s.md:15 — "On macOS 27.0.1 Spotlight still catalogues every file under abcd's home by name, although the .noindex name keeps it from reading their contents"
  - Opening a working copy there sets off no indexing burst. Shown only for eight copies in a .noindex probe folder outside abcd's home, taken before the rename, against an old-shape control that itself did not burst, so the receipt calls the suffix's effect inconclusive
    evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:5 — "eight working copies in a `.noindex`-suffixed probe folder shaped as the renamed store, `~/.d6-probe.noindex/worktrees/<root-sha>/`"
    evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:6 — "So the receipt passes D6's threshold, and the comparison is inconclusive about the suffix's effect, because the old shape did not burst."
- missing:
  - The computer stays responsive while abcd works: D6's check that no run log carries a stop naming indexing was never taken; the receipt defers it to the next run's log, and no later receipt in the repository records it
    evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:28 — "The run-log half of D6 cannot be read before the rename: no run has opened its lanes in the renamed store. The audit reads the next run's log against this criterion."

Scope-condition dispositions:
- cond-2610031309233205 — narrowed: The only dated receipt is macOS 27.0, and it shows the indexer-CPU bar met by a .noindex probe folder outside the home but calls the suffix's effect inconclusive because its control did not burst, having measured CPU rather than what the index holds; the later finding on macOS 27.0.1, which has no receipt, shows the name keeping file contents out of the index while every file under ~/.abcd.noindex is still catalogued by name
  narrowing: Holds for macOS 27.0 alone, and there only as indexer CPU under 20% for eight copies in a .noindex probe folder outside the home measured before the rename, not as the name shown honoured at scan time against a control that burst; where the name has since been observed (macOS 27.0.1, no receipt dated) it is honoured for file contents only, every file's name and dates still being catalogued
  evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:6 — "So the receipt passes D6's threshold, and the comparison is inconclusive about the suffix's effect, because the old shape did not burst."
  evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:7 — "Taken on macOS 27.0 (build 26A428), 2026-10-04, with the product thinker's consent"
  evidence: .abcd/work/issues/open/iss-2610161806164689-on-macos-27-0-1-spotlight-indexes-abcd-s.md:15 — "the receipt that exists measured indexer CPU, not what the index holds"
- cond-2610031309237054 — survived: The receipt sums, per sample, the CPU of exactly the three processes the condition names, corespotlightd, mds_stores and every mdworker, and nothing in the delivered code, receipt or press release claims anything of any other program that scans the home
  evidence: .abcd/development/research/notes/2026-10-04-noindex-scan-receipt.md:14 — "summing the CPU of corespotlightd, mds_stores and every mdworker per sample"
- cond-2610031309234392 — survived: No Windows change ships: the home is one name on every platform with no platform branch in the resolver, no Go file sets a file attribute, and FILE_ATTRIBUTE_NOT_CONTENT_INDEXED is named only in the shipped intent and the closed spec's Out list as the later change, which is the deferral the condition assumed
  evidence: internal/abcdhome/abcdhome.go:38 — "const name = ".abcd.noindex""
  evidence: .abcd/development/specs/closed/spc-2610031309233367-abcd-keeps-its-home-folder-out-of.md:107 — "- Windows, a later change with `FILE_ATTRIBUTE_NOT_CONTENT_INDEXED` named for"
- cond-2610031309236184 — untested: Nothing in the delivery runs on, measures or receipts a Linux indexer: abcd makes no Linux change and no code names a Baloo or Tracker setting, but the condition's premise, that Tracker's default skip of a directory holding .git leaves abcd's home alone, was never exercised, and nothing measured the home's stores that hold no .git (transcripts, run logs, sources)
<!-- abcd-review-end receipt=rcp-d5bf5bbbdcef -->

## Grounds

- pursued: a run that opens many lanes no longer waits on the indexer, as it did on 2 October (the product thinker, 2026-10-03).
