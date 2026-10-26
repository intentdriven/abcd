---
id: itd-2609151138388536
slug: append-only-logs-conflict-on-every-merge
spec_id: spc-2610110931381316
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
related_issues: [iss-2609100507439414]
origin: extracted-from-record
production_mode: hand-written
related_adrs: [adr-2610110925465593, adr-2609151138420062]
---

# Every Decision Is Its Own Record, and the Log Stops Conflicting

Typed links: `related_adrs` [adr-2610110925465593](../../decisions/adrs/2610110925465593-decisions-are-one-flat-file-each-and-the.md) (the shape rule this intent enacts) and [adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted.md) (the earlier rule it supersedes); `refines` [adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and.md) (the timestamp-numeric id seam, which this adds a sixth family to and changes in no way); `related_issues` [iss-2609100507439414](../../../work/issues/resolved/iss-2609100507439414-append-only-logs-conflict-on-every-merge.md) (the negative finding: the append-only log conflicted on the first two of 27 branch merges). Prose cross-references, not typed links, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed.md)): [itd-2609150819432059](itd-2609150819432059-abcd-launch-cannot-set-up-the-release.md), the companion that closes the OTHER conflict class — this intent refines nothing of its, and it refines nothing of this intent's, but the two together close both classes and neither closes both alone; and [iss-2609100508570803](../../../work/issues/resolved/iss-2609100508570803-the-record-verbs-worked-from-worktrees.md), the positive finding from the same run that makes this a shape question — five one-file-per-entry record families, zero conflicts, same 27 merges.

## Press Release

> **abcd's decision log becomes a folder of minted records, and the merge
> conflict it produced on every parallel branch goes away.** A decision is no
> longer a line appended to the bottom of one shared file. It is a record with
> an id, allocated through the same seam that mints captures, intents, specs and
> ADRs, written to its own file in one flat folder, `.abcd/work/decisions/`.
> A command and the site list every decision in date order. The old single file,
> `.abcd/work/DECISIONS.md`, is retired, and every link and citation that named
> it now points at the decision's own file. Two branches that each record a
> decision now merge the way two branches that each capture an issue already do:
> git sees two new files and has nothing to reconcile. A repository abcd manages
> records its new decisions the same way; abcd never touches its existing log,
> which its owner can convert with the same one-off script.
>
> "I landed twenty-seven branches in a day and the decision log fought me on the
> first two merges — I ended up hand-writing a merge attribute into
> `.gitattributes` at two in the morning to make it stop," said Maya,
> autonomous-development practitioner. "The issue ledger never once did that, and
> nobody had to think about why. Now the decisions behave like the issues do."

## Why This Matters

Graduated from [iss-2609100507439414](../../../work/issues/resolved/iss-2609100507439414-append-only-logs-conflict-on-every-merge.md):
a managed repository's shared append-only files conflicted on nearly every merge
across an autonomous run that landed 27 worker branches through one integration
branch in a single day. `.abcd/work/DECISIONS.md` conflicted on the first two
merges and would have conflicted on every later one; the session stopped it by
hand, by adding a `merge=union` attribute for that path to `.gitattributes`.

The positive finding from the same run,
[iss-2609100508570803](../../../work/issues/resolved/iss-2609100508570803-the-record-verbs-worked-from-worktrees.md),
is what makes this a shape question rather than a plumbing question. In the same
27 merges — 33 issue resolutions, 5 intents shipped, 3 ADRs superseded, 1 ideate
verdict — the record store did not conflict once. Every one of those five
families is one file per entry. The two files that conflicted are the two that
are single, append-to-the-bottom files. The conflict is a property of the file
shape, not of the content or of the branching.

The log itself has said so since it was written. `.abcd/work/DECISIONS.md`'s own
header carries the instruction: "Graduate this file to per-file
`decisions/<date>--<slug>.md` if size or parallel-agent merge contention bites."
It has now bitten, in a measured run, at 330 entries and 2,489 lines.

The remedy applied in the field — the union merge attribute — works, and it is
the reason the log stopped conflicting for the rest of that run. It is still a
patch on the shape rather than a fix of it: it needs a `.gitattributes` entry
written into a repository abcd adopts, it makes the ordering of two concurrent
appends non-deterministic, and it is safe only for entries that are independent
and dated, which is a property nothing checks. A folder needs none of that.

