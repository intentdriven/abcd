---
id: spc-2610161044461448
slug: drain-many-accounts
intent: itd-2610161016394967
intents: [itd-2610161016394967, itd-2610161016391855, itd-2610161016401115]
bundle: drain-many-accounts
origin: researcher-authored
production_mode: hand-written
---
# drain-many-accounts

## Summary

This spec delivers the drain-many-accounts bundle:

- [itd-2610161016394967](../../intents/shipped/itd-2610161016394967-several-people-each-on-their-own-github.md): drains on several accounts share the work (critical);
- [itd-2610161016391855](../../intents/shipped/itd-2610161016391855-a-drain-lane-that-cannot-finish-no.md): a stuck drain lane hands itself back (major);
- [itd-2610161016401115](../../intents/shipped/itd-2610161016401115-a-drain-runs-unattended-on-a-hosted.md): drains run unattended and visibly (minor).

The first two block a drain run on several GitHub accounts at once, and the third can follow. The members ship together when this spec closes. Steps 1 to 13 can be used with a host-answered judgement before steps 14 to 18 land. Every planning decision was made facilitator-on-delegation and is recorded in the decision store; each member's `## Decisions` names the records it rests on.

**Approach.** Everything a drain chooses by is read from origin, the one place every account shares: the ledger as origin's default branch holds it, the reservations, and the hand-backs. It is read once per move, inside the drain's lock, before any reservation or judgement. A drain reserves an issue before it judges it, so the reservation push is the compare-and-swap between accounts. A lane that cannot finish is read from evidence that changes, the refusals in the run's state and the pull request's state on the forge, and is handed back after a bound.

New state on the remote rides on the reservation family the claim package already owns: a hand-back kind and a lane stage are closed-vocabulary trailers on the reservation's own commits. They use the trailer collector the session-announcement work generalised (spc-2610150728396006 step 1, on the default branch). No second primitive is added ([`one-canonical-primitive`](../../principles/one-canonical-primitive.md)), and no free text reaches the remote (dcn-2610150000000001).

**When abcd touches the network.** Brief invariant 7 names what `drain` and `implement step` fetch and push. Each step that adds a network act amends that sentence in the same change ([`spec-moves-with-the-surface`](../../principles/spec-moves-with-the-surface.md)):

