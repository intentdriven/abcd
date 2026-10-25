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

<!-- abcd-review: INGESTED receipt=rcp-cf03d6bdefc1 -->
Fidelity review — receipt rcp-cf03d6bdefc1 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e5fc73c86bafbce8b71e1ec703f5997d44d1df7522d223d4e97d949841fa259d
Input attestations: diff:ca99ea28b..80764362b, git diff --binary from the spec's plan commit to its close commit on main@sha256:3dd8bdc15c06e682367717938ba552c6d0ff87892ad2ab67d827b858c61e5b91; commits:this intent's step commits and follow-ups: f87eea869, 7837234d7, d0bd60137, 77bf8def1, ffda8dd52, d7d49ff2f, 764f355de, 6add08b19, 4358e61c8, 1dd3e4a0d, 9d61af9ed, bab97a0ae, d8c42ece7, 058d35c81, 0efecf94c, aae1f72a3, 3e1fa4f06, 385972ec4, d13e39bed, 9f9976a5d, a26850419, 00e95902b, da43b25ba, a1c16e21f; post-close 730aca57b@-; tree:origin/main at 2e03c61e4, read in an audit worktree at eac09d94f, which adds only audit notes; file:line refs below are at that tree@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Routed to the claude runner in the person's own layer, the move that writes the host's request also starts the runner on its own brief, reads its digest-less answer, fills in the remedy digest abcd computed and takes the answer in that move with no --judgement given (tested against a fake claude harness). Concern: the delivery is narrower than 'a command-line model': a route to the opencode runner is skipped with a diagnostic and the judgement stays on the host, and the judge is never started where the checkout's or the temporary directory's real path holds a parenthesis or a comma (iss-2610171043574168, still open), so in those setups a person answers as before.
  evidence: internal/core/implement/loop/drain_judge.go:127 — "return o.Runners != nil && d.Transcripts != nil && o.Runners.RouteFor(RoleDrainJudge).Runner != runner.Host"
  evidence: internal/core/implement/loop/drain_judge.go:293 — "a := JudgementAnswer{SchemaVersion: ra.SchemaVersion, Issue: ra.Issue, RemedySHA256: j.RemedySHA256,"
  evidence: internal/core/implement/loop/drain.go:1020 — "if taken, clause, started, err = routedJudgement(repoRoot, iss, v, judging, plan, &st, o, d, now); err != nil {"
  evidence: internal/core/implement/loop/drain_judge_test.go:57 — "if j.RemedySHA256 != remedyDigest(issueRemedy) {"
  evidence: internal/surface/cli/drain.go:258 — "o := loop.Options{Roots: &roots, Runners: cfg, TakeOver: takeOver}"
  evidence: internal/core/runner/config.go:253 — "if role == roleDrainJudge && name == OpenCode {"
  evidence: .abcd/work/issues/open/iss-2610171043574168-the-routed-drain-judge-cannot-run-for-a.md:15 — "The routed drain judge cannot run for a checkout or temporary directory whose real path holds a parenthesis or a comma"
- ac-2 — MET_WITH_CONCERNS: With --take-over, the move's view lets this person's own other-copy reservation of the issue through and the reservation taken before the judgement takes it over for this copy, the lane opening on it (tested with a GitHub no-reply git identity). Concern, neither tested nor documented: the let-through is keyed on capHolder, the holder read with no network act, which is empty for a person whose username only the forge token can name (git email not a no-reply address and no machine-layer reservation.github_username), so for that person the view still passes every such issue over and nothing is taken over; and only the issue's own reservation is exempted, so a lost-copy reservation whose footprint overlaps another of that copy's live reservations is still passed over, in the view and again by claim's crowding, until one lapses.
  evidence: internal/core/implement/loop/drain.go:939 — "takeOver = capHolder"
  evidence: internal/core/implement/loop/drain.go:863 — "capHolder = claim.ReaderHolder(repoRoot, claimOptions(now))"
  evidence: internal/core/claim/render.go:168 — "h, err := resolveHolder(root, who, o, false)"
  evidence: internal/core/claim/holder.go:163 — "cannot tell which GitHub user you are without asking GitHub: user.email is not a GitHub no-reply address"
  evidence: internal/core/claim/crowd.go:257 — "!(takeOver != "" && r.State == StateLive && sameHolder(r.Holder, takeOver))"
  evidence: internal/core/claim/crowd.go:266 — "if r.State != StateLive || r.Record == id || r.Copy == CopyThis {"
  evidence: internal/core/implement/loop/drain_takeover_test.go:60 — "if !ok || r.Copy != claim.CopyThis || r.State != claim.StateLive || r.Holder != "octo" {"
  evidence: internal/core/implement/loop/reservation_test.go:26 — "users.noreply.github.com"
