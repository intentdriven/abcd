---
id: itd-2610161016394967
slug: several-people-each-on-their-own-github
spec_id: spc-2610161044461448
kind: bundle-member
suggested_kind: null
reclassification_history: []
builds_on: []
severity: critical
impact: fix
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610160957404827, iss-2610161015370582, iss-2610161015387318, iss-2610161015371794, iss-2610161015393523, iss-2610161015400509, iss-2610161015397586, iss-2610161015422836, iss-2610152125101468]
bundle: drain-many-accounts
---

# Drains on several accounts share the work

## Press Release

> Several people, each on their own GitHub account and machine, run abcd drain on one repository at the same time, and the drains share the work rather than repeat it. Each drain plans from what origin's default branch holds now, never from a checkout that has fallen behind, so it never reserves an issue another drain already fixed. It skips an issue another account holds before spending a judgement on it, and each account starts at a different point in the queue. An issue one drain hands back stays handed back for every drain, recorded on origin. At the per-person reservation limit a drain waits for a pull request to merge instead of refusing, a lane paused for a while keeps its reservation, a drain whose origin is a fork refuses rather than reserving where nobody looks, one unreadable worktree stops the move once with its name instead of passing everything over, and a drain stops as empty only when nothing is left for anyone. A generated file never locks another account out, and an issue scoped to a whole directory is taken after the issues naming a file and narrows to the files its lane changed.

## Why This Matters

The product thinker wants several people, each on their own GitHub account and machine, to run `abcd drain` on this repository at the same time and get more fixes for it, not the same fixes several times over. Today the drain was built for one account. Every guard against repeated work lives in the checkout that made it, and each account's checkout falls behind origin within hours on a busy day.

A lab with three accounts on one origin, run on 2026-10-16, showed what happens. One account resolved the head issue. A second account, one commit behind, still listed it as eligible, reserved it, opened a lane, and then stalled at the brief, because the issue was already resolved where the lane was cut from (iss-2610160957404827). Every drain judges the same head of the queue, because every drain takes the same order and asks its judgement before checking who holds the issue (iss-2610161015387318). A hand-back is recorded only on the machine that made it, so the other accounts judge the same issue again (iss-2610161015370582). The cap of ten reservations is the ordinary state of a drain whose pull requests wait on a merge, yet it refuses the move and loses the judgement just given (iss-2610161015371794). A lane paused past the two-hour lease is taken over and turns into a duplicate public pull request (iss-2610161015393523). A fork origin reserves where nobody looks (iss-2610161015400509). One unreadable worktree makes every issue read as contended (iss-2610161015397586). A drain stops as empty while other accounts still hold work, and throws its judgements away (iss-2610161015422836). An issue scoped to a directory locks every other account out of every file under it (iss-2610152125101468). Each of these alone wastes a day of several people's runs.

## Mechanism

_Facilitator-on-delegation, 2026-10-16: the product thinker delegated the planning decisions of this bundle; this claim is the facilitator's, grounded in the evidence of the issues it closes._

We expect drains on several accounts to share the work, because every fact a drain chooses by is read from origin, the one place every account shares: the ledger as origin's default branch holds it, the reservations, and the hand-backs. The drain looks at those facts before it spends a judgement, and each account starts at its own point in the queue. This is the competing-consumers pattern with skip-locked selection: a consumer reads the row it locks, and passes over a locked row without waiting on it. It would be shown wrong if, in the first week of drains run on two or more accounts, two accounts open lanes or pull requests for one issue, a drain reserves an issue already resolved on origin's default branch, or an issue one drain handed back is judged again by another while its record is unchanged.

## Scope Conditions

- Holds for accounts that can push to origin, the repository the pull requests target, and that each resolve to their own GitHub username. A person's laptop and cloud session count as one account. <!-- cond: cond-2610161044460239 -->
- Holds while origin answers a fetch at the start of each move. A move that cannot fetch, or cannot read origin's default branch, plans nothing and waits as for an outage. It never plans from what the checkout last saw. <!-- cond: cond-2610161044469355 -->
- Holds while every account draining at once runs a build that carries this bundle. An older build reads a published hand-back as a plain release and may reserve it. <!-- cond: cond-2610161044467946 -->
- Holds for up to about ten accounts draining one repository at once, each opening one lane at a time. Beyond that, the per-move fetch of every reservation, and the race window between two reservation pushes, have not been measured. <!-- cond: cond-2610161044469265 -->
- Holds for a ledger whose issue records name a file or a directory in `found_at`. A record that names neither is handed back before any judgement, as the drain already does. <!-- cond: cond-2610161044463540 -->

## Acceptance Criteria