This intent closes one of the run's two conflict classes. The other is the
changelog, whose `[Unreleased]` section has the same shape and where union is
**not** safe — it duplicates the `###` headings. abcd answered that half for
itself by deriving the changelog from records at the cut and refusing a non-empty
`## [Unreleased]`; what a managed repository lacks is the ability to use that
flow, which is
[itd-2609150819432059](itd-2609150819432059-abcd-launch-cannot-set-up-the-release.md).
This intent `refines` nothing of itd-2609150819432059's, and itd-2609150819432059
`refines` nothing of this one's: they are disjoint. Together they close both
conflict classes, and neither closes both alone. The typed link is a companion
link, recorded so that a reader who finds one half does not conclude the other
half was overlooked.

The shape rule this delivers is
[adr-2610110925465593](../../decisions/adrs/2610110925465593-decisions-are-one-flat-file-each-and-the.md),
which supersedes
[adr-2609151138420062](../../decisions/adrs/2609151138420062-the-decisions-log-is-a-folder-of-minted.md).
The id seam it mints through is
[adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and.md),
which is already the allocator for captures, intents, specs and — since the
ruling of 2026-09-01 — ADRs; a sixth family costs the seam nothing, which is the
point of having a seam.

## Mechanism

We expect this to remove the conflict class rather than reduce it **because a
merge conflict requires two sides to have edited overlapping regions of the same
file, and a folder of one-file-per-decision gives two concurrent decisions no
shared region at all.** The prediction is falsifiable in the shape of the
artefact, not in a statistic: if the assembled index is a committed file that
every decision also writes to, the conflict simply moves from `DECISIONS.md` to
the index, and the class is not closed — it is renamed. The index must therefore
be derived at read time or regenerated from the folder, never appended to by the
minting verb, and an acceptance criterion below is written to catch exactly that
failure.

The secondary claim is that the readability cost is recoverable. A folder is
harder to read end to end than a file, and **we expect the listing command and
the site page to recover that**, because each renders the whole history in date
order from the folder, and every citation that named the old file now names the
decision's own file. If a reader has to open hundreds of files to answer a
question the single file answered, the listing has failed and the claim is
wrong.

## Scope Conditions

- **Repositories using git.** The claim is about git's merge behaviour on file <!-- cond: cond-2610110931385884 -->
  boundaries; nothing here is asserted about another version-control system.
- **abcd itself and every repository abcd manages.** New decisions take the <!-- cond: cond-2610110931387631 -->
  shape in both. abcd never rewrites a managed repository's existing log; until
  its owner converts it by hand, it stays as frozen history, so the claim covers
  the decisions recorded after the shape is adopted.
- **Parallel branch work.** The benefit is proportional to concurrency. At one <!-- cond: cond-2610110931381195 -->
  branch at a time the single file never conflicts and the folder buys nothing
  but consistency with the other five families.
- **Decisions that are independent and dated.** This is the same condition the <!-- cond: cond-2610110931383463 -->
  union attribute silently assumed. A decision family where one entry amends
  another in place is outside this claim — an amendment is a new record, on the
  standing rule that a durable record is superseded rather than edited.

## Acceptance Criteria

> _BDD (the [itd-1](../disciplines/itd-1-acceptance-gates.md) discipline)._

- **Given** a repository on the new shape, **when** a decision is recorded,
  **then** it is written as one file under the decisions folder carrying an id
  allocated through the adr-45 seam, and no other committed file is appended to
  by that act.
- **Given** a decisions folder holding entries, **when** a person runs the
  listing command or opens the site's decisions page, **then** every entry is
  listed in date order with its id, derived from the folder, and no committed
  file is written by recording a decision.
- **Given** the shape has landed in abcd, **when** the repository is searched,
  **then** `.abcd/work/DECISIONS.md` does not exist, no committed file names it
  as a live path, and every citation that named a dated entry of it points at
  that entry's own file. A citation the migration could not map with certainty
  is listed for review in the change, never guessed.
- **Given** two branches cut from the same base that each record one decision,
  **when** both are merged into the integration branch, **then** both merges
  complete with no conflict and with both decision files present, and this holds
  with no `merge=union` attribute and no merge driver configured for the path.
- **Given** a repository abcd manages that already has a `DECISIONS.md`,
  **when** abcd's install runs, **then** the existing file is left byte for byte
  as it was, new decisions are recorded in the decisions folder, and no merge
  attribute is scaffolded into its `.gitattributes`.
- **Given** a repository adopting abcd with no decision log, **when** adoption
  runs, **then** it gets the decisions folder and no single log file.
