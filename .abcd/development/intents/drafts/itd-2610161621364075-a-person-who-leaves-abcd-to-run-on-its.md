---
id: itd-2610161621364075
slug: a-person-who-leaves-abcd-to-run-on-its
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609091014076309]
related_intents: [itd-148, itd-2609221017023290]
related_adrs: [adr-2610161621415708, adr-42, adr-2609091248200336]
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The agents abcd launches on its own work inside a sandbox the operating system enforces

Typed links: `builds_on` [itd-2609091014076309](../planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md) (the worktree store, which gives every launched agent the separate checkout this sandbox confines it to); `related_intents` [itd-148](../planned/itd-148-every-change-starts-in-its-own-worktree.md) (the read-only primary checkout, the same isolation enforced on sessions rather than on launched agents) and [itd-2609221017023290](../shipped/itd-2609221017023290-abcd-keeps-every-external-credential-the.md) (the credential store, whose keychain home a sandboxed launch must still reach); `related_adrs` [adr-2610161621415708](../../decisions/adrs/2610161621415708-abcd-s-command-guard-steers-agents-an-os.md) (the proposed rule this intent enacts), [adr-42](../../decisions/adrs/0042-guard-parse-layer-is-a-mistake-filter.md) (the guard is a mistake filter, and a boundary needs an execution-layer control) and [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user.md) (machine-scoped scratch, where the worktrees this sandbox opens live). Related issues, named here rather than in `related_issues` because none of them is promoted into this intent: [iss-2610161612203555](../../../work/issues/open/iss-2610161612203555-an-agent-abcd-starts-without-a-terminal.md) (restores abcd's guard to each launch; this intent depends on it, see Open Questions), and [iss-2608230847432285](../../../work/issues/open/iss-2608230847432285-per-agent-worktrees-do-not-isolate-a.md) with [iss-213](../../../work/issues/open/iss-213-several-agents-sharing-one-git-worktree.md) (worktrees alone did not isolate sessions; itd-148 carries both).

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

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

1. **Split into three parts, or keep one record?** The two adversarial reviews proposed, and this draft does not yet adopt, a split:
   - (A) the agents abcd's runner launches run inside the sandbox: this draft's capability as written;
   - (B) the drains and builds a person's own session drives report how isolated they are, before they start, since abcd cannot fence a session it did not launch;
   - (C) network access limited to a declared list of hosts, as a later intent (Open Question 3).
   Whether B is a second intent, a criterion of this one, or dropped, and whether C is filed now or later, is the product thinker's call in the planning interview.
2. **Who is covered?** As written: an agent abcd itself starts, for a role the person routed to a runner on their own machine. Not covered: the person's own interactive session, a drain or build that session drives on the host route (the default), and any agent the host starts as a sub-agent. Does the press release's "the agents it starts on its own" read that narrowly to the person, or does it promise B as well?
3. **"Can't reach other sites" or "can't send your code elsewhere"?** The honest network claim is "network limited to a declared list of hosts". It narrows mistakes and is not a barrier against sending code out: allowing the forge still allows a push to any repository the credential reaches, and the host's own sandbox documentation warns that broad domains such as `github.com` leave exfiltration paths. On the platforms themselves, Seatbelt cannot name hosts, Landlock limits by port only, and bubblewrap's network is all or nothing, so host filtering needs a proxy running outside the sandbox. Which promise does the product thinker want made, and does it belong to part C?
4. **Which mechanism?** Candidates, none chosen:
   - a whole-process wrapper at the runner's launch point, around the launched agent: Anthropic's sandbox-runtime (`srt`; Seatbelt on macOS, bubblewrap with a host-side proxy on Linux; Apache-2.0; a beta research preview, and a new dependency that needs explicit sign-off), or abcd's own Seatbelt and bubblewrap or Landlock profiles. It covers the file tools, hooks and servers the agent runs, not only its shell commands;
   - the host's own sandbox, passed for one launch with `--settings`, which leaves the person's settings untouched but covers shell commands only: the file tools, hooks and servers run outside it. A possible inner layer, not the boundary on its own;
   - a container, a virtual machine, or a cloud session: the strongest separation, and the heaviest, tying abcd to a container runtime or a hosted service and making the worktree and receipt folder harder to share.
   Sources the reviewers read: https://code.claude.com/docs/en/sandboxing, https://code.claude.com/docs/en/sandbox-environments, https://code.claude.com/docs/en/headless, https://github.com/anthropics/sandbox-runtime, https://docs.kernel.org/userspace-api/landlock.html.
5. **What may a launched agent write?** The writable set is derived from the run, not "the worktree plus abcd's home". The reviewers' derivation of what a lane needs:
   - its own worktree;
   - its receipt folder in the main checkout's local tier, `.abcd/.work.local/run/<run-id>/<lane>/`, with the run's `state.json` readable and not writable;
   - the main checkout's `.git`, without `hooks/` and `config`, since a worktree's commits land there;
   - the Go caches, either read-only or one per run;
   - the temporary directory;
   - the host's own state, with the paths it loads configuration from held read-only.
   abcd's home as a whole stays read-only to the agent: it holds `config.json` (the person's runner routes), `rules.json`, `trusted-roots` and other repositories' worktrees, and an agent that could write them would change the policy of the next launch. Does this list stand, and what does a repository whose work needs more declare?
6. **A machine with no sandbox: stop or warn?** Before the first launch abcd says the sandbox is missing; does the run then stop, or go ahead with the warning? If it stops by default, the waiver that lets it run belongs to the person alone, in their user layer under `~/.abcd.noindex/` as `trusted-roots` does, and never in a file inside a repository, which could vouch for itself.
7. **Scope conditions the reviewers expect, to confirm at planning.** Each is a place the sandbox may break a run or mislead abcd:
   - inside a sandbox `ps` sees only the sandbox's own processes, so abcd's machine-load gate and any `pgrep` waiter would read a busy machine as clear: the gate runs in abcd before the launch, outside the sandbox;
   - Go tools may fail TLS verification under Seatbelt (reported by the review, not yet reproduced here);
   - git over SSH fails under the host-side proxy (the host's documentation says plain `ssh` cannot connect even to an allowed host), so a launch pushes over HTTPS;
   - the keychain credential helper, and the credential store's keychain home, stay reachable from inside;
   - `sandbox-exec` is deprecated on macOS 27 (reported by the review), though custom profiles still run, so a macOS mechanism built on it carries a removal risk;
   - a nested bubblewrap fails, so a launch inside a container or another sandbox needs its own path;
   - Ubuntu 24.04 and later restrict unprivileged user namespaces by default, which blocks bubblewrap until the person allows it.
8. **Depends on the guard reaching each launch.** abcd's guard keeps teaching inside the sandbox only once iss-2610161612203555, being built now, restores it to each launch; today the launch loads no hooks. This is recorded in words rather than as `blocked_by`, which names intents only. Should planning wait for that issue to resolve, or plan now and build after?

## Decisions

Ruled by the product thinker on 2026-10-16, in the planning interview:

1. **Split into three parts.** (A) The agents abcd's runner launches run inside a sandbox the operating system enforces; this record carries part A. (B) Drains and builds the person's own session drives report how isolated they are, since abcd never changes the person's host settings and cannot fence that session. (C) Limiting which hosts a run may reach is a later, separate feature. Parts B and C are filed as their own drafts before planning goes on.
2. **Kept apart from the worktree store, building on it.** The store (itd-2609091014076309) says where a launched agent's copy lives; this record says what the agent may reach from it. This one waits for the store to ship.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