- **Given** Maya, an autonomous-development practitioner, whose checkout is behind origin's default branch, where Bob, a staff engineer on another account, has since resolved the issue at the head of the queue, **when** Maya's drain makes its next move, **then** it neither reserves, judges nor opens a lane for that issue. (iss-2610160957404827)
- **Given** the same behind checkout, **when** Maya runs `abcd drain --dry-run`, **then** it names the issue as finished on origin's default branch, says how old its view of origin is and how many commits the checkout is behind, and writes nothing. (iss-2610160957404827)
- **Given** an issue that is resolved or wontfix on origin's default branch, whether or not a reservation tip exists for it, **when** a drain or any person reserves it, **then** the reservation backs off as finished on the default branch, and nothing is pushed. (iss-2610160957404827)
- **Given** an issue that is open on origin's default branch but not yet in Maya's checkout, **when** Maya's drain reaches it, **then** the drain reserves, judges and opens a lane for it from origin's record. (iss-2610160957404827)
- **Given** Bob's drain handed an issue back (promoted it to a draft, flagged it for a decision record, named its home, or found it stuck), **when** Maya's drain reaches that issue while its record on origin's default branch is unchanged, **then** Maya's drain passes it over as handed back by Bob, without reserving, judging or promoting it. (iss-2610161015370582)
- **Given** an issue Bob's drain handed back, **when** a person reserves it by hand with `abcd reserve`, **then** the reservation is made, with a note naming the hand-back. (iss-2610161015370582)
- **Given** an open issue whose record names an intent in `related_intents` on origin's default branch, **when** any drain plans its move, **then** the issue is not eligible, and the dry run names the intent that holds it. (iss-2610161015370582)
- **Given** the head of the queue is reserved by another account, **when** Maya's drain makes a move, **then** it passes over that issue, naming its holder, and makes no reservation push and no judgement request for it. (iss-2610161015387318)
- **Given** Maya's and Bob's drains start together on a group of ten eligible issues of one category and severity, **when** each opens its first lane, **then** neither has to back off from the other's first issue on most runs, because each takes the group in its own order, and both take a more urgent group first. (iss-2610161015387318)
- **Given** Maya holds ten live reservations, most of them lanes whose pull requests wait to merge, **when** their drain reaches its next issue, **then** the move does not refuse and asks no judgement: it writes when it will look again, names the pull requests it waits on, and advises waiting for one to merge rather than releasing a reservation. (iss-2610161015371794)
- **Given** Maya's lane is in progress, **when** their drain makes any move, waiting or not, with the lane's lease past its half, **then** the drain renews the lane's reservation. (iss-2610161015393523)
- **Given** Maya's lane whose reservation is now another account's, **when** the lane reaches its push, **then** nothing is pushed, no pull request is opened, and the lane is handed back. (iss-2610161015393523)
- **Given** Bob's checkout has a fork as its remote named origin, **when** Bob's drain starts, or Bob reserves an issue, **then** it refuses before reserving anything, naming the repository the fork was made from and the command that sets origin to it. (iss-2610161015400509)
- **Given** one sibling worktree that abcd cannot read, **when** Maya's drain makes a move, **then** the move refuses once, before any reservation or judgement, naming that worktree and how to repair or remove it, and passes no issue over as contended. (iss-2610161015397586)
- **Given** every remaining eligible issue is reserved by another account, **when** Maya's drain makes a move, **then** it does not stop: it waits, says when it will look again, and keeps every judgement it has taken. (iss-2610161015422836)
- **Given** no eligible issue is left, held or not, **when** Maya's drain makes a move, **then** it stops as empty. (iss-2610161015422836)
- **Given** Bob's lane holds an issue whose footprint includes a file the repository declares as generated, **when** Maya's drain reserves an issue whose footprint includes that file too, **then** that file is not the overlap that refuses it. (iss-2610152125101468)
- **Given** a group of one category and severity holding issues whose `found_at` names a file and issues whose `found_at` names a directory, **when** a drain takes from that group, **then** it takes the file-scoped issues first. (iss-2610152125101468)
- **Given** Bob's lane reserved a directory, **when** its implement stage has finished, **then** the lane's next renewal narrows its reservation to the files its branch changed. (iss-2610152125101468)
- **Given** Maya reserves by hand from a second copy a footprint that overlaps one of Maya's own reservations, **when** the reservation is made, **then** it succeeds with a warning naming the overlap, while a drain's reservations still back off from Maya's other copy. (iss-2610152125101468)

## Decomposition

Run by hand under [`decompose-before-filing`](../../principles/decompose-before-filing.md) on 2026-10-16, facilitator-on-delegation (dcn-2610161040352440), and graded in the decomposition-calibration note.

| Part | Type | Home | Verdict |
| --- | --- | --- | --- |
| Planning from origin's default branch, and backing off any reservation of a record finished there (iss-2610160957404827) | capability | this intent; the per-move fetch is a clause in brief invariant 7's drain sentence | file here |
| Reserving before judging, passing over held issues, and an order of its own for each account (iss-2610161015387318) | capability | this intent | file here |
| Hand-backs published on origin as tombstones, and issues naming an intent made ineligible (iss-2610161015370582) | capability | this intent, with the ineligibility as a clause in brief invariant 19, inside the reservation's own record family on the shared remote (adr-2610031352212697); fixed-form, so no new trust rule (dcn-2610150000000001) | file here |
| Waiting at the cap and when only held work is left, and stopping as empty only when nothing is left for anyone (iss-2610161015371794, iss-2610161015422836) | capability | this intent | file here |
| Renewing the lane in progress on every move, and the reservation check before the push (iss-2610161015393523) | capability | this intent, beside the land fence of itd-2609091034175565 | file here |
| Refusing a fork origin (iss-2610161015400509), and refusing once on an unreadable worktree (iss-2610161015397586) | capability | this intent | file here |
| Generated files shared, same-holder overlaps warned, directory-scoped issues taken last and narrowed (iss-2610152125101468) | capability | this intent; the declaration of generated files is shared with itd-2610161016391855 | file here |

Verdict: FILE-AS-IS, as a member of the drain-many-accounts bundle. itd-2610161016401115 builds on it. Every part is a drain capability over records that already exist, so nothing routes elsewhere.

## SOTA

Facilitator-on-delegation, 2026-10-16, from the grounds the decisions cite:

