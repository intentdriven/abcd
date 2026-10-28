---
id: itd-2610222215485265
slug: abcd-drain-run-from-a-plain-terminal
spec_id: spc-2610230813348813
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610222000271964]
related_intents: [itd-2610161621364075, itd-2610170634394993, itd-2610100704270254]
refines: [itd-2610161016401115]
origin: extracted-from-record
production_mode: hand-written
impact: additive
---

# abcd drain --drive works through the ledger from a plain terminal

## Press Release

> **Type `abcd drain --drive` in a plain terminal and it works through the ledger on its own.** Frank opens a terminal on a spare machine and types `abcd drain --drive`. Before it touches anything, abcd tells him what it will do: work through up to eight issues, start Claude sessions on his own subscription, walled off from his own setup, about forty agent runs at most. It asks him once. He says yes. The terminal then shows one line per step: an issue judged, a lane opened, its checks run, its pull request queued. Then it moves to the next. It stops at the cap, when nothing eligible is left, or when Frank presses Ctrl-C, and it tells him what it left and how to pick it up again. Whatever it cannot finish, it hands back with the reason. It never leaves an issue held while it waits for someone who isn't coming.
>
> _Facilitator-proposed on 2026-10-22 from iss-2610222000271964 and revised after the Opus and Sonnet review round 1 the same night; signed off by the product thinker on 2026-10-23 (dcn-2610230813064058)._

## Why This Matters

On 2026-10-22, after the crash, the product thinker typed `abcd drain` in a plain terminal on a second machine. The drain reserved the first of 213 eligible issues, wrote a judgement request, and waited for an agent session that never came. It opened no lanes, and the issue stayed reserved for two hours (iss-2610222000271964).

The shipped drain bundle (itd-2610161016401115) promised a drain that "runs unattended on a hosted session or a plain terminal for a day". It delivered a judge routable to a runner, and lane roles that `implement step` starts through a runner. But a host session still drives every step: each `abcd drain` is one move, a lane's preflight is run by the host, and a runner that fails hands its role to the host. In a plain terminal there is no host, so the first step that needs one waits for good. This intent closes that gap, so it `refines` the shipped one rather than reopening it.

Doing so reverses the host-delegated boundary for an opt-in path. itd-2609201916151817's decision 5 flagged that reversal and the product thinker confirmed it for the opt-in path. The record it owes is adr-2610222222326104, accepted at this intent's sign-off on 2026-10-23 (dcn-2610230813064058).

## Mechanism

_Facilitator-proposed; signed off by the product thinker on 2026-10-23 (dcn-2610230813064058)._

We expect a terminal drive to land work with no person and no host session, because each thing a host does between steps becomes an act the driver does itself, or a hand-back:
- it calls the next move;
- it runs a lane's preflight behind the load gate;
- it starts each role through the person's runner;
- it hands back what fails.

The claim is shown wrong if a terminal drive on a scratch repository seeded with three minor issues, on a machine with the runner set up, needs any input after its one yes before it reaches its cap, or ends with an issue still reserved and no lane working on it.

## Scope Conditions

_Facilitator-proposed; signed off by the product thinker on 2026-10-23 (dcn-2610230813064058)._

- Holds on a machine where the person has enabled the claude runner (`runner.claude`) and logged in once to the walled configuration directory abcd uses for drives, or has set an API key for drives in their configuration (dcn-2610230716408975, narrowing ruling RN2 for this route). <!-- cond: cond-2610230813347476 -->
- Holds only while every wall on the subscription route holds in a live run on that machine; a drive refuses to start when any fails. <!-- cond: cond-2610230813346479 -->
- Holds for one drive per working copy, one lane at a time, as the drain's lock and its one-lane rule already enforce. <!-- cond: cond-2610230813348149 -->
- Holds while the terminal stays open, or under a tool the person chooses to keep it running (`tmux`, `nohup`). abcd starts no background service. <!-- cond: cond-2610230813342528 -->
- Once they ship, the sandbox's fence check (itd-2610161621364075) and the coordination intent's login check apply to the helpers abcd starts: such a drive holds only where its fence passes and away from the person's GitHub login. A drive the person starts and watches at the terminal may count as attended and run under their login (dcn-2610230710469123, dcn-2610230813064058). <!-- cond: cond-2610230813349250 -->

