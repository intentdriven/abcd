---
id: adr-2611241456294452
slug: an-independent-checker-is-optional-its
status: accepted
date: 2026-11-24
supersedes: null
superseded_by: null
related_intents: [itd-2611011325430933, itd-2611241456272313, itd-2610261025516272]
related_rfcs: []
related_adrs: [adr-2611241456284353, adr-2611011327164895, adr-2610120937435856]
---

# ADR-2611241456294452: An independent checker is optional: its verdict is reported, never required and never waited for

## Context

A person whose agents write the work wants it checked by something the agents do not control, and often owns a spare computer that could do it, sometimes one that runs an operating system the project ships to and the person does not work on. itd-2611011325430933 gives that machine the job: it fetches each pushed change by itself, runs the checks the default branch declares, and reports under an identity whose key stays on the machine.

The first plan made its verdict a required check, with landings held until it answered. A machine in someone's home is switched off, asleep or busy some of the time, and a required check that does not report drops every waiting change at the forge's timeout. The product thinker ruled on 2026-11-24 that the machine is an optional enhancement: abcd looks for it, uses it when it is there, ignores it and says so when it is not, and it never holds up development. Several such machines may be known, and the first one free takes a change.

One rule was already fixed: the check of the merged result is the one run a landing rests on (adr-2611241456284353). The same ruling keeps whatever a person's machines say to each other on the person's own network, with no relay anywhere else.

## Decision

We will treat an independent checker as optional everywhere. Its verdict is reported and never required: abcd writes no forge rule that requires it, no landing waits for it, and a change it has not reached lands and is checked afterwards. A missed or late check is named in the landing's report, with the machine and how far behind it is, and a failure found after a landing is reported as soon as it is known.

A checker takes no work and no instruction from the machines whose work it checks. It learns of changes from the shared remote, reads the checks to run from the default branch, and reports through the forge under its own identity. The one thing it answers on the network is a fixed question about its own state (reachable, free or busy, how far behind), on the local network only. A verdict is believed only from the checker's forge identity, never from that answer.

A person may run several checkers. They are found by a list the person keeps on their own machine, or, where the person switches it on, by discovery on the local network. No repository file names a host. A checker claims a change before it runs it, so the first one free takes it and no two run the same commit.

A checker's verdict never stands in for the check of the merged result. It may stand beside a landing as the statement that something other than the lander's own machine judged that exact commit.

## Alternatives Considered

- **Optional, reported, never waited for (chosen).** Development never depends on one machine in a person's home, and a fault only that machine can see is still found the same hour and not the same week. Its cost, stated plainly: a faulty change can already be in the default branch when the checker reports it.
- **A required check from the checker's identity, with landings held.** Nothing lands without the second opinion. One unplugged cable stops every landing, and on a forge queue a silent checker drops entries on the shared path. Ruled out by the product thinker.
- **Wait for a set time, then land with a note.** Keeps the protection on ordinary days. It still puts a timer on every landing for a machine that may be off, and adds a second behaviour to explain. Not taken; the report of a late verdict covers the same ground without the wait.
- **The working machine hands changes to a checker it finds.** Simple to pool. It gives the agents a way to choose what the checker runs, which is the one thing the checker exists to deny. Rejected: checkers pull, and a claim among themselves decides who runs what.

## Consequences

- No rule on the forge names the checker, so setting one up or switching one off changes nothing about how a change lands.
- The checker cannot be the reason a hosted run is removed where a landing depends on that run. It adds a point to a change's plan (itd-2611241456272313) and coverage of an operating system no other run reaches per change.
- Every surface that reports a landing must say which of three things stood behind it: the check of the merged result, an independent verdict on that exact commit, or the lander's own machine alone.
- Discovery is opt-in, bound to the local network, and carries no verdict. A person who wants no listening service keeps a declared list or none.
- The upgrade path adr-2611011327164895 names stays open: a person moves from a self-attested result to an independent one by setting up a checker, and nothing else about landing changes.