- **Competing consumers with skip-locked selection.** A queue shared by many workers is safe when each worker reads the row it locks and passes over a locked row without waiting, as `SELECT ... FOR UPDATE SKIP LOCKED` does in PostgreSQL and MySQL. This is mature and widely deployed. Adopted, path 2: the reservation push is the lock, and the drain passes over a held issue before it spends a judgement.
- **Spreading workers without a coordinator.** Rendezvous (highest random weight) hashing gives each worker its own stable order over a shared set, and randomised work stealing does the same for schedulers. Both are mature. Adopted, path 2: a hash of holder, copy and issue orders each group.
- **Reasoned tombstones.** Cassandra and Kafka keep a deletion marker so that every replica agrees the item is gone. This is mature. Adopted, path 2: a hand-back is a released reservation carrying its kind and the record's blob id.
- **Lease keep-alives and fencing.** Chubby and etcd renew a lease independently of the work's progress, and a fencing check before a write stops a holder whose lease lapsed. This is mature. Adopted, path 2.
- **Not adopted.** A central work broker or a hosted queue, because every account already shares only origin, and abcd adds no service.

## Open Questions

_None open; the product thinker delegated this bundle's planning decisions on 2026-10-16, and each is recorded under Decisions. The technical questions the build meets are in the bundle's spec._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-288b7637ec71 -->
Fidelity review — receipt rcp-288b7637ec71 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e3f43db9a1d967cf2367b4036feaa9336da88af15acdbd0424dc6501d261e834
Input attestations: diff:ca99ea28b..80764362b, from the commit that planned spc-2610161044461448 to the commit that closed it (the range also carries other work merged beside it; the paths judged are internal/core/implement/loop, internal/core/claim, internal/core/capture, internal/core/peers, internal/surface/cli, .gitattributes and the brief); digest over git diff --binary --no-renames of that range@sha256:9b0ba7c9f46e56b9a4655d8cf25ce111e34851bdfe9329bc72bdd9ea098eead9; commits:730aca57b, accb6ada6, 59c8b54b0, 561e07e95, dcc1e24d1 and 09c4ad251, follow-ups naming spc-2610161044461448 that landed on origin/main after the close@-; tree:origin/main 2e03c61e454bf09ecda8cee0ae2f377157b522f7, the state judged; the worktree HEAD eac09d94f adds only other intents' audit notes and issue records under .abcd/@-;

Acceptance rollup: MET 13 · MET_WITH_CONCERNS 7 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Each move fetches origin under the drain's lock and plans from one snapshot of origin's default branch, so an issue another clone resolved there is not in the plan, and claim's decideReserve backs off any reservation of a record finished there; the behind-clone test shows a checkout one commit behind neither reserves, judges nor opens a run for the resolved head issue, and takes the next one.
  evidence: internal/core/implement/loop/drain.go:409 — "plan, origin, wait, recovered, probed, err := movePlan(repoRoot, &st, rule, now)"
  evidence: internal/core/implement/loop/drain_origin.go:468 — "if plan, origin, err = planFromOrigin(repoRoot, v, now, drainSpread(repoRoot, key)); err != nil {"
  evidence: internal/core/claim/remote.go:356 — "so there is nothing to reserve; take other work"
  evidence: internal/core/implement/loop/drain_origin_test.go:101 — "func TestABehindClonesMovePassesOverAnIssueResolvedOnOrigin(t *testing.T) {"
- ac-2 — MET: DrainDryRun plans from origin's default branch as last fetched without fetching, and planFromOrigin records when abcd last fetched, the view's age, the commits HEAD lacks and the issues open here that origin finished; the text render prints all three; the test finds origin/main one commit behind, 30 minutes old, the resolved issue named, and the local tier and git status unchanged.
  evidence: internal/core/implement/loop/drain_origin.go:118 — "branch as last fetched, and fetches and writes nothing."
  evidence: internal/core/implement/loop/drain_origin.go:234 — ""rev-list", "--count", "HEAD.."+v.commit"
  evidence: internal/surface/cli/drain.go:498 — "origin: planned from %s at %s, as last fetched %s; this checkout is %d commit(s) behind it"
  evidence: internal/surface/cli/drain.go:501 — "finished on %s, still open in this checkout: %s"
  evidence: internal/core/implement/loop/drain_origin_test.go:300 — "func TestTheDryRunNamesTheFinishedIssueTheViewsAgeAndTheCommitsBehind(t *testing.T) {"
- ac-3 — MET: claim.judge reads origin's default branch whether or not a tip exists, and decideReserve's first row backs off a record finished there before any commit or push, for a drain's reservation and a person's alike, which reserve exits 3 on; the claim test covers no tip, a live tip and a tombstone, resolved and wontfix, asserting the back-off names the record as finished on origin's default branch and origin's tip does not move.
  evidence: internal/core/claim/remote.go:599 — "// The default branch is read whether or not a tip exists, since a"
  evidence: internal/core/claim/remote.go:356 — "return Result{}, message{}, "", backOff("%s, so there is nothing to reserve; take other work; %s","
  evidence: internal/core/claim/finished_test.go:43 — "func TestAReservationOfARecordFinishedOnTheDefaultBranchBacksOff(t *testing.T) {"
  evidence: internal/surface/cli/reserve.go:291 — "case errors.Is(err, claim.ErrBackOff):"
- ac-4 — MET: The plan's ledger is read from origin's commit through treeLedger, and the same plan feeds the lane's start check and the judgement's remedy; the test files an issue on origin alone, and the move judges it over origin's remedy, reserves origin's found_at, opens the lane without adding a record to the checkout, and the lane's brief carries origin's remedy.
  evidence: internal/core/implement/loop/drain_origin.go:259 — "func treeLedger(repoRoot string, v originView) (*capture.TreeLedger, error) {"
  evidence: internal/core/implement/loop/drain.go:1048 — "lane.plan = &plan"
  evidence: internal/core/implement/loop/drain_origin_test.go:149 — "func TestAnIssueOpenOnOriginAndAbsentHereIsJudgedAndOpened(t *testing.T) {"