## Acceptance Criteria

_Facilitator-proposed on 2026-10-22 and revised after review round 1; walked and kept in full by the product thinker on 2026-10-23 (dcn-2610230813064058)._

- **Given** Frank types `abcd drain --drive` in a plain terminal, **when** it starts, **then** before its first move it shows the cap (eight unless `--max` says otherwise), an estimate of the agent runs that cap allows, and the runner and the route the agents will use (his subscription, walled, or the API key his configuration names), and starts nothing without his yes. Without a terminal, `--drive` alone is not enough; the scripted yes is `--drive --yes`.
- **Given** a machine where the runner is not enabled, the walled configuration directory has no login (or, on the key route, the API key is missing), the judge is routed to opencode, or the repository pins a role to the host, **when** `abcd drain --drive` starts, **then** it refuses before its first move and names what is missing. It reserves nothing.
- **Given** the subscription route, **when** a drive starts, **then** it first proves in a live run that each wall holds: the person's own instructions, memory, MCP servers and connectors, hooks and plugin hooks, and saved sessions are out of the agent's reach, and no API key is set; it refuses to start and names the wall that failed when any does. abcd never reads, stores or forwards the login token (dcn-2610230716408975).
- **Given** Frank said yes, **when** the drive runs, **then** it judges each eligible issue through the runner, opens its lane, starts each role through the runner, runs the lane's preflight itself behind the load gate, lands it, and moves to the next, with no further input. It prints one line per step and logs the detail under the local tier.
- **Given** a role whose runner fails or answers out of shape during a lane, **when** the driver meets that fallback, **then** the lane is handed back as stuck with the runner's reason, its reservation is released, and the drive moves on. No lane is left waiting for a host.
- **Given** three runner failures in a row, or a judgement whose runner fails, **when** the driver meets them, **then** it treats them as an outage: it waits 1, 5 and 10 minutes, then hourly, saying so each time. It stops after eight hours of failures, and hands no issue back for an outage.
- **Given** the drain's own waits (its pace window, its cap wait, a wait for origin), **when** a move returns one, **then** the driver sleeps until the earliest time the move named, never less than thirty seconds, and holds the drain's lock only during a move.
- **Given** a drive, **when** Frank presses Ctrl-C or the terminal closes, **then** a running agent is stopped, the open lane is left where it stands with its reservation, and the drive prints the run, the lane and the time its reservation lapses, so the next `abcd drain --drive` or any host picks it up. A second Ctrl-C exits at once.
- **Given** `abcd drain` without `--drive`, **when** it runs, in a terminal or a host session, **then** it does what it does today, one move per call. The host's page always calls it that way.

## Open Questions

_All resolved: both of the product thinker's choices were settled on 2026-10-23, and the draft was walked and signed off the same day (dcn-2610230813064058)._

- **What a drive runs on: settled** (dcn-2610230716408975, decision 10). The person's subscription by default, walled; an API key when their configuration asks for it. Whether the build runner takes the same default is iss-2610230716454430, for the product thinker.
- **Where a drive may run: settled** (dcn-2610230710469123, decision 9). Only the helpers abcd launches on its own must run away from the person's GitHub login; a drive the person starts and watches may count as attended.

## Decisions

Each is the facilitator's, on the product thinker's delegation of 2026-10-14 and 2026-10-16 for technical choices, recorded with `abcd decide` on 2026-10-22 and revised after review round 1:

1. One intent, refining the shipped drain bundle (itd-2610161016401115). The reversal of the host-delegated boundary it carries is recorded as adr-2610222222326104, accepted at the product thinker's sign-off of 2026-10-23 (dcn-2610230813064058).
2. The driver is a loop inside `abcd drain`, entered only by `--drive`, not a new verb and not a background service.
3. Driving is never switched on by detecting the absence of a host. `--drive` is the only way in, so a host session or a terminal inherited from one never starts a drive by accident.
4. Consent comes before the first move, names the cap, the estimated agent runs, the runner and its route (the walled subscription or the API key, per decision 10), and is never remembered. `--drive --yes` is the scripted yes. The default cap while driving is eight, the cap the product thinker's own drains use.
5. Roles the person left unrouted run on `runner.fallback_host`, the existing key for a run with no host session, or on the claude runner for this drive alone when the person's yes names it. A role the repository pins to the host refuses the start.
6. The driver runs the lane's preflight itself, through `abcd implement load --wait`, when the landing waits for a receipt.
7. A runner fallback while driving hands the lane back as stuck. Runner failures in a row are an outage the driver waits through, never a hand-back.
8. Ctrl-C or a hang-up stops the running agent and leaves the open lane resumable with its reservation, naming when the reservation lapses. Nothing is handed back for an interruption.

The product thinker's, on 2026-10-23:

9. **Only abcd's own helpers run away from the person's login (narrowing C1).** The build and drain lanes abcd launches on its own must run where the person's GitHub login is absent; the person's own sessions working while they are away keep running on their own computer under their login, and a drain the person starts and watches may count as attended (dcn-2610230710469123). It answers this intent's second choice and the coordination intent's first in one ruling.
10. **A drive runs on the person's subscription by default, walled (narrowing RN2 for this route).** After the product thinker challenged the premise that a subscription cannot run agents abcd starts, a check of the vendor's current docs found that the person may sign in to the unmodified CLI with their own subscription, and that only other apps may not use or handle that login. `--bare`, the one documented full seal, reads no subscription login. So a drive runs the person's `claude -p` on their subscription by default, behind a wall, and on an API key with `--bare` when their configuration asks for it (dcn-2610230716408975). The wall:
   - a dedicated `CLAUDE_CONFIG_DIR` the person logs into once;
   - `CLAUDE_CODE_DISABLE_CLAUDE_MDS=1` and `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`;
   - `--strict-mcp-config` with an empty `--mcp-config`;
   - `--settings` with `disableAllHooks` true;
   - `--no-session-persistence`;
   - `ANTHROPIC_API_KEY` unset.

   abcd never reads, stores or forwards the login token. The docs are silent on three edges (`settings.json` under `--bare`, claude.ai connectors under `--strict-mcp-config`, and plugin hooks under `disableAllHooks`), so the walls are proved live before anything else is built, and a drive refuses to start when any fails. The limit that a person with only a subscription cannot drive is dropped. The build runner keeps RN2's key-only rule until the product thinker answers iss-2610230716454430.

The product thinker's walk and sign-off of 2026-10-23 (dcn-2610230813064058):

11. **Signed off as written.** The six stories stand, both parts of the technical list are accepted, and the press release, the mechanism, the scope conditions and the reason for doing it now are signed off with no wording change. With the plan, adr-2610222222326104 is accepted, and AGENTS.md's host-delegated boundary names it as the one written exception, not built until this intent ships. The scope condition now says, as the product thinker was shown, that the fence and the login check apply to the helpers abcd starts.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a terminal drive lands work with no person and no host session once each thing a host does between steps becomes the driver's own act or a hand-back; shown wrong if a drive on a scratch repository seeded with three minor issues needs any input after its one yes before its cap, or ends with an issue still reserved and no lane working on it. Occasioned by 2026-10-22, when a drain typed in a plain terminal reserved the first of 213 issues, opened none, and held it for two hours (iss-2610222000271964).