- ac-3 — MET_WITH_CONCERNS: A plain move that passes over an issue held by this person's other copy appends 'to move it to this one, run `abcd drain --take-over`', and the drain registers that flag, listed in its help and generated reference (tested). Concern: the advice is gated on the same offline holder, so a person only the forge token names is told only who holds the issue and nothing of the flag; a hold the reservation push itself meets is passed over in claim's own words, 'you are already building ... on another computer ... pass --take-over', not `abcd drain --take-over`; and under --take-over a pass-over for a footprint overlap repeats the advice although the flag was given.
  evidence: internal/core/implement/loop/drain.go:946 — "to move it to this one, run `abcd drain --take-over`"
  evidence: internal/core/implement/loop/drain.go:945 — "claim.SameHolder(r.Holder, capHolder)"
  evidence: internal/core/implement/loop/drain_takeover_test.go:41 — "if why := plain.Passed[i].Reason; !strings.Contains(why, "abcd drain --take-over") {"
  evidence: internal/surface/cli/drain.go:234 — "cmd.Flags().BoolVar(&takeOver, "take-over", false,"
  evidence: internal/core/claim/remote.go:403 — "you are already building %s on another computer, until %s; pass --take-over to continue here"
  evidence: internal/core/implement/loop/drain.go:994 — "res.Passed = append(res.Passed, Excluded{ID: v.ID, Check: r.Check, Reason: termsafe.Sanitize(r.Reason)})"
- ac-4 — MET: Every lane the drain starts is handed the drain's stored pace (lane.pace = st.Pace) with the flag fields cleared, and a lane's start takes o.pace instead of resolving configuration; the test begins a drain with --pace 120/30 --sub-agents 4 and finds its second lane, opened in a move given no flag, on 120/30 with four sub-agents.
  evidence: internal/core/implement/loop/drain.go:1046 — "lane.Pace, lane.SubAgents, lane.FixRounds, lane.pace = nil, nil, nil, st.Pace"
  evidence: internal/core/implement/loop/loop.go:376 — "pace = *o.pace"
  evidence: internal/core/implement/loop/drain_repace_test.go:49 — "if got := paceNumbers(p); got != [4]int{120, 30, 4, BundledFixRounds} {"
- ac-5 — MET: A move of a running drain given --pace or --sub-agents calls repaceDrain where it used to refuse through the resume check, sets the drain state's pace and writes it with the move; the test moves a running drain with --pace 60/15 --sub-agents 3 without error and reads the new numbers back from the drain's state.
  evidence: internal/core/implement/loop/drain.go:398 — "// A running drain takes a new pace rather than refusing it"
  evidence: internal/core/implement/loop/drain.go:401 — "if repace, repaced, err = repaceDrain(repoRoot, &st, o, flags); err != nil {"
  evidence: internal/core/implement/loop/drain_pace.go:61 — "st.Pace = &next"
  evidence: internal/core/implement/loop/drain_repace_test.go:63 — "moved, err := Drain(repo.Root(), Options{Pace: strp("60/15"), SubAgents: strp("3")}, DrainOptions{})"
  evidence: internal/core/implement/loop/drain_repace_test.go:74 — "if err != nil || st.Pace == nil || paceNumbers(*st.Pace) != [4]int{60, 15, 3, BundledFixRounds} {"
