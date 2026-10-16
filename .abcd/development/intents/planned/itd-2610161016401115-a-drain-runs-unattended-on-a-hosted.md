---
id: itd-2610161016401115
slug: a-drain-runs-unattended-on-a-hosted
spec_id: spc-2610161044461448
kind: bundle-member
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610161016394967, itd-2610161016391855]
severity: minor
impact: fix
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2610161015422306, iss-2610161015417715, iss-2610161015457468, iss-2610161015434746, iss-2610161015448977, iss-2610161015433495]
bundle: drain-many-accounts
---

# Drains run unattended and visibly

## Press Release

> A drain runs unattended on a hosted session or a plain terminal for a day. Its judgement of each remedy routes through the same model routing the build roles use, so no person needs to write the answer by hand, and a hosted session that restarts takes its own reservations back with one flag rather than waiting for them to lapse. The pace and sub-agents a drain was started with reach every lane it opens and can be changed while it runs, an issue it handed back is looked at again once its record changes, a lost connection to origin waits like any other outage, and abcd peers shows each drain lane's stage so every person can see what the other accounts' drains are doing.

## Why This Matters

The product thinker wants drains to run on several accounts through a day without a person minding each one. A person on any of those accounts should be able to see what the others are doing. Today a drain needs a person or a host agent at every judgement: in a plain terminal someone writes the JSON answer, its sha256 digest included, by hand, and a hosted session that restarts loses its local tier, so its new copy reads its own reservations as another copy's, and the advice names a flag the drain does not take (iss-2610161015422306). The pace and sub-agents a drain was started with reach only its first lane (iss-2610161015417715). A drain that has run for a day cannot be re-paced, and keeps the issues it handed back excluded even after their records change (iss-2610161015457468). A lost connection to origin refuses the move instead of waiting (iss-2610161015448977). Each reservation act asks the forge who the person is again (iss-2610161015433495). And `abcd peers` lists each account's reservations but nothing of what a drain lane is doing, so one person cannot tell a working lane from a stuck one on someone else's account (iss-2610161015434746).

These matter less than the faults that stop a drain outright, which the bundle's other two members close. They decide whether a day's drain needs minding.

## Mechanism

_Facilitator-on-delegation, 2026-10-16: the product thinker delegated the planning decisions of this bundle; this claim is the facilitator's, grounded in the evidence of the issues it closes._

We expect a drain to run unattended for a day, because each step that today needs a person either goes through a route abcd already has or becomes a wait that resumes by itself. The judgement goes through the model routing the build roles already use, and abcd computes the digest itself. A lost connection becomes the outage wait the loop already has, and a restart is recovered with the take-over the build already offers. We expect it to be visible, because a lane's stage rides on the reservation every account already fetches. It would be shown wrong if a drain started in a plain terminal with a routed judge needs a person's answer before it ends, or if a person on one account cannot name the stage of another account's drain lane from `abcd peers`.

## Scope Conditions

- Holds where the person has routed the drain's judge to a runner on their own machine. With no route, the judgement goes to the host as it does today. <!-- cond: cond-2610161044461615 -->
- Holds for a hosted session whose restart keeps the clone's history on origin but not its local tier. Its own reservations are moved to the new copy with `--take-over`; work not pushed before the restart is lost. <!-- cond: cond-2610161044469300 -->
- Holds for the stage names the loop already has. A stage is shown as it was at the reservation's last renewal, so it can be up to half a lease old. <!-- cond: cond-2610161044467671 -->
- Holds while `abcd peers` lists the reservations. When the people's status view takes that listing over, the same stage and hand-back rows go with it. <!-- cond: cond-2610161044460285 -->

## Acceptance Criteria

- **Given** Frank, a DevOps engineer, has routed the drain's judge to a command-line model on Frank's machine, **when** Frank runs `abcd drain` in a plain terminal and it reaches an issue's judgement, **then** the judgement is asked of that model and its answer taken, with abcd computing the remedy's digest itself, and no person writes the answer. (iss-2610161015422306)
- **Given** Frank's hosted session restarted mid-drain and lost its local tier, **when** Frank runs `abcd drain --take-over`, **then** the drain moves Frank's own reservations from the lost copy to this one as it reaches their issues. (iss-2610161015422306)
- **Given** a drain whose reservation is held by the same person's other copy, **when** the drain refuses to take it, **then** the advice names `abcd drain --take-over`, a flag the drain accepts. (iss-2610161015422306)
- **Given** Maya started a drain with `--pace 120/30 --sub-agents 4`, **when** the drain opens its second and later lanes, **then** each lane runs on that pace and sub-agent ceiling, not on what configuration resolves. (iss-2610161015417715)
- **Given** Maya's drain is running, **when** Maya runs `abcd drain --pace 60/15 --sub-agents 2`, **then** the drain takes and records the new values instead of refusing them. (iss-2610161015457468)
- **Given** a running drain that has taken new values, **when** it opens its next lane, or its lane in progress reaches its next stage, **then** that lane runs on the new values. (iss-2610161015457468)
- **Given** an issue Maya's drain handed back whose record has since changed on origin's default branch, **when** the drain next moves, **then** the issue is eligible again and judged afresh, and an old handed-back run for it no longer passes it over. (iss-2610161015457468)
- **Given** origin does not answer the fetch at the start of a drain move, **when** the drain moves, **then** it enters the drain's outage wait and probes again later, as for any other lost connection, instead of refusing the move. (iss-2610161015448977)
- **Given** a person whose git email is not a GitHub no-reply address, **when** one drain move makes several reservation acts, **then** the forge is asked for the person's username once in that move. (iss-2610161015433495)
- **Given** Carol, a tech lead, on another account from Maya's, **when** Carol runs `abcd peers`, **then** each of Maya's drain lanes is listed with its stage as of its last renewal. (iss-2610161015434746)
- **Given** Maya's drain handed issues back in the last day, **when** Carol runs `abcd peers`, **then** those hand-backs are listed with their kind. (iss-2610161015434746)