- **Given** the new shape is in force, **when** the repository is inspected,
  **then** the decisions-append gate (`record-lint decisions-append`, rules
  DA001–DA004, in `internal/core/lint/decisionsappend.go`) is gone, no CI job invokes it, `make preflight` does not run it,
  and nothing in the record or the conventions names it as a live gate — a folder
  of minted files has no mid-file position to police, so the gate is retired by
  the shape rather than disabled.
- **Given** abcd's existing log, **when** the migration has run,
  **then** every entry is present as its own file with its original date and text
  preserved verbatim, the index renders them in date order, and the
  count matches — a migration that silently drops or reorders history fails this
  criterion.

## Open Questions

None open: the product thinker settled them on 2026-10-11 (Decisions, below).

## Decisions

- 2026-10-11, the product thinker, in the planning interview: `.abcd/work/DECISIONS.md` is retired outright, not kept as a symlink or a signpost page, and every link to it across the repository is rewritten, since a single decisions file is not a common standard (MADR and adr-tools keep one file per decision). This supersedes the earlier ADR's symlink (adr-2610110925465593).
- 2026-10-11, the product thinker: the folder is flat, `dec-<id>-<slug>.md`, with no year, month or day subfolders; the id is the timestamp, so a listing sorts by date.
- 2026-10-11, the product thinker: a repository abcd manages but did not create keeps its existing `DECISIONS.md` as frozen history, never rewritten; new decisions go to the folder.
- 2026-10-11, the product thinker, in the criteria walk: all eight acceptance criteria stand as written (four walked as stories, four accepted as a list).
- 2026-10-11, the product thinker: the migration of abcd's own log (splitting its entries into the folder and rewriting every link and citation) is a one-off script run once in this repository, not a verb or any code in abcd. It is needed once and only here; a managed repository is never migrated, since its log stays as frozen history.
- 2026-10-11, the technical facilitator, with one answer defensible on the record: the folder is `.abcd/work/decisions/` (the home the press release named) and its records are `dec-`, a prefix no family uses; the union attribute for `.abcd/work/DECISIONS.md` goes in the same change; the changelog's attribute stays out of scope as a separate conflict class.

