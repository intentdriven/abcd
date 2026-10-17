---
id: adr-2610161621415708
slug: abcd-s-command-guard-steers-agents-an-os
status: proposed
date: 2026-10-16
supersedes: null
superseded_by: null
related_intents: [itd-2610161621364075]
related_rfcs: []
related_adrs: [adr-42]
---

# ADR-2610161621415708: Unattended runs abcd launches execute inside an OS sandbox

## Context

[adr-42](0042-guard-parse-layer-is-a-mistake-filter.md) decided that abcd's command guard is a mistake filter and that anything needing an enforced boundary needs an execution-layer control behind it. This record applies that decision to the one place abcd itself now runs agents nobody watches.

abcd starts a process only for a role the person has routed to a runner on their own machine, and it does so at one launch point, `internal/core/runner/proc.go`. The launched agent runs without a terminal, in print mode with the bare flag, with its role's tools granted up front and every other tool denied, because no one is there to answer a prompt. What confines it today is that tool list and nothing else: an allowed edit or shell command reaches whatever the person's account can write. The bare flag also means the launch loads no hooks, so abcd's guard does not reach it; iss-2610161612203555, being built now, restores the guard per launch.

Everything else abcd calls autonomous runs elsewhere. A drain or build is driven by the person's own session on the host route by default (`internal/core/implement/loop/drive.go`), under the host's settings, which abcd never edits (dcn-2610161612269647). For those runs abcd can detect and report how isolated they are, and cannot fence them.

The host's own documentation sets the bar for the unattended case. Its shell sandbox covers shell commands only, while the file tools, hooks and servers run outside it, and it calls that sandbox on its own "not sufficient for fully unattended runs"; a run without prompts belongs inside a container, a virtual machine, or a sandbox around the whole process ([sandboxing](https://code.claude.com/docs/en/sandboxing), [sandbox environments](https://code.claude.com/docs/en/sandbox-environments)).

On 2026-10-16 the product thinker named sandboxing these runs an important feature (dcn-2610161619296615, which corrects dcn-2610161612269647's "does not sandbox"), and intent itd-2610161621364075 asks for it.

## Decision

We will run every agent abcd's runner launches inside a boundary the operating system enforces around the whole launched process, so that its file tools, hooks and servers are confined along with its shell commands.

We will derive what the launched agent may write from the run itself: its worktree; its receipt folder in the main checkout's local tier, with the run's state readable only; the main checkout's git directory without its hooks and configuration; the Go caches, read-only or one per run; the temporary directory; and the host's own state, with the paths the host loads configuration from held read-only. abcd's home stays readable to the agent and writable only at that agent's own worktree, so the next launch's policy (the runner routes, the rules, the trusted roots) is the person's alone to change.

We will run the machine-load gate and every other check that reads the machine in abcd, before the launch and outside the boundary, where the whole machine is visible.

We will keep abcd's guard teaching inside the boundary, once iss-2610161612203555 brings it to each launch, as the mistake filter adr-42 describes.

We will say, before a run's first launch, whether its launches will be confined, and name the reason when the machine offers no boundary. Where a person chooses to run without one, that choice lives in their own user layer under `~/.abcd.noindex/`, as `trusted-roots` does.

We will describe a launch as confined only where the boundary demonstrably runs, per `enforcement-claims-are-facts`.

We will decide limiting the network as a separate, later record.

## Alternatives Considered

- **The status quo: the per-launch tool grant and the guard.** It costs nothing and already narrows which tools a launched agent may call. Rejected because it decides which tools run and not what they reach: an allowed edit or command acts with the account's full reach, and the guard, a mistake filter by adr-42, does not reach the launch at all until iss-2610161612203555 lands.
- **A stricter guard.** adr-42 records a fail-closed allowlist as the shape a stricter guard would take, and it would tighten shell commands. Rejected as the boundary because the guard reads shell commands only, so the file tools pass it untouched, and what an allowed command does once running is outside what it reads.
- **Per-launch approval alone** (iss-2610161612203555's approval tool, backed by the guard's registry). Kept as the first step for answering the host's prompts. Rejected as the boundary because approval is a decision before each call, not a confinement during it: it does not confine the file tools, or anything an approved command goes on to start.
- **A whole-process sandbox at the runner's launch point.** Anthropic's sandbox-runtime wraps a process in Seatbelt on macOS and bubblewrap with a host-side proxy on Linux, and denies writes to `.git/hooks` and `.git/config` on its own ([sandbox-runtime](https://github.com/anthropics/sandbox-runtime)); it is a beta research preview, Apache-2.0, and a new dependency that needs explicit sign-off. abcd's own Seatbelt and bubblewrap or Landlock ([Landlock](https://docs.kernel.org/userspace-api/landlock.html)) profiles avoid the dependency and carry their maintenance. Either meets the decision; the intent chooses.
- **The host's own sandbox, passed per launch with `--settings`** ([sandboxing](https://code.claude.com/docs/en/sandboxing), [headless](https://code.claude.com/docs/en/headless)). It leaves the person's settings untouched and is maintained by the host's vendor. Rejected as the boundary because it covers shell commands only, and it is one runner's mechanism, not the other's; it remains a possible inner layer.
- **A container, a virtual machine, or a cloud session.** The strongest separation, with its own kernel in the last two. It ties abcd to a container runtime or a hosted service, and sharing the worktree and receipt folder with abcd across that boundary is harder. A candidate the intent weighs beside the whole-process sandbox.

## Consequences

- The machine-load gate and every `pgrep`-style waiter run in abcd, outside the boundary: inside a sandbox `ps` sees only the sandbox's own processes, so a check run there would read a busy machine as clear.
- A run without a boundary is the person's choice, recorded in their user layer under `~/.abcd.noindex/`; no repository file can make it for them.
- Network limiting is deferred to its own record. When it comes, its honest claim is a declared list of hosts, which narrows mistakes and does not stop code being sent elsewhere: allowing the forge allows a push to any repository the credential reaches.
- Drains and builds a person's own session drives stay unconfined by abcd; reporting their isolation is proposed as a separate part of the intent.
- The guard keeping its place inside the boundary depends on iss-2610161612203555.
- Platform limits become the intent's scope conditions: Go tools' TLS under Seatbelt, git over SSH behind a proxy, the keychain credential helper, `sandbox-exec`'s deprecation on macOS, nested bubblewrap, and Ubuntu's restriction on unprivileged user namespaces.
- A brief invariant states the rule once this record is accepted.
- The boundary sits behind a seam (`internal/core/fence`): a fence-neutral policy and one interface each mechanism implements, so the mechanism the spec picks (sandbox-runtime, `spc-2610161751056096`) can be replaced by another without changing the runner or the loop.
- The mechanism is a program abcd does not ship, and a boundary has no native default to degrade to, so on the runner route it is a requirement: [adr-22](0022-bundled-deps-as-pluggable-adapters.md)'s first declared exception, proposed until the product thinker signs it off, with the person's off switch in their user layer in place of degradation.
- A signing key is not opened to a confined agent. Where the person's commits are signed, the spec proposes that a confined agent commits unsigned and abcd signs outside the boundary when it lands the work, a question for the product thinker.
