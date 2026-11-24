---
id: adr-2611241456284353
slug: the-check-of-the-merged-result-runs
status: accepted
date: 2026-11-24
supersedes: null
superseded_by: null
related_intents: [itd-2611241456272313, itd-2610110940216682, itd-2610261025516272, itd-2611011325430933]
related_rfcs: []
related_adrs: [adr-2610120937435856, adr-2611011327164895, adr-2611241456294452]
---

# ADR-2611241456284353: The check of the merged result runs every declared check for every change, and no earlier result stands in for it

## Context

A change in a repository abcd manages can be checked at several points before it lands: on the contributor's machine before the push, on the proposal (the pull request), and on the merged result before the default branch moves. Each point was set up on its own, so the same tests often run at all three. abcd's own repository measured it on 2026-11-24: about 50 minutes on the contributor's machine, about an hour on the proposal, and 40 to 60 minutes on the merged result, with 195 of the last 200 runs on the merged result passing.

Four intents now cut that repetition from four sides: a budget per change (itd-2611241456272313), a shorter proposal run (itd-2610110940216682), a landing check on the person's own machine where the forge tests no merge (itd-2610261025516272), and an optional checker on a machine the person owns (itd-2611011325430933). Each removes or moves a run. Without one rule that none of them may cross, two of them can each remove the same safety.

Three rulings were already in force. No receipt minted in the cloud is trusted in place of the local check (adr-2610120937435856). A result the pusher reports about its own work is a label, never a stand-in for an independent run (adr-2611011327164895, proposed). The receipt a push carries is the pusher's word (iss-2610110939559446). The forge offers no way to bind a result produced on a contributor's machine to a commit so that others can verify it: its attestations are signed only inside its own hosted runs. itd-2610110940216682 has owed this rule as a decision since 2026-10-11.

## Decision

We will treat the check of the merged result as the one run a landing rests on. For every change, whoever made it and whatever it touched, every check the default branch declares for the landing runs in full on the exact tree that will land, before the default branch moves. Where the forge tests the merged result (a merge queue, a merge train), that run is the check. Where it tests none, abcd's own landing check on the candidate tree is (itd-2610261025516272).

No setting, no earlier run, no local result, no contributor's statement and no verdict from another machine lets a change land without it, shortens it, or selects among its checks. Every earlier run is an early warning. A plan for a change may move a check later, up to this run, and may never drop it.

Two things stay outside the rule. A release runs every check on every platform the product ships to with nothing reused. A platform the landing run does not cover is stated as not covered in every plan and report, never implied.

## Alternatives Considered

- **The run on the merged result is whole for every change; earlier runs are warnings (chosen).** It is the only run that tests the tree that lands, so it is the one that cannot be replaced. With it fixed, a wrong plan for the earlier runs costs a later warning and never an unchecked landing, and nothing a contributor's machine says has to be trusted.
- **Size the run on the merged result to the change as well.** Cheaper again, and a selection error there lands a fault with a green result. Selection inside this run is left out until reuse in hosted runs has its own measured record (the follow-up itd-2611011325421304 names). Rejected for now.
- **Let a passed local check or an independent checker's verdict stand in for the run on the merged result.** Neither tested the merged tree, the first is the pusher's word, and the second belongs to one person's network and cannot answer for other contributors. Rejected.
- **Keep every run whole at every point.** Safe, and it is what makes a change wait through the same tests three times. Rejected as the only choice; it stays the answer for a change from outside the project and whenever a plan cannot be proven.

## Consequences

- The earlier runs can be cut to what gives an early answer, per change, without a trust decision each time: the proposal run of a project's own contributors keeps its quick gates, and the long test stages wait for the landing.
- A fault the earlier runs would have caught is now found at the landing, where it delays the changes waiting behind it. That cost is bounded by a limit each project can set, past which the earlier run comes back by itself (itd-2611241456272313).
- A repository whose forge tests no merged result has no such run until abcd's own landing check supplies one, and until then no earlier run is cut there.
- A hosted workflow that skips a declared check on the landing event is a defect, however green it reports. A test reads the workflow and fails on it, and the brief gains the rule as an invariant in the first change that builds on this record.
- The landing run still executes the hosted definition as the change itself writes it, so a change that edits that definition can weaken its own run. The workflow audit and review hold that today; reading the declaration from the default branch closes it for the checks abcd runs itself.