- ac-5 — MET: Every hand-back releases the drain's reservation with a closed-vocabulary kind (user-visible, trust-rule, design-finding, second-package, unachievable, stuck, closed) and the blob of the record on origin's default branch; another drain's move passes such a tip over from its view before any push or judgement while that blob is unchanged, and claim backs a drain's reservation off it; the cross-clone test has Quinn's drain pass Pat's trust-rule hand-back over naming it, push nothing and write no judgement request, and the stuck test asserts a stuck tombstone.
  evidence: internal/core/implement/loop/drain.go:928 — "if why := view.HandBack(v.ID, now); why != "" {"
  evidence: internal/core/claim/handback.go:289 — "func (s Snapshot) HandBack(id string, now time.Time) string {"
  evidence: internal/core/claim/remote.go:366 — "if p.req.Drain {"
  evidence: internal/core/implement/loop/drain_handback_test.go:74 — "func TestOneClonesHandBackIsPassedOverByAnotherClonesDrain(t *testing.T) {"
  evidence: internal/core/implement/loop/drain_stuck_test.go:84 — "if kind, _ := tombstoneTrailers(t, bare, secondIssue); kind != claim.HandBackStuck {"
- ac-6 — MET: decideReserve makes a person's own reservation over a standing hand-back and attaches the hand-back to the result, and reserve prints HandBack.Note naming who handed it back, as which kind and when; the CLI test has Q's abcd reserve over P's user-visible hand-back exit 0 with that note, and --json carry handed_back.
  evidence: internal/core/claim/remote.go:369 — "res.HandedBack = &HandBack{Kind: j.HandedBack, Holder: j.Holder, At: j.ReleasedAt}"
  evidence: internal/core/claim/handback.go:76 — "this reservation is a person's own, so it was made"
  evidence: internal/surface/cli/reserve_handback_test.go:15 — "func TestReserveOverAHandBackGoesAheadWithANote(t *testing.T) {"
- ac-7 — MET_WITH_CONCERNS: eligibility hands back by the rule promoted any open issue whose related_intents names an intent, ahead of category and severity, judged on the ledger read from origin's default branch, and the dry run prints the reason naming the intents. Concern: a record the consistency pass filed is exempt unless an intent names it back (dcn-2610161425496537, now in brief invariant 19), so such a record naming an intent stays eligible; and a reason names at most three intents, then a count.
  evidence: internal/core/capture/eligible.go:147 — "case !skipPromoted && len(promotedIntents(iss)) > 0:"
  evidence: internal/core/capture/eligible.go:213 — "if consistencyFinding(iss) {"
  evidence: internal/core/capture/eligible_test.go:264 — "func TestAnIssueNamingAnIntentIsHandedBackAsPromoted(t *testing.T) {"
  evidence: internal/surface/cli/drain.go:471 — "termsafe.Sanitize(v.Reason)"
  evidence: .abcd/development/brief/02-constraints/03-invariants.md:175 — "a record the consistency pass files is not one"
- ac-8 — MET: Before any reservation act or judgement the move checks its once-fetched view with Snapshot.Contention and passes over an issue another copy holds, naming the holder, with nothing pushed and nothing asked; the test leaves Quinn's reservation of the head untouched, counts one push (the next issue's reservation), finds the pass-over naming quinn, and the judgement asked over the next issue.
  evidence: internal/core/implement/loop/drain.go:941 — "if why := view.Contention(v.ID, footprint, now, takeOver); why != "" {"
  evidence: internal/core/claim/crowd.go:260 — "return fmt.Sprintf("%s is reserved by %s until %s, %s", id, r.Holder, stamp(r.EffectiveExpiry), s.AsOf(now))"
  evidence: internal/core/implement/loop/drain_held_test.go:62 — "func TestAHeldIssueGetsNoPushAndNoJudgement(t *testing.T) {"
- ac-9 — MET_WITH_CONCERNS: Within each group of one category and severity the plan orders issues by a rendezvous hash of the holder git names, the copy's key and the issue id, under the rule's shared group order, so both accounts take a more urgent group first, and since the reservation precedes the judgement a collision costs one passed-over push, not a judgement. Concern: the first issues differ only by chance (two keys agree on a group's head about once in the group's size, and a fixed pair agrees on a fixed group every run), the test proves only that two keys' whole orders differ, and no two-clone move or lab rerun measures the 'most runs' claim.
  evidence: internal/core/capture/eligible.go:293 — "if spread != "" {"
  evidence: internal/core/capture/eligible.go:308 — "func spreadWeight(spread, id string) uint64 {"
  evidence: internal/core/capture/drainspread_test.go:90 — "func TestTwoHoldersOrderAGroupDifferently(t *testing.T) {"
  evidence: internal/core/capture/drainspread_test.go:111 — "if slices.Equal(a, b) {"
- ac-10 — MET_WITH_CONCERNS: The cap is counted in the move's view before any reservation or judgement, and a reservation act's typed cap refusal ends in the same wait: waiting_until fifteen minutes on, apart from next_eligible_at, naming the holds, the pull requests and the advice to wait for a merge rather than release one; the test at ten live reservations sees no refusal and no judgement, the pace window untouched, #7 named and the judgements kept. Concern: only pull requests of lanes this checkout ran are named, so a lane of the same person's other copy counts toward the cap without its pull request being named.
  evidence: internal/core/implement/loop/drain.go:982 — "if mine := view.LiveHeldBy(capHolder, v.ID); len(mine) >= capLimit {"
  evidence: internal/core/implement/loop/drain_wait.go:98 — "wait for one of their pull requests to merge, which ends its reservation, rather than releasing one"
  evidence: internal/core/implement/loop/drain_wait.go:94 — "waiting := "none of them a lane's pull request in this checkout""
  evidence: internal/core/implement/loop/drain_wait_test.go:26 — "func TestADrainAtTheCapWaitsAndNamesThePullRequests(t *testing.T) {"
