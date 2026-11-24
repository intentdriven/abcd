---
id: itd-2610261025516272
slug: an-abcd-managed-private-repository-lands
spec_id: spc-2611241502278381
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610110940216682]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2609012313465609, iss-2610110939559446, iss-172]
supersedes: [itd-115]
refines: [itd-106]
blocked_by: [itd-2610110940216682]
---

# A project with no merge queue still lands only what was tested

## Press Release

> **A project whose forge offers no merge queue lands every change the way a queue would: tested as it will land, one at a time, with exactly the tested tree reaching the default branch.** When Alice's agent finishes a change, abcd joins it to the default branch as it stands, runs the project's own landing check on that result on her machine, and lands that result and nothing else. If the default branch moved in the meantime, the landing is refused and checked again. When her laptop and her desktop both try to land, one goes first and the other waits its turn, then tests against what the first landed. abcd says plainly what stood behind each landing: an independent check, or her own machine's word. Where the forge already runs a queue, nothing here switches on, and the queue stays the judge.
>
> "My project is private, so the forge gives me no queue, and I used to choose between paying for a run on every pull request and merging on trust," said Alice, a solo founder. "Now two machines can land all day, nothing lands untested, and I can see which landings only my own machine vouched for."

## Why This Matters

A merge queue gives one guarantee that nothing else on a forge gives: the tree that lands is a tree that was tested. Most projects abcd manages cannot have one. On the largest forge a queue exists for a private repository only on the top plan, and a personal account has none at all; on the next largest it needs a paid tier. Such a project has three poor choices today. It can require every branch to be up to date and pay for a full hosted run each time the default branch moves. It can drop that rule and land merges nobody tested. Or it can land on trust.

abcd's own repository shows what the checks cost when they are laid out without a budget (measured 2026-11-24): the local check takes about 50 minutes, the hosted pull-request run about an hour, and the queue run 40 to 60 minutes, so each change is tested three times, and 195 queue runs in 200 pass. A project without a queue pays the first two and still lacks the third, the only one that tests what lands.

`abcd land` cannot fill the gap as built. It runs one repository's check by name, hands the merge to the forge, and where no queue exists the forge merges a tree no one tested. Nothing orders two machines. And nothing abcd installs gives a managed project a landing check to run at all.

## Mechanism

