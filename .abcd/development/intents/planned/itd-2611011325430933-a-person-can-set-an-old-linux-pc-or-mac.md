---
id: itd-2611011325430933
slug: a-person-can-set-an-old-linux-pc-or-mac
spec_id: spc-2611241502296629
kind: standalone
suggested_kind: null
reclassification_history: []
refines: [itd-2610261025516272]
builds_on: [itd-2610261025516272]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2610261025516272]
---

# A machine the person owns checks each change independently

## Press Release

> **A person can set a spare computer on their own network to check every change their agents push, and nothing ever waits for it.** The machine finds each change by itself, runs the checks the project's default branch declares, and reports the result under an identity that lives only on that machine, so the verdict is one no agent could have supplied. It is a second opinion on a second machine, and where that machine runs an operating system the project ships to but the person does not work on, every change is checked there within the hour and not once a night. It is optional throughout. abcd looks for the machine, uses it when it is there, and when it is switched off or busy the work lands as usual and abcd says that the second opinion is still owed. A person with several spare machines can name them all, and the first one free takes the next change. The check of the merged result still runs on every change and is never replaced. A project with nothing for the machine to add is told so and nothing is set up.
>
> "My agents wrote the change and my agents ran the checks, on the same laptop," said Alice, who builds a small desktop product alone. "Now the old computer under my desk runs them again, and it is the one thing in the loop my agents cannot talk to. When I take it away for a weekend, nothing stops."

## Why This Matters

A person whose agents write the work wants it checked by something the agents do not control. Today there are two choices. One is a hosted run on every pull request, which repeats what the person's machine has just run and what the check of the merged result will run again: in the repository this was measured in, each change is tested three times, at about 50 minutes, an hour, and 40 to 60 minutes (2026-11-24). The other is the agents' own word: a local result posted to the forge can be posted by anything holding the person's token (adr-2611011327164895, iss-2610110939559446). A hosted run is also the only way to reach an operating system the person does not work on, and where that runner is scarce or priced by the minute it is moved off the per-change path: in the same repository the second operating system is tested nightly and at a release, and intermittent tests found only there cost a release about seven hours.

A self-hosted runner looks like the answer and is not: the pull request decides what the runner executes, the forge advises against one on a public repository, and the forge cannot tell the runner's verdict from any other workflow's. A small checker that pulls, runs the default branch's declared checks, and reports under an identity whose key never leaves the machine gives a verdict bound to one source.

A machine in someone's home is also switched off, asleep or busy some of the time. If landing depended on it, one unplugged cable would stop all work, so it must be an addition and never a gate (adr-2611241456294452).

## Mechanism

We expect a change checked this way to carry a verdict no agent could have produced, and a defect that shows only on the checker's operating system to be reported within the hour of the push and not at the next scheduled run, with no landing ever delayed, because nothing the agents hold reaches the machine and nothing waits on it: it takes no work and no instruction over the network, it reads the checks to run from the default branch and not from the change, the checks run under an account that holds no credential, the verdict counts only from the identity whose key a separate signer on the machine holds, and no rule on the forge and no step of a landing requires it. Shown wrong if any pushed change alters what the machine runs or obtains its key; if a verdict under the machine's name is accepted from any other source; if, over twenty consecutive landings with the machine switched off, any landing takes longer or is refused because of it; if, over twenty consecutive landings with it on, fewer than three in four changes have its verdict within an hour of the push, or its backlog grows from one day to the next at the project's ordinary rate of change; or if, over a month, defects specific to the checker's operating system are first found by a scheduled run or a release as often as before.

## Scope Conditions