- ac-6 — MET: After the move's state is written, each lane in progress has its run's pace replaced and logged, and its next stage reads the run's pace for the sub-agent ceiling and the window, while the next lane starts on st.Pace; the test finds the lane in progress and the next lane both on the new numbers, and a refused move takes none.
  evidence: internal/core/implement/loop/drain_pace.go:83 — "this lane runs on it from its next stage"
  evidence: internal/core/implement/loop/drain.go:469 — "if err := repace.lanes(repoRoot, st, now); err != nil {"
  evidence: internal/core/implement/loop/schedule.go:64 — "return s.Pace.SubAgents.Value"
  evidence: internal/core/implement/loop/loop.go:818 — "end := st.WindowStartedAt.Add(time.Duration(st.Pace.WorkMinutes.Value) * time.Minute)"
  evidence: internal/core/implement/loop/drain_repace_test.go:78 — "the lane in progress takes the new pace for its next stage"
  evidence: internal/core/implement/loop/drain_repace_test.go:86 — "the next lane opens on the new pace"
- ac-7 — MET: Each hand-back route keeps the blob of the issue's record on origin's default branch, an issue whose blob there has changed is no longer counted as taken, and a run that was handed back no longer passes its issue over; the tests change the record on origin and see the issue judged afresh and a new run opened. The blob on the closed-pull-request route landed only after the close, in 730aca57b, which is on main.
  evidence: internal/core/implement/loop/drain_readmit.go:115 — "return r.Blob == "" || now == "" || now == r.Blob"
  evidence: internal/core/implement/loop/drain.go:893 — "return r.Key == v.ID && !r.issueHandedBack()"
  evidence: internal/core/implement/loop/drain_repace_test.go:119 — "once its record changed on origin the issue is judged again"
  evidence: internal/core/implement/loop/drain_repace_test.go:150 — "the finished run no longer passes the issue over"
  evidence: git:730aca57b — "fix: keep the judged record on a closed pull request's hand-back route"
- ac-8 — MET: A fetch that fails at a move's start records an origin probe in the drain's state on the outage gate's schedule (one, five and ten minutes, then hourly, giving up eight hours into the hourly stage), plans nothing, asks nothing and returns without a refusal; tests cover the failed fetch, the probe schedule, the recovery and the give-up.
  evidence: internal/core/implement/loop/drain_origin.go:427 — "kind, why = DrainOriginFetch, "could not fetch "+claim.Remote+": "+termsafe.Sanitize(ferr.Error())"
  evidence: internal/core/implement/loop/drain_origin.go:444 — "s := implement.DefaultSchedule"
  evidence: internal/core/implement/loop/drain.go:550 — "return finish(started || probed)"
  evidence: internal/core/implement/loop/drain_origin_test.go:222 — "a move that cannot read origin: %v; want it to wait"
  evidence: internal/core/implement/loop/drain_origin_test.go:390 — "func TestADrainWaitingOnOriginFollowsTheOutageScheduleAndGivesUp(t *testing.T) {"
- ac-9 — MET: The real user-endpoint call keeps a definite answer for the process, keyed on the checkout and a digest of the forge credential's environment and gh's account store, and every reservation act a drain move makes resolves against the same checkout root; the test makes a reserve, a renewal and a release as a person whose email is not a no-reply address and counts one gh call.
  evidence: internal/core/claim/holder.go:278 — "if a, ok := holderCache.get(k); ok {"
  evidence: internal/core/claim/holder.go:283 — "holderCache.put(k, forgeAnswer{login: login, id: id})"
  evidence: internal/core/claim/holdercache_test.go:89 — "func TestThreeReservationActsAskTheForgeForTheHolderOnce(t *testing.T) {"
  evidence: internal/core/claim/holdercache_test.go:106 — "if n := gh.calls(); n != 1 {"
