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

<!-- abcd-review: OWED receipt=rcp-288b7637ec71 -->
Fidelity review OWED (receipt rcp-288b7637ec71).
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
