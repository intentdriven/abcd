---
id: itd-2610261025516272
slug: an-abcd-managed-private-repository-lands
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610161647248332]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2609012313465609, iss-2610110939559446, iss-172]
supersedes: [itd-115]
refines: [itd-106]
---

# Private repos land checked work without renting CI

## Press Release

> **An abcd-managed private repository lands each change after one check sized to what the change touches, run on the person's own machine, and buys hosted CI minutes only where the person chose to.** While agents work, a check reruns only the tests whose inputs changed and reuses every result whose inputs are the same, whichever worktree produced it. A feature is checked in full once, as it will land, not once per step, and a change to records alone reruns only the tests that read those records. To land a change, abcd takes the default branch as it stands, tests the merge on the same machine, again only as far as the default branch's new commits reach, and merges exactly the tree it tested, one change at a time, so no pull request waits behind another for a re-run. abcd finds out what the repository's plan allows (a merge queue, required checks, or nothing at all), says plainly what enforces the check and what does not, and shows what each hosted run would cost per landing before the person picks one. A contribution from anyone but the person and their own agents still runs in hosted CI.
>
> "My repositories are private, my laptop has ten cores, and I was paying a cloud runner to repeat what my agents had just run," said Alice, a solo founder. "Now the agents check their own work on my machine, a change to one package tests that package, and the only minutes I buy are for the one Linux run I asked for."

## Why This Matters

abcd's own repository is the one repository abcd manages with the whole regime (a preflight, a push receipt, a merge queue), and it shows where the time goes ([research note](../../research/notes/2026-10-26-local-first-landing-for-private-repos.md), "What abcd's own repository measures"):

- One preflight runs for about 24 minutes, and the wait at the machine's load gate before it took 37 to 62 minutes, with about 20 waiters across seven sessions on 2026-10-16 (iss-2610161639025702).
- Every lane starts with a cold cache. A fresh worktree's first preflight reused 0 of about 110 packages' test results, and a rerun in the same worktree reused 47 to 88, because the go command keys a build on the package's directory unless `-trimpath` is set.
- Every reviewed step of a feature pays its own full run, and a change to records alone pays the same as a change to code.
- The hosted pull-request run repeats the local preflight (iss-2610110939559446), and the queue then runs again, 1.27 times per merged pull request, with the macOS leg dropping entries at its 45-minute cap (iss-2609012313465609).

The record has owed this intent since 2026-09-29: iss-2609012313465609 was deferred on the ruling that an intent should choose among its directions, starting with which comes first.

In a private repository the same habits cost money, and the route abcd planned is closed:

- Hosted minutes beyond the plan's 2,000 (Free) or 3,000 (Pro, Team) a month cost $0.006 a minute on Linux and $0.062 on macOS, so a macOS minute costs about ten Linux minutes.
- GitHub's merge queue, the route itd-115 configures, exists for a private repository only when its organization is on Enterprise Cloud. A personal account or a Team organization has no queue.
- Rulesets, and with them required checks, need Pro or Team for a private repository. On GitHub Free nothing on the forge enforces a check at all.
- Nothing abcd installs gives a managed repository a preflight, a receipt or a CI check workflow: `ahoy` installs four commit hooks, and `launch scaffold` writes the release workflows.

Two drafts already hold parts of this, and neither reaches that repository. itd-115 configures GitHub's merge queue by default, with a fallback that polls and updates a branch left behind. itd-2610110940216682 runs the full hosted suite once per change in the queue, and by its own scope condition a repository without a queue keeps the full suite on its pull requests. Neither makes a local check cheaper, and neither serves a private repository without a queue, which is most of the repositories abcd manages.

Local CI with a commit status, the gh-signoff pattern that Rails 8.1 made a framework default, answers the cost for a single developer: the laptop runs the suite and tells the forge so. It leaves two things to answer: an agent's sign-off is the agent's word, and a branch tested alone is not its merge.

## Mechanism

We expect the local check per landing to cost in proportion to what a change touches, and the hosted minutes per landing to fall to none, or to the one Linux run the person chose, because three costs paid on every landing today can be avoided. Results whose inputs are byte-identical are rerun only because the cache key carries the worktree's directory. A feature's steps each pay a full run that one check of the landing tree covers. And the merge, the one thing a branch's own run cannot prove, is tested in hosted CI after a local run whose result a status could have carried. Shown wrong if, over a measured run of at least twenty landings in one private managed repository and twenty in abcd's own, the local check time per landing, the wait before it or the hosted minutes per landing is not lower than the 2026-10 baseline in the research note; or if a landed tree ever differs from the tree its check ran on; or if the release cut's run without reuse finds a defect that a reused result had passed; or if escapes, hosted failures a check run before the push would have caught, do not fall from the 13 of 32 failed runs of 2026-10-18 to 2026-10-26 (dcn-2610261152329451).

