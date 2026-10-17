---
id: itd-2610161621364075
slug: a-person-who-leaves-abcd-to-run-on-its
spec_id: spc-2610161751056096
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091014076309]
related_intents: [itd-148, itd-2609221017023290]
related_adrs: [adr-2610161621415708, adr-42, adr-2609091248200336]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
refines: [itd-148]
---

# The agents abcd launches on its own work inside a sandbox the operating system enforces

Typed links: `builds_on` [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a.md) (the worktree store, which gives every launched agent the separate checkout this sandbox confines it to); `related_intents` [itd-148](itd-148-every-change-starts-in-its-own-worktree.md) (the read-only primary checkout, the same isolation enforced on sessions rather than on launched agents) and [itd-2609221017023290](../shipped/itd-2609221017023290-abcd-keeps-every-external-credential-the.md) (the credential store, whose keychain home a sandboxed launch must still reach); `related_adrs` [adr-2610161621415708](../../decisions/adrs/2610161621415708-abcd-s-command-guard-steers-agents-an-os.md) (the proposed rule this intent enacts), [adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md) (the guard is a mistake filter, and a boundary needs an execution-layer control) and [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user.md) (machine-scoped scratch, where the worktrees this sandbox opens live). Related issues, named here rather than in `related_issues` because none of them is promoted into this intent: [iss-2610161612203555](../../../work/issues/open/iss-2610161612203555-an-agent-abcd-starts-without-a-terminal.md) (restores abcd's guard to each launch; this intent depends on it, see Open Questions), and [iss-2608230847432285](../../../work/issues/open/iss-2608230847432285-per-agent-worktrees-do-not-isolate-a.md) with [iss-213](../../../work/issues/open/iss-213-several-agents-sharing-one-git-worktree.md) (worktrees alone did not isolate sessions; itd-148 carries both).

## Press Release

> **When you leave abcd working through the night, the agents it starts on its own can change their own work and nothing else on your computer.** Each agent abcd launches runs inside a fence your computer's operating system enforces, not only abcd's own checks. The agent can edit the copy of the project it was given and the notes abcd keeps for that piece of work. Your other projects, your settings, and the choices you made about how abcd runs stay out of its reach. On a computer that cannot put up the fence, abcd says so before the first agent starts, so you never find out in the morning.
>
> "I used to leave a run going and wonder over breakfast what else it had touched," said Maya, an autonomous-development practitioner. "Now the agents abcd starts can only change the work they were given. The one night my laptop couldn't fence them in, abcd told me before it began, and the choice was mine."

## Why This Matters

abcd now starts agents that nobody watches. When a person routes a role to a runner on their own machine, abcd launches that agent itself, without a terminal, with its tools granted up front because no one is there to answer a prompt. Two things stand between such an agent and the rest of the computer today: the list of tools it was granted, and abcd's command guard. Neither confines what an allowed tool touches. The guard is a mistake filter, not a boundary: [adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md) decided so, and the guard's chapter of the brief says that anything needing an enforced boundary needs a control at the execution layer, "a sandbox, a permission system, a restricted shell", with the guard in front of it to teach. Both recorded the principle; no record enacted it. And a launched agent currently carries no guard at all, because the launch loads no hooks (iss-2610161612203555 restores it).

The vendors whose agents abcd sits beside draw the same line, as the research note of 2026-08-18 on the guard's parse-layer limits records. Claude Code documents its command patterns as steering and says its sandbox holds even when a prompt injection gets past the model; Codex keeps an OS-enforced sandbox mode apart from the approval policy layered on top; GitHub Copilot's coding agent runs in a throwaway, firewalled environment rather than behind a command list. Cursor's history shows why the line matters: its command denylist, and then the allowlist that replaced it, were both bypassed. Claude Code's own guide to sandbox environments is blunter for the unattended case: a run without permission prompts belongs inside a container, a virtual machine or a whole-process sandbox, and its shell-only sandbox "is not sufficient for fully unattended runs".

The product thinker named sandboxing these runs an important feature on 2026-10-16. The decision recorded earlier that day, dcn-2610161612269647, says abcd "does not sandbox"; dcn-2610161619296615 corrected it, because the question that day had dropped the sandbox option to fit the screen and no ruling was made on it.

What this intent can honestly promise is bounded by where abcd starts processes. abcd launches a process only for a role the person routed to a runner, through one launch point in its runner. The drains and builds a person's own session drives run on that session, under the host's settings, and abcd never edits those settings, so for them abcd can detect and report how isolated the run is but cannot fence it. The network is a separate question: limiting it to a list of hosts narrows mistakes but does not stop code being sent elsewhere, so it is proposed as its own later part (Open Questions 1 and 3).

## Mechanism

We expect the fence to stop an agent abcd launches from changing anything outside its own work, because the operating system enforces it below anything the agent, or a command the agent runs, can change; abcd's guard still steers the agent inside it. It would be shown wrong if a launched agent, with the fence on, changed a file outside the paths its run was given. (Facilitator-proposed, 2026-10-16.)

## Scope Conditions

- macOS and Linux; Windows is out. <!-- cond: cond-2610161751057372 -->
- Agents abcd's runner launches, not a session the person drives. <!-- cond: cond-2610161751059926 -->
- Local runs on the person's own computer, not cloud sessions. <!-- cond: cond-2610161751054152 -->
- One project per run. <!-- cond: cond-2610161751050328 -->
- The operating system's fence can start: on Linux, unprivileged user namespaces are not blocked. <!-- cond: cond-2610161751054881 -->
- The machine-load gate and every check that reads the whole machine run in abcd, outside the fence. <!-- cond: cond-2610161751056717 -->

## Acceptance Criteria

Seeded by the facilitator on 2026-10-16; proposals until the product thinker confirms them.

- **Given** a run whose agent abcd launches through its runner, **when** the agent tries to write a file outside its worktree and its run's receipt folder, **then** the operating system refuses the write and the run records the refusal.
- **Given** the same agent, **when** it edits files in its own worktree and commits there, **then** the edit and the commit succeed, and so does pushing its branch to origin.
- **Given** a launched agent, **when** it tries to change the person's abcd settings, rules, trusted roots or another project's worktree under `~/.abcd.noindex/`, **then** the write is refused.
- **Given** a computer on which the fence cannot start, **when** a run is about to launch its first agent, **then** abcd refuses to start the run, names the reason and how the person may turn the fence off for their own computer, and launches nothing.
- **Given** a person who has turned the fence off for their computer in their user layer, **when** a run starts, **then** abcd says once that its launches are unfenced, and a project's own files can never turn the fence off.
- **Given** a fenced run, **when** its lane builds and tests the project, **then** it uses a fresh Go build cache of its own and leaves the person's caches unchanged.
- **Given** a fenced run on a loaded machine, **when** abcd's load gate decides whether a preflight may start, **then** the gate reads the whole machine from outside the fence and is not misled by the fence's own view.
- **Given** abcd's guard is active in a launched agent, **when** the agent runs a command the guard refuses, **then** the refusal and its advice reach the agent inside the fence as they do outside.
- **Given** a fenced run, **when** it finishes, **then** the person's host settings are byte-identical to before the run.

## Open Questions

None open. Each question the two reviews raised is settled in Decisions 1 to 8. How opencode launches are fenced is left to the spec, under decision 6.

## Decisions

Ruled by the product thinker on 2026-10-16, in the planning interview:

1. **Split into three parts.** (A) The agents abcd's runner launches run inside a sandbox the operating system enforces; this record carries part A. (B) Drains and builds the person's own session drives report how isolated they are, since abcd never changes the person's host settings and cannot fence that session. (C) Limiting which hosts a run may reach is a later, separate feature. Parts B and C are filed as their own drafts before planning goes on.
2. **Kept apart from the worktree store, building on it.** The store (itd-2609091014076309) says where a launched agent's copy lives; this record says what the agent may reach from it. This one waits for the store to ship.
3. **The fence refines the read-only primary checkout (itd-148) for launched agents.** itd-148 states the rule, that the main copy of the files is not edited outside a declared coordinator, and its check guides every session; this record is that rule's enforcing half for the agents abcd launches, so a change to the rule changes both.

4. **Who is covered and the network limit are settled by the split (facilitator, recorded without a question).** Part A covers only the agents abcd's runner launches; the person's own session is part B's; the network promise belongs to part C. No question was put because ruling 1 leaves one defensible answer for each.
5. **Planned now, built after its two dependencies (facilitator, recorded without a question).** It builds on the worktree store, and the guard reaches each launched agent only once iss-2610161612203555 lands; planning does not wait on either, building does. One defensible answer, so no question was put.

6. **Which fence (Q8).** While abcd launches Claude, through the Claude Code harness or the claude CLI, the fence is Anthropic's own. The product thinker prefers a state-of-the-art open-source fence abcd adopts, and abcd does not build its own, which would be beyond what abcd is. The choice is revisited when abcd supports another harness (Codex, for one), and how opencode launches are fenced today is open. Anthropic's sandbox-runtime is itself open source (Apache-2.0) and wraps any process, so planning checks first whether it already meets the preference for every harness, against the other open-source options, before anything is built.
7. **A computer that cannot put up the fence (Q9): the run stops until the person acts.** The person can turn the fence off for their own computer only, in their user layer under `~/.abcd.noindex/`, never in a project.
8. **Go build caches inside the fence (Tech Q1): a fresh cache per run.** A run cannot plant anything in the person's later builds, at the cost of cold builds and race runs.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: abcd is starting to run unattended across several accounts at once, and the agents it launches today reach anything the person's account can write; it would be shown wrong if fenced runs broke ordinary work, such as tests, git or the tool's own checks, often enough that people switched the fence off