- 2026-10-14, the product thinker, in a second planning interview that reconciled this record with later rulings: the press release's last sentence now reads "abcd never touches its existing log, which its owner can convert with the same one-off script"; a person may run the one-off script by hand on a repository they own, and abcd never runs it there; the migration of abcd's own log lands in the same change as the folder; the migrated entries take date order (the 48 written out of date order move to their dates), so the migration criterion reads "date order"; the site page is the "Decision log". The mechanism, the scope conditions and the eight criteria were walked again and stand.
- 2026-10-14, the technical facilitator: the prefix is `dcn`, not `dec`, so a decision record reads apart from the lifeboat's `dec-L<line>` keys; `abcd decide "<text>"` (and `/abcd:decide`) records a decision, `abcd decide` with no operand lists them, and `abcd decide --adr "<title>"` mints an ADR, which bare `abcd decide "<title>"` did before. Accepted cost: an older page or an earlier release's binary still mints an ADR for the bare form until both catch up.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-72909a4e917b -->
Fidelity review — receipt rcp-72909a4e917b (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:17e87db78710bffb85a47c32fc345f272bc3664c6b1ac38269149f83d854a9c8
Input attestations: diff:89d8d9110..64661b186 (PR #1026 land/decisions-folder, merged 2026-10-16 as 64661b18623a3307b00581557d975b3b94209749; digest over git diff --binary --no-renames of that range)@sha256:4792750bab070e96cf4f41f9feefc1122a17ba46eca985da9ba55f6901920276; commits:1fab6e304 and 7a1494402, follow-ups naming the intent, landed in PR #1038 as 2696127a8@-; tree:origin/main 2e03c61e454bf09ecda8cee0ae2f377157b522f7, the state judged (the worktree HEAD 18170d03f adds only other intents' audit notes)@-; forge:PR #1026 body, comments, reviews and its 14 status checks (all SUCCESS), read with gh pr view@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Both writers, abcd decide "< text>" and ideate's public-verdict pointer, go through decisions.Record. It mints the id with recordid.Minter.Mint over the dcn family, which is registered in the adr-45 seam's familyRoots, and creates exactly one file in .abcd/work/decisions/ by exclusive create, under a lock held on the folder itself that leaves no artefact. TestRecordWritesOneFileAndNothingElse asserts that git status names only the new record path, and PR #1026 merged with every check green.
  evidence: internal/core/decisions/record.go:96 — "id, err := minter.Mint(Family)"
  evidence: internal/core/decisions/record.go:108 — "if err := fsutil.CreateExclusiveIn(r, rel, []byte(content), 0o644); err != nil {"
  evidence: internal/core/decisions/record.go:160 — "The lock is on the directory, so it leaves no"
  evidence: internal/core/recordid/resolve.go:237 — "{"dcn", DecisionsRelDir},"
  evidence: internal/core/decisions/record_test.go:81 — "if got := status(t, root); len(got) != 1 || got[0] != "?? "+d.Path {"
  evidence: internal/surface/cli/decide.go:108 — "d, err := decisions.Record(repoRoot, text)"
  evidence: internal/core/ideate/record.go:215 — "d, err := decisions.Record(repoRoot, pointer)"
- ac-2 — MET: Bare abcd decide reads the folder on every run. List sorts by id, and the id is the date stamp; ReadListing reverses that to newest first. Each block is the id and then the body, and the JSON form carries id, date, slug, path and text. TestListingIsDateOrderedAndDerived asserts the order in both forms and that listing leaves the tree unchanged. The site's Decision log renders every exported dcn record with no cap, grouped by month, each row its id, date and full body; its test builds developmentDeckCap+5 records and finds every one, and abcd's site.json opts in. No index is committed, because site-render builds into the local tier, so recording a decision writes nothing else. The order is newest first, by the product thinker's 2026-10-15 ruling (spec decision 21), which is still date order.
  evidence: internal/core/decisions/decisions.go:209 — "sort.SliceStable(out, func(i, j int) bool { return idLess(out[i].ID, out[j].ID) })"
  evidence: internal/core/decisions/record.go:226 — "for i, j := 0, len(ds)-1; i < j; i, j = i+1, j-1 {"
  evidence: internal/core/decisions/record_test.go:218 — "t.Errorf("reading the listing changed the tree: %q", s)"
  evidence: internal/core/site/decisionlog.go:62 — "func (e *explorer) decisionLogPage() (string, error) {"
  evidence: internal/core/site/decisionlog_test.go:22 — "const decisionCount = developmentDeckCap + 5"
  evidence: internal/core/site/decisionlog_test.go:63 — "func TestDecisionLogPageListsEveryEntry(t *testing.T) {"
  evidence: .abcd/site.json:63 — ""decisions": true"
  evidence: Makefile:223 — "@go run ./cmd/abcd site build --out .abcd/.work.local/scratch/site-render-check >/dev/null"
- ac-3 — MET_WITH_CONCERNS: The log is absent at origin/main; c2fdca9e1 removed it. The only live-surface mentions left are the frozen-log handling for managed repositories, through decisions.FrozenLog, and absence assertions in tests. A sweep of every tracked file outside the history set finds no dated-entry citation of the log. Every dcn handle cited outside test fixtures resolves, all 182 relative links to decision records reach a file, and 12 sampled rewrites each point at the entry whose words they quote. First concern: the review list is not itemised in the change. Commit 88fdb204b gives only totals (435 items: 210 mapped, 15 to the folder, 40 reworded, 170 left as history). PR #1026 has no comments and carries no list, no per-class counts and no script source or sha256, although spec step 6 promised all of them. So 'never guessed' can be spot-checked but not audited item by item. Second concern: the reviews charter's live link to ../DECISIONS.md survived the merge. It was fixed the next day in 1fab6e304, and only because the release docs review found it. TestNoLiveDecisionsLogPath exempts the whole of .abcd/work/reviews/ as history, so it would not catch a repeat.
  evidence: git show origin/main:.abcd/work/DECISIONS.md — "fatal: path '.abcd/work/DECISIONS.md' does not exist in 'origin/main'"
  evidence: internal/core/decisions/frozenlog.go:18 — "const FrozenLog = ".abcd/work/DECISIONS.md""
  evidence: internal/core/decisions/livepath_tree_test.go:78 — "func TestNoLiveDecisionsLogPath(t *testing.T) {"
  evidence: .abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:153 — "2026-09-29 (recorded in [dcn-2609300000000011]"
  evidence: git show 88fdb204b (commit message) — "The citations the migration script could not map with certainty, 435 on"
  evidence: .abcd/development/specs/closed/spc-2610110931381316-append-only-logs-conflict-on-every-merge.md:292 — "round-trip verdict, the citation counts per class, and the review list with"
  evidence: PR #1026 body (gh pr view 1026) — "every citation of the log rewritten or resolved by hand"
  evidence: internal/core/decisions/livepath_tree_test.go:28 — "".abcd/work/reviews/","
  evidence: git show 1fab6e304 -- .abcd/work/reviews/README.md — "decisions pending graduation to ADRs"
- ac-4 — MET: TestTwoBranchesRecordingMergeCleanly cuts two branches from one base in a repository with no .gitattributes and records one decision on each through Record. It merges both into an integration branch; the git helper fails on any non-zero exit, a conflict included. It then asserts that both files are present and that git check-attr reports merge unspecified for each. abcd's own .gitattributes lost the log's union line in 69235f66a, and its only merge attribute left is CHANGELOG.md's. On the forge, 10 main-line merges since landing had decision records added on both sides after their merge base, among them PR #1114 (24 on the branch, 3 on main) and PR #1112 (1 against 35). Every one of them landed.
  evidence: internal/core/decisions/record_test.go:123 — "func TestTwoBranchesRecordingMergeCleanly(t *testing.T) {"
  evidence: internal/core/decisions/record_test.go:144 — "git(t, root, "merge", "--no-ff", "-q", "-m", "merge "+branch, branch)"
  evidence: internal/core/decisions/record_test.go:150 — "got != p+": merge: unspecified""
  evidence: .gitattributes:12 — "CHANGELOG.md merge=union"
  evidence: git show 69235f66a -- .gitattributes — "-.abcd/work/DECISIONS.md merge=union"
  evidence: merge 6fe563eeb (PR #1114) — "Merge pull request #1114 from intentdriven/chore/coordination-checklist-plans"
  evidence: merge 34d9b582d (PR #1112) — "Merge pull request #1112 from intentdriven/fix/lane-e-captures"
- ac-5 — MET: TestInstallLeavesAFrozenLogByteForByte installs over a repository whose DECISIONS.md has a CRLF line and no trailing newline. It asserts the file's sha256 is unchanged, the receipt claims no write of it, and the decisions folder is seeded beside it. TestInstallScaffoldsNoMergeAttribute asserts no merge= line in .gitattributes, and merge unspecified for a record, the README and the frozen log's path; ahoy's only attribute writer is the hooks' EOL pin. New decisions go to the folder whether or not a frozen log is present: TestListingNamesAFrozenLog records one beside a log. The shipped rewriter cannot touch the frozen log, because relink holds it in its append-only set.
  evidence: internal/core/ahoy/record_tiers_test.go:249 — "func TestInstallLeavesAFrozenLogByteForByte(t *testing.T) {"
  evidence: internal/core/ahoy/record_tiers_test.go:275 — "if sha256.Sum256(got) != before {"
  evidence: internal/core/ahoy/record_tiers_test.go:293 — "func TestInstallScaffoldsNoMergeAttribute(t *testing.T) {"
  evidence: internal/core/ahoy/record_tiers_test.go:308 — "if strings.Contains(line, "merge=") {"
  evidence: internal/core/ahoy/banlist_scaffold.go:949 — "// pinHookEOL appends the hooks' EOL attribute to .gitattributes, creating the file"
  evidence: internal/core/decisions/record_test.go:228 — "func TestListingNamesAFrozenLog(t *testing.T) {"
  evidence: internal/core/relink/relink.go:72 — "decisions.FrozenLog: true,"
- ac-6 — MET: On a repository holding only .git, TestInstallCreatesTheCommittedTiers has install create both committed tiers and seed .abcd/work/decisions/README.md, the folder's static seed, leaving nothing for detection to report. TestInstallSeedsTheDecisionsFolder closes skeleton.decisions_folder_missing on an existing shared tier and asserts that no .abcd/work/DECISIONS.md is laid. The seed table names the folder's README and no log. That replaces the header-seeded single log an earlier release laid, which 8a88783a1 removed, and the host-run adoption page lays the same folder.
  evidence: internal/core/ahoy/record_tiers_test.go:21 — "func TestInstallCreatesTheCommittedTiers(t *testing.T) {"
  evidence: internal/core/ahoy/record_tiers_test.go:202 — "func TestInstallSeedsTheDecisionsFolder(t *testing.T) {"
  evidence: internal/core/ahoy/record_tiers_test.go:237 — "t.Errorf("install laid a single decisions log: %v", err)"
  evidence: internal/core/ahoy/record_tiers.go:98 — "tier: path.Dir(decisions.RelDir), file: decisions.RelDir + "/README.md", present: decisions.RelDir,"
  evidence: internal/core/ahoy/record_tiers.go:33 — "// No single decisions log is seeded, and one a repository already keeps is never"
  evidence: commands/ahoy.md:250 — "`decisions/` folder, one file per decision, with a `README.md` so git keeps"
- ac-7 — MET_WITH_CONCERNS: The gate is gone. 69235f66a deleted internal/core/lint/decisionsappend.go and its suite, record-lint's decisions-append mode, the Makefile's lint-decisions target and the two CI steps, and none of them is present at origin/main. make preflight lists no lint-decisions, and no CI step invokes the mode. AGENTS.md, CONTRIBUTING, the brief, the principles and the rules files never name it; only its two subject ADRs do, as retired. TestNoLiveDecisionsLogPath refuses the name on live surfaces. Concern: three open issues still name it as a live gate, and the tree test does not scan the ledger. The deferral_reason of iss-2608271804494611 says lint-decisions reads DECISIONS.md. The remedies of iss-2608271804496010 and iss-2608271817300337 ask for proof by lint-decisions green, a target that no longer exists. This delivery's own commit f6a899b20 edited that last remedy line and left the gate's name in place.
  evidence: git show 69235f66a (commit message) — "the log: internal/core/lint/decisionsappend.go and its suite, record-lint's"
  evidence: Makefile:321 — "preflight: load-check fmt-check lint-reviews lint-issues record-lint issue-drift docs-lint check-attribution site-render smoke evals-cold-reading"
  evidence: .github/workflows/ci.yml:541 — "go run ./cmd/record-lint"
  evidence: internal/core/decisions/livepath_tree_test.go:66 — "liveGateRe = regexp.MustCompile(`decisions-append|lint-decisions`)"
  evidence: .abcd/work/issues/open/iss-2608271804494611-work-tier-and-root-prose-sit-under-no.md:12 — "lint-decisions reads DECISIONS.md"
  evidence: .abcd/work/issues/open/iss-2608271804496010-four-pruned-adrs-are-still-cited-by-name.md:10 — "proven by lint-decisions and links_resolve green"
  evidence: .abcd/work/issues/open/iss-2608271817300337-adr-seed-pilot-gate-arming-trust-rules.md:10 — "proven by lint-decisions and record-lint green"
  evidence: git log -1 -- .abcd/work/issues/open/iss-2608271817300337-adr-seed-pilot-gate-arming-trust-rules.md — "f6a899b20 2026-10-15 docs: name the ADR mint as abcd decide --adr everywhere it is cited"
- ac-8 — MET: I rebuilt the migration independently from the log as it stood on main just before the merge (89d8d9110, the same 538 entries as the migration commit's parent). The 538 dated top-level entries map to the 538 dcn records the merge added, with none missing and none extra. Each record's id is its entry's date at 000000 plus the entry's place, in file order, among the entries of that date, and its frontmatter date is the entry's own. In all 538 cases the body equals its entry byte for byte once the leading '- ' and the two-space continuation indent are removed. The 54 entries written out of date order now sort to their dates, which the PR states as '54 moved'. The listing and the site order by id and so render in date order, and all 739 records in the folder today parse, so the reader skips none.
  evidence: git show 89d8d9110:.abcd/work/DECISIONS.md — "Append-only, one line per decision, newest last. Date-prefixed. Architecture-shaping"
  evidence: git show c2fdca9e1 (commit message) — "become 538 records under .abcd/work/decisions/, each id carrying its"
  evidence: .abcd/work/decisions/dcn-2607060000000001-rebuild-abcd-from-scratch-in-go-no.md:6 — "2026-07-06 — Rebuild abcd from scratch in Go, no external tools (specstory,"
  evidence: internal/core/decisions/decisions.go:209 — "sort.SliceStable(out, func(i, j int) bool { return idLess(out[i].ID, out[j].ID) })"
  evidence: PR #1026 body (gh pr view 1026) — "538 entries in, 538 records out, a byte-for-byte round trip, entries in date order (54 moved)"

Gap audit:
- honoured:
  - A decision is a record with an id allocated through the same seam that mints captures, intents, specs and ADRs, written to its own file in one flat folder, .abcd/work/decisions/
    evidence: internal/core/decisions/record.go:96 — "id, err := minter.Mint(Family)"
    evidence: internal/core/recordid/resolve.go:237 — "{"dcn", DecisionsRelDir},"
  - Two branches that each record a decision merge the way two branches that each capture an issue do: git sees two new files and has nothing to reconcile, with no merge attribute
    evidence: internal/core/decisions/record_test.go:123 — "func TestTwoBranchesRecordingMergeCleanly(t *testing.T) {"
    evidence: merge 6fe563eeb (PR #1114) — "Merge pull request #1114 from intentdriven/chore/coordination-checklist-plans"
  - A command and the site list every decision in date order, read from the folder each time
    evidence: internal/surface/cli/decide.go:120 — "func listDecisions(cmd *cobra.Command, asJSON bool) error {"
    evidence: internal/core/site/decisionlog.go:62 — "func (e *explorer) decisionLogPage() (string, error) {"
  - The old single file, .abcd/work/DECISIONS.md, is retired, and abcd's own history moved over intact: 538 entries became 538 records, verbatim and in date order
    evidence: internal/core/decisions/livepath_tree_test.go:81 — "if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(decisions.FrozenLog))); !errors.Is(err, fs.ErrNotExist) {"
    evidence: git show c2fdca9e1 (commit message) — "become 538 records under .abcd/work/decisions/, each id carrying its"
  - A repository abcd manages records its new decisions the same way, and abcd never touches its existing log
    evidence: internal/core/ahoy/record_tiers_test.go:275 — "if sha256.Sum256(got) != before {"
    evidence: internal/core/relink/relink.go:72 — "decisions.FrozenLog: true,"
    evidence: internal/core/repolint/rule_decisions.go:65 — "/ folder to go to; run `abcd ahoy install` to lay it (the log stays as it is)"
  - No hand-written merge attribute is needed any more: the union line abcd carried for the log is gone, and install scaffolds none
    evidence: git show 69235f66a -- .gitattributes — "-.abcd/work/DECISIONS.md merge=union"
    evidence: internal/core/ahoy/record_tiers_test.go:308 — "if strings.Contains(line, "merge=") {"
- diverged:
  - Every link and citation that named the old file now points at the decision's own file. Delivered: a citation of a dated entry points at that entry's record, but of the 435 hand-reviewed items 15 now name the folder, 40 were reworded and 170 still name the log as history. In all, 209 lines in 107 tracked files still name DECISIONS.md: the history set, the log's subject records, and the frozen-log handling for managed repositories.
    evidence: git show 88fdb204b (commit message) — "reworded, and 170 that narrate what the log was are left as history."
    evidence: internal/core/decisions/frozenlog.go:18 — "const FrozenLog = ".abcd/work/DECISIONS.md""
  - The change shows its work (spec step 6): the review list with each item's resolution, the citation counts per class, and the script's source with its sha256, so that a reviewer can rerun it. Delivered: PR #1026 has no comments and states only totals and the round-trip verdict, with no list, no per-class counts and no script. Commit 88fdb204b gives the review list only as totals.
    evidence: .abcd/development/specs/closed/spc-2610110931381316-append-only-logs-conflict-on-every-merge.md:293 — "the resolution of each item. It also carries the script's source in a"
    evidence: PR #1026 body (gh pr view 1026) — "every citation of the log rewritten or resolved by hand"
  - The decisions-append gate is retired by the shape and named as live nowhere. Delivered: it is gone from code, CI, preflight and the conventions, but three open issues still name lint-decisions as a live gate, outside the tree test's scan. One of those remedy lines was edited by this delivery's commit f6a899b20.
    evidence: .abcd/work/issues/open/iss-2608271817300337-adr-seed-pilot-gate-arming-trust-rules.md:10 — "proven by lint-decisions and record-lint green"
    evidence: .abcd/work/issues/open/iss-2608271804494611-work-tier-and-root-prose-sit-under-no.md:12 — "lint-decisions reads DECISIONS.md"
  - TestNoLiveDecisionsLogPath holds the tree to the retirement from then on. Delivered: it exempts all of .abcd/work/reviews/ as history, the live charter README included. So the charter's link to ../DECISIONS.md passed it at the merge and was caught only by the release docs review, which led to the fix in 1fab6e304.
    evidence: internal/core/decisions/livepath_tree_test.go:28 — "".abcd/work/reviews/","
    evidence: git show 1fab6e304 -- .abcd/work/reviews/README.md — "decisions pending graduation to ADRs"
- missing:
  - A managed repository's owner can convert the existing log with the same one-off script. Nothing an owner could run exists. The script stayed in a gitignored local tier and was never committed, and PR #1026 does not carry its source as the spec required. No user doc names a way to convert, and no copy was found in the main checkout's local tier or in the machine's abcd store.
    evidence: .abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:34 — "> which its owner can convert with the same one-off script."
    evidence: .abcd/development/specs/closed/spc-2610110931381316-append-only-logs-conflict-on-every-merge.md:293 — "the resolution of each item. It also carries the script's source in a"
    evidence: git ls-tree -r --name-only origin/main — "no tracked path matches decisions-migrate"

Scope-condition dispositions:
- cond-2610110931385884 — survived: The delivery is bound to git throughout. Record resolves the checkout root through gitutil.CheckoutRoot and refuses outside a repository. The merge proof and every field merge run through git, and nothing is claimed for another version-control system.
  evidence: internal/core/decisions/record.go:65 — "root, err := gitutil.CheckoutRoot(dir, "the decisions folder")"
  evidence: internal/core/decisions/record_test.go:112 — "t.Error("Record outside a repository succeeded; it must refuse")"
  evidence: internal/core/decisions/record_test.go:144 — "git(t, root, "merge", "--no-ff", "-q", "-m", "merge "+branch, branch)"
- cond-2610110931387631 — survived: In abcd the log is gone and 201 records with minted ids have been added since landing. In a managed repository, install seeds the folder and leaves an existing log byte for byte, and every recording goes to the folder. The readers only read the frozen log, relink holds it append-only, and the lint warns without asking it to change. So the shape governs exactly the decisions recorded after adoption. There are two edges, neither a contradiction. An earlier release's binary still files an ADR for the bare form until it is updated, an accepted cost (spec decision 17). And embark carries neither the folder nor a frozen log into a target repository (iss-2610150733180751, open), which concerns history, not new decisions.
  evidence: internal/core/ahoy/record_tiers_test.go:275 — "if sha256.Sum256(got) != before {"
  evidence: internal/core/relink/relink.go:72 — "decisions.FrozenLog: true,"
  evidence: internal/core/repolint/rule_decisions.go:65 — "/ folder to go to; run `abcd ahoy install` to lay it (the log stays as it is)"
  evidence: .abcd/work/decisions/dcn-2610162257477571-refines-dcn-2610162239308903-the-load.md:6 — "2026-10-16 — Refines dcn-2610162239308903, the load gate's queue"
  evidence: .abcd/work/issues/open/iss-2610150733180751-embark-carries-neither-a-source.md:15 — "embark carries neither a source repository's decision records nor its frozen decision log"
- cond-2610110931381195 — survived: Concurrency came and the folder absorbed it. Of the 19 main-line merges since landing whose branch added decision records, 10 met a main that had added records of its own since their merge base, among them PR #1114 (24 against 3), PR #1112 (1 against 35) and PR #1074 (12 against 2). All of them landed with no conflict on the folder. No issue captured since landing reports a decision conflict, which is the falsifier the intent's Grounds name.
  evidence: merge 6fe563eeb (PR #1114) — "Merge pull request #1114 from intentdriven/chore/coordination-checklist-plans"
  evidence: merge 34d9b582d (PR #1112) — "Merge pull request #1112 from intentdriven/fix/lane-e-captures"
  evidence: merge 83f037c28 (PR #1074) — "Merge pull request #1074 from intentdriven/feat/headless-approvals"
  evidence: internal/core/decisions/record_test.go:123 — "func TestTwoBranchesRecordingMergeCleanly(t *testing.T) {"
- cond-2610110931383463 — survived: All 739 records are dated, and Parse refuses a body that does not open with its date. No record already on main has been edited in place: each edit or deletion of a decision file since landing (11 commits modifying, 2 deleting) acted on a record its own branch had added and not yet merged. Amendments arrive as new records; 39 are named as corrections, refinements, supersessions or amendments, for example dcn-2610162257477571, which refines dcn-2610162239308903. Immutability is a convention that nothing enforces mechanically, as the spec says, but it held.
  evidence: internal/core/decisions/decisions.go:146 — "return Decision{}, fmt.Errorf("decisions: %s: the body must open with its date, %s", rel, d.Date)"
  evidence: .abcd/work/decisions/dcn-2610162257477571-refines-dcn-2610162239308903-the-load.md:6 — "2026-10-16 — Refines dcn-2610162239308903, the load gate's queue"
  evidence: .abcd/development/specs/closed/spc-2610110931381316-append-only-logs-conflict-on-every-merge.md:95 — "- **Immutable, like every other record.** A correction is a new record. Nothing"
<!-- abcd-review-end receipt=rcp-72909a4e917b -->

## Grounds

- pursued: We expect two pieces of work that each record a decision to land without a collision on decisions. Shown wrong if such a collision happens again after the change lands.
- pursued: decisions recorded on branches from two accounts must merge without anyone resolving a conflict, or working across accounts stalls. Shown wrong if, after this ships, a pull request that records a decision is still dropped from the merge queue over a decision conflict.