- ac-11 — MET: drainMove renews the lane in progress at half its lease after the merged-lane finish and before the window gate, so a move the pace window holds still renews it; the test pauses the drain, moves 61 minutes into a two-hour lease with no step run, and finds origin's tip renewed and the run record logging it.
  evidence: internal/core/implement/loop/drain.go:678 — "leaseNotes = renewInProgress(repoRoot, st, view, now)"
  evidence: internal/core/implement/loop/reservation.go:158 — "half := own.ClaimedAt.Add(own.EffectiveExpiry.Sub(own.ClaimedAt) / 2)"
  evidence: internal/core/implement/loop/drain_lease_test.go:24 — "func TestAWaitingDrainMoveRenewsTheLaneInProgress(t *testing.T) {"
- ac-12 — MET: pushFence makes the land fence's reservation check before the landing's push and refuses when the hold is another's, so nothing is pushed and no pull request opened, and the drain hands that lane back as stuck at once; the test moves the reservation to quinn before the push and finds no branch on origin, no pull request, the lane routed back and Quinn's reservation left as it stands.
  evidence: internal/core/implement/loop/reservation.go:254 — "func pushFence(c Context, lane Lane) (time.Time, string, error) {"
  evidence: internal/core/implement/loop/reservation.go:233 — "stays: "nothing is pushed and no pull request is opened""
  evidence: internal/core/implement/loop/drain_lease_test.go:102 — "func TestALaneWhoseReservationIsAnothersPushesNothingAndIsHandedBack(t *testing.T) {"
- ac-13 — MET_WITH_CONCERNS: Each fresh reservation act, reserve's and the one a drain makes before its first judgement, asks the forge whether each github.com push URL of origin is a fork after its fetch and before its push, and refuses a fork naming its parent and the git remote set-url command; a new drain refused there writes no state. Concern: the drain's start itself asks nothing (spec step 10), so a drain with nothing to reserve does not refuse; a forge that does not answer warns and lets the reservation onto the fork (dcn-2610161040378209); reservation.fork_origin can allow one named fork; and only claim's tests show the refusal, no drain-level test.
  evidence: internal/core/claim/remote.go:208 — "if !p.req.Release && (res.Act != ActRenewed || p.req.Drain && !res.RenewedLive) {"
  evidence: internal/core/claim/fork.go:181 — "a fork of %s: a reservation there is invisible to everyone working on %s, so it holds nothing for them"
  evidence: internal/core/claim/fork.go:156 — "if a.err != nil {"
  evidence: internal/core/implement/loop/reservation.go:593 — "res, err := claim.Reserve(repoRoot, claim.Request{Record: id, Footprint: footprint, TakeOver: takeOver, Drain: true, Stage: stage}, claimOptions(now))"
  evidence: internal/core/claim/fork_test.go:59 — "func TestAForkOriginRefusesAFreshReservationNamingItsParent(t *testing.T) {"
- ac-14 — MET_WITH_CONCERNS: unreadablePeers runs once per move, after the housekeeping and before any issue is reserved or judged, and refuses with the drain's own peers check, not contention, naming each unreadable peer once with a remedy for its reason; the test with a broken git file sees the worktree named, `git worktree repair` as the remedy, no issue passed over, nothing reserved, no drain state, and the repair letting the next move judge. Concern: for an unreadable worktree whose reason is not a broken git file, a split ledger or another owner, the remedy gives no repair or removal command, only the `git status --ignored` check before anyone removes it.
  evidence: internal/core/implement/loop/drain.go:822 — "if err := unreadablePeers(repoRoot); err != nil {"
  evidence: internal/core/implement/loop/drain_unreadable.go:49 — "could not be read, so what each holds is unknown and every issue would read as held by it"
  evidence: internal/core/implement/loop/drain_unreadable.go:91 — "settle what its reason names; no command to remove it is offered"
  evidence: internal/core/implement/loop/drain_unreadable_test.go:62 — "func TestAnUnreadableWorktreeRefusesTheDrainMoveOnce(t *testing.T) {"
- ac-15 — MET: When every eligible issue left was passed over as held, the move ends in a held wait: waiting_until fifteen minutes on, naming the holds and the judgements kept, never the empty stop; the test hands Pat's lane back with Quinn holding the last issue and sees no stop, the wait time, reserved by quinn named and the judgement kept, the wait kept at a later move, and the issue judged once Quinn releases it.
  evidence: internal/core/implement/loop/drain.go:1132 — "return finish(waitMove(&st, &res, prevWait, DrainWaitHeld, heldWaitText(held, len(st.Judgements)), now, cooling))"
  evidence: internal/core/implement/loop/drain_wait.go:105 — "every eligible issue left is held by another (%s), so the drain does not end, and keeps its %d judgement(s)"
  evidence: internal/core/implement/loop/drain_wait_test.go:82 — "func TestADrainWithOnlyHeldWorkLeftWaitsAndKeepsItsJudgements(t *testing.T) {"
- ac-16 — MET: The empty stop is written only once the issue loop leaves nothing capped, held or cooling; the same test ends with the last issue handed back on a yes and the drain stopped as empty.
  evidence: internal/core/implement/loop/drain.go:1139 — "st.Stopped, st.EndedAt = DrainStoppedEmpty, &now"
  evidence: internal/core/implement/loop/drain_wait_test.go:134 — "if !done.Complete || done.Stopped != DrainStoppedEmpty {"
