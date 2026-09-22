---
id: itd-2609201916056194
slug: abcd-runs-a-delegated-agent-through-a-command-line-model-run
spec_id: spc-2609221533057881
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609170822093401]
related_intents: [itd-22, itd-2609081951381895, itd-6, itd-51]
supersedes: [itd-2]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_adrs: [adr-2609221009491186]
---

# A role runs through the command-line harness the operator chose, and every fall back to the host is recorded

## Press Release

> **Any role can run through a command-line harness the operator names, with the same brief, contract and transcript store as the host's own sub-agent; where it cannot, the host takes it and the record says so.**
>
> "My reviews were the scarcest thing in a run, and a second harness was sitting on the machine doing nothing," said a technical facilitator. "Now a role names its runner, the review goes out through it, and the receipt reads the same as one the host ran. When the runner is not there the host picks it up, and the run tells me how often that happened, which is how I learned which roles to move."

## Why This Matters

The loop's process driver names a command-line rung that nothing reads. Every harness checked on 2026-09-22 can be driven headlessly with a prompt in, structured events out, an exit code and a non-interactive permission switch: Claude Code's print mode, opencode's run mode and its server, Codex's exec and app-server, Gemini's headless mode, Cursor's CLI, Aider's message mode. What is missing is abcd's side: one runner interface, a route per role, and the discipline that a fallback is evidence rather than a silent recovery. Two hazards are recorded: a print-mode run without the bare flag executes the target repository's hooks and configured servers with no trust dialog, and a runner's model route must obey the provider allowlist.

## Mechanism

We expect a role routed through another harness's command line to land in the record indistinguishably from a host-run one, because the runner is handed the same brief and held to the same output contract; shown wrong if a reader cannot tell which route ran from the receipt alone, or if no role is routed off the host in a release of it shipping.

## Scope Conditions

- Holds for a harness with a documented non-interactive mode that takes a prompt, returns structured output and an exit code, and grants permissions without a prompt. <!-- cond: cond-2609221533059253 -->
- Holds where the runner's binary is on the machine and its own credential is already configured; abcd never logs a harness in. <!-- cond: cond-2609221533051474 -->
- Holds under adr-2609221009491186: a runner's model route is one its provider's allowlist admits. <!-- cond: cond-2609221533057668 -->

## What's In Scope

- **The route per role**: `roles.<role>.runner` names `host` (the default) or a configured runner; nothing changes for a role left unset.
- **The runner interface**: the same brief, inputs and output contract the host sub-agent gets; the answer validated the same way; the transcript captured into abcd's own store, whatever the harness keeps of its own.
- **The shipped runners**: the claude CLI in print mode with the bare flag (so a target repository's hooks and servers do not run untrusted) and opencode, through its run mode or its server; a third is configuration of the same interface where the harness's shape allows.
- **The fallback, and its record**: an unavailable or failing runner hands the role to the host, which is the default route, and where abcd runs as the binary with no host session the fallback is a host the operator configured (their main connector, the claude CLI for example) so there is always a landing. Every fallback writes a receipt naming the role, the runner asked for, the reason, and the route that ran.
- **The intel**: the run's summary reports the fallback count per runner and per role, so a route that never works is visible without reading transcripts.
- **The allowlist**: a runner's model route is resolved against its provider's list before the lane starts.
- **The first proof**: a ruthless review of a real lane sent through the opencode runner from a lane the host drives, its receipt indistinguishable from a host-run review but for the route it names.
- **Security review** before it ships: it starts processes with a prompt and a repository path.

## What's Out of Scope

- Choosing the runner for a role (the model tier's, itd-2609170822093401).
- Installing or authenticating a harness (itd-63 explains; the person installs).
- A served console over the loop (reframed as the operator console, research note of 2026-09-22).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609221009491186 records the vocabulary rulings it rests on):

1. The host is the default route; a runner is opt-in per role (adr-25).
2. Where abcd runs with no host session, the fallback is a host the operator configured, never nothing (ruled 2026-09-22).
3. Every fallback is recorded as intel: a receipt per event and a count per runner and per role in the run record (ruled 2026-09-22).
4. The claude CLI runner passes the bare flag, so a target repository's hooks and configured servers do not run untrusted.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a role whose runner is set to a configured command-line harness, **when** the lane reaches that role, **then** the runner gets the same brief, inputs and output contract the host sub-agent would get, its answer is validated the same way, and its transcript lands in abcd's own store.
- **Given** a role left unset, **when** the lane reaches it, **then** the host runs it and nothing differs from today.
- **Given** a runner that is absent, refuses or fails, **when** the lane reaches that role, **then** the host runs it instead; with no host session the configured fallback host runs it; and a receipt names the role, the runner asked for, the reason and the route that ran.
- **Given** a completed run, **when** its summary is read, **then** it reports the fallback count per runner and per role.
- **Given** a runner whose model route is not on its provider's allowlist, **when** the lane is about to start, **then** it is refused before the runner is launched.
- **Given** the claude CLI runner, **when** it launches, **then** it passes the bare flag, so the target repository's hooks and configured servers do not run untrusted.
- **Given** a review run through the opencode runner, **when** its receipt is read beside a host-run review's, **then** the two differ only in the route named.
- **Given** the lane, **when** it ships, **then** a security review of the process launch, the repository path and the permission flags is on its record.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the goal is abcd in the preferred harness with the others reachable, and the run's reviews are its scarcest step; we expect a review sent through a second harness to be indistinguishable in the record; shown wrong if a reader cannot tell the route from the receipt alone, or if no role is routed off the host in a release