- Holds for a machine the person owns and keeps on their own network, running an operating system the project is built or tested on. <!-- cond: cond-2611241502291059 -->
- Holds for a project that declares its checks on the default branch (itd-2611241456272313). Where there is no declared check, or the machine would add nothing the project's other runs do not already give, abcd sets nothing up and says why. <!-- cond: cond-2611241502298584 -->
- Holds for a project on a forge that can show a check as coming from one reporting identity. On a forge that cannot, the verdict is reported and labelled as unpinned, and abcd says that nothing on the forge vouches for its source. <!-- cond: cond-2611241502299698 -->
- Holds for changes pushed to the project's own branches by people with write access. A change from a fork is never fetched to the machine. <!-- cond: cond-2611241502299312 -->
- Holds where the machine can check changes about as fast as they arrive once a check is sized to what a change can affect (itd-2611011325421304). It is not spare capacity for the pusher's own check, and no result of the pusher's is accepted in its place. <!-- cond: cond-2611241502290237 -->
- The verdict is never required. The check of the merged result is outside this intent and is never replaced, shortened or waited on by it. <!-- cond: cond-2611241502297239 -->
- Machines are found from a list the person keeps on their own computer or, where the person switches it on, by discovery on the local network only. Nothing in a project's files names a machine. <!-- cond: cond-2611241502295926 -->
- Registering the machine's identity takes one step only the person can take, in a browser (dcn-2610180720389628). <!-- cond: cond-2611241502297380 -->

## Acceptance Criteria

- Given a machine the person has set up, when a change is pushed to a branch of the project, then the machine fetches it without any agent's action, runs the checks the default branch declares, and reports the result on that commit under its own identity.
- Given a pushed change that edits a file the default branch declares as part of its checks, when the machine runs, then it runs the default branch's copy of that file, and the verdict names the edit.
- Given a pushed change whose code tries to read the machine's key, report a verdict itself, or reach other machines on the person's network, when the checks run, then the account they run under holds no credential and no such route, and only the signer reports, after the result is in.
- Given a change ready to land and a machine that is switched off, unreachable, busy or behind, when abcd lands the change, then the landing proceeds exactly as it would with no machine at all, and its report names the machine, says the independent check is still owed, and says how far behind the machine is and since when.
- Given a change that landed before the machine reached it, when the machine is next free, then it checks that commit or the default branch's tip that contains it and reports the result; and given that result is a failure, then abcd reports it the next time anyone reads the project's status, naming the commit and the check.
- Given the person has named several machines, or switched discovery on and several answer, when a change is pushed, then the first machine free takes it, no second machine runs the same commit, and a machine that stops mid-check loses its claim so another takes the change.
- Given discovery is off and no machine is named, when abcd runs, then it looks for nothing on the network; given discovery is on, then abcd asks only on the local network, and a machine it finds is used only for its state, never for a verdict.
- Given no rule on the forge and no step of abcd's landing, when a machine is set up, changed or removed, then nothing about how a change lands differs, and abcd offers no setting that makes the verdict required.
- Given a branch pushed again before the machine reached its earlier commit, when the machine picks its next work, then it checks the newest commit and marks the older one superseded, not passed.
- Given the machine loses its connection, when it is back, then it reports every result it owes.
- Given a forge that cannot show which identity reported a check, when the person sets the machine up, then abcd says so before anything is installed and labels every verdict as unpinned.
- Given a change from a fork, when it is opened, then the machine never fetches it.
- Given a project with no declared check, when the person asks for a checker, then abcd installs nothing and says what is missing.
- Given a change the machine checked, when abcd reports that landing anywhere, then the report names the machine, its operating system, the commit tested and when, and distinguishes it from a hosted check and from the lander's own word.

## SOTA

From the [local-first landing note](../../research/notes/2026-10-26-local-first-landing-for-private-repos.md) and the forge's documentation as read on 2026-11-24.

- **Self-hosted runners:** mature. Not adopted: the workflow a runner executes comes from the change, the forge advises against one on a public repository, and its verdict is indistinguishable from any other workflow's.
- **A forge application as a check's reporting identity:** built into the largest forge, where only an application may create a check and a rule may name the expected source. Adopted for the identity and its key on the machine; the rule that would require the check is deliberately not written.
- **Pull-based build workers (the build-farm lineage):** mature. Adopted as the shape: the worker asks for work and no one pushes work to it.
- **Service discovery on the local network (DNS-SD over multicast DNS):** built into the operating systems this targets. Adopted through the operating system's own tools, opt-in, for a machine's state only.
- **Commit statuses posted from the person's token:** rejected for the reason adr-2611011327164895 gives.