## Scope Conditions

- Holds for repositories whose pushers are the person and the agents working under the person's identity. A contribution from anyone else runs the full hosted check.
- Holds for GitHub at the plan tiers its documentation states on 2026-10-26: a merge queue for organization-owned public repositories and for private repositories on Enterprise Cloud only, and rulesets on private repositories from Pro and Team up. Another forge reaches the same routes through the forge seam.
- Holds for tests that are deterministic given their inputs. A flaky test breaks reuse and selection alike, and is a defect to fix, not a reason to rerun everything.
- Holds where the person's machine runs the repository's full check in tens of minutes, not hours, under the machine's load gate.
- Holds where the person's own operating system is one the project ships to. Another operating system is covered by a hosted run the person chooses, or by the nightly and release runs.

## Acceptance Criteria

- Given a commit whose check has passed in one worktree, when the same commit is checked in a fresh worktree on the same machine, then every test whose inputs are unchanged is reused rather than rerun, and the check's report counts the tests it ran and the tests it reused.
- Given a test that reads an input the result cache cannot see (a file a subprocess reads, or a file outside the repository), when the repository declares that test, then it reruns on every check; and given a release cut, when its gates run, then every test runs without reuse on the tree being tagged.
- Given a change to records alone, when its check runs, then only the tests whose inputs include those records rerun, and the report says which ran and why.
- Given a check that ran only a part of the gates, as an agent's quick check of one step does, when anyone lands the branch, then the landing is refused, naming the gates the check left out; only a check that covered every gate on the landing tree, each run or reused, lets it land.
- Given a feature built in several reviewed steps, when it lands, then it lands as one change with one landing check, and each step in between was checked with reuse rather than with a full run.
- Given a private repository with no merge queue and two branches ready to land, when abcd lands them, then it lands one at a time, each tested merged onto the default branch as it stands at that moment, and the tree that reaches the default branch is the tree that check ran on.
- Given the default branch moves between a branch's landing check and its merge, when abcd merges, then the stale merge is refused, and abcd tests the merge again, rerunning only what the new commits on the default branch reach.
- Given landings started from two computers at once, when both reach their merge, then they land one at a time, ordered by a hold on the shared remote, and neither lands an untested merge.
- Given a repository whose plan allows rulesets, when a landing check passes, then abcd records it on the tested commit as a status the ruleset requires; and given a plan that allows no rulesets, then abcd says plainly that nothing on the forge enforces the check and the local hooks are the whole fence.
- Given a repository that runs hosted CI and a pull request whose head carries a passed landing check, when the pull request is opened or updated, then the hosted pull-request run does no work beyond confirming that status.
- Given a pull request from anyone but the person and their agents, when it is opened, then the full hosted check runs on a hosted runner, a status its author posts is not accepted in place of it, and no self-hosted runner runs its code.
- Given abcd sets up a managed repository, when it offers the landing routes the plan allows, then each route states the hosted minutes and their price per landing at the published rates, with the date those rates were read, and states what the forge enforces under it.
- Given a private repository, when abcd sets up its hosted runs, then no macOS run is placed on every landing unless the person chooses it with its price shown; macOS runs nightly and at the release cut instead.
- Given a hosted run that fails, when abcd reads it, then it names the check that failed and whether that check could have run before the push (an escape), and given the failures since a stated date, then it counts escapes per check and names, for each check with any, the earliest local point that sees its input (the commit, the posted text, the landing, or the preflight before its test stages).
- Given the baseline in the research note, when this intent ships, then a measured run of at least twenty landings reports the local check time, the wait before it and the hosted minutes per landing against that baseline.

## SOTA

Surveyed 2026-10-26 against primary sources ([research note](../../research/notes/2026-10-26-local-first-landing-for-private-repos.md)):