- ac-17 — MET_WITH_CONCERNS: overlap, shared by the reservation act's crowding and the drain's view-only Contention, drops a shared path that origin's default branch declares linguist-generated, and abcd declares its CLI reference and surface manifest; the claim tests reserve a declared file in two people's footprints and show Contention agreeing with crowding. Concern: only a plain root .gitattributes on origin's default branch is read, so a file at or below a nested .gitattributes, or declared beside a quoted pattern, an [attr] line, ident or working-tree-encoding, or a non-ASCII line, still counts as the overlap.
  evidence: internal/core/claim/crowd.go:128 — "if !ok || g.Is(shared) {"
  evidence: internal/core/claim/generated.go:18 — "A nested .gitattributes, .git/info/attributes and the"
  evidence: internal/core/claim/generated.go:97 — "abcd shares a generated file only on a declaration whose every line git"
  evidence: internal/core/claim/generated_test.go:51 — "func TestAGeneratedFileIsNoOverlap(t *testing.T) {"
  evidence: internal/core/claim/generated_test.go:198 — "func TestContentionAgreesWithCrowdingOnGeneratedFiles(t *testing.T) {"
  evidence: .gitattributes:58 — "docs/reference/cli/commands.md linguist-generated"
- ac-18 — MET: orderEligible sorts by category, then severity, then the file-scoped issues before the rest, then the spread order, with found_at classified against origin's commit; the test orders a minor bug group's two file issues ahead of its two directory issues whatever their age.
  evidence: internal/core/capture/eligible.go:290 — "if fa, fb := fileScoped[a.ID], fileScoped[b.ID]; fa != fb {"
  evidence: internal/core/implement/loop/drain_origin.go:200 — "func treeFiles(repoRoot string, v originView) func([]string) (map[string]bool, error) {"
  evidence: internal/core/capture/drainspread_test.go:69 — "func TestAGroupTakesTheIssuesNamingAFileFirst(t *testing.T) {"
- ac-19 — MET_WITH_CONCERNS: An issue lane's directory reservation is narrowed to the files its branch changed since its merge base with origin's default branch, within the footprint and never wider. Concern: the narrowing is taken at the first due renewal once the lane is at its landing with its validators passed, not at the next renewal after the implement stage (dcn-2610170919040486), so a renewal during validation keeps the whole directory, as the test asserts; a lane its landing sends back reserves its whole found_at again (dcn-2610170947246937); and the closed spec still says after the implement stage.
  evidence: internal/core/implement/loop/narrow.go:52 — "if l.Stage != StageLand || !gitutil.IsFullSHA(l.HeadSHA) {"
  evidence: internal/core/implement/loop/narrow_test.go:253 — "// While the validators judge the lane, a renewal keeps the directory."
  evidence: internal/core/implement/loop/narrow_test.go:238 — "func TestALanesDirectoryReservationNarrowsToItsChangedFiles(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-2610161044461448-drain-many-accounts.md:65 — "After the implement stage, the lane's next renewal narrows its footprint to the files its branch changed"
- ac-20 — MET: crowding lets an act that asks for the warning, the reserve verb's, go ahead over the same person's other copy and returns each shared path, which reserve prints as a warning; every other act, a drain's included, backs off it, and the drain's view passes it over with the take-over advice; the claim test makes the laptop's reserve with the overlap named, then has a drain's and a build's reservation over that copy back off, and the CLI test shows the warning in both renders.
  evidence: internal/core/claim/crowd.go:103 — "case mine && p.req.WarnOwnOverlap:"
  evidence: internal/core/claim/generated_test.go:163 — "func TestOnesOwnReserveOverlappingOnesOtherCopyWarns(t *testing.T) {"
  evidence: internal/surface/cli/reserve_ownoverlap_test.go:14 — "func TestReserveOverOnesOtherCopyWarnsInBothRenders(t *testing.T) {"
  evidence: internal/core/implement/loop/drain.go:946 — "it is your own reservation from another copy: to move it to this one, run `abcd drain --take-over`"