**Path 2:** a native floor (the pull-only checker, its signer, the claim among several) with the forge's identity as the seam. No new dependency: the token is signed with the standard library and discovery uses the operating system's tools.

## Decisions

- 2026-11-24, the product thinker: a checking machine is optional. abcd uses one when it can find it and carries on without it when it cannot, saying so, and development never waits for it. A person may make several known to abcd, and the first one free takes a change. Recorded as adr-2611241456294452.
- 2026-11-24, the technical facilitator on auto, following that ruling: no rule requires the check and landing takes no hold for it. With no required check there is nothing to carry over onto a merge-queue entry, so that is not built either. The earlier criterion that a landing waits for a machine that is offline is withdrawn.
- 2026-11-24, the technical facilitator: the checker never replaces the check of the merged result (adr-2611241456284353). This answers the first open question.
- 2026-11-24, the technical facilitator: machines are declared first, in the person's own configuration, and discovery is opt-in and local only. A checker answers one fixed question about its state on the local network and takes no work that way; checkers pull, and a claim made under the checker's identity decides which of several runs a commit.
- 2026-11-24, the technical facilitator: "the default branch's definition" is the declared checks and the declared list of files that define them (itd-2611241456272313), both read from the default branch's tip and laid over the change before it runs.
- 2026-11-24, the technical facilitator: two operating-system accounts on the machine, a signer and a runner. The runner holds nothing and is denied the private address ranges where the system allows it. A virtual machine is offered as the stronger option and not required.
- 2026-11-24, the technical facilitator: which changes are the project's own is decided by where a branch lives (the project's own branches, never a fork's), not by any name written in a commit.
- 2026-11-24, the technical facilitator: the first version is proven on a Mac, since the ruling speaks of Macs and that is the coverage no other per-change run gives. The Linux service is built and tested alongside; its proof on a real machine is not a condition of finishing.
- 2026-11-24, the technical facilitator: the earlier criterion that named an unbuilt landing report and the hosted minutes saved is reworded to "wherever abcd reports a landing"; minutes saved are left to itd-2611011325403127.

## Open Questions

_All resolved; the rulings are in the Decisions section._

1. resolved: the machine never replaces the check of the merged result, and no longer replaces any required run (2026-11-24).
2. resolved: the first proof is on a Mac, on the repository that measured the gap (2026-11-24).
3. resolved: a landing never waits for the machine; abcd names it as off or behind in the landing's report and reports a late verdict when it arrives (the product thinker, 2026-11-24).

## Decomposition

Hand-run four-piece table, 2026-11-01, revised at planning on 2026-11-24.

| Part | Kind | Home |
|---|---|---|
| the pull-only checker, its identity, several machines and the first one free, catch-up after an absence, the report of a late verdict | capability | this intent |
| an independent checker is optional: reported, never required, never waited for, and it takes no work from the machines it checks | trust rule | adr-2611241456294452 |
| the checker never runs a change's own definition of its checks | trust rule | already standing in the principle evaluator-outside-the-loop; cited, no new ADR |
| the service, the fetch key, the identity's setup, discovery | plumbing | a brief chapter for the checker |

Verdict: FILE-AS-IS. Typed links: `refines` itd-2610261025516272 (it answers that intent's question about a self-hosted runner, and fills its "an independent check judged that exact commit"); it resumes the route itd-2610120738075547 deferred ("a machine the person runs"); `builds_on` itd-2610261025516272.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: agents write the work and run its checks on one machine, and a second operating system is checked only nightly; we expect a pull-only checker under its own identity to give a verdict no agent could produce within the hour while no landing waits for it, and a landing delayed by a checker that is off, or a change that alters what the checker runs, would show it wrong