- **Local CI with a commit status (gh-signoff; Rails 8.1 `bin/ci`):** usable, and a framework default since Rails 8.1. Adopted as the pattern: the local result is posted as a status a ruleset can require. Not adopted as a dependency: posting a status is one forge call abcd makes itself.
- **The not-rocket-science rule and its queues (bors, Jane Street, GitHub's merge queue):** mature. Adopted: test the merge before the default branch moves, one landing at a time, with fewer and larger landings. GitHub's queue is used where the plan offers it; where it does not, abcd runs the queue on the person's machine.
- **Selection by dependency (Pants `--changed-dependents`, Nx affected):** mature inside their build systems, each a toolchain a repository would have to adopt. Not adopted now. The seam is the declared gate list's inputs, which a repository already on one of them can fill from it.
- **Reuse of results whose inputs are unchanged (Go's test cache; build-system caches; `GOCACHEPROG`):** mature. Adopted as the first lever: Go's own cache, shared across worktrees by `-trimpath`, with the tests whose inputs it cannot see declared. A cache shared between machines is a later backend behind `GOCACHEPROG`.
- **Learned test selection (Meta's predictive test selection):** mature at Meta's scale, and needs a change history one repository lacks. Not adopted.
- **Self-hosted runners:** mature, free today with a postponed fee, and unsafe for code from outside contributors. Offered only as a route for the person's own changes.

Path taken: 2. A native floor (the declared gate list and its inputs, the receipt posted as a status, the local landing queue, reuse across worktrees) with seams for the field's backends: a forge merge queue where the plan offers one, and a dependency selector or a shared cache where a repository already runs one. No new dependency. The independent fit challenge runs at planning, with the collisions the research note lists.

## Decomposition (itd-84 hand-run, 2026-10-26)

Confirmed by the product thinker on 2026-10-26 (decision 1 below). Verdict: **SPLIT**.

| Part | Type | Home |
|------|------|------|
| How a managed repository's changes are checked and landed: reuse, one check per landing, the local queue, the status, the route and its price | capability | this intent |
| A self-reported local result may replace duplicate hosted work, and may gate a landing, only for the person's and their agents' changes; it never gates an outside contribution, and a release cut never reuses a result | trust rule | an ADR at planning, then a brief invariant |
| Each check runs once, where it is cheapest to trust | stance | a principle at planning |
| The gate list's schema, the receipt's fields, `-trimpath`, the declared tests | plumbing | the spec and the brief |

Typed links, as confirmed:

- `supersedes` itd-115: its want, a ready pull request that never waits behind another, is met here on every plan, through GitHub's queue where the plan has one and through the local queue elsewhere, which replaces its poll-and-update fallback.
- itd-2610110940216682 stays the shape of hosted CI where a queue gates the default branch, as a sibling with no typed link. This intent's pull-request criterion (no work beyond confirming a passed landing check) tightens that draft's first criterion (the quick checks on one platform), flagged for the product thinker as a refinement of it, not a reversal.
- `refines` itd-106: that intent sets up which checks a repository has; this one decides where each runs and how a landing uses them.
- `builds_on` itd-2610161647248332: the route is chosen from what the forge can do, which that intent finds out.
- The issues it answers, joined from both ends: iss-2609012313465609 (the intent that issue's deferral says is owed), iss-2610110939559446 (the receipt as a status) and iss-172 (pull-request queue hygiene in managed repositories). iss-2610161639025702 (preflight throughput) is answered here too, but is already promoted to itd-2610170634394993, so it is cited rather than joined.

No reversal flag. iss-172's invariant protects testing the merge before the default branch moves, and every route here still does that. The strict up-to-date setting stays on wherever the plan allows it; where the plan allows no rulesets, the local queue's own refusal of a stale merge does that job.

## Open Questions

- **The default route for a private repository with rulesets.** The local queue with its self-reported status, which spends no hosted minutes, or the local queue plus one hosted Linux run of the merge per landing, which is an independent check at about $0.006 a minute. Either way the person sees both prices at setup.
- **Who counts as the person's own.** The person's identity and the agents working under it only, or every collaborator with write access.
- **The self-hosted runner route.** Offer it for the person's own changes, since it gives a check that the pushing agent does not run itself and costs no minutes, or leave it out, since it shares the machine and the load gate with the local checks.
- **abcd's own repository.** It is public, so its minutes are free and its queue gives an independent check of every merge. Keep the queue and adopt only reuse and one check per landing, or move it to the local queue too.

## Decisions

1. **The routing is confirmed as proposed (2026-10-26).** The product thinker accepted the split and the typed links with a plain yes, given the agent's recommendation. This intent holds the capability alone. The trust rule (a self-reported local result may replace duplicate hosted work and gate a landing only for the person's and their agents' changes, never an outside contribution, and a release cut never reuses a result) becomes an ADR at planning, then a brief invariant. The stance that each check runs once, where it is cheapest to trust, becomes a principle at planning. itd-115 is superseded by this intent, the link to itd-106 is a refinement, and itd-2610161647248332 is a builds-on edge. The graded table is in the decomposition-calibration note under this date.

2. **Escapes are recorded and the fixes are not bundled (2026-10-26).** Asked whether to record what wastes hosted runs, the product thinker accepted the recommendation: the recording, counting each hosted failure a check run before the push would have caught, is a criterion of this intent, and the fixes for the escapes found so far are issues worked first, outside it (dcn-2610261152329451): iss-2609061525437969, iss-220, iss-2610241424265464, iss-2610261151381604, iss-2610261151481013 and iss-2610261152125261.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