Gap audit:
- honoured:
  - Each drain plans from what origin's default branch holds now, never from a checkout that has fallen behind, so it never reserves an issue another drain already fixed
    evidence: internal/core/implement/loop/drain_origin.go:458 — "so the move plans nothing: it never plans from what this checkout last saw"
    evidence: internal/core/claim/remote.go:356 — "so there is nothing to reserve; take other work"
    evidence: internal/core/implement/loop/drain_origin_test.go:101 — "func TestABehindClonesMovePassesOverAnIssueResolvedOnOrigin(t *testing.T) {"
  - It skips an issue another account holds before spending a judgement on it
    evidence: internal/core/implement/loop/drain.go:941 — "if why := view.Contention(v.ID, footprint, now, takeOver); why != "" {"
    evidence: internal/core/implement/loop/drain_held_test.go:62 — "func TestAHeldIssueGetsNoPushAndNoJudgement(t *testing.T) {"
  - An issue one drain hands back stays handed back for every drain, recorded on origin
    evidence: internal/core/claim/handback.go:289 — "func (s Snapshot) HandBack(id string, now time.Time) string {"
    evidence: internal/core/implement/loop/drain_handback_test.go:74 — "func TestOneClonesHandBackIsPassedOverByAnotherClonesDrain(t *testing.T) {"
  - At the per-person reservation limit a drain waits for a pull request to merge instead of refusing
    evidence: internal/core/implement/loop/drain_wait.go:98 — "wait for one of their pull requests to merge, which ends its reservation, rather than releasing one"
    evidence: internal/core/implement/loop/drain_wait_test.go:26 — "func TestADrainAtTheCapWaitsAndNamesThePullRequests(t *testing.T) {"
  - A lane paused for a while keeps its reservation
    evidence: internal/core/implement/loop/drain.go:678 — "leaseNotes = renewInProgress(repoRoot, st, view, now)"
    evidence: internal/core/implement/loop/drain_lease_test.go:24 — "func TestAWaitingDrainMoveRenewsTheLaneInProgress(t *testing.T) {"
  - One unreadable worktree stops the move once with its name instead of passing everything over
    evidence: internal/core/implement/loop/drain_unreadable.go:49 — "could not be read, so what each holds is unknown and every issue would read as held by it"
    evidence: internal/core/implement/loop/drain_unreadable_test.go:62 — "func TestAnUnreadableWorktreeRefusesTheDrainMoveOnce(t *testing.T) {"
  - A drain stops as empty only when nothing is left for anyone
    evidence: internal/core/implement/loop/drain.go:1139 — "st.Stopped, st.EndedAt = DrainStoppedEmpty, &now"
    evidence: internal/core/implement/loop/drain_wait_test.go:82 — "func TestADrainWithOnlyHeldWorkLeftWaitsAndKeepsItsJudgements(t *testing.T) {"
  - An issue scoped to a whole directory is taken after the issues naming a file
    evidence: internal/core/capture/eligible.go:290 — "if fa, fb := fileScoped[a.ID], fileScoped[b.ID]; fa != fb {"
    evidence: internal/core/capture/drainspread_test.go:69 — "func TestAGroupTakesTheIssuesNamingAFileFirst(t *testing.T) {"
- diverged:
  - Each account starts at a different point in the queue: only within each group of one category and severity, by a rendezvous hash of holder, copy key and issue id; the group order is shared, so every account starts in the same group, and two accounts share a group's first issue by chance
    evidence: internal/core/capture/eligible.go:293 — "if spread != "" {"
    evidence: internal/core/capture/drainspread_test.go:103 — "the nitpick group comes first for every account: %v"
  - A drain whose origin is a fork refuses rather than reserving where nobody looks: it refuses at each fresh reservation rather than at the drain's start, a forge that does not answer lets the reservation onto the fork with a warning, reservation.fork_origin can allow one named fork, and an origin off github.com is not asked
    evidence: internal/core/claim/fork.go:156 — "if a.err != nil {"
    evidence: internal/core/claim/fork.go:172 — "if strings.EqualFold(allow, fork) {"
    evidence: internal/core/claim/remote.go:208 — "if !p.req.Release && (res.Act != ActRenewed || p.req.Drain && !res.RenewedLive) {"
  - A generated file never locks another account out: only a file a plain root .gitattributes on origin's default branch declares linguist-generated, with no nested .gitattributes at or above it, counts as shared, and abcd's own declaration leaves out the brief's surfaces page the spec named
    evidence: internal/core/claim/generated.go:18 — "A nested .gitattributes, .git/info/attributes and the"
    evidence: internal/core/claim/generated.go:97 — "abcd shares a generated file only on a declaration whose every line git"
    evidence: .gitattributes:56 — "# The brief's surface chapters are not named: each mixes hand-written prose with"
    evidence: .abcd/development/specs/closed/spc-2610161044461448-drain-many-accounts.md:63 — "abcd's own declaration names the generated CLI reference, the surface manifest and the brief's surfaces page"
  - A directory-scoped issue narrows to the files its lane changed: only at the first due renewal once the lane is at its landing with its validators passed, and back to its whole found_at if the landing sends it to a fresh round
    evidence: internal/core/implement/loop/narrow.go:52 — "if l.Stage != StageLand || !gitutil.IsFullSHA(l.HeadSHA) {"
    evidence: internal/core/implement/loop/narrow_test.go:253 — "// While the validators judge the lane, a renewal keeps the directory."
- missing:
  - Several people's drains on one repository share the work rather than repeat it, shown end to end: the spec made a rerun of the 2026-10-16 three-account lab on the built binary, every account on one build, the evidence before close, and no record of that rerun exists in the history, the close commit or this machine's lab store; the claim rests on per-step tests with two or three clones of one local bare origin
    evidence: .abcd/development/specs/closed/spc-2610161044461448-drain-many-accounts.md:75 — "Before this spec closes, the three-account lab of 2026-10-16 is run again on the built binary"
    evidence: 80764362b — "chore: close the drain-many-accounts spec and ship its three intents"

Scope-condition dispositions:
- cond-2610161044460239 — survived: Holders are GitHub usernames, the cap counts one username across every copy, one person's copies are told apart by copy key (a drain backs off the other copy, a person's reserve warns), and the fork check keeps reservations on the repository the pull requests target; nothing delivered assumes less.
  evidence: internal/core/claim/crowd.go:15 — "username across every record in the view: a laptop and a cloud session that"
  evidence: internal/core/implement/loop/drain.go:945 — "claim.SameHolder(r.Holder, capHolder)"
  evidence: internal/core/claim/fork.go:181 — "a fork of %s: a reservation there is invisible to everyone working on %s, so it holds nothing for them"
- cond-2610161044469355 — survived: Every move fetches inside the drain's lock; a failed fetch, a default-branch gap or a missing origin/HEAD records an origin wait on the outage schedule and the move returns before any judgement, lane, renewal or reservation; the test covers all three kinds and the recovery; only the dry run, which is not a move, reads the view as last fetched, and says how old it is.
  evidence: internal/core/implement/loop/drain_origin.go:458 — "so the move plans nothing: it never plans from what this checkout last saw"
  evidence: internal/core/implement/loop/drain.go:545 — "if wait != "" {"
  evidence: internal/core/implement/loop/drain_origin_test.go:195 — "func TestADrainMoveThatCannotReadOriginWaits(t *testing.T) {"
