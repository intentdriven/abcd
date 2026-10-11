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

<!-- abcd-review: OWED receipt=rcp-d5bf5bbbdcef -->
Fidelity review OWED (receipt rcp-d5bf5bbbdcef).
<!-- abcd-review-end receipt=rcp-d5bf5bbbdcef -->

## Grounds

- pursued: a run that opens many lanes no longer waits on the indexer, as it did on 2 October (the product thinker, 2026-10-03).
