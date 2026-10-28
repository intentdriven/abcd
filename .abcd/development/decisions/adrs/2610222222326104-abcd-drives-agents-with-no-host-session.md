---
id: adr-2610222222326104
slug: abcd-drives-agents-with-no-host-session
status: accepted
date: 2026-10-22
supersedes: null
superseded_by: null
related_intents: [itd-2610222215485265, itd-2609201916151817, itd-2609201916056194, itd-2610161016401115]
related_rfcs: []
related_adrs: [adr-25]
---

# ADR-2610222222326104: abcd drives agents with no host session only on the person's own invocation, on the person's own account behind a wall, and hands back what it cannot finish

## Context

[adr-25](0025-host-delegated-llm-default.md) makes the language model host-delegated by default: abcd prepares the work, and the person's host session starts the agents. Runners (itd-2609201916056194) are opt-in adapters a role is routed to. A host session still drives every step today. `abcd implement step` and each `abcd drain` move return to the host between steps.

itd-2609201916151817's decision 5 flagged a process driver as a reversal of that boundary. In that driver, abcd runs alone and starts every agent itself. The product thinker confirmed it for the opt-in path only, and the decision asked for a record before any such path ships. `drive.go`'s header says the same: the no-host path owes this record.

On 2026-10-22 the product thinker asked for exactly that path: typing `abcd drain` in a plain terminal should work through the ledger by starting Claude sessions itself (iss-2610222000271964, itd-2610222215485265). Two earlier rulings bound it:
- **RN2 (2026-10-02):** the claude runner runs in print mode with the bare flag on an API key, so a role reaches a runner only on the person's own route, and a subscription account runs roles in its own session. Its grounds are Claude Code's CLI reference and legal and compliance pages. For this drive the product thinker narrowed it on 2026-10-23 (dcn-2610230716408975), after a check of the vendor's current docs found that the person may run the unmodified CLI on their own subscription and that only other apps may not use or handle that login: a drive runs on the person's subscription by default, behind a wall, and on an API key with the bare flag when their configuration asks for it.
- **Coordination C1 (2026-10-22):** unattended runs happen only where the person's forge login is absent. On 2026-10-23 the product thinker narrowed it to the build and drain lanes abcd launches on its own; a drive the person starts and watches may count as attended (dcn-2610230710469123).

## Decision

abcd drives agents with no host session only under four conditions.

1. **On the person's own invocation.**
   - A no-host drive starts only from a command the person typed in a terminal, after a yes given at that terminal, or with an explicit flag that is the scripted yes.
   - The yes covers that one drive and is never remembered.
   - A repository's configuration can never start one or widen one.
2. **On the person's own account, behind a wall, through their own route.**
   - Every agent a no-host drive starts runs through a runner the person enabled on their own machine: the role's own route, or `runner.fallback_host` for a role they left unrouted.
   - It runs the person's own unmodified CLI on their subscription by default, walled off from their own setup by a dedicated configuration directory they log into once and the switches that keep their instructions, memory, tools, hooks and saved sessions out of reach, with no API key set; or on an API key with the bare flag when their configuration asks for it (dcn-2610230716408975). abcd never reads, stores or forwards the login token.
   - Before the first move the drive proves each wall in a live check and refuses to start, naming the wall, when any fails.
   - The yes names the route, the cap on the work, and an estimate of the agent runs it allows.
   - A role a repository pins to the host has nowhere to run without a host, so the drive refuses to start and names that role.
3. **Bounded.**
   - A no-host drive has a cap, a default one when none is given.
   - It ends at the cap, at an empty queue, or on the person's signal.
   - Every wait inside it has a deadline (adr-2610222026262980, rule 1).
4. **What it cannot finish goes back.**
   - A role that falls back with no host to take it is a step the drive cannot finish, so its work is handed back with the reason.
   - A run of runner failures in a row is treated as an outage, which the drive waits through and then stops. It never hands work back permanently for a failure that may be passing.

## Alternatives Considered

- **A no-host drive on these four conditions (chosen).** It gives the person what they asked for. It keeps the spend theirs and visible, on their own account. It keeps a repository from starting one. And it never leaves work held for a host that is not coming.
- **Keep every agent host-started, and tell a terminal user to open a Claude session.** Rejected: this is what iss-2610222000271964 found, a drain that reserved an issue and waited for two hours.
- **An API key only, as RN2 ruled.** Rejected for this drive on 2026-10-23: the vendor's current docs allow the person's own unmodified CLI on their own subscription, and a person with only a subscription would otherwise have no terminal drive. The key stays available by configuration.
- **The subscription with no wall.** Rejected: the agent would read the person's own instructions, memory, tools and hooks, and the bare flag, the one documented full seal, reads no subscription login.
- **A background service that keeps drives running.** Rejected: no process must stay alive (dcn-2610180720366282). The terminal session is the drive's lifetime.

## Consequences

- `abcd drain` gains a driving mode. `abcd build` can adopt the same four conditions later, under this record.
- AGENTS.md's host-delegated boundary gains its first written exception. The boundary's text names this record from its acceptance at the product thinker's sign-off of itd-2610222215485265 on 2026-10-23 (dcn-2610230813064058), and says the exception is not built until that intent ships.
- A person on a subscription alone can run a no-host drive, behind the wall. If a wall cannot be made to hold in its live check, the subscription route goes back to the product thinker before anything else is built.
- The sandbox's fence (itd-2610161621364075) applies to every agent such a drive starts, since each one goes through the runner's one launch point. So does the coordination intent's login check, for the helpers abcd launches on its own; a drive the person starts and watches may count as attended.