We expect that in a project with no merge queue no untested tree reaches the default branch, and that a landing costs one check of the landing tree plus a recheck only of what newer commits reach, because the landing is a compare and swap on the default branch: abcd builds the candidate (the change joined to the default branch's tip), checks the candidate, and moves the default branch only by an act the shared remote refuses when the tip is no longer the one the candidate contains. The hold on the shared remote adds order, not safety: it stops two machines from spoiling each other's check. Shown wrong if, over twenty landings in a project without a queue, the tree on the default branch after any landing differs from the tree its check ran on; or two landings started together from two computers ever leave a merge neither tested; or the time a landing waits behind another exceeds the other's check plus the recheck of what it changed; or a landing lapsed mid-check (a machine switched off) blocks the next one for longer than its hold's stated end.

## Scope Conditions

- Holds where the forge tests no merge before it lands: no merge queue, or no forge at all. Where a queue gates the default branch, this route stays off and the queue's run is never skipped or replaced. <!-- cond: cond-2611241502273892 -->
- Holds where the project declares its landing check on the default branch, as commands abcd can run on the person's machine whatever the toolchain (the declaration of itd-2611241456272313). A project that declares none gets no local landing, and abcd says so. <!-- cond: cond-2611241502279011 -->
- Holds for changes by people with write access and the agents working for them. A change from anyone else is never fetched and run on the person's machine, and takes the forge's own route. <!-- cond: cond-2611241502270018 -->
- Holds where the full landing check runs in tens of minutes on the person's machine, or reruns only what a change can affect (itd-2611011325421304); one landing at a time is too slow otherwise. <!-- cond: cond-2611241502279316 -->
- Holds for tests that are deterministic given their inputs. An intermittent test fails a landing as it would fail a queue entry. <!-- cond: cond-2611241502271801 -->
- A local landing is the lander's own word unless an independent check also judged that exact commit. This intent never turns a hosted run off; which hosted runs a project keeps is the person's choice, made elsewhere (adr-2611011327164895). <!-- cond: cond-2611241502270363 -->
- Shown first on a practice project made for the proof, and counted as finished only after one real private project has landed through it. <!-- cond: cond-2611241502270671 -->

## Acceptance Criteria

- Given a project that declares its landing check on the default branch, when abcd lands a change, then it runs the check as the default branch declares it, not as the change does, and a project in any language lands the same way.
- Given a check that ran only part of the declared landing check, when anyone lands the branch, then the landing is refused, naming the parts left out.
- Given a project with no merge queue and a branch ready to land, when abcd lands it, then it checks the change joined to the default branch as it stands at that moment, and the tree that reaches the default branch is identical to the tree the check ran on; abcd compares the two after the landing and reports a difference as a failure, never silently.
- Given the default branch moves between a landing's check and its merge, when abcd merges, then the stale landing is refused by the shared remote or by abcd, and abcd checks the new candidate, rerunning only what the new commits reach where the project's check can reuse results.
- Given landings started from two computers at once, when both reach their merge, then they land one at a time, ordered by a hold on the shared remote; the one that waits is told who holds the turn and until when; and neither lands a merge it did not test.
- Given a machine that takes the hold and then stops, when the hold's stated end passes, then the next landing proceeds without anyone clearing it by hand.
- Given a release being cut that holds merges, when a landing starts, then it waits for the release as every other landing does.
- Given a project whose forge gates the default branch with a merge queue, when abcd lands a change, then it uses the queue and takes no hold, and the queue's run on the merged tree is the judge.
- Given a change from someone without write access, when anyone asks abcd to land it locally, then abcd refuses before fetching or running its code, and names the forge's own route.
- Given any landing, when abcd reports it, then the report says what stood behind it: an independent check that judged that exact commit, named, or the lander's own machine alone; and where nothing on the forge enforces any check, abcd says that the local route is the whole fence.
- Given a practice project with no queue, made for the proof, when twenty changes have landed through this route from two computers, then the report states, per landing, the check time, the wait behind other landings, the reruns after a moved default branch, and that the landed tree matched the tested tree.
- Given one real private project the product thinker names, when twenty of its changes have landed through this route, then the same report is given for it against that project's own figures from before, and only then is this intent finished.

## SOTA

Surveyed 2026-10-26 against primary sources ([research note](../../research/notes/2026-10-26-local-first-landing-for-private-repos.md)), and read again at planning on 2026-11-24:

- **The not-rocket-science rule and its queues (bors, merge trains, the forges' merge queues):** mature. Adopted: test the merge before the default branch moves, one landing at a time. The forge's queue is used wherever the plan offers it; where it does not, abcd keeps the same rule on the person's machine.
- **Local CI with a commit status (the sign-off pattern a web framework made its default in 2025):** usable for a single developer. Not adopted as a required check: a status posted with the person's token is the agent's word (adr-2611011327164895), so the lander's own result is a label in abcd's reports and nothing here posts it.
- **Compare and swap on a ref (a fast-forward-only push; a forge merge pinned to a head commit):** built into git and the forges. Adopted as the safety of the landing; the hold on the shared remote only orders it.
- **Reuse of results whose inputs are unchanged:** mature, and delivered by itd-2611011325421304, whose fixed check directory the candidate is built and checked in.
- **Self-hosted runners:** mature, and unsafe for code from outside contributors. Not offered (dcn-2611042202509334); the independent route is the optional checker of itd-2611011325430933.

Path taken: 2. A native floor (the candidate, the pinned landing, the turn on the shared remote) with a seam for the forge's own queue where the plan offers one. No new dependency. The fit challenge the note owed was run at planning: the collisions it lists are answered by the Decisions below.

## Decomposition (itd-84 hand-run, 2026-10-26, revised at planning 2026-11-24)

| Part | Type | Home |
|------|------|------|
| Landing where the forge tests no merge: the candidate, the pinned landing, the turn between machines, what stood behind each landing | capability | this intent |
| A result the lander reports about its own work is a label and never a stand-in for an independent run; it never applies to an outside contribution | trust rule | adr-2611011327164895 (proposed) |
| The check of the tree that lands runs every declared check and is never cut | trust rule | adr-2611241456284353 |
| Each check runs once, where it is cheapest to trust | stance | the principle `each-check-runs-once-where-it-is-cheapest-to-trust` |
| The declared landing check, the receipt in the binary, the turn's reservation subject | plumbing | the spec and the brief |
| Reuse of unchanged results | capability | itd-2611011325421304 |
| Which checks a change needs, escapes counted per check, what is saved first | capability | itd-2611241456272313 |

Typed links: `supersedes` itd-115 (its want, a ready change that never waits behind an untested merge, is met here wherever no queue exists); `refines` itd-106; `builds_on` itd-2610110940216682 and, through it, itd-2611241456272313, whose declaration names the landing check. The earlier `builds_on` edge to itd-2610161647248332 is dropped: whether a queue gates the default branch is read from the ruleset mirror `land` already reads, so this intent does not wait for that draft.

## Open Questions

_All resolved; the rulings are in the Decisions section._

1. resolved: hosted or independent checking stays the default, and nothing here switches a hosted run off (2026-11-24, from adr-2611011327164895).
2. resolved: anyone with write access and their agents may use the route and its hold; a landing counts as independently checked only with an independent verdict (2026-11-24).
3. resolved: no self-hosted runner is offered (dcn-2611042202509334); the optional checker of itd-2611011325430933 is the independent route (2026-11-24).
4. resolved: a repository whose forge runs a queue keeps it and gains only the declared landing check (2026-11-24).
5. resolved: the route is proven on a practice project first, then on one real private project before the intent is finished (the product thinker, 2026-11-24).

## Decisions

1. **The routing is confirmed as proposed (2026-10-26).** The product thinker accepted the split and the typed links with a plain yes, given the agent's recommendation. This intent holds the capability alone. The trust rule (a self-reported local result may replace duplicate hosted work and gate a landing only for the person's and their agents' changes, never an outside contribution, and a release cut never reuses a result) becomes an ADR at planning, then a brief invariant. The stance that each check runs once, where it is cheapest to trust, becomes a principle at planning. itd-115 is superseded by this intent, the link to itd-106 is a refinement, and itd-2610161647248332 is a builds-on edge. The graded table is in the decomposition-calibration note under this date.

2. **Escapes are recorded and the fixes are not bundled (2026-10-26).** Asked whether to record what wastes hosted runs, the product thinker accepted the recommendation: the recording, counting each hosted failure a check run before the push would have caught, is a criterion of this intent, and the fixes for the escapes found so far are issues worked first, outside it (dcn-2610261152329451): iss-2609061525437969, iss-220, iss-2610241424265464, iss-2610261151381604, iss-2610261151481013 and iss-2610261152125261.

3. **The route is chosen by what the forge tests, not by visibility (2026-11-24, the technical facilitator on auto).** It switches on only where no queue gates the default branch. A public repository on a personal account has no queue either, and a private one on the top plan has.

4. **Safety is a compare and swap; the hold is only order (2026-11-24, the technical facilitator).** The default branch moves by a fast-forward push of the candidate where it accepts a push, or by the forge's merge pinned to the candidate's head where a pull request is required. Both are refused by the remote when the tip moved. Leases lapse and clocks differ, so a hold is never what keeps an untested merge out.

5. **No status is posted by this intent (2026-11-24, the technical facilitator).** The earlier criteria that recorded the local result as a status a ruleset requires, and that reduced the hosted pull-request run to confirming it, are withdrawn: they are the pattern adr-2611011327164895 rejects. The own-word label lives in abcd's reports; posting it stays with itd-2611011325428560 behind the person's switch.

6. **The reuse criteria live in itd-2611011325421304, and three more criteria leave (2026-11-24, the technical facilitator).** Escapes counted per check move to itd-2611241456272313, so the product thinker's ruling of 2026-10-26 that they be recorded stands with a new home. No priced platform on every landing of a private project follows from that intent's default of lowest spend for a private repository. Prices per route at setup belong to the drafts itd-2611011325403127 and itd-2611011325449761, and no surface shows a price until one of them is planned.

7. **One candidate, one change, and no retry (2026-11-24, the technical facilitator).** Landings are not batched, and an intermittent failure fails the landing; a second `land` reruns only what failed where results are reused.

8. **Where it is proven (2026-11-24, the product thinker).** A practice project first, then one real private project before the intent is called finished. The product thinker names the real project when the time comes, and the record of the proof names neither the project nor its owner.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a project without a merge queue either pays a hosted run each time its default branch moves or lands merges nobody tested; we expect a compare and swap on the default branch to land only the tested tree, one landing at a time, and twenty landings in which a landed tree differs from its checked tree, or two machines leave an untested merge, would show it wrong
