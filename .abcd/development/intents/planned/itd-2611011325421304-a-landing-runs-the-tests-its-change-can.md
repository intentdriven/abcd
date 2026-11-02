---
id: itd-2611011325421304
slug: a-landing-runs-the-tests-its-change-can
spec_id: spc-2611012047503720
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611011325403127]
refines: [itd-2610261025516272]
related_issues: [iss-2611011343052330]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: fix
---

# A one-package change is checked in minutes, not the whole suite

## Press Release

> **A change to one package is checked in minutes, because abcd reruns only what the change could have affected.** Maya's agent fixes a bug in one package and runs the check before pushing. Instead of an hour of every test in the project twice over, abcd reuses every result that cannot have changed and reruns the rest, and the report says how many tests ran and how many were reused. Nothing the check guarded is skipped: a test whose inputs changed always runs, and a test declared as reading what abcd cannot see always runs too.

## Why This Matters

> On 2026-11-01 a full check of abcd's own repository reused no cached test result at all: every package ran from scratch, then again under the race detector. Go's test cache keys on the test binary, and each worktree's absolute source paths change it; with `-trimpath` the same unchanged package in a second worktree came straight from the cache. `-trimpath` cannot simply be switched on, because nine test files find the repository through `runtime.Caller` and fail under it (iss-2611011343052330). On a busy day the check waited one to two hours for its turn on the machine, so every minute it does not repeat is a minute a waiting agent gets back. The product thinker relies on these checks as the only human in the loop, so the saving must come from not repeating what cannot have changed, never from running fewer of the checks that matter. A change to records alone already takes a shorter path, which still runs the tests that read records (dcn-2608160000000002 ruled that skipping them is fail-open).

## Mechanism

> We expect most unchanged packages to be reused because Go already records, per package, the test binary, the flags, and the files and environment variables a test reads, and reuses a result only when all are unchanged (cmd/go test caching); the change is to stop defeating it with worktree paths. It would be shown wrong by a measured run in which few results are reused: Go keys an opened file on its size and modification time, and a fresh worktree gives every file a new time, so a test that reads the repository's tree reruns in a new worktree even when nothing it reads changed. The measure decides how large the saving is before any figure is claimed.

## Scope Conditions

- The person's own machine, with one Go cache shared by the worktrees of a checkout; hosted CI is out of scope here, because a fresh checkout's new file times defeat the cache there too (left to a follow-up intent). <!-- cond: cond-2611012047502529 -->
- Go repositories; another language keeps the records-only path and whatever reuse its own tooling offers. <!-- cond: cond-2611012047501260 -->
- Tests that are deterministic given their inputs; a test that reads state the cache cannot see (a subprocess's output, a binary it builds, the clock, the network) is declared and always reruns. <!-- cond: cond-2611012047500069 -->
- The full check, uncached, still runs where it runs today in hosted CI (pull request, merge queue, nightly); this intent changes what the person's machine repeats, not what lands untested. <!-- cond: cond-2611012047500513 -->

## Acceptance Criteria

- Given a commit whose check has passed in one worktree, when the same commit is checked in a fresh worktree on the same machine, then every test whose inputs are unchanged is reused rather than rerun, and the check's report counts the tests it ran and the tests it reused.
- Given a test that reads an input the result cache cannot see (a file a subprocess reads, or a file outside the repository), when it is named on the list of always-rerun tests kept on the default branch, then it reruns on every check; and given a release cut, when its gates run, then every test runs without reuse on the tree being tagged.
- Given a change to records alone, when its check runs, then only the tests whose inputs include those records rerun, and the report says which ran and why.
- Given the test helpers that find the repository through `runtime.Caller`, when the whole suite runs under `-trimpath`, then every test passes, and no test finds the repository through its own source path.
- Given a check, when it ends, then its report names how many packages ran and how many were reused, for the unit stage and the race stage alike.
- Given ten consecutive landings after this ships, when their reports are read, then the share of reused packages and the check time are recorded against the baseline of 2026-11-01 (no reuse), and that figure, not an estimate, is the saving stated.

## Decisions

- 2026-11-01, the product thinker in the planning interview: reuse of unchanged test results becomes its own intent, shipped first. This intent refines itd-2610261025516272 by taking its three reuse criteria, the first and third verbatim and the second reworded to name the list on the default branch; that draft keeps a pointer and the landing route.
- 2026-11-01, the product thinker: hosted CI reuse (restoring file times in a fresh checkout) is left to a follow-up intent; this one covers the person's machine.
- 2026-11-01, the product thinker: the always-rerun tests are named on a list on the default branch, never by a test or a change itself.
- 2026-11-01, the product thinker: the press release, the mechanism claim, the four scope conditions and all six acceptance criteria were confirmed, the second worded to name the list on the default branch and its release-cut half confirmed separately; impact is fix.
- 2026-11-02, after the interview, from the landing reviews: the press release's 'a test abcd cannot see the inputs of always runs too' was made precise as 'a test declared as reading what abcd cannot see always runs too' (declared meaning named on the list kept on the default branch), since only a declared test can be rerun on purpose; an undeclared test that reads a subprocess, the clock or the network is reused on the person's machine and caught only by the hosted full run. Flagged for the product thinker's review.
- 2026-11-01, the product thinker: reuse is planned alone and first. The drafts itd-2610110940216682 (every change checked in full once) and itd-2610262141001890 (how many checks run at once) are kept separate from it and are to be planned together later; itd-2610262141001890 waits on itd-2610262140595217, so bundling them with reuse would have held reuse back.

## Open Questions

1. resolved: reuse on the person's machine is this intent; restoring file times so hosted CI can reuse results is a follow-up intent of its own.
3. open (2026-11-02, from the landing reviews): Go hashes a file a test opens by a relative name joined to the package's absolute directory (cmd/go/internal/test/test.go, computeTestInputsID), so such a test misses the cache in a worktree at another path even when nothing it reads changed; a file opened by an absolute name outside the module root is not hashed at all, so a test that reads one can be reused with a stale result. Keep criterion 1 for tests that open files by running the test stages from one fixed directory per checkout, or narrow criterion 1 to tests whose test log names no path inside the module (no open, stat, lstat or chdir of one)?
2. resolved: the tests that always rerun are named on a list kept on the default branch, so a change cannot declare its own tests away.

## Decomposition

Hand-run four-piece table, 2026-11-01, revised after two pre-interview reviews (design and feasibility; record discipline), which found the first draft repeated three of itd-2610261025516272's criteria, contradicted dcn-2608160000000002 by skipping the suite for record-only changes, and bundled macOS scheduling and batching that belong elsewhere.

| Part | Kind | Home |
|---|---|---|
| reuse of unchanged results across worktrees on the person's machine, the declared always-rerun tests, the reuse count, the measured saving | capability | this intent |
| a change never declares its own tests away; the list of always-rerun tests is read from the default branch | trust rule | already standing in the principle evaluator-outside-the-loop; cited, no new ADR |
| `-trimpath`, the `runtime.Caller` sweep | plumbing | iss-2611011343052330, delivered by this intent |
| macOS off every landing in a private repository; one landing check for a feature built in several steps | capability | stays in itd-2610261025516272 |

Verdict: FILE-AS-IS after the revision. Typed links: `refines` itd-2610261025516272, moving its three reuse criteria here verbatim, the parent keeping a pointer; `builds_on` itd-2611011325403127 for the landing report. Advisory reversal flags: none.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: checks wait one to two hours for the machine and each repeats work that cannot have changed; we expect a one-package check to take minutes, and ten landings that show little reuse would show it wrong
