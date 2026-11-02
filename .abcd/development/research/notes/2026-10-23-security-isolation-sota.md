# Integrity, secrets, sandboxing and isolation: state of the art

Dated 2026-10-23; recency is judged against that date. A state-of-the-art pass
over the open intents that share one question, what abcd can honestly promise
about what an agent, a prompt or a packed record can reach or alter:
[itd-35](../../intents/drafts/itd-35-lifeboat-integrity-audit.md) (lifeboats
carry their own integrity proof),
[itd-16](../../intents/drafts/itd-16-hash-chain-merkle-audit.md) (the audit
umbrella and its hash chain),
[itd-62](../../intents/drafts/itd-62-pluggable-safety-gate.md) (a fail-closed
gate wrapping a trusted scanner),
[itd-2609292109005937](../../intents/drafts/itd-2609292109005937-a-person-who-opts-in-has-every-prompt.md)
(an opt-in local check on every prompt),
[itd-2609151516525843](../../intents/drafts/itd-2609151516525843-the-public-banlist-cannot-exist-when-a.md)
(a banlist on the first commit, and one for the whole machine),
[itd-2610071545369520](../../intents/drafts/itd-2610071545369520-when-abcd-s-banned-names-check-warns-on.md)
(a person confirms the assistant's verdict on each warned line),
[itd-2610161621364075](../../intents/planned/itd-2610161621364075-a-person-who-leaves-abcd-to-run-on-its.md)
(launched agents run inside an OS-enforced sandbox),
[itd-2610161710315026](../../intents/drafts/itd-2610161710315026-a-person-who-lets-abcd-s-agents-run.md)
(unattended runs reach only a declared list of hosts),
[itd-2610161710316538](../../intents/drafts/itd-2610161710316538-when-abcd-drives-a-drain-or-a-build-from.md)
(drains and builds report how isolated they were),
[itd-18](../../intents/drafts/itd-18-permission-template-per-project-type.md)
(permissions from day one) and
[itd-148](../../intents/planned/itd-148-every-change-starts-in-its-own-worktree.md)
(the primary checkout is read-only; researched for its enforcement gap only,
since its store sibling carries its own prior art). Every source below was
opened in this pass; where a page could not be retrieved the text says so and
gives no link. Fetched pages were read as data. Earlier notes this one builds
on rather than repeats: the guard's parse-layer limits
([2026-08-18](2026-08-18-guard-wrapper-family-and-parse-layer-limits.md)),
the same-machine coordination survey
([2026-09-30](2026-09-30-same-machine-agent-coordination-sota.md)), the
local-models survey ([2026-08-22](2026-08-22-local-models-mlx-sota.md)) and
the headless-coordination note
([2026-10-17](2026-10-17-parallel-headless-agent-coordination.md)).

Tiers, as the researcher method sets them: **[EVIDENCE]** an evaluation,
experiment or large dataset, and also the primary text of a standard or a
vendor's documentation of its own tool's behaviour, marked `standard`, `docs`
or `study`; **[CONSENSUS]** several independent practitioners or vendors
agreeing; **[CONTESTED]** credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** a single report or a vendor claim with no method
behind it. The proposer wrote this note; the independent fit-challenge is
still owed at plan time.

## One finding that runs through the cluster

Every vendor whose agent abcd sits beside now draws the same line the guard's
2026-08-18 note drew: a command list steers, an operating-system boundary
confines, and the two are different controls. Claude Code's permissions page
says argument-constraining Bash patterns "are fragile" and that guidance
"shapes what Claude tries but doesn't enforce a boundary"; Codex says "The
sandbox defines technical boundaries" and the approval policy says when to
ask; Copilot's agent runs behind a firewall and says even that "should not be
considered a comprehensive security solution". The honest shape for every
intent here is the same: an enforced boundary where one exists, a stated
limit where it does not, and a report that says which was in force.
[CONSENSUS] ([Claude Code permissions](https://code.claude.com/docs/en/permissions),
[Codex approvals and security](https://learn.chatgpt.com/docs/agent-approvals-security),
[Copilot agent firewall](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/customize-the-agent-firewall)).

## itd-2610161621364075: launched agents run inside an OS-enforced sandbox

The intent is planned with eight rulings; decision 6 asks planning to check
first whether Anthropic's sandbox-runtime meets the preference for an adopted
open-source fence before anything is built. This section is half of that
check: it weighs `srt` against a microVM, native profiles and the host's Bash
sandbox, not against the other open-source fences, and it does not try `srt`
on a non-Claude launch (Codex, opencode); the other half is finished at plan
time.

### Findings

- **What sandbox-runtime is.** `@anthropic-ai/sandbox-runtime` (`srt`) wraps
  any command: "Uses `sandbox-exec` with dynamically generated Seatbelt
  profiles" on macOS; "Uses bubblewrap for containerization with network
  namespace isolation" on Linux, with seccomp for unix-socket blocking
  ("x64/arm64 only"); Windows is alpha, under a dedicated local user and a
  Windows Filtering Platform fence. Apache-2.0, "Beta Research Preview",
  installed with `npm install -g`; Linux also needs `bubblewrap`, `socat` and
  `ripgrep`, and the README's requirements list names `ripgrep` for macOS
  too, where its own platform summary and the harness's page say no extra
  package is needed. [EVIDENCE: docs]
  ([sandbox-runtime README](https://github.com/anthropics/sandbox-runtime))
- **Its defaults match the ADR's policy.** Writes are allow-only ("By default,
  write access is denied everywhere"), reads deny-then-allow, network denied
  until a domain is listed. Without a settings file it still starts, blocking
  network and confining writes to `/tmp/claude`, `~/.npm/_logs` and
  `~/.claude/debug`; a settings file that exists but is empty or invalid "is an
  error, not a fallback". [EVIDENCE: docs] (same README)
- **It denies the paths adr-2610161621415708 names, on its own.** At the
  project root it denies `.git/hooks`, `.git/config` (unless
  `allowGitConfig: true`), `.mcp.json`, `.claude/commands`, `.claude/agents`
  and shell start-up files, "even with `allowWrite: [\".\"]`". On macOS the
  denies are checked at write time and so cover nested repositories made
  during the session; on Linux "the runtime builds the deny list once at
  launch", scanning `mandatoryDenySearchDepth` levels (default 3) with
  ripgrep, and "does not cover anything the session creates later, such as
  `git init`, `git clone`, or scaffolding". [EVIDENCE: docs]
  ([Claude Code sandbox environments](https://code.claude.com/docs/en/sandbox-environments),
  README)
- **Whole-process is the documented bar for unattended runs.** The host's own
  comparison says the shell-only sandbox "is not sufficient for fully
  unattended runs in either mode", and that a run without prompts belongs
  "inside a container, a VM, or the sandbox runtime, so that file tools, MCP
  servers, and hooks are also inside the boundary". It also warns that
  isolation "does not change what is sent to the model". [EVIDENCE: docs]
  (sandbox environments)
- **Launch-time facts for the runner.** `claude -p --bare` skips hooks,
  skills, plugins, MCP servers and `CLAUDE.md`, "will become the default for
  `-p`", and never reads OAuth credentials or the keychain, so a bare launch
  needs an API key or an `apiKeyHelper` in `--settings` JSON; `--settings`
  applies above user and project files and below managed settings; `-p` runs
  skip the workspace-trust check. [EVIDENCE: docs]
  ([Claude Code headless](https://code.claude.com/docs/en/headless),
  [settings precedence](https://code.claude.com/docs/en/settings))
- **The three vendors use the same two primitives.** Codex: "Locally, Codex
  uses an OS-enforced sandbox", Seatbelt on macOS, and on Linux "Uses
  bubblewrap. Codex uses the first `bwrap` found on `PATH`, with a bundled
  fallback that requires unprivileged user namespace support"; network is off
  in `workspace-write` unless enabled. Gemini CLI: `sandbox-exec` with six
  built-in Seatbelt profiles, or Docker/Podman. [CONSENSUS]
  ([Codex sandboxing](https://learn.chatgpt.com/docs/sandboxing),
  [Codex approvals](https://learn.chatgpt.com/docs/agent-approvals-security),
  [Gemini CLI sandbox](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/sandbox.md))
- **`sandbox-exec` is deprecated and universally used.** The man page:
  "The `sandbox-exec` command is DEPRECATED", pointing at App Sandbox, which
  has no command-line form. Community reports place the primitive working on
  macOS 26 while profiles break on tools that read sysctls the profile omits;
  no Apple statement on removal was found. [CONTESTED: deprecated in text,
  relied on in practice]
  ([sandbox-exec(1)](https://keith.github.io/xcode-man-pages/sandbox-exec.1.html))
- **Linux platform limits.** bubblewrap needs unprivileged user namespaces,
  and "is not a complete, ready-made sandbox with a specific security policy";
  Ubuntu restricts unprivileged user namespaces through
  `kernel.apparmor_restrict_unprivileged_userns`, and Claude Code's guide says
  Ubuntu 24.04's default policy "prevents bubblewrap from creating the user
  namespaces it needs", giving an AppArmor profile for `bwrap` as the fix.
  Landlock is the kernel's own unprivileged alternative ("Landlock empowers
  any process, including unprivileged ones, to securely restrict themselves",
  5.13 onward), restricting file hierarchies and TCP/UDP ports but not
  hostnames, and "Files or directories opened before the sandboxing are not
  subject to these restrictions". A Go binding exists (`go-landlock`, MIT),
  a Go dependency that would need sign-off and that covers Linux only.
  [EVIDENCE: docs]
  ([bubblewrap](https://github.com/containers/bubblewrap),
  [Ubuntu user namespaces](https://ubuntu.com/blog/ubuntu-23-10-restricted-unprivileged-user-namespaces),
  [Claude Code sandboxing](https://code.claude.com/docs/en/sandboxing),
  [Landlock](https://docs.kernel.org/userspace-api/landlock.html),
  [go-landlock](https://github.com/landlock-lsm/go-landlock))
- **Nesting and what breaks inside.** srt: "What stops working as a result is
  whatever sandboxes itself with user namespaces: Chromium's and Electron's
  namespace sandbox, rootless podman"; `enableWeakerNestedSandbox` is the
  escape and weakens the fence. Jest needs `--no-watchman`. Unix sockets are
  blocked by default on both platforms, so an `ssh-agent` socket for a push
  over SSH must be allowed by path on macOS or wholesale on Linux. [EVIDENCE:
  docs] (README)
- **A microVM is the stronger, heavier alternative.** Docker Sandboxes put
  the agent in a microVM with "Full (hypervisor)" isolation, a host-side
  proxy for all outbound TCP and a credential sentinel swapped by the proxy
  "after the request has left the microVM"; the local CLI is free for
  commercial use. It needs the Docker tooling installed and had its own
  egress hole (below). [EVIDENCE: docs]
  ([Docker Sandboxes](https://docs.docker.com/ai/sandboxes/),
  [architecture](https://docs.docker.com/ai/sandboxes/architecture/))

### Options weighed

1. **Adopt `srt` behind the `internal/core/fence` seam.** Meets every clause
   of the ADR's policy with configuration alone; maintained by the vendor of
   the harness abcd launches most; a runtime dependency on Node and npm, not
   a Go import; beta, "APIs and configuration formats may evolve".
2. **Native Seatbelt profiles plus Landlock or bubblewrap.** No external
   program; abcd carries two policy languages and every platform regression;
   on macOS it still rests on the deprecated `sandbox-exec`.
3. **A microVM (Docker Sandboxes) or a container.** Strongest boundary, with
   its own egress proxy and credential sentinel; a Docker dependency, worse
   sharing of the worktree and receipt folder, and the Go build cache ruling
   (decision 8) becomes moot inside it.
4. **The host's own Bash sandbox per launch.** Already rejected by the ADR:
   shell commands only.

### Recommendation

**Path 1: adopt sandbox-runtime as the first fence behind the seam the ADR
already names**, with the ADR's policy expressed as an `srt` settings file
abcd generates per launch and passes with `--settings`. It is a new runtime
dependency (Node, npm, ripgrep on macOS and Linux by the README, and on Linux
bubblewrap and socat) and the
first exception to adr-22 that the ADR proposes, so it **needs explicit
sign-off**. Three consequences for the spec: on Linux the deny list is built
once at launch, so a worktree the agent creates after launch is outside the
mandatory denies and the run must create every worktree before the fence
goes up; the run must allow the forge's domains and the `ssh-agent` socket
(or route pushes over HTTPS through the credential store) for criterion two to
hold; and the detection that criterion four demands is a probe, not a config
read: the host's own documented check is a write outside the allowed paths
failing with "Operation not permitted" (macOS) or "Read-only file system"
(Linux).

### Local fit risks

- `basics-built-in-adapters-bring-power` and adr-22: a fence with no native
  default to degrade to is a requirement, not an adapter; the ADR says so and
  the product thinker has not yet signed the exception.
- `enforcement-claims-are-facts`: "confined" may be said only where the probe
  proves it; srt starts silently with built-in defaults when no settings file
  is found, and the host says "Don't take a clean start as proof your settings
  loaded".
- `the-users-directory-is-theirs`: srt's own defaults write under `~/.npm` and
  `~/.claude`; the generated policy must not widen that.
- Host-agnostic prose: the fence is Anthropic's; user-facing text names the
  mechanism, never the harness.

## itd-2610161710315026: unattended runs reach only a declared list of hosts

### Findings

- **Allow-only by name, deny first.** srt: "By default, all network access is
  denied. You must explicitly allow domains"; `deniedDomains` "takes
  precedence over allowedDomains"; entries take wildcards and ports. On Linux
  "The network namespace of the sandboxed process is removed entirely, so all
  network traffic must go through the proxies"; on macOS "The Seatbelt profile
  allows communication only to a specific localhost port". [EVIDENCE: docs]
  ([sandbox-runtime README](https://github.com/anthropics/sandbox-runtime))
- **Name allowlists need a resolved-address check.** srt resolves each
  permitted name and drops loopback, link-local, multicast, cloud metadata and
  the host's own addresses, because "whoever controls a permitted name's
  DNS... controls what it resolves to"; IP-literal entries and `parentProxy`
  routes skip the check; domain fronting is named as a possible bypass.
  [EVIDENCE: docs] (README)
- **Proxies filter by hostname and do not inspect.** Claude Code's proxy
  "enforces the allowlist based on the requested hostname and, by default,
  does not terminate or inspect TLS traffic"; with `strictAllowlist` set in
  user, managed or `--settings` scope, hosts outside the list are denied
  rather than prompted and a repository's `allowedDomains` entries are
  ignored; a repository's files cannot widen an admin-required sandbox
  either. [EVIDENCE: docs]
  ([Claude Code sandboxing](https://code.claude.com/docs/en/sandboxing))
- **DNS has been left outside an egress allowlist before, and the fence
  leaves it open on macOS and Windows.** Docker Sandboxes "enforces an
  HTTP/S-only egress allowlist but does not apply it to DNS resolution", so a
  workload could "encode data into DNS labels" and exfiltrate
  (CVE-2026-12039, versions 0.13.0 to before 0.33.0, CVSS 5.7, the default
  configuration unaffected). srt's Windows notes say "DNS resolution via the
  system resolver is not fenced", because `getaddrinfo()` is answered by a
  system service outside the sandboxed process, and add "This mirrors the
  macOS behaviour". On Linux, srt removing the network namespace closes this
  channel by construction; on macOS and Windows a lookup through the system
  resolver succeeds even though the connection that follows is blocked, so
  data can still leave in DNS labels there. [EVIDENCE: advisory, docs]
  ([CVE-2026-12039](https://fieldguide.lutrasecurity.com/CVE-2026-12039/))
- **Report what was blocked.** Copilot's firewall is on by default with a
  recommended allowlist, organisation rules a repository cannot remove, and
  when a request is blocked "a warning is added to the pull request body"
  showing "the blocked address and the command that tried to make the
  request"; it applies to the agent's shell, "does not apply directly to"
  MCP server processes, and "should not be considered a comprehensive security
  solution". Claude Code "names the denied host in the command's result".
  [EVIDENCE: docs] ([Copilot firewall](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/customize-the-agent-firewall),
  sandboxing)
- **The press release's limit is the vendors' limit.** Allowing the forge
  allows a push to any repository the credential reaches; Claude's guide:
  "Any approach that allows network egress can still leak data the agent can
  read". [CONSENSUS] (sandbox environments, Copilot firewall)

### Options weighed

1. srt's `network.allowedDomains` per launch, the same mechanism as the fence
   (one dependency, one policy file).
2. A firewall abcd configures itself (`pf` on macOS, nftables on Linux):
   needs root, machine-wide, and the user's directory is theirs.
3. A host-level `strictAllowlist` through Claude's `--settings`: Bash only
   and one harness only.

### Recommendation

**Path 1: the declared list is srt's `allowedDomains` for the launch**, with
one rule from the field and one question for the record. The rule: the run's
receipt lists every denied host with the command that tried it (the Copilot
shape) and names that the system resolver sits outside the fence on macOS and
Windows. The question: who may put a host on the list. The field's strict
shapes do not let a project propose: Claude Code's `strictAllowlist` and
admin-required sandbox take the list from the user or managed layer and
ignore a repository's entries, and Copilot merges a repository's additions
with organisation entries it cannot remove. A project file that declares
hosts a user layer can only narrow would let a checked-out project widen its
own reach, the opposite of what the list is for; the record decides whether
a project file declares, proposes, or neither. The honest claim stays the
press release's: it narrows mistakes. IP literals in the list are a hole and
should be refused.

### Local fit risks

- One canonical primitive: the fence intent owns the settings file; this
  intent adds a section to it, not a second mechanism.
- Loud staging: a hidden widening is worse than no list; the receipt must
  show the list in force, not the list declared.
- Script-first MVP: the first delivery can be the generated settings file and
  the receipt line; no proxy of abcd's own.

## itd-2610161710316538: drains and builds report how isolated they were

### Findings

- **A hook knows the permission mode on most events; nothing documented tells
  it the sandbox state.** Hook input lists `permission_mode` among its common
  fields (`session_id`, `transcript_path`, `cwd`, `hook_event_name`), with
  the caveat "Not all events receive this field"; the `UserPromptSubmit`
  example carries it, so the probe reads it from an event that does. The
  environment-variables page documents `CLAUDECODE=1` and
  `CLAUDE_CODE_CHILD_SESSION=1` as markers that a process was spawned by the
  host, and says nothing that exposes whether the sandbox is on. [EVIDENCE:
  docs] ([Claude Code hooks](https://code.claude.com/docs/en/hooks),
  [environment variables](https://code.claude.com/docs/en/env-vars))
- **The host's own test is a probe.** "Confirm commands run inside the
  sandbox": `touch ~/sandbox-probe` "Fails with `Operation not permitted` on
  macOS, or `Read-only file system` on Linux and WSL2". `/sandbox` shows the
  state and the missing dependencies, but it is an interactive panel.
  [EVIDENCE: docs] ([sandboxing](https://code.claude.com/docs/en/sandboxing))
- **What a session's sandbox leaves outside.** Even when on, the Bash sandbox
  leaves file tools, MCP servers, hooks and the status-line command running
  "with your full access", and an allow rule "also approves an unsandboxed
  retry". So "the host's protections were on" has at least three distinct
  answers: mode, Bash sandbox, unsandboxed retry allowed. [EVIDENCE: docs]
  (sandboxing)
- **Precedent for the report line.** Copilot writes blocked requests into the
  pull-request body; the model-matrix criteria already say "for a copy made
  in the host's harness, the report says this isolation is asserted, not
  proven". [EVIDENCE: docs] (Copilot firewall; model-matrix intent)

### Options weighed

1. Read the host's merged settings files and infer.
2. Probe at run start: attempt a write outside the allowed set and an
   outbound connection to a host not on any list, inside the host's Bash tool,
   and report the outcomes with the hook-reported `permission_mode`.
3. Say nothing abcd cannot prove.

### Recommendation

**Path 2: a native probe shaped to the host's own documented check**, run at
the start of a drain or build through the same tool the run uses, reporting
three facts separately (permission mode from the hook input; whether a write
outside the worktree was refused; whether an unlisted host was refused) and
repeating them in the morning summary. A config read is the seam for a host
that exposes its state later. The report says "asserted" for anything only
the host claims and "proven" only for a probe that failed as it should.

### Local fit risks

- `enforcement-claims-are-facts` and `guards-prove-themselves` pull the same
  way as the recommendation; the risk is the opposite temptation, inferring
  from settings files because a probe costs a tool call.
- The probe itself is a write attempt and a connection attempt; it must target
  a path and a host that are harmless when the fence is absent.

## itd-148: the primary checkout is read-only (enforcement gap only)

### Findings

- **The host now enforces worktree isolation itself, with the same honesty
  this intent promises.** Claude Code blocks "an `Edit`, `Write`, or
  `NotebookEdit` that targets a path in the main checkout", a command "whose
  working directory resolves to the main checkout, or whose working directory
  it can't verify stays outside it", git redirected "through `git -C`,
  `--git-dir`, a `GIT_DIR` or `GIT_WORK_TREE` variable, or a `cd`", and any
  command whose text it cannot parse ("You can't turn this check off"). It
  then says: "None of them tracks which files a shell command writes, so a
  command that writes into the main checkout without running git there, such
  as `cp` or a shell redirect, isn't refused". Refusals name the worktree and
  how to proceed. The checks run "while a session is isolated in a worktree"
  and protect the main checkout from that session; a session started in the
  primary checkout, this intent's case, gets none of them. [EVIDENCE: docs]
  ([Claude Code worktrees](https://code.claude.com/docs/en/worktrees.md))
- **The host's worktrees live inside the checkout, abcd's do not.** Default
  location `.claude/worktrees/<name>/`; hook `${CLAUDE_PROJECT_DIR}` stays at
  the main checkout while the hook's `cwd` field follows the worktree; a
  `WorktreeCreate` hook may place worktrees elsewhere; `-p` runs leave their
  worktrees and locks for a later sweep. [EVIDENCE: docs] (worktrees)
- **The host's own binary refuses a worktree whose git identity resolves into
  the main checkout** ("Refusing to use <path> as an isolation worktree"),
  which is the git-common-directory check this intent specifies, in the other
  direction. [EVIDENCE: docs] (worktrees)
- **A `PreToolUse` hook can refuse, and silence approves nothing.** Decision
  fields `permissionDecision` allow/deny/ask with a reason, or exit 2 with
  stderr; "The hook can deny the call, but staying silent doesn't approve
  it"; a hook that fails or times out lets the action proceed by default,
  and `onFailure: "block"` on a command or HTTP hook (v2.1.295 or later)
  makes it block instead. [EVIDENCE: docs]
  ([hooks](https://code.claude.com/docs/en/hooks))
- **The host writes into the main checkout from a worktree session.** A
  standing approval ("Yes, and don't ask again") given in a worktree session
  "saves the rule to the main checkout's `.claude/settings.local.json`", so
  it applies in every worktree and survives the worktree's removal. The write
  is the host's own, not a tool call, so no hook sees it. [EVIDENCE: docs]
  ([Claude Code worktrees](https://code.claude.com/docs/en/worktrees.md))
- **The field's answer to "the tree is remembered, not known" is unchanged
  from 2026-09-30:** one checkout per agent created by the tool, every
  command anchored by absolute path. That note found no source measuring how
  often agents lose their directory; none was found now either. [CONSENSUS]
  ([2026-09-30 note](2026-09-30-same-machine-agent-coordination-sota.md))
- **An OS-level backstop exists and is the wrong one.** macOS `chflags uchg`
  sets a user-immutable flag the owner may set and clear; it would stop every
  writer, git's fast-forward included, and the allow-set needs git to move.
  [EVIDENCE: docs] ([chflags(1)](https://keith.github.io/xcode-man-pages/chflags.1.html))
- **The git backstop's reach.** `pre-commit` and `pre-merge-commit` "can be
  bypassed with the `--no-verify` option"; the hooks page documents no
  pre-commit run for rebase or cherry-pick; `core.hooksPath` moves the hooks
  directory. [EVIDENCE: docs] ([githooks](https://git-scm.com/docs/githooks))
- **No field precedent for a "declared coordinator".** The closest shape is
  the single writer the 2026-10-17 note documents: a lease with an expiry,
  renewed by its holder, with a fencing token so a stale holder's late write
  is refused. [EVIDENCE: docs via that note]
  ([2026-10-17 note](2026-10-17-parallel-headless-agent-coordination.md))

### Options weighed

1. Native hook check in the binary shaped to the host's four checks, plus the
   git pre-commit backstop (the intent as written).
2. Delegate to the host's own isolation by running every abcd session with
   `--worktree`: worktrees land inside the checkout, against
   adr-2609091248200336, and only one harness has it.
3. `chflags`/`chmod` on the primary checkout: stops the allow-set too.

### Recommendation

**Path 2: build the block natively, shaped to the host's four checks and
saying what they say**, that a `cp` or a redirect is not caught; the hook is
the seam, set fail-closed with `onFailure: "block"`, so a host that grows
its own isolation can take over a check at a time. The host's own write of a
standing approval into the main checkout's `.claude/settings.local.json` is
outside any hook's reach: the allow-set names it, or the record states it as
a known exception. For the open question, the coordinator declaration should be a lease
file in the main checkout's local tier holding the session id, pid and start
time, renewed while the session lives and expiring without it, so a second
declaration is refused while the first holds and a dead coordinator does not
lock the checkout forever; the same shape the load gate already uses.

### Local fit risks

- The host places worktrees inside the checkout and reads project settings
  from there; abcd's store is elsewhere by ruling, so the host's
  `--worktree` path and abcd's block must not fight over which tree is
  "main".
- `one-writer-per-file`: a lease file in the local tier is per-worktree by
  construction; the coordinator's lease must live in the primary checkout's
  tier, which the block must then allow.

## itd-16: the audit umbrella and its hash chain

### Findings

- **The primitives the intent names are the right ones, and are standards.**
  RFC 8785 (JCS) is Informational and defines sorting ("JSON object
  properties MUST be sorted recursively"), ECMAScript number serialisation
  and UTF-8 output, for data that must be "expressed in an invariant format";
  numbers "MUST be expressible as IEEE 754 double-precision values".
  RFC 9562 (Proposed Standard) gives UUIDv7 a 48-bit millisecond timestamp
  and "a time-ordered value field", with counters left to the application
  ("Counter rollovers MUST be handled by the application"). [EVIDENCE:
  standard] ([RFC 8785](https://www.rfc-editor.org/rfc/rfc8785),
  [RFC 9562](https://www.rfc-editor.org/rfc/rfc9562))
- **The tree shape is RFC 6962, and compatibility is cheap.** Leaf hash
  `SHA-256(0x00 || d)`, node hash `SHA-256(0x01 || left || right)`, "This
  domain separation is required to give second preimage resistance"; audit
  paths prove inclusion, consistency proofs "prove the append-only property".
  Go's own `golang.org/x/mod/sumdb/tlog` "follows the design of Certificate
  Transparency (RFC 6962) and its proofs are compatible with that system",
  with `TreeHash`, `ProveRecord`, `CheckTree` and tiles; it is in `x/mod`,
  not the standard library, so importing it is a Go dependency needing
  sign-off. [EVIDENCE: standard, docs]
  ([RFC 6962](https://www.rfc-editor.org/rfc/rfc6962),
  [x/mod/sumdb/tlog](https://pkg.go.dev/golang.org/x/mod/sumdb/tlog))
- **A chain head has a standard text form.** A C2SP checkpoint is a signed
  note of at least three lines: origin, tree size "as an ASCII decimal", and
  the base64 root "of the RFC 6962 Merkle hash tree at the specified tree
  size", followed by signature lines; the Go checksum database serves exactly
  this through `/latest` as "a note, which is text that has been signed by one
  or more server keys". [EVIDENCE: standard, docs]
  ([tlog-checkpoint](https://c2sp.org/tlog-checkpoint),
  [Go modules reference](https://go.dev/ref/mod))
- **Tamper-evidence needs a reader who holds an earlier head.** A log's
  cosigner "MUST NOT sign any checkpoint which is inconsistent with any
  checkpoint it previously signed"; a witness keeps the latest checkpoint
  per log and cosigns a new one only after checking "that the old size
  matches the size of the latest checkpoint it cosigned" and the consistency
  proof. Without someone outside the filesystem holding an earlier head, a
  party who can edit a transcript can also recompute the chain; what a
  chain-only design detects is a partial edit, not a rewrite. [EVIDENCE:
  standard] ([tlog-witness](https://c2sp.org/tlog-witness), tlog-checkpoint)
- **Signing without a new dependency.** `ssh-keygen -Y sign` signs a file or
  stdin with an SSH key under a namespace "to prevent signature confusion
  across different domains of use" (recommended form `NAMESPACE@YOUR.DOMAIN`);
  `-Y verify` checks against an allowed-signers file. Git's `gpg.format` takes
  `ssh` with `gpg.ssh.allowedSignersFile`. The intent scopes signing out; the
  cheapest anchor is nonetheless a signed checkpoint, and the second cheapest
  is pushing the head to the forge as a tag. A pushed tag is not a witness on
  its own: anyone with push access can delete or move a tag and the forge
  exposes no tag history to a reader, so it anchors only where the forge
  refuses to move or delete it (a tag ruleset or an immutable release) or a
  third party keeps a copy. [EVIDENCE: docs]
  ([ssh-keygen(1)](https://man.openbsd.org/ssh-keygen),
  [git gpg config](https://raw.githubusercontent.com/git/git/master/Documentation/config/gpg.adoc))
- **Time anchoring exists as a free service.** OpenTimestamps: "A timestamp
  proves that some data existed prior to some point in time", via calendar
  servers that "don't require any registration or api key"; it is an
  external call and a Bitcoin dependency in the proof. [EVIDENCE: docs]
  ([OpenTimestamps](https://opentimestamps.org/))
- **Attestation shape, if the entries are ever to interoperate.** An in-toto
  Statement is `_type: https://in-toto.io/Statement/v1`, a `subject` list
  where "Each element MUST have `digest` set", a `predicateType` URI and a
  `predicate`; SLSA provenance is one predicate type
  (`https://slsa.dev/provenance/v1`) attesting "that a particular build
  platform produced a set of software artifacts"; DSSE is the envelope
  ("payload", "payloadType", "signatures"), and "Consumers MUST ignore
  unrecognized fields". Rekor is the public log for such metadata, "an
  immutable, tamper-resistant ledger", backed by Trillian. The framework is
  "still under development". [EVIDENCE: standard, docs]
  ([in-toto Statement](https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md),
  [SLSA provenance](https://slsa.dev/spec/v1.1/provenance),
  [DSSE envelope](https://github.com/secure-systems-lab/dsse/blob/master/envelope.md),
  [Rekor](https://docs.sigstore.dev/logging/overview/))
- **Earlier local framing.** The design-principles survey already placed
  abcd's provenance as PROV-like rather than SLSA/in-toto verifiable, "a
  deliberate lighter-weight" choice; this intent is where that choice is
  tested. [CONSENSUS, local] ([design principles note](design-principles-sota.md))

### Options weighed

1. Native chain: JCS entries, RFC 6962 hashing, daily segments, C2SP
   checkpoint text as the head, no signature (the intent as written).
2. Option 1 plus a signed checkpoint via `ssh-keygen -Y` and an optional push
   of the head to the forge; `x/mod/sumdb/tlog` adopted later for proofs.
3. Full attestation stack (in-toto Statements in DSSE, Rekor): a public log
   for private transcripts, and the launch rule already excludes the chain
   from public payloads.

### Recommendation

**Path 2: native, shaped to RFC 6962 and the C2SP checkpoint, with the
head's signature and its external anchor as the first seam.** Hash with the
`0x00`/`0x01` prefixes so proofs stay compatible with `x/mod/sumdb/tlog`
should it be adopted (a Go dependency, sign-off needed); write the head as a
checkpoint note; keep JCS and UUIDv7 as drafted, and keep criterion 1's
per-entry `prev_hash`, so the tree adds inclusion and consistency proofs over
a chain it does not replace and no criterion changes. The press release's
"non-repudiability" is not reachable without an anchor outside the
filesystem; the SOTA section says so and the ruling stands.

### Local fit risks

- Host-delegated LLM work is untouched; the dependency rule is the risk,
  since `x/mod` is tempting and small.
- The history store redacts on write, so the chain hashes redacted
  transcripts; an auditor reading "every conversation hashed" may expect the
  raw text. The claim must name what is hashed.
- `one-canonical-primitive`: the preflight receipt, the lab pins and this
  chain all hash trees; one hashing primitive should serve all three.

## itd-35: lifeboats carry their own integrity proof

### Findings

- **A directory hash already has a canonical form in Go.** `dirhash` H1 is
  "the base64-encoded SHA-256 of a summary" equal to
  `sha256sum $(find . -type f | sort) | sha256sum`, one line per file, sorted;
  `HashZip` ignores "per-file modification times, and other metadata". The
  package is in `x/mod` (dependency), but the definition is a dozen lines.
  [EVIDENCE: docs] ([x/mod/sumdb/dirhash](https://pkg.go.dev/golang.org/x/mod/sumdb/dirhash))
- **The manifest shape that other tools read is a subject list of digests.**
  In-toto's Statement: subjects "are matched purely by digest, regardless of
  content type", and a Statement's `_type`, `subject` and `predicateType` are
  all required; a lifeboat manifest written as `[{name, digest:{sha256}}]`
  becomes readable by an in-toto consumer once wrapped in a Statement, and
  is not an in-toto artefact on its own. [EVIDENCE: standard] ([in-toto Statement](https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md))
- **Go's module zips answer the archive question.** The checksum database
  hashes "the files within the module's `.zip` file", not the zip bytes, so a
  re-packed archive with the same files verifies; the same rule makes
  `.tar.gz` verification a verify-on-extract. [EVIDENCE: docs]
  ([Go modules reference](https://go.dev/ref/mod))
- **The proof's weakness is the same as itd-16's.** A root stored inside the
  lifeboat it covers is recomputable by whoever edits the lifeboat; the
  witness rule applies (above). Signing with `ssh-keygen -Y sign` over the
  manifest, or conveying the root separately, is what makes "prove
  provenance" true. [EVIDENCE: standard] ([tlog-witness](https://c2sp.org/tlog-witness))

### Options weighed

1. The intent's tree (artefact, section, root) under itd-16's primitives.
2. A flat sorted manifest (dirhash shape, in-toto subject shape) with an
   RFC 6962 root over it; sections are derived from paths, not hashed as a
   layer.
3. Signed manifest only, no tree: enough for the compliance quote, loses the
   "which file" report on mismatch, which a manifest diff gives anyway.

### Recommendation

**Path 2: a flat, sorted, digest-per-file manifest in the in-toto subject
shape, rooted with RFC 6962 hashing from itd-16's primitive**, verified on
extracted files so a directory and an archive verify alike; sections become
a report grouping, not a hash layer, which amends the tree shape the record's
scope and first criterion name. Signing is out of scope by the record's
Out-of-Scope bullet, shared with itd-16, not by a ruling; the SOTA section
records that the compliance claim needs it or an out-of-band root.

### Local fit risks

- Blocked by itd-16: this intent should not grow its own hasher
  (`one-canonical-primitive`).
- itd-9 migration re-hashing: a flat manifest makes "original hash preserved
  beside the new" a two-column file rather than a tree rewrite.

## itd-62: a pluggable fail-closed gate wrapping a trusted scanner

### Findings

- **The motivating number is real, vendor-run and task-skewed.** Veracode's
  2025 report: "In 45 percent of all test cases, LLMs introduced
  vulnerabilities classified within the OWASP" Top 10, over "80 curated
  coding tasks across more than 100 large language models", scored by the
  vendor's own analyser on tasks built around known weakness patterns; Java
  "over 70 percent", XSS failed in 86% and log injection in 88% of cases.
  [EVIDENCE: study, vendor-run]
  ([Veracode press release](https://www.veracode.com/press-release/ai-generated-code-poses-major-security-risks-in-nearly-half-of-all-development-tasks-veracode-research-reveals/))
- **Secret scanners: high recall, low precision, measured.** Basak, Cox,
  Reaves and Williams (ESEM 2023) ran five open-source and four proprietary
  tools on a benchmark: Gitleaks precision 46%, recall 88%; TruffleHog recall
  52%; GitHub's scanner precision 75%; false positives traced to "employing
  generic regular expressions and ineffective entropy calculation".
  [EVIDENCE: study] ([arXiv 2307.00714](https://arxiv.org/abs/2307.00714))
- **Gitleaks.** MIT; modes `git`, `dir`, `stdin`; `detect`/`protect`
  deprecated from v8.19.0; output `json, csv, junit, sarif, template`; exit
  "0 - no leaks present", "1 - leaks or error encountered"; allowlisting by
  `gitleaks:allow` comment, `.gitleaksignore` fingerprints ("experimental and
  subject to change") and config allowlists; and the maintainer's notice
  "Gitleaks is feature complete. I'm not merging new features into Gitleaks",
  security patches only, and "I'm shifting my focus to Betterleaks"
  (github.com/betterleaks/betterleaks, MIT, the same maintainers; its SARIF
  support was not confirmed in this pass). [EVIDENCE: docs]
  ([gitleaks](https://github.com/gitleaks/gitleaks))
- **TruffleHog.** AGPL-3 since v3; "For every secret TruffleHog can classify,
  it can also log in to confirm if that secret is live or not", which means
  the scanner calls the provider's API with the found credential; clones local
  repositories to a temporary directory before scanning; exit 183 with
  `--fail` when results are found. [EVIDENCE: docs]
  ([trufflehog](https://github.com/trufflesecurity/trufflehog))
- **detect-secrets.** Apache-2.0; a baseline of accepted findings plus an
  `audit` step that "allows analysts to label results"; inline
  `# pragma: allowlist secret`. [EVIDENCE: docs]
  ([detect-secrets](https://github.com/Yelp/detect-secrets))
- **Dependency scanning.** OSV-Scanner is Go, Apache-2.0, "supports 11+
  language ecosystems and 19+ lockfile types", scans container images, and
  has an offline mode ("No network connection is required after the initial
  database download"); its guided remediation "can be risky when run on
  untrusted projects". [EVIDENCE: docs]
  ([osv-scanner](https://github.com/google/osv-scanner))
- **One interchange format.** SARIF 2.1.0 is an OASIS Standard (errata
  2023) with `result`, `location`, `fingerprints`/`partialFingerprints` and
  `suppression` objects; gitleaks, TruffleHog and GitHub's uploader all speak
  it. [EVIDENCE: standard]
  ([SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html))
- **Why gates get ignored.** Johnson, Song, Murphy-Hill and Bowdidge (ICSE
  2013), twenty developers interviewed: "false positives and the way in which
  the warnings are presented" are the barriers, though "all of our
  participants felt that use is beneficial". [EVIDENCE: study, small]
  ([Google Research record](https://research.google/pubs/pub43477/))
- **The local scanner seam already exists.** `adapter/scanner` is a native
  pattern set that fails closed on an unreadable override and treats external
  scanners as "config-selected plug-ins behind the same seam"; this intent
  widens that seam to the downstream app and to a second scanner kind.
  [CONSENSUS, local] (`internal/README.md`)

### Options weighed

1. Wrap gitleaks (secrets) and OSV-Scanner (dependencies) as executables
   found on `PATH`, parse their SARIF, map to the four verdicts; scanner
   absent or SARIF unparsable is INCONCLUSIVE and refuses.
2. Import scanner libraries into the binary: Go dependencies, licence mixing
   (AGPL for TruffleHog), and a fork-by-stealth of their rule sets.
3. Delegate to the forge's scanners: not local-first, nothing on a laptop.

### Recommendation

**Path 1: adopt gitleaks and OSV-Scanner as the first wrapped scanners,
executed, never imported, with SARIF 2.1.0 as the only shape the gate
parses**, so a third scanner is a configuration line. No Go dependency.
Two warnings for the spec: gitleaks is in maintenance with Betterleaks named
as its successor, so the seam's swappability is load-bearing from day one
and the first-scanner choice is revisited at plan time, and TruffleHog's live
verification makes outbound calls with the found credential, which puts it
outside a local-first default. The 46% precision means the gate needs the
same confirm-and-record flow as itd-2610071545369520, or it will be ignored,
as the ICSE study predicts.

### Local fit risks

- `basics-built-in-adapters-bring-power`: the native scanner is the floor;
  this gate must not make the native floor look like the external scanner.
- Host-agnostic prose: the gate's docs name tools by category, with the
  concrete scanner in configuration.
- The brief change (framework-provided disciplines) remains the sequencing
  question the intent already records.

## itd-2609292109005937: an opt-in local check on every prompt

### Findings

- **The prompt seam can hold, and cannot rewrite.** `UserPromptSubmit` "Blocks
  the prompt, so it never reaches Claude" on exit 2 or `decision: "block"`,
  and "can't replace the prompt; it only injects `additionalContext` alongside
  it". So on this host "redact" is not available in the hook; "hold and
  tell" is. The host lowers the default hook timeout to 30 seconds on this
  event. [EVIDENCE: docs] ([Claude Code hooks](https://code.claude.com/docs/en/hooks))
- **The prompt is the smaller channel.** "Isolation also does not change what
  is sent to the model. Your prompts and the files Claude reads are
  transmitted to the Anthropic API or your configured provider with or
  without a sandbox." A prompt-only check leaves file reads and tool results
  unchecked. [EVIDENCE: docs]
  ([sandbox environments](https://code.claude.com/docs/en/sandbox-environments))
- **Rule-based detection misses what matters most; small models help and do
  not suffice.** REDACT (June 2026; 13,427 records, 51 entity types, 25
  languages; detectors scored on a locked 1,000-record sample): partial
  micro-F1 Presidio 0.195, GLiNER-multi 0.320, OpenAI Privacy Filter 0.512,
  GPT-4.1 0.597, Claude Sonnet 4.6 0.636; recall on HIGH-sensitivity
  categories Presidio 0.07, GLiNER 0.42, the Privacy Filter 0.57, the two
  hosted models 0.74 and 0.77.
  PIIBench (April 2026; 2.37 million sequences, 48 types) found every system
  under F1 0.14, Presidio highest at 0.1385 "yet it produces zero recall on
  most entity types". The two benchmarks rank differently; both say the same
  about a local-only gate. [EVIDENCE: study; CONTESTED on ranking]
  ([REDACT](https://arxiv.org/abs/2606.19881),
  [REDACT results](https://arxiv.org/html/2606.19881),
  [PIIBench](https://arxiv.org/abs/2604.15776))
- **The earlier local survey's hybrid figure stands.** The 2026-08-22 note
  recorded a practitioner benchmark at ~69% recall for a deterministic layer
  and ~83% for the hybrid, and set the deterministic layer as the gate with a
  local model as an assist. [ANECDOTE, practitioner]
  ([2026-08-22 note](2026-08-22-local-models-mlx-sota.md))
- **Presidio itself.** Analyzer and anonymizer with regex, checksum and NER
  recognisers (spaCy, transformers, GLiNER); "there is no guarantee that
  Presidio will find all sensitive information"; Python; now "transitioning
  to a community-owned project" under a new home. [EVIDENCE: docs]
  ([Presidio](https://presidio.dataprivacystack.org/))
- **Secret detectors at the same seam.** Gitleaks' `stdin` mode exists, with
  its measured 88% recall and 46% precision (itd-62 above). [EVIDENCE: study]
- **The fail-open question is a configuration choice at this host.** The
  `UserPromptSubmit` default timeout is 30 seconds, settable per hook ("If
  your hook needs more time, set the `timeout` field"); a hook that reaches
  it is cancelled and "The prompt still reaches Claude", unless the hook sets
  `onFailure: "block"` (v2.1.295 or later): "To block the prompt instead,
  set `onFailure: "block"` on a command or HTTP hook." So a slow local model
  fails open by default and fails closed by configuration. A blocked prompt
  is not gone: "By default the block message shown to the user ends with
  `Original prompt:` followed by the submitted text, and Claude Code writes
  that message to the session's transcript file on disk", which a
  privacy-motivated hold should disclose. [EVIDENCE: docs] (hooks)

### Options weighed

1. Deterministic first (scanner plus banned names on stdin), hold on a hit,
   tell the person; the local model as a second, opt-in pass over the same
   text with a strict time budget.
2. Local model first: measured recall says it would miss most of what the
   press release promises to catch.
3. Rewrite the prompt: not possible at this host's seam.

### Recommendation

**Path 2: native, deterministic-first, hold-and-tell at the hook, with the
local model as the opt-in adapter behind the existing scanner seam** (adr-22
shape). The "redact or hold" question is answered by the host: hold. The
press release's "nothing leaves this machine unchecked" is false as a
promise, because file reads and tool results leave unchecked; the record
should promise "no prompt leaves unchecked". The owed ADR on failing open
starts from a host that offers both postures: `onFailure: "block"` holds the
prompt when the hook fails or times out, and the default lets it through.
Hold-within-budget is a latency discipline, not the only fail-closed option,
and a held prompt's text still reaches the local transcript.

### Local fit risks

- Host-delegated LLM work: a local model at the prompt seam is the first
  model abcd would run in its own loop on every turn; latency and consent
  (the LOAD rule) apply.
- `loud-staging`: a check that did not run must say so on every prompt it
  missed, or the opt-in reads as protection.
- Host-agnostic prose: the hook's capabilities are one harness's; the
  intent's claim must be phrased by capability ("where the host lets a hook
  hold a prompt").

## itd-2609151516525843: a banlist on the first commit, and one for the machine

### Findings

- **The un-ignore candidate is ruled out by git's own text.** "It is not
  possible to re-include a file if a parent directory of that file is
  excluded. Git doesn't list excluded directories for performance reasons, so
  any patterns on contained files have no effect, no matter where they are
  defined." A fence that excludes the record directory cannot be carved by a
  `!` line; the declaration must live outside the fenced namespace or the
  fence must be rewritten to exclude files, not the directory. Precedence of
  sources: command line, `.gitignore` files, `$GIT_COMMON_DIR/info/exclude`,
  `core.excludesFile`. [EVIDENCE: docs] ([gitignore](https://git-scm.com/docs/gitignore))
- **Machine-wide patterns have one shipped precedent.** `git secrets --add
  --global` "Adds patterns to the global git config", read by hooks in every
  repository; `register-aws` ensures "keys present in `~/.aws/credentials` are
  not found in any commit"; patterns are "not guaranteed to catch them all".
  The store is `~/.gitconfig`, which dotfile repositories commit. [EVIDENCE:
  docs] ([git-secrets](https://github.com/awslabs/git-secrets))
- **Machine-wide hooks have two.** pre-commit's `init-templatedir` sets
  `init.templateDir` so "any newly cloned repository will automatically have
  the hooks set up"; repositories without a config "will be skipped";
  `core.hooksPath` moves the hooks directory for a repository or, set
  globally, for all; `git init may copy hooks to the new repository`.
  [EVIDENCE: docs] ([pre-commit](https://pre-commit.com/),
  [githooks](https://git-scm.com/docs/githooks))
- **The host already keeps a per-user ignore file.** Claude Code adds
  `**/.claude/settings.local.json` to the user's `core.excludesFile` (or
  `~/.config/git/ignore`) "so the file stays out of your commits in every
  repository", a user-level exclude as a machine-wide fence precedent.
  [EVIDENCE: docs] ([Claude Code settings](https://code.claude.com/docs/en/settings))
- **The CI boundary is the record's own scope condition.** A runner "clones
  the repository and has neither the home nor any right to it"; nothing in
  the field contradicts it, and no outside source is needed to state it.
  [LOCAL]

### Options weighed

1. Machine-global store in `~/.abcd.noindex/` in the existing keyed format,
   read by the committed hooks beside the per-repository private store;
   declaration as a committed file outside the fenced namespace.
2. Store the patterns in global git config as git-secrets does: puts the
   machine's most sensitive strings in a file people commit to dotfiles.
3. Global `core.hooksPath` or `init.templateDir` so even repositories abcd
   never prepared run the check: reaches "ones that do not exist yet" for
   real, but overrides every other tool's hooks on the machine.

### Recommendation

**Path 2 for the machine-global half: native, shaped to git-secrets'
machine-wide pattern layer for the reach and to pre-commit's template
directory for the "repositories that do not exist yet" claim, with the store
kept in abcd's home and never in git config.** For the declaration half the
survey has one negative finding and no shape: gitignore's rule puts the
declaration outside the fenced directory, and the ruling owed on invariant 5
chooses what it is. The template-directory route
(option 3) is the only one that honestly covers a repository abcd has never
touched; it should be offered as a separate, explicit opt-in because it
replaces the user's `init.templateDir`.

### Local fit risks

- `the-users-directory-is-theirs`: option 3 edits the user's global git
  configuration; the intent's own invariant that no project file reaches the
  home list is the mirror of that rule.
- Invariant 5 (one visibility switch) is a ruling owed before the spec; the
  gitignore finding narrows the shapes but does not choose one.
- Multi-user homes stay out of scope; git-secrets has the same gap.

## itd-2610071545369520: a person confirms the assistant's verdict per warned line

### Findings

- **Automation bias is robust and only partly mitigated.** Parasuraman and
  Manzey (Human Factors 2010, 52(3):381–410) review the evidence that
  complacency and automation bias "represent different manifestations of
  overlapping automation-induced phenomena", with commission and omission
  errors when aids are imperfect; Goddard, Roudsari and Wyatt (JAMIA 2012,
  19(1):121–127), a systematic review of 74 papers from 13,821 retrieved,
  found trust, workload, task complexity and time pressure raise it, and
  listed as mitigators "training and emphasizing user accountability", "the
  position of advice on the screen, updated confidence levels attached to
  DSS output", and "the provision of information versus recommendation".
  The two reviews differ on training: the bias the first calls robust to it
  is one the second lists it against. [EVIDENCE: study]
  ([Europe PMC record, Parasuraman](https://www.ebi.ac.uk/europepmc/webservices/rest/search?query=TITLE%3A%22Complacency%20and%20bias%20in%20human%20use%20of%20automation%22&format=json&resultType=core),
  [Europe PMC record, Goddard](https://www.ebi.ac.uk/europepmc/webservices/rest/search?query=%22automation%20bias%22%20AND%20AUTH%3A%22Goddard%20K%22%20AND%20PUB_YEAR%3A2012&format=json&resultType=core))
- **Explanations help only when checking is cheaper than deciding.**
  Vasconcelos et al. (CSCW 2023, N = 731 over five studies): overreliance is a
  cost-benefit choice; explanations reduce it when they "lower the effort
  needed to verify the AI's prediction", and null results elsewhere may be
  explanations that did not. Buçinca, Malaya and Gajos (N = 199): "cognitive
  forcing significantly reduced overreliance compared to the simple
  explainable AI approaches", at the cost of the least favourable ratings.
  [EVIDENCE: study] ([Vasconcelos](https://arxiv.org/abs/2212.06823),
  [Buçinca](https://arxiv.org/abs/2102.09692))
- **Line identity: hash the content, not the number.** GitHub code scanning
  "uses fingerprints to match results across various runs", "possible to
  match alerts to the correct line of code when files are edited", using
  `partialFingerprints.primaryLocationLineHash` with a stable `ruleId` and
  file path; the uploader computes it when a tool omits it. SARIF's
  `fingerprints`/`partialFingerprints` are "a stable value that can be used
  by a result management system to uniquely identify a result over time".
  Gitleaks' `.gitleaksignore` fingerprint ends in the line number and is
  "experimental"; that is the shape to avoid. [EVIDENCE: docs, standard]
  ([SARIF support](https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning),
  [SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html),
  [gitleaks](https://github.com/gitleaks/gitleaks))
- **Recording who decided and why.** GitHub: "you can comment on a dismissal
  to record the context", the comment "is added to the alert timeline and can
  be used as justification during auditing"; dismissed "in all branches";
  reopenable. detect-secrets persists labels in the baseline so a rescan
  "will also preserve any labelled secrets you have". Semgrep's `nosemgrep`
  takes a rule id but no reason, and "Ignoring code through this method still
  generates a finding". [EVIDENCE: docs]
  ([resolving alerts](https://docs.github.com/en/code-security/code-scanning/managing-code-scanning-alerts/resolving-code-scanning-alerts),
  [detect-secrets](https://github.com/Yelp/detect-secrets),
  [Semgrep ignores](https://docs.semgrep.dev/ignoring-files-folders-code))
- **The originating evidence is one test.** The managed repository's lab:
  frontier judge 97.8% with no missed protected use on 280 held-out lines,
  the local decision model 90.0% with ten misses; the report itself says "One
  test is not a track record". [EVIDENCE: study, single run, local]
  (iss-2610071538075997)

### Options weighed

1. Per-line verdict with a one-line reason, explicit approve or override per
   line, no accept-all; the allow marker carries a content hash, the
   decider's role, the time and the reason; overrides are counted.
2. Batch approval of all "ordinary" verdicts with one keystroke: the
   automation-bias literature's worst case.
3. Silent allow of high-confidence verdicts: the ruling already declined the
   silent local judge.

### Recommendation

**Path 2: native, shaped to GitHub's line fingerprint for identity and its
dismissal record for provenance, with the interaction designed from the
overreliance evidence**: the verdict and reason appear beside the line so
checking is cheaper than re-reading; each line needs its own answer; and
the marker records who, when and why. Whether a count of overrides is the
calibration signal the lab asked for, or the miss-rate review the
2026-10-07 ruling declined by another name, is the product thinker's to
say, so it is an open question rather than part of the path. Both open
questions in the press
release are answered this way: a content hash with the rule id identifies the
line, and the marker does record the decision.

### Local fit risks

- `widen-options-never-recommend` and the GRILL rules govern how the verdict
  is put to the person; a proposed verdict is a recommendation by construction,
  so the question must still quote the line in full and offer override first.
- Host-delegated: the judge is the session's model; the local shadow log the
  ruling declined stays declined.
- `one-canonical-primitive`: the allow marker should be the docs-lint escape
  that already exists, extended, not a second marker.

## itd-18: permissions from day one

### Findings

- **The host now says what the intent proposes is the fragile layer.** Bash
  permission patterns "that try to constrain command arguments are fragile";
  a deny rule "doesn't match the same program by path or inside `sh -c`"; and
  "For filesystem and network enforcement that doesn't depend on the command
  text, use sandboxing". An allow rule also "approves an unsandboxed retry".
  `Write(path)` rules "never match"; only `Edit(path)` and `Read(path)` do.
  [EVIDENCE: docs] ([permissions](https://code.claude.com/docs/en/permissions))
- **The file's own lifecycle.** `.claude/settings.local.json` is "You, in this
  one project only"; precedence managed > `--settings` > local > project >
  user; list keys merge across files; standing approvals are written to the
  file at the repository root, resolved "through worktrees to the main
  checkout"; the host adds it to the user's global excludes. [EVIDENCE: docs]
  ([settings](https://code.claude.com/docs/en/settings))
- **Each harness has a different file and grammar.** Codex: `sandbox_mode`
  plus `approval_policy`, writable roots under `sandbox_workspace_write`;
  opencode: `permission` rules per tool in `opencode.json`, "Most permissions
  default to `"allow"`" with `.env` reads denied; Gemini: `tools.sandbox` and
  profiles. [EVIDENCE: docs] ([Codex](https://learn.chatgpt.com/docs/sandboxing),
  [opencode](https://opencode.ai/docs/permissions/),
  [Gemini](https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/sandbox.md))
- **Project-type detection has a reference implementation.** Linguist
  detects by extension, filename and heuristics, reports the strategy per
  file, and honours `.gitattributes` `linguist-language` overrides. [EVIDENCE:
  docs] ([Linguist](https://github.com/github-linguist/linguist))

### Options weighed

1. Pre-populate `.claude/settings.local.json` per project type (the intent).
2. One abcd-owned project-type profile (tools, test and build commands)
   rendered to each harness's file by an adapter; in the Claude host, prefer
   the sandbox with `autoAllowBashIfSandboxed` to prefix allows.
3. Do nothing; the host's "don't ask again" accumulates the file anyway.

### Recommendation

**Path 2: a native project profile with a per-host renderer as the seam**,
shaped to the vendors' own advice that an OS boundary replaces most prefix
allows. The intent as drafted writes one harness's private file; the
host-agnostic rule and the reach to Codex and opencode both argue for the
profile first.

### Local fit risks

- Host-agnostic prose: the intent's press release names one host's file.
- `the-users-directory-is-theirs`: the local file is the person's; abcd
  proposes with a transparent confirm, never rewrites.
- Overlap with the sandbox intents: a template that allows `curl` is what the
  egress list exists to narrow.

## Not worth adopting

- **TruffleHog as the default secret scanner**: AGPL and live verification
  that sends found credentials to providers; keep as a configured option.
- **A full in-toto/SLSA/Rekor stack for the audit chain**: a public log for
  private transcripts, against the launch rule, and the framework is "still
  under development".
- **OpenTimestamps as the chain's anchor**: an outbound call per anchor and a
  Bitcoin dependency in every proof; a signed checkpoint pushed to the forge
  gives the same ordering guarantee for abcd's threat model.
- **`chflags uchg` or `chmod` on the primary checkout**: blocks git's own
  fast-forward and the allow-set.
- **A local model as the only prompt gate**: REDACT and PIIBench both put
  small detectors' recall where the press release cannot stand on it.
- **Global `init.templateDir` by default**: honest reach, but it replaces the
  user's template directory; opt-in only.
- **Writing banned names into global git config** (git-secrets' shape):
  dotfile repositories commit that file.

## Not verified

- Whether srt's macOS profile closes the DNS channel (the Seatbelt profile
  permits only the proxy port; the resolver path was not traced).
- Whether Codex still uses Landlock and seccomp anywhere on Linux; its
  current page names bubblewrap, the 2026-08-18 note named Landlock.
- Apple's intent for `sandbox-exec`: deprecated in the man page, no removal
  statement found; the macOS 26 failure reports are community posts.
- Johnson et al. (ICSE 2013) was read from the publisher's record, not the
  paper; the two automation-bias papers from Europe PMC records; the PMC and
  SAGE pages refused the fetch.
- gitleaks' and TruffleHog's precision in IssueGuard's comparison: the
  abstract page gave no per-tool numbers; the search snippet's figures were
  not confirmed on an opened page and are not used above.
- Whether the keychain credential helper and Go's TLS work under srt's
  profile for a push from inside the fence; the ADR lists both as platform
  limits and this pass found no primary source either way.
- The Veracode report itself; figures come from the vendor's press release.
- Presidio's licence and the `presidio-research` evaluation package: the new
  home page states neither.
- The Claude Code hooks page's event-specific input for `UserPromptSubmit`
  (the page was read to its first hundred thousand characters).