## Decomposition

Run by hand under [`decompose-before-filing`](../../principles/decompose-before-filing.md) on 2026-10-16, facilitator-on-delegation (dcn-2610161040352440), and graded in the decomposition-calibration note.

| Part | Type | Home | Verdict |
| --- | --- | --- | --- |
| The drain's judgement routed to a command-line model, and a restart recovered with take-over (iss-2610161015422306) | capability | this intent, on the runner routing the build roles use | file here |
| Pace and sub-agents carried into every lane and changeable while the drain runs; handed-back issues re-admitted when their records change (iss-2610161015417715, iss-2610161015457468) | capability | this intent | file here |
| A failed origin fetch waits as an outage (iss-2610161015448977); the holder cached per process (iss-2610161015433495) | capability | this intent | file here |
| A drain lane's stage and recent hand-backs shown in abcd peers (iss-2610161015434746) | capability | this intent, riding on the reservation; the people's status view of itd-2609151838312703 inherits the rows when it takes the listing over (iss-2610150726331622) | keep both |

Verdict: FILE-AS-IS, as a member of the drain-many-accounts bundle. It builds on itd-2610161016394967 (the hand-back tombstones it re-admits from, and the per-move fetch its outage wait wraps) and itd-2610161016391855 (the stage a stuck lane reports).

## SOTA

Facilitator-on-delegation, 2026-10-16:

- **Routing a model-run role.** abcd's build roles already go through one runner routing, so a new role reuses it (path 1, adopt the native seam). A hosted model API used directly was not adopted, because the person's runner is the existing route.
- **Resuming on another machine.** CI systems and hosted agents keep resumable state on the shared remote or in a store the platform documents as surviving. abcd's take-over already moves a reservation between copies, so it is adopted (path 2). State kept outside the checkout was declined under `durable-state-lives-where-the-platform-says-it-survives`.
- **Showing what workers are doing.** Job dashboards read status from the record every worker writes. A trailer on the reservation renewal costs no new act, so it is adopted (path 2).

## Open Questions

_None open; the product thinker delegated this bundle's planning decisions on 2026-10-16, and each is recorded under Decisions. The technical questions the build meets are in the bundle's spec._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-16, facilitator-on-delegation (dcn-2610161040352440): the three drafts are one bundle, drain-many-accounts; this one is minor and builds on the other two.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040390001): the judgement is a drain-judge role on the runner routing; with no route it goes to the host as today; abcd computes the digest.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040398791): a restarted hosted drain recovers with `abcd drain --take-over`; abcd keeps no drain state outside the checkout. The press release sentence that said a restart keeps the drain state and copy key was narrowed to this on 2026-10-16.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040399117): pace and sub-agents reach every lane and can be changed while the drain runs; changed records are re-admitted.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040394970): the lane stage rides on the reservation renewal and shows in abcd peers with recent hand-backs; the holder is cached per process. The wait on a failed fetch is delivered with the bundle's first step, under dcn-2610161040350256.
- 2026-10-16, facilitator-on-delegation (dcn-2610161044187673): the pre-pass's questions are answered. Brief invariant 7 is amended for each new drain network act in the step that adds it, invariant 19 gains the clause on issues naming an intent, a published hand-back is a judgement kept on the reservation and not a classification written onto the issue, and each sibling overlap is kept both.

## Grounds

- pursued: facilitator-on-delegation, 2026-10-16: the product thinker wants drains on several accounts to run through a day without minding and to be seen by everyone; we expect routing the judgement, waiting out lost connections and showing lane stages on the reservation to remove the remaining reasons a person must step in. Shown wrong if a drain with a routed judge needs a person's answer before it ends, or a person cannot name another account's lane stage from abcd peers.