- ac-10 — MET_WITH_CONCERNS: Every renewal the loop pushes for a lane (a drain move's, a lane step's, the land fence's) carries Abcd-Stage from the loop's own stage names, and abcd peers shows it beside a live reservation in text and JSON; the surface test lists another holder's lane at the implement stage. Concern: a lane opened on the reservation its judgement took, or on a take-over, names no stage until its first renewal, about half a lease after the hold was taken, so a new lane is listed with no stage (iss-2610170948520608, still open).
  evidence: internal/core/implement/loop/reservation.go:96 — "req := claim.Request{Record: st.Key, Renew: true, Lease: renewalTerm(*own), Stage: renewalStage(*st)}"
  evidence: internal/surface/cli/peers.go:438 — "if r.State == claim.StateLive && claim.ValidLaneStage(r.Stage) {"
  evidence: internal/surface/cli/peers.go:517 — "line += " stage " + r.Stage"
  evidence: internal/surface/cli/peers_stage_test.go:120 — "if stages["iss-200"] != "implement""
  evidence: .abcd/work/issues/open/iss-2610170948520608-in-a-drain-a-lane-opens-on-the.md:15 — "peers see none for the lane until its first renewal, about half a lease later."
  evidence: internal/core/claim/prepare.go:211 — "if req.Stage != "" && !req.Renew && (!req.Drain || req.TakeOver) {"
- ac-11 — MET_WITH_CONCERNS: abcd peers reads each holder's hand-back tombstones released in the last day from the fetched view and prints each with its record, kind and time, newest first; the surface test lists another holder's hand-backs with their kinds. Concern: only the newest ten per holder are listed with a kind and the rest only counted (dcn-2610170757251547), and only a hand-back published as a tombstone still at the issue's tip is listed, so one the drain keeps pending, one for an issue with no found_at, which reserves nothing, or one since reserved again does not appear.
  evidence: internal/core/claim/recenthandbacks.go:56 — "if r.State != StateReleased || r.ReleasedBy != ReleasedByTombstone || r.HandedBack == "" {"
  evidence: internal/surface/cli/peers.go:550 — "hb.Record, hb.Kind, hb.ReleasedAt)"
  evidence: internal/core/claim/recenthandbacks.go:24 — "const HandBacksShown = 10"
  evidence: internal/surface/cli/peers.go:553 — "and %d more"
  evidence: internal/surface/cli/peers_stage_test.go:132 — "if len(qh.HandBacks) != claim.HandBacksShown || qh.More != recent-claim.HandBacksShown {"
  evidence: internal/core/implement/loop/drain.go:916 — "r := noFoundAtHandBack(v.ID, now)"
  evidence: .abcd/development/brief/04-surfaces/35-drain.md:322 — "route that the hand-back is not yet published, and why; the drain keeps it"

Gap audit:
- honoured:
  - The drain's judgement routes through the runner routing the build roles use, as a drain-judge role, with abcd computing the remedy digest itself
    evidence: internal/core/runner/config.go:213 — "known := map[string]bool{roleImplementer: true, roleDrainJudge: true}"
    evidence: internal/core/implement/loop/drain_judge.go:293 — "RemedySHA256: j.RemedySHA256,"
  - The pace and sub-agents a drain was started with reach every lane it opens
    evidence: internal/core/implement/loop/drain.go:1046 — "lane.Pace, lane.SubAgents, lane.FixRounds, lane.pace = nil, nil, nil, st.Pace"
  - A running drain's pace and sub-agents can be changed while it runs, applied from the next lane and the lane in progress's next stage
    evidence: internal/core/implement/loop/drain.go:401 — "if repace, repaced, err = repaceDrain(repoRoot, &st, o, flags); err != nil {"
    evidence: internal/core/implement/loop/drain_pace.go:83 — "this lane runs on it from its next stage"
  - An issue the drain handed back is looked at again once its record changes on origin
    evidence: internal/core/implement/loop/drain_readmit.go:115 — "return r.Blob == "" || now == "" || now == r.Blob"
  - A fetch of origin that fails at a move's start becomes a wait on the outage schedule rather than a refusal
    evidence: internal/core/implement/loop/drain_origin.go:444 — "s := implement.DefaultSchedule"
    evidence: internal/core/implement/loop/drain.go:550 — "return finish(started || probed)"
  - The holder the forge names is asked once per process
    evidence: internal/core/claim/holder.go:283 — "holderCache.put(k, forgeAnswer{login: login, id: id})"
  - abcd peers shows a live drain lane's stage, riding on the reservation renewal every account already fetches
    evidence: internal/core/implement/loop/reservation.go:96 — "Stage: renewalStage(*st)}"
    evidence: internal/surface/cli/peers.go:517 — "line += " stage " + r.Stage"
- diverged:
  - A hosted session that restarts takes its own reservations back with one flag: delivered only where the person's username resolves without the forge and the lost copy's holds do not overlap; a person only the forge token names gets no take-over and no advice naming the flag, and overlapping holds wait until they lapse
    evidence: internal/core/implement/loop/drain.go:863 — "capHolder = claim.ReaderHolder(repoRoot, claimOptions(now))"
    evidence: internal/core/implement/loop/drain.go:939 — "takeOver = capHolder"
    evidence: internal/core/claim/crowd.go:266 — "if r.State != StateLive || r.Record == id || r.Copy == CopyThis {"
  - A lost connection to origin waits like any other outage: delivered for the move's opening fetch, but a reservation push whose own fetch fails mid-move refuses the move
    evidence: internal/core/implement/loop/reservation.go:618 — "settle what it names, then run the verb again"
    evidence: internal/core/implement/loop/drain.go:998 — "return refuseMoved(err)"
    evidence: .abcd/development/brief/04-surfaces/35-drain.md:364 — "`origin` not answering"
  - No person needs to write the judgement's answer by hand: only the claude runner can take the role, the judge cannot start under a path holding a parenthesis or comma, and a runner that does not answer is not started again for that request, so the host's request stands until its hold lapses
    evidence: internal/core/runner/config.go:253 — "if role == roleDrainJudge && name == OpenCode {"
    evidence: internal/core/runner/runner.go:234 — "toolRe = regexp.MustCompile("
    evidence: internal/core/implement/loop/drain_judge.go:249 — "so it is not started again and the request is the host's"
  - abcd peers shows each drain lane's stage: a lane opened on its judgement hold or on a take-over shows none until its first renewal
    evidence: .abcd/work/issues/open/iss-2610170948520608-in-a-drain-a-lane-opens-on-the.md:15 — "peers see none for the lane until its first renewal, about half a lease later."
  - abcd peers lists each holder's recent hand-backs with their kind: bounded to the newest ten per holder with a count of the rest, and only tombstones still at the tip
    evidence: internal/core/claim/recenthandbacks.go:24 — "const HandBacksShown = 10"
    evidence: .abcd/work/decisions/dcn-2610170757251547-abcd-peers-shows-at-most-ten-hand-backs.md:6 — "abcd peers shows at most ten hand-backs per holder"
- missing:
  - Evidence that a drain runs unattended for a day: the spec names the three-account lab, rerun on the built binary before close, as that evidence, and no rerun is recorded in the close or anywhere in the record tree
    evidence: .abcd/development/specs/closed/spc-2610161044461448-drain-many-accounts.md:75 — "Before this spec closes, the three-account lab of 2026-10-16 is run again on the built binary, every account on the same build."
    evidence: git:80764362b — "chore: close the drain-many-accounts spec and ship its three intents"

Scope-condition dispositions:
- cond-2610161044461615 — narrowed: A route in the person's own layer starts the runner and, with no route, the host's request is written as before (tested), but only the claude runner can take the role and the judge cannot start under a path holding a parenthesis or a comma.
  narrowing: holds where the person's own machine layer routes roles.drain-judge.runner to the claude runner and the real paths of the checkout and the system temporary directory hold no parenthesis or comma; an opencode route, or such a path, leaves the judgement with the host exactly as with no route
  evidence: internal/core/runner/config.go:248 — "which is skipped: only the person's own"
  evidence: internal/core/runner/config.go:253 — "if role == roleDrainJudge && name == OpenCode {"
  evidence: internal/core/implement/loop/drain_judge_test.go:111 — "func TestWithNoRouteTheDrainJudgementIsTheHostsRequest(t *testing.T) {"
  evidence: .abcd/work/issues/open/iss-2610171043574168-the-routed-drain-judge-cannot-run-for-a.md:15 — "so every move hands the request to the host."
- cond-2610161044469300 — narrowed: With a GitHub no-reply git identity the take-over moves the lost copy's reservation as the drain reaches it (tested), but the let-through keys on the holder read without the network and exempts only the issue's own reservation, not the footprints of the lost copy's other holds, which that copy was allowed to overlap.
  narrowing: holds where the person's GitHub username resolves without asking the forge (a GitHub no-reply user.email, or reservation.github_username in the machine layer) and for each lost-copy reservation whose footprint shares no path with another live reservation of that copy; otherwise the new copy passes those issues over until the lost copy's reservations lapse
  evidence: internal/core/implement/loop/drain_takeover_test.go:27 — "lost := otherCopy(t, bare, patEmail)"
  evidence: internal/core/implement/loop/drain.go:863 — "capHolder = claim.ReaderHolder(repoRoot, claimOptions(now))"
  evidence: internal/core/claim/render.go:168 — "h, err := resolveHolder(root, who, o, false)"
  evidence: internal/core/claim/crowd.go:266 — "if r.State != StateLive || r.Record == id || r.Copy == CopyThis {"
  evidence: internal/core/claim/crowd.go:95 — "if r.Copy == CopyThis && mine {"
- cond-2610161044467671 — narrowed: The renewal's stage vocabulary is exactly the loop's performed stages, held together by a test, and a drain move renews the lane at half its lease, but a lane shows no stage before its first renewal and a stopped drain's last stage stays shown until the reservation lapses.
  narrowing: holds while the lane's drain keeps moving (each move renews at half the lease, one hour at the two-hour default); a lane opened on its judgement hold or a take-over names no stage until its first renewal, and once moves stop the last stage stays on show for up to a full lease, until the reservation lapses
  evidence: internal/core/claim/stage.go:24 — "var LaneStages = []string{LaneStageWorktree, LaneStageBrief, LaneStageImplement, LaneStageValidate, LaneStageLand}"
  evidence: internal/core/implement/loop/renewal_stage_test.go:26 — "func TestEveryStageTheLoopRenewsWithIsAClaimLaneStage(t *testing.T) {"
  evidence: internal/core/implement/loop/reservation.go:159 — "return own, !now.Before(half), """
  evidence: internal/core/lease/lease.go:19 — "const Default = 2 * time.Hour"
  evidence: internal/surface/cli/peers.go:438 — "if r.State == claim.StateLive && claim.ValidLaneStage(r.Stage) {"
  evidence: .abcd/work/issues/open/iss-2610170948520608-in-a-drain-a-lane-opens-on-the.md:15 — "peers see none for the lane until its first renewal, about half a lease later."
- cond-2610161044460285 — survived: abcd peers lists the stage and hand-back rows, and the people's status view, abcd status, already embeds, fills and renders the same reservations block, so the rows travel with the listing.
  evidence: internal/surface/cli/peers.go:438 — "if r.State == claim.StateLive && claim.ValidLaneStage(r.Stage) {"
  evidence: internal/surface/cli/status.go:65 — "reservationsBlock"
  evidence: internal/surface/cli/status.go:166 — "failed, listed := out.reservationsBlock.fillView(root, now, fetched.Gap, ferr)"
  evidence: internal/surface/cli/status.go:309 — "renderReservations(w, out.reservationsBlock)"
  evidence: internal/surface/cli/status.go:115 — "drain lane's stage and each holder's hand-backs from the last day."
<!-- abcd-review-end receipt=rcp-cf03d6bdefc1 -->

## Decisions

- 2026-10-16, facilitator-on-delegation (dcn-2610161040352440): the three drafts are one bundle, drain-many-accounts; this one is minor and builds on the other two.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040390001): the judgement is a drain-judge role on the runner routing; with no route it goes to the host as today; abcd computes the digest.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040398791): a restarted hosted drain recovers with `abcd drain --take-over`; abcd keeps no drain state outside the checkout. The press release sentence that said a restart keeps the drain state and copy key was narrowed to this on 2026-10-16.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040399117): pace and sub-agents reach every lane and can be changed while the drain runs; changed records are re-admitted.
- 2026-10-16, facilitator-on-delegation (dcn-2610161040394970): the lane stage rides on the reservation renewal and shows in abcd peers with recent hand-backs; the holder is cached per process. The wait on a failed fetch is delivered with the bundle's first step, under dcn-2610161040350256.
- 2026-10-16, facilitator-on-delegation (dcn-2610161044187673): the pre-pass's questions are answered. Brief invariant 7 is amended for each new drain network act in the step that adds it, invariant 19 gains the clause on issues naming an intent, a published hand-back is a judgement kept on the reservation and not a classification written onto the issue, and each sibling overlap is kept both.

## Grounds

- pursued: facilitator-on-delegation, 2026-10-16: the product thinker wants drains on several accounts to run through a day without minding and to be seen by everyone; we expect routing the judgement, waiting out lost connections and showing lane stages on the reservation to remove the remaining reasons a person must step in. Shown wrong if a drain with a routed judge needs a person's answer before it ends, or a person cannot name another account's lane stage from abcd peers.