- step 1: one fetch of the reservations and origin's default branch at the start of every move; `--dry-run` still fetches nothing and reads the view as last fetched, with its age;
- step 2: the reservation push before a judgement, in place of the reservation at the lane's start;
- step 3: the hand-back tombstone, a release push carrying its kind;
- step 7: the renewal of the lane in progress on every move, and the reservation check before the land stage's push;
- step 8: one pull-request listing per move for the parked lanes;
- step 10: on each act that reserves afresh (a `reserve`, a `build` start, each `drain` lane's own reservation), after its fetch and before its push, one forge question for each of origin's push URLs on github.com, whether it is a fork; an answer is kept for the process and a failed question is asked again after a minute;
- step 13: the sync step's fetch of origin's default branch, at each sync;
- step 14: the routed judge, which runs on the person's runner and is not abcd's network act.

Steps 4, 11, 12 and 15 to 18 read only what the step 1 fetch brought, and change no network act. Step 4 adds one clause to invariant 19: an issue whose `related_intents` names an intent is handed back by the rule `promoted`, so a machine never takes alone an issue an intent already owns. `TestAbcdsOwnDrainRuleIsTheStrictBaseline` is updated with it.

**How each criterion is met.** Each criterion names its issue; the step that resolves that issue meets it.

- **Planning from origin (iss-2610160957404827, iss-2610161015448977).** The fetch moves inside `drainMove`, ahead of `capture.PlanDrain`, which runs before the lock today. One ledger snapshot is read from origin's default-branch tree, as the lane brief reads its base (`baseTree` in `internal/core/implement/loop/issuebrief.go`), and classified by the same `eligibility` function. The same snapshot feeds:
  - `issueCheck`, through `Options`, so an issue new on origin is not refused as absent here;
  - the remedy digest in `requestJudgement` and `takeJudgement`;
  - `runFootprint`;
  - the promote route.

  A promotion this checkout cannot make, because it does not hold the record as origin does, publishes the tombstone alone and says so. A failed fetch, a non-empty default-branch `gap`, or a missing `origin/HEAD` cannot plan. Each is recorded as an origin outage probe, and the move waits as the outage gate waits; for a missing `origin/HEAD` the move also names `git remote set-head origin --auto`. In `claim.decide`, any reservation act other than a release or a renewal backs off when the record is finished on origin's default branch (`j.derived`), with or without a tip, and `derived` is computed for a missing tip too. It is a back-off, not a refusal, so the drain passes over the issue rather than stopping its move (`drain.go`, the `StageReservation` row). The dry run warns how many commits the checkout is behind (dcn-2610161040350256).
- **Reserving before judging (iss-2610161015387318).** `Start`'s reservation is split out, and the drain reserves the next issue before `requestJudgement`. A "yes" releases that reservation as a hand-back; a "no" opens the lane on the reservation already held. Before any push, the move's view is consulted: an issue held by another copy, or a footprint overlapping a held one, is passed over naming the holder, with no push.

  Within each group of one category and one severity, file-scoped issues come first. Each part is then ordered by a rendezvous hash of the holder, the copy key and the issue id (dcn-2610161040368890). The plan's view is fetched once per move, and passing over held issues adds no fetch. A reservation push keeps its own fetch, which is its compare-and-swap.
- **Hand-backs on origin (iss-2610161015370582).** The release of a handed-back issue carries `Abcd-Handed-Back: <kind>` and `Abcd-Record: <blob id of the record on origin's default branch>`. The kinds are the drain's hand-back kinds plus `stuck` and `closed`. A reservation a drain makes (a `Request` flag the drain sets) backs off such a tip while the blob is unchanged. A person's own `abcd reserve` or `abcd build` is let through, with a note naming the hand-back (dcn-2610161040365515).
- **Stuck lanes (iss-2610161015368630).** Each refused step is logged in the run's record with its stage, its check and a count. Refusals that are waits by design never count: those carrying `checkWait` (the preflight-receipt wait) and the outage waits. `laneOutcome` reads a lane as stuck after three identical refusals, or sixty minutes refusing at one stage; both bounds are read from the pace's layers. A refusal that cannot change makes the lane stuck at once: the record finished at the base or on the default branch, or the land fence finding the hold another's. A stuck lane is routed as a hand-back of kind `stuck`, and its reservation released with the tombstone. `implement step --discard` accepts it (dcn-2610161040382067).
- **The cap and held-only waits (iss-2610161015371794, iss-2610161015422836).** The holder's live reservations are counted in the move's view before any reservation or judgement. `capRefusal` returns a typed cap error whose advice says to wait for a pull request to merge.

  At the cap, or when every eligible issue left is held, the drain writes `waiting_until` fifteen minutes on. That field is kept apart from `next_eligible_at`, so the pace window is not reset. The drain names the pull requests or holds it waits on, and keeps its judgements. `DrainStoppedEmpty` is written only when no eligible issue is left, held or not (dcn-2610161040364312).
- **Leases mid-lane (iss-2610161015393523).** `drainMove` renews the lane in progress through `renewReservation`, ahead of its window gate, so a waiting move renews too. `landPush` checks the reservation as `landFence` does before it pushes. When the hold is another's, the lane is handed back at once (dcn-2610161040374343).
- **Parked lanes (iss-2610161015381750).** `finishMerged` reads every parked lane's pull request in one `gh pr list` call: number, state, mergeability, and merge-queue entry. The outcomes:
  - a closed pull request releases the lane's reservation and hands the lane back as `closed`;
  - a conflicting one is no longer renewed, and is named in `next`; a conflict confined to generated files goes to step 13's sync instead;
  - one still open and mergeable is renewed and named as waiting (dcn-2610161040380948).
- **Unreadable worktrees (iss-2610161015397586).** `peers.Scan`'s unjudged peers are checked once per move, where per-issue contention begins: after the move's housekeeping (a pending judgement, the outage stop, merged lanes, the lane in progress), which is written first. Any present refuses the move before an issue is judged or reserved, naming each with the repair its reason calls for: a plain `git worktree repair` for a broken git file, the ledger's own move-or-remove advice for one id in two folders, the account that owns a worktree another account owns, and otherwise no removal at all: removing a worktree deletes its ignored files without asking, `.abcd/.work.local/` among them, so the remedy gives the `git status --ignored` check that lists them and leaves removal to whoever owns that work. A worktree whose owner cannot be read gets no command. No remedy advises `--force`. A start whose own peers check meets one stops the move the same way (dcn-2610161040378713).
- **Fork origins (iss-2610161015400509).** An act the judgement finds a fresh reservation (a reserve, a `build` start, the reservation a `drain` takes on an issue before asking its judgement, which the lane a no opens then starts on, not the drain's start) asks the forge whether each of origin's github.com push URLs is a fork, after its fetch and before its push. An answer is kept for the process; a failed question is asked again by a later act after a minute, at most once a minute for a repository. A fork refuses, naming the repository it was made from and the command that points that push URL there, unless `reservation.fork_origin` names that fork as `"<owner>/<name>"` (dcn-2610161141338923, dcn-2610161223212218), which the output then says it waived; a fork of the allowed fork is refused naming both. Renewals, a plain reserve that renews included, and releases never ask; an origin that is not on github.com is not asked, and one that names a host and an owner and a name on a host not github.com by name warns that it was not checked. A forge that does not answer gives a warning and lets the act through, in the drain's run record and output as in reserve's (dcn-2610161040378209). Tests stub the forge's answer; the bare-origin tests stay offline.
- **Directory footprints and generated files (iss-2610152125101468).** Generated files are declared with `linguist-generated` in `.gitattributes`, read from origin's default branch. abcd's own declaration names the generated CLI reference, the surface manifest and the brief's surfaces page. `crowding` never counts a declared generated file as an overlap.

  A person's own `abcd reserve` that overlaps another of their copies warns instead of refusing. A reservation a drain makes keeps that back-off. After the implement stage, the lane's next renewal narrows its footprint to the files its branch changed (dcn-2610161040385152). The file-before-directory order is part of step 2.
- **Merging the default branch (iss-2610161015410527).** `syncLane` merges only the run's landed sibling lanes today. It also merges origin's default branch whenever that has moved, before the records and the push, and the merge head is validated as a sibling merge's is. The land stage enters the sync again from arming, and from the merge wait, when the open pull request starts to conflict.

  A conflict confined to declared generated files goes back to the implementer as a fix round, under the host's permissions, with the validators judging the result; abcd itself runs no command the repository declares. Any other conflict hands the lane back, naming the files (dcn-2610161040384068).
- **The routed judge and take-over (iss-2610161015422306).** A new role, `drain-judge`, is routed through `runner.Dispatcher`, as the build roles are. With a route, the drain sends the judgement request and takes the answer, and abcd computes the remedy digest itself. With no route, today's host request stands (dcn-2610161040390001). `abcd drain --take-over` passes `Options.TakeOver` to the drain's reservations, and the advice for this person's other copy names it (dcn-2610161040398791).
- **Pace, sub-agents and re-admission (iss-2610161015417715, iss-2610161015457468).** `drainMove` passes the stored pace and sub-agent ceiling into every lane start. A running drain takes `--pace` and `--sub-agents` instead of refusing them. The new values are written to the drain's state, and applied from the next lane it opens and the next stage of the lane in progress.

  A handed-back issue whose record blob on origin has changed is eligible again. A handed-back run that has finished no longer passes its issue over (dcn-2610161040399117).
- **Holder and visibility (iss-2610161015433495, iss-2610161015434746).** `ResolveHolder` caches its forge answer for the process. Each reservation renewal a drain lane makes carries `Abcd-Stage: <stage>`, from the loop's stage names. `abcd peers` lists each live reservation's stage, and each holder's hand-back tombstones from the last day with their kind (dcn-2610161040394970).

**Proof.** Each step's tests run against a local bare repository standing in for origin, with clones standing in for accounts, as the claim package's tests do. Fakes are used only for the forge's answers. Before this spec closes, the three-account lab of 2026-10-16 is run again on the built binary, every account on the same build. It is the evidence that no issue is worked twice and that no drain stalls.

**Coordination.**

- The session-announcement spec (spc-2610150728396006) has steps 2 to 4 in flight, held by a reservation whose footprint covers `internal/core/claim`, `internal/core/implement/loop` and the brief's invariants page. Each step here is reserved with its footprint before it starts.
- Most steps change `internal/core/implement/loop`, so the steps land one after another, in this order.
- Steps 10, 11 and 17 change only `internal/core/claim` and its forge client, and can be built beside the others.

**Out of scope.**

- `abcd peers` keeps its name and its other listings (iss-2610150726331622).
- The people's status view of itd-2609151838312703 is not built here. It inherits the stage and hand-back rows when it takes the reservation listing over.

## Footprint

- packages: internal/core/implement/loop, internal/core/claim, internal/core/capture, internal/core/peers, internal/core/github, internal/core/runner, internal/surface/cli, commands, docs, .abcd/development/brief, .gitattributes
- tests: internal/core/implement/loop (drain move, judgement, sync, land and finish tests against a local bare origin with several clones), internal/core/claim (reserve, decide, crowding, message, holder and fork tests), internal/core/capture (eligibility tests), internal/surface/cli (drain and peers surface tests), the three-account drain lab rerun on the built binary

## Steps

1. Plan every move from one snapshot of origin's default branch, wait out a failed fetch, and back off any reservation of a finished record (itd-2610161016394967, itd-2610161016401115; resolves iss-2610160957404827, iss-2610161015448977)
   - packages: internal/core/implement/loop, internal/core/capture, internal/core/claim, internal/surface/cli, commands, .abcd/development/brief
   - tests: loop: a behind clone's move neither reserves, judges nor opens a lane for an issue another clone resolved on origin; an issue open on origin and absent from the checkout is judged and opened from origin's record; a failed fetch, a default-branch gap and a missing origin/HEAD each record an outage probe and plan nothing; the dry run names the finished issue, its view's age and the commits behind, and writes nothing; claim: a reservation of a record resolved on origin's default branch backs off with no tip, with a live tip and with a tombstone, and when a lost race re-judges it
2. Reserve before judging, pass over held issues, and order each group per account (itd-2610161016394967; resolves iss-2610161015387318)
   - packages: internal/core/implement/loop, internal/core/capture, .abcd/development/brief
   - tests: loop: an issue another holder reserved in the move's view gets no push and no judgement request and is passed naming the holder; a judgement "yes" releases the reservation taken before it, and a "no" opens the lane on it; passing over held issues adds no fetch; capture or loop: within a group file-scoped issues come first, and two holders' rendezvous orders differ on a ten-issue group while both take a more urgent group first
3. Publish hand-backs on origin as tombstones that drains back off (itd-2610161016394967; part of iss-2610161015370582)
   - packages: internal/core/claim, internal/core/implement/loop, .abcd/development/brief
   - tests: claim: a released tip carrying each hand-back kind, stuck and closed included, and a record blob round-trips, and free text in it is refused; a drain's reservation backs off it while the blob is unchanged and succeeds once it changes; a person's own reserve passes with a note; loop: one clone's hand-back is passed over by another clone's drain with no judgement request; a promotion the checkout cannot make publishes the tombstone alone
4. Make issues naming an intent ineligible (itd-2610161016394967; resolves iss-2610161015370582)
   - packages: internal/core/capture, .abcd/development/brief
   - tests: capture: an issue naming an intent in related_intents is handed back by the rule promoted, and the dry run names the intent; TestAbcdsOwnDrainRuleIsTheStrictBaseline holds the amended invariant 19 clause
5. Hand a stuck lane back after its bound (itd-2610161016391855; resolves iss-2610161015368630)
   - packages: internal/core/implement/loop, internal/surface/cli, commands
   - tests: loop: three identical refusals, and sixty minutes refusing at one stage, each hand the lane back as stuck, release it with its tombstone and open the next lane in the same move; a ninety-minute preflight-receipt wait and an outage wait are not stuck; a brief refused for an issue finished at its base hands back at once; implement step --discard accepts a stuck lane
6. Wait at the cap and when only held work is left (itd-2610161016394967; resolves iss-2610161015371794, iss-2610161015422836)
   - packages: internal/core/claim, internal/core/implement/loop
   - tests: claim: the cap refusal is typed and its advice names waiting for a merge; loop: at ten live reservations the move asks no judgement, writes waiting_until fifteen minutes on without resetting the pace window, and names the pull requests; with only held issues left the drain is not complete and keeps its judgements; with none left, held or not, it stops as empty
7. Renew the lane in progress every move, and check the reservation before the push (itd-2610161016394967; resolves iss-2610161015393523)
   - packages: internal/core/implement/loop, .abcd/development/brief
   - tests: loop: a waiting drain move, with the lane past half its lease and no step run, pushes the lane's renewal; a lane whose reservation moved to another holder refuses its push, pushes nothing, opens no pull request and is handed back
8. Read parked lanes from their pull requests (itd-2610161016391855; resolves iss-2610161015381750)
   - packages: internal/core/implement/loop, .abcd/development/brief
   - tests: loop: every parked lane is read in one forge listing; a closed pull request releases and hands back its lane; a conflicting one is not renewed and is named; an open lane not yet on the default branch is renewed and named as waiting
9. Refuse the move once on an unreadable worktree (itd-2610161016394967; resolves iss-2610161015397586)
   - packages: internal/core/implement/loop
   - tests: loop: a sibling worktree whose git file points nowhere refuses the move before any reservation or judgement, naming it, and no issue is passed as contended
10. Refuse a fork origin (itd-2610161016394967; resolves iss-2610161015400509)
   - packages: internal/core/claim, internal/core/github, internal/core/implement/loop, internal/surface/cli, .abcd/development/brief
   - tests: claim: a forge answer naming a fork refuses a fresh reservation, naming the parent and the command; a renewal, a plain reserve that renews, and a release ask nothing; an origin off github.com is not asked, a host not github.com by name warns; every push URL is asked, the push URL rather than the fetch URL; no answer warns and lets the act through; an answer is kept for the process and a failure for a minute; `reservation.fork_origin` naming the fork lets it through with a warning, a fork of the allowed fork is refused, and a malformed value refuses once a fork is named; loop: a drain lane started while the forge fails names the skipped check, and a later act asks again; a drain lane on the allowed fork names the waived check
11. Share declared generated files, and warn on one person's own overlapping reserve (itd-2610161016394967; part of iss-2610152125101468)
   - packages: internal/core/claim, .gitattributes
   - tests: claim: a linguist-generated path in two footprints is no overlap; a person's own reserve overlapping their other copy warns and succeeds; a drain's reservation overlapping that copy still backs off
12. Narrow a lane's reservation to the files it changed (itd-2610161016394967; resolves iss-2610152125101468)
   - packages: internal/core/implement/loop
   - tests: loop: a lane that reserved a directory has its footprint narrowed to its branch's changed files at the first renewal after the implement stage, and a narrowed footprint never names a path outside the first
13. Merge origin's default branch in the lane's sync, and sync again when a waiting pull request conflicts (itd-2610161016391855; resolves iss-2610161015410527)
   - packages: internal/core/implement/loop, docs, .abcd/development/brief
   - tests: loop: a lane whose default branch moved merges it in sync and the merge head is validated; a conflict confined to a declared generated file goes to the implementer as a fix round; a conflict on an undeclared file hands back naming it and pushes nothing; a pull request that starts to conflict while armed or waiting re-enters sync rather than being handed back
14. Route the drain's judge through the runner (itd-2610161016401115; part of iss-2610161015422306)
   - packages: internal/core/implement/loop, internal/core/runner, commands, docs
   - tests: loop: a routed drain-judge's answer is taken with abcd computing the digest and no --judgement given; with no route the host request is written as today
15. Accept --take-over on drain (itd-2610161016401115; resolves iss-2610161015422306)
   - packages: internal/core/implement/loop, internal/surface/cli, commands
   - tests: loop: drain --take-over moves this holder's reservation from another copy; the advice for that case names the flag; cli: the flag is listed in help and the generated reference
   - note: the move's view passes over an issue this person's other copy holds (`Snapshot.Contention` in `internal/core/claim/crowd.go`) before any reservation act, so as it stands `--take-over` never reaches that copy's reservation; with the flag given, the pass-over lets this person's other copy through to the reservation act, which takes it over.
16. Carry pace and sub-agents into every lane, take new ones while running, and re-admit changed records (itd-2610161016401115; resolves iss-2610161015417715, iss-2610161015457468)
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: loop: the second and later lanes run on the drain's pace and ceiling; a running drain takes and records a new --pace and --sub-agents and applies them from the next lane and stage; a handed-back issue whose record blob changed on origin is judged again; a finished handed-back run no longer passes its issue over
17. Cache the holder for the process (itd-2610161016401115; resolves iss-2610161015433495)
   - packages: internal/core/claim
   - tests: claim: three reservation acts in one process ask the forge for the holder once
18. Show each drain lane's stage and recent hand-backs in abcd peers (itd-2610161016401115; resolves iss-2610161015434746)
   - packages: internal/core/claim, internal/core/implement/loop, internal/core/peers, internal/surface/cli, commands
   - tests: claim: a renewal carries a stage from the loop's names and an unknown stage is refused; cli: peers lists another holder's lane with its stage and that holder's hand-backs of the last day with their kinds