- cond-2610161044467946 — survived: A hand-back rides as two optional trailers on an ordinary release tombstone, read by this build's validateHandBack, and the reservation reader before this bundle ignores any key it does not know, so an older build reads the tombstone as a plain release it may reserve over, as the condition says; nothing delivered checks the build another account runs.
  evidence: internal/core/claim/handback.go:217 — "func validateHandBack(m *message, t trailers) string {"
  evidence: internal/core/claim/message.go:27 — "= "Abcd-Handed-Back""
  evidence: ca99ea28b:internal/core/claim/message.go:286 — "does not know, and any text outside the trailers, is ignored."
- cond-2610161044469265 — untested: Every multi-account test runs one local bare origin with two or three clones, and the three-account lab rerun the spec named as proof left no record; nothing measures the per-move reservation fetch or the race between two reservation pushes at about ten accounts.
  evidence: .abcd/development/specs/closed/spc-2610161044461448-drain-many-accounts.md:75 — "Before this spec closes, the three-account lab of 2026-10-16 is run again on the built binary"
- cond-2610161044463540 — narrowed: A record whose found_at names no path is handed back before any judgement with nothing reserved, but one whose found_at names a path origin's default branch holds as neither a file nor a directory (deleted since) is not handed back: it sorts with the directory-scoped issues, is reserved with that path and judged.
  narrowing: The hand-back before judgement holds only for a found_at that names no path at all; a found_at naming a path origin's default branch no longer holds is taken as if it named a directory, after its group's file-scoped issues, reserved with that path and judged.
  evidence: internal/core/implement/loop/drain.go:915 — "if footprint, ok = issueFootprint(iss); !ok {"
  evidence: internal/core/implement/loop/drain_reserve_test.go:105 — "func TestADrainHandsBackAnIssueWithNoFoundAtAndMovesOn(t *testing.T) {"
  evidence: internal/core/capture/eligible.go:317 — "naming a path that is not there (deleted since, or never present): each"
  evidence: internal/core/capture/drainspread_test.go:133 — "func TestAPathAbsentWhereTheLedgerWasReadIsNotFileScoped(t *testing.T) {"
<!-- abcd-review-end receipt=rcp-288b7637ec71 -->

## Decisions

- 2026-10-16, facilitator-on-delegation (dcn-2610161040352440): the three drafts are one bundle, drain-many-accounts; this one is critical, since it blocks the first run on several accounts.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040350256): each move fetches origin inside the lock and reads one snapshot of the ledger from origin's default branch, for the plan, the start check, the digest and the footprint; a failed fetch, an unread default branch or a missing origin/HEAD is an outage wait; the dry run reads the last fetched view and writes nothing; any reservation of a finished record backs off.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040368890): a drain reserves before it judges; held issues and footprints are passed over with no push; within a group, file-scoped issues come first, each part in a rendezvous-hash order of holder, copy and issue.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040364312): at the cap, counted before any judgement, or with only held work left, the drain waits fifteen minutes on a clock apart from the pace window and keeps its judgements; it stops as empty only when nothing eligible is left for anyone.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040365515): a hand-back is the release of the drain's reservation, carrying a closed-vocabulary kind and the record's blob id; drains back off it while the record is unchanged, a person's own reserve passes; an issue naming an intent is ineligible (brief invariant 19); every account runs one build.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040374343): every drain move renews the lane in progress before its window gate; the land stage checks the reservation before its push and hands the lane back when the hold is another's.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040378209): a fresh reservation and a drain start ask the forge once per process whether origin is a fork, and refuse one; renewals never ask, and no answer warns and proceeds.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040378713): an unreadable sibling worktree refuses the move once, before any reservation or judgement.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040385152): generated files are declared with the forge's linguist-generated attribute and never count as an overlap; a person's own overlapping reserve warns; drain reservations keep the back-off between one person's copies; a lane's reservation narrows to the files it changed.
- 2026-10-16, facilitator-on-delegation (dcn-2610161044187673): the pre-pass's questions are answered. Brief invariant 7 is amended for each new drain network act in the step that adds it, invariant 19 gains the clause on issues naming an intent, a published hand-back is a judgement kept on the reservation and not a classification written onto the issue, and each sibling overlap is kept both.
- 2026-10-16, facilitator-on-delegation (dcn-2610161412287479): a hand-back tombstone names one of seven kinds, the drain's four, unachievable, stuck and closed, and a tip carrying anything else on those trailers is unreadable.
- 2026-10-16, facilitator-on-delegation (dcn-2610161412283450): a hand-back stands while origin's default branch holds the record at the blob its tombstone names; only a blob read there that differs lifts it, and a record that branch does not hold in one file gets a plain tombstone, saying why.
- 2026-10-16, facilitator-on-delegation (dcn-2610161412284018): every issue lane's hand-back is published, `abcd build`'s included; a drain passes over a standing one from its view before any push, its reservation backing off one the view missed; a checkout holds a record as origin does when git's object name for its file equals origin's blob.

## Grounds

- pursued: facilitator-on-delegation, 2026-10-16: the product thinker wants drains on several GitHub accounts at once to add up to more fixes, starting the next day; we expect that reading the ledger, the holds and the hand-backs from origin before any judgement stops accounts repeating each other's work. Shown wrong if, in the first week of drains on two or more accounts, two accounts open lanes for one issue, or a drain reserves an issue already resolved on origin's default branch.
