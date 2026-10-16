---
id: spc-2610161751056096
slug: a-person-who-leaves-abcd-to-run-on-its
intent: itd-2610161621364075
origin: researcher-authored
production_mode: hand-written
---
# a-person-who-leaves-abcd-to-run-on-its

## Summary

This spec delivers part A of [itd-2610161621364075](../../intents/planned/itd-2610161621364075-a-person-who-leaves-abcd-to-run-on-its.md): every agent abcd's runner launches runs inside a fence the operating system enforces around the whole launched process. It enacts [adr-2610161621415708](../../decisions/adrs/2610161621415708-abcd-s-command-guard-steers-agents-an-os.md). Part B (reporting how isolated a person's own session is) and part C (the network promise) are separate records, by the intent's decision 1.

The launches it covers are the ones that pass through the runner's one launch point, `launcher.run` in `internal/core/runner/proc.go`:

- the loop's roles (implementer, reviewers, auditor) started by `implement/loop.Drive`, for `build` and for each `drain` lane;
- the planning interviewer started by `interview.Written`;
- a role that falls back to `runner.fallback_host`, which is another runner launch.

`runner.Exec`, which runs a vendor binary for abcd's own short calls such as a quota reading, starts no agent and is not fenced here.

Building waits on two records, by decision 5: the worktree store (itd-2609091014076309) and the guard's per-launch approver (iss-2610161612203555, on branch `feat/headless-approvals` with its decision record). Steps 3 to 8 can be built before the approver lands. Step 9 cannot.

### Which fence

Ruling 6 asks for Anthropic's sandbox while abcd launches Claude, prefers a state-of-the-art open-source fence abcd adopts, and rules out a fence of abcd's own. Planning was to check first whether Anthropic's sandbox already meets that preference for every harness. It does, and it is the recommendation for both runners.

**Anthropic's sandbox-runtime (`srt`)** ([repository](https://github.com/anthropics/sandbox-runtime), read 2026-10-16; npm `@anthropic-ai/sandbox-runtime` 0.0.79, published 2026-10-07):

- Apache-2.0. It is the code Claude Code's own shell sandbox is built on, and the vendor's guide names it as the way to put a whole Claude Code process behind one boundary, file tools, MCP servers and hooks included ([sandbox environments](https://code.claude.com/docs/en/sandbox-environments), [sandboxing](https://code.claude.com/docs/en/sandboxing)).
- It wraps any process: its README names "agents, local MCP servers, bash commands and arbitrary processes". So the same fence serves opencode, which settles how opencode launches are fenced, the question decision 6 left open.
- On macOS it runs `sandbox-exec` with a Seatbelt profile it generates. On Linux it runs bubblewrap without a network namespace, bridges a host-side proxy in over `socat`, and adds a seccomp filter that blocks creating Unix sockets.
- Writes are denied everywhere by default, and a deny wins over an allow. Reads are allowed everywhere by default.
- Some writes are denied even inside an allowed path: `.git/hooks`, `.git/config`, the shell's startup files, `.gitconfig`, `.gitmodules`, `.mcp.json`, `.claude/commands` and `.claude/agents`, from the wrapped process's directory down.
- A settings file that is named but missing, empty or invalid makes it refuse to start rather than fall back to its defaults.
- Maturity: the README calls it a "Beta Research Preview" whose "APIs and configuration formats may evolve". It needs Node 22.12 or later. On Linux it also needs bubblewrap 0.4 or later, `socat`, `ripgrep`, and unprivileged user namespaces; Ubuntu 24.04 restricts those by default, which is the intent's fifth scope condition.

The other open-source options, against the same need (a whole-process boundary on macOS and Linux that abcd adopts rather than writes):

- **nono** ([always-further/nono](https://github.com/always-further/nono)): Apache-2.0, a single Rust binary, Seatbelt on macOS and Landlock on Linux, with profiles for Claude Code and OpenCode. It is the nearest rival and needs no user namespaces. It is pre-1.0 ("APIs are stabilizing"), it is not Anthropic's, and ruling 6 asks for Anthropic's while abcd launches Claude. Two fences for two runners would be two policies to keep in step ([`one-canonical-primitive`](../../principles/one-canonical-primitive.md)). It stays the first alternative if srt fails the probes in step 2.
- **landrun** ([Zouuup/landrun](https://github.com/Zouuup/landrun)) and **go-landlock** ([landlock-lsm/go-landlock](https://github.com/landlock-lsm/go-landlock)): MIT, Go, Landlock only. That makes them Linux only, with no macOS half. Linux 5.13 or later is needed for files and 6.7 for TCP. Landlock needs no user namespaces, which is its real advantage over bubblewrap. Using the library would leave abcd writing its own policy layer, and ruling 6 rules out a fence of abcd's own.
- **bubblewrap** ([containers/bubblewrap](https://github.com/containers/bubblewrap)): LGPL-2.1+, mature (Flatpak, rpm-ostree), Linux only. By its own README it "is not a complete, ready-made sandbox with a specific security policy". It is the building block srt already uses.
- **nsjail** ([google/nsjail](https://github.com/google/nsjail)): Apache-2.0, Linux only, namespaces and seccomp configured in protobuf. It is built to jail services and fuzz targets, needs user namespaces as bubblewrap does, and has no macOS half.
- **firejail**: GPL-2.0 and a setuid-root binary. Its attack surface is the reverse of what this record wants, as CVE-2022-31214 showed: a local root escalation through its join path ([Debian tracker](https://security-tracker.debian.org/tracker/CVE-2022-31214)).
- **`sandbox-exec` with a profile of abcd's own**: no dependency, but abcd would author and maintain the Seatbelt policy, which is the fence of its own ruling 6 rules out. Its man page also marks the command "DEPRECATED". Every macOS option above runs on the same mechanism, so this deprecation is a risk they share, recorded in the ADR's consequences.
- **Codex's sandbox**: Seatbelt on macOS and bubblewrap or Landlock on Linux, Apache-2.0. It ships inside the Codex CLI, so adopting it would mean installing another vendor's agent. It is the candidate to reconsider when abcd launches Codex, by decision 6.

**Recommendation: srt for both the claude and the opencode runner, on macOS and Linux.** It is the fence ruling 6 names for Claude, it already wraps any process, and one fence means one policy derivation. The choice is recorded in dcn-2610161809259623.

[`prefer-sota`](../../principles/prefer-sota.md) asks for an adversarial fit-challenge before the verdict, and one was run on 2026-10-16. It kept srt for both runners, since most of what it found about opencode comes from opencode's own state and would hit any fence. It changed the design in six places, each written in below:

- the main git directory is narrowed, with a ref check after each launch;
- srt's interpreter and package are admitted, not only its script;
- the prompt's passage through srt's shell is tested;
- srt's leftovers are cleared after a killed launch;
- opencode's state is seeded rather than dropped;
- reads of the credential stores a launch never needs are denied.

It also named two things to probe rather than assert, the macOS hard-link gap and the harnesses' behaviour under a fresh state folder (step 2). nono is reopened only if a probe finds a gap in srt that cannot be worked around.

**The dependency.** abcd does not vendor srt and `go.mod` does not change. The person installs it once, like the harness it wraps. It is a new external program on the runner route, though, and on that route it is a requirement, not an adapter that degrades: by ruling 7 a run without it stops. That reverses the dependencies chapter's rule that a missing tool degrades to a native path, for this one route. Step 1 waits for the person's sign-off on it, and amends the chapter when given.

**What the fence brings that part A did not ask for.** srt filters the network to a list of hosts, and it refuses `*` in that list ("Overly broad patterns like "*.com" or "*" are not allowed for security reasons", `sandbox-config.js` in 0.0.79). So a fenced launch reaches only the hosts it is given, whether or not part C has been built. The list here is derived for function, not as part C's promise:

- the runner's model endpoint: `api.anthropic.com` for claude, or the host of `ANTHROPIC_BASE_URL` when it is set; for opencode, the endpoint of the provider its model route names;
- the hosts of `GOPROXY` and `GOSUMDB` when the worktree holds a `go.mod`;
- the hosts the person adds in `runner.sandbox.hosts`, in their machine layer only.

A lane's agent never pushes: its brief says "Never push", and the land stage pushes from abcd, outside the fence (`landPush`, `internal/core/implement/loop/land.go`). So the forge is not on the list: only abcd's own push, outside the fence, reaches it. Whether a fenced launch may reach the network only through a host list is the second point the person's sign-off covers in step 1.

### What a launched agent may write

The write list is derived from the launch itself; nothing in the repository widens it. Every other path is read-only. On Linux srt takes literal paths only, so each entry is resolved to a real absolute path before the settings file is written (`fsutil.EnsureRealDirAll`, so a symlinked component is refused).

- **The worktree** the role runs in (`Request.Dir`). For a lane that is `~/.abcd.noindex/worktrees/<root-sha>/<run-id>-<lane-id>`. For the planning interviewer, which runs in the checkout, it is not the checkout: the interviewer gets the folder holding the intent record it is editing, and the interview's changed-paths check keeps refusing every other change, as today.
- **The receipt folder**: the lane's directory `.abcd/.work.local/run/<run-id>/<lane-id>/` (its `brief.md` and `receipt.json`), or the interview's turn directory. The run's `state.json`, its lock and its fingerprint stay readable and are not writable.
- **The main checkout's git directory**, narrowed to what a commit in the lane writes:
  - `objects/`;
  - the lane's own `worktrees/<name>/`, with its `config.worktree`, `gitdir` and `commondir` denied;
  - the lane branch namespace `refs/heads/build/` and its reflogs under `logs/refs/heads/build/`.

  Nothing grants `hooks/`, `config`, `packed-refs`, `HEAD`, the main index or any other worktree's folder, so what decides what git runs, and every branch outside the loop's namespace, stays out of reach. srt's own `.git` denies do not cover this: they reach from the wrapped process's directory down, and in a linked worktree `.git` is a file pointing at the main one.

  On Linux srt grants folders, not single files, and `build/` holds every lane's branch. So after each launch abcd compares every ref, read outside the fence, with the list it took before. A launch that moved any ref but its lane's own branch is recorded, and its lane refuses to land.
- **A fresh Go cache per run**, by ruling 8: `GOCACHE` and `GOMODCACHE` point into the run's fence directory. The module cache is filled from the person's own cache through `GOPROXY=file://<their GOMODCACHE>/cache/download,<their GOPROXY>`, a form `go help goproxy` documents. A cached module is copied in without the network and checked against `go.sum` as usual. The person's caches are only read.
- **Temp**: a fresh directory per launch, set as `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR`. The system's `/tmp` is not granted, except srt's own built-in paths.
- **The host's state**, kept apart from its configuration:
  - claude: a fresh `CLAUDE_CONFIG_DIR` per launch. Under `--bare` the harness reads no OAuth credential and no keychain, and takes its key from `ANTHROPIC_API_KEY` or an `apiKeyHelper` passed with `--settings` ([headless](https://code.claude.com/docs/en/headless)), so the person's `~/.claude` and `~/.claude.json` are never needed and stay read-only. That also covers the paths the vendor warns about: "a sandboxed session that can write them can persist hooks, permission rules, or MCP servers that run unsandboxed the next time you launch Claude Code".
  - opencode: fresh `XDG_DATA_HOME`, `XDG_STATE_HOME` and `XDG_CACHE_HOME` per launch, because opencode downloads and later runs programs from its data and cache folders. abcd seeds the fresh cache with a copy of the person's, outside the fence, so a launch does not fetch its provider packages again, and nothing it writes there flows back. The person's configuration folder stays where it is, read-only, since it names the model and the server (`opencode.go`). The person's credential file is reached through a read-only link from the fresh data folder. A provider whose sign-in rewrites that file cannot refresh inside the fence: the launch fails, naming it, and the person uses a key for runner launches. abcd sees only the `<provider>/<model>` route, not a base URL set in opencode's configuration, so a provider abcd does not know, or a local model server, is reached by adding its host to `runner.sandbox.hosts`. Step 2 measures all of this.
- **abcd's home** (`~/.abcd.noindex/`) is readable and is not writable, except the launch's own worktree inside it. So the next launch's policy (`config.json`, `rules.json`, `trusted-roots`, `load-limits`, the other worktrees) is the person's alone to change.

**Reads.** Reads stay open, as srt's default leaves them, except for the credential stores a launch never needs. Those are denied:

- `~/.ssh` and `~/.gnupg`;
- `~/.aws`, `~/.config/gh`, `~/.netrc`, `~/.git-credentials` and `~/.docker/config.json`;
- abcd's own credential file under its home.

The vendor's guide says why the two layers go together: "Without network isolation, a compromised agent could exfiltrate sensitive files like SSH keys". The model endpoint is reachable, so this narrows what a misled agent can send there. It is not part C's promise.

Writes srt already allows outside the list (the child's standard streams, `/tmp/claude`, `~/.npm/_logs`, `~/.claude/debug`) are named in the run record's fence block as the fence's own, so the list abcd states is the list that holds.

The policy goes into a settings file abcd writes for each launch, inside the run's fence directory but outside every path the launch may write. It is never `~/.srt-settings.json`, and `--control-fd` is not used, so nothing can change the policy while the launch runs.

### A machine without a fence

Ruling 7: the run stops until the person acts. When a run is about to start and any role it may launch is routed to a runner, abcd checks the fence before creating the run. That covers a `build` start, each `drain` lane's start, and an `intent interview` start. The check:

- finds `srt` on PATH and admits it as a harness is admitted (`launcher.admit`): an absolute path, not writable by group or other, not inside the repository or the checkout. srt's own README warns that a copy inside a writable path can be rewritten from inside the sandbox. srt is a Node script, so the check also covers the interpreter its first line names, found on PATH, and the package folder the script resolves into, and none of them may sit inside a path any launch may write;
- reads its version and refuses any but the one abcd was tested against, 0.0.79, naming the version found and the one to install. A 0.0.x version promises nothing between releases, so a range would be a guess. The pin moves in a change that runs the `fence` test lane on the new version;
- runs a trial launch, `srt --settings <trial> -- /bin/sh -c ': > "$1"; : > "$2"' sh <inside> <outside>`, and checks from outside that the write inside landed and the write outside did not. That makes "fenced" a fact, not an assumption, per [`enforcement-claims-are-facts`](../../principles/enforcement-claims-are-facts.md).

Any failure refuses the run: exit 2, nothing created and nothing launched. The refusal names the reason (srt not found, refused, the wrong version, bubblewrap or user namespaces missing as srt reports, or the trial writing outside), how to fix it, and the off switch.

The off switch is `runner.sandbox.off: true` in `~/.abcd.noindex/config.json`. The `runner` namespace is already the machine's alone, and a repository's `.abcd/config.json` that sets it is refused, naming the machine's file (`runner.Load`). No flag sets it either, so a project's files and a command an agent can type can never turn the fence off. With it set, the run says once, at its start, that its launches are not fenced and where that was set.

A wrap that fails during a run (srt removed, its settings refused) is a new runner reason, `fenced`. It is not a fallback: like a rate limit, it hands the role to no other route. The lane refuses at its runner stage with the same remedy.

### The guard inside the fence

The approver runs **inside** the fence. The harness starts `abcd guard approve` itself, as the one MCP server on stdio its `--mcp-config` names (`approverConfig` on `feat/headless-approvals`), and a child of the wrapped process is inside the wrapper. Keeping it inside costs nothing and is the safer side:

- it only reads: abcd's own binary, the worktree's `.abcd/guard.json` and the bundled registry, all readable in the fence. It writes nothing;
- its answer can only narrow what the agent does. An allow it gives still runs inside the fence, so an approver that is wrong, or fooled by the input it judges, can never let a command past the boundary;
- putting it outside would mean a channel crossing the fence, carrying input the agent controls, to a process with the person's full reach. That channel would be a way out, and abcd would have to build it, which ruling 6 rules out. On Linux, srt's seccomp filter blocks exactly that kind of Unix-socket bridge.

So the guard keeps its place as adr-42's mistake filter, teaching inside the boundary the OS enforces. One limit is stated, not fixed: the approver judges against the registry of the working tree, which the agent can edit. A registry switched off is refused (`registry-not-in-force`). An entry the agent deletes stops teaching for the rest of that launch, and the fence still bounds what follows.

### Checks that read the machine run outside

The load gate, the peers scan, and the reservation renewals and fences the loop takes between stages already run in abcd, outside any launch. What changes is what happens to a gate an agent reaches inside the fence: `make preflight` runs its test stages through `abcd implement load --wait`, and inside the fence that waiter would read the fence's view of the process table. On Linux the process namespace shows it only its own processes. It also could not write its hold under abcd's home.

So a waiter never judges the machine from inside a fence:

- a waiter that finds its launch's gate directory (an `ABCD_FENCE_GATE` the launch sets) asks through it, and waits for an answer;
- a waiter that cannot write its hold refuses, naming the fence, rather than opening. That covers an agent that unsets the variable;
- for each request, the runner starts, outside the fence, an ordinary `abcd implement load --wait --gate <dir>` process. It judges the whole machine as every waiter does, visible in the process table to every other account's waiters. When the gate opens it takes the hold, and writes its answer where the fenced side can read it and cannot write.
- that outer waiter then holds the gate for the inner command's length, as `--wait -- <command>` does, until the inner side writes its done marker or the launch ends;
- one request may be outstanding per launch, and the outer waiter's own timeout bounds it.

A preflight run inside the fence therefore runs its commands inside the fence, under a gate read outside it. Its receipt lands in the worktree's local tier, which the agent may write.

### How each criterion is met

1. **A write outside is refused and recorded.** The derived list (step 3) is what the wrapped launch can write (step 7). The fence's refusals are read back from the launch's stderr and the tool errors in its transcript, and written to the lane's run record under `fence.refusals`, each with its path and the tool that tried (step 8).
2. **Edit, commit and push succeed.** The worktree, and the parts of the main git directory a lane's commit writes, are writable (step 3), so the edit and the commit land. The push is the land stage's, outside the fence, as it is today. Step 13's live run shows all three.
3. **abcd's settings, rules, trusted roots and other worktrees are refused.** abcd's home is read-only except the launch's own worktree (step 3). Step 8's real-fence test writes each of them and asserts the refusal.
4. **No fence: the run is refused before its first launch.** Steps 5 and 6.
5. **The off switch is the person's alone and said once.** Steps 4 and 6.
6. **A fresh Go cache of its own.** Step 11.
7. **The load gate reads the whole machine from outside.** Step 10.
8. **The guard's refusal reaches the agent inside.** Step 9.
9. **Host settings byte-identical.** Step 12.

### Proof

The tests that matter run against the real fence. A test lane under the build tag `fence` runs real srt on macOS and on a Linux runner with bubblewrap installed, wrapping a stand-in harness: a script named `claude` or `opencode` on a PATH the test controls, which tries the writes, commits and gates each criterion names. Fakes stand in only for what a test cannot cause, such as a missing srt or an out-of-range version. Before this spec closes, one `build` of a real intent runs fenced on the built binary with the real claude runner (and the real opencode runner where routed), on macOS and on Linux, and its run record and the before-and-after hashes of the person's host settings are the evidence (step 13), per the product thinker's rule that real runs are the proof.

### Known limits, stated, not claimed

- On macOS srt's README gives "no guarantee" for a hard link, inside an allowed path, to a file outside it.
- On Linux srt's own deny scan is fixed at launch. abcd's explicit denies cover the main git directory, which exists before the launch, and not a repository the agent creates later inside its worktree.
- Go programs that check TLS through the system's trust service can fail under Seatbelt (the vendor's guide names `gh`, `gcloud` and `terraform`). Step 2 measures whether `go` itself downloads modules inside the fence, and records what it finds.
- On Linux, srt covers a protected path that does not yet exist with a placeholder on the host, and a SIGKILL skips its cleanup. abcd's timeout kill is the group's SIGKILL (`procgroup.Kill`), so a timeout first sends SIGTERM to srt and kills the group only after a grace period. After every launch abcd also removes, outside the fence, any empty placeholder that appeared at one of srt's protected names and that git does not track, so none is committed or left in the person's worktree. Step 2 checks whether bubblewrap's own session takes descendants out of the group abcd kills.
- srt quotes the argument vector into a shell string on macOS and Linux, which amends `proc.go`'s rule that the argv is never a shell line. abcd's wrapped vector is fixed words, a role name from a closed list, and paths abcd made. A path carrying a quote or a control character is refused before the launch, and a test sends a prompt through with every shell metacharacter (step 7).
- The intent's press release says the person's other projects and settings "stay out of its reach". With this design that holds for changes. It holds for reads only where the read denies above reach.

### Coordination

- Steps 3 to 8, 10 and 11 change `internal/core/runner` and the new `internal/core/fence`, so they land in order.
- Step 9 waits for `feat/headless-approvals` to merge, and reserves `internal/core/runner/claude.go` after it.
- Step 10 changes `internal/core/implement/load_wait.go`, which every preflight runs through, so it lands alone, after a reservation.
- Every step reserves its footprint with `abcd reserve` before it starts.

### Out of scope

- Part B: the isolation report for drains and builds a person's own session drives. Part C: the network promise and its claim (the host list here is functional, not a promise).
- Confining reads. Secrets the account can read stay readable to a fenced agent.
- Windows, cloud sessions, and more than one project per run (the intent's scope conditions).
- Caches of other toolchains (npm, cargo, pip): writes to them are refused, and a run that needs one shows up in step 13's evidence before anything is added for it.

## Footprint

- packages: internal/core/fence, internal/core/runner, internal/core/implement, internal/core/implement/loop, internal/core/interview, internal/procgroup, internal/surface/cli, commands, docs, ACKNOWLEDGEMENTS.md, .abcd/development/brief, .abcd/development/research/notes, .abcd/development/decisions/adrs
- tests: internal/core/fence (policy derivation, admission and trial tables), internal/core/runner (fenced argv, environment, kill order, the fenced reason, config tests), internal/core/implement (the fenced waiter and its outer gate), internal/core/implement/loop and internal/core/interview (refusal before a run starts, the fence block of the run record), internal/surface/cli (build, drain, interview and status surface tests), the `fence`-tagged real-srt lane on macOS and Linux, and step 13's live fenced build

## Steps

1. Sign-off: the person approves srt as a required program on the runner route, and the host list it brings (waits for the person; nothing is built before it)
   - packages: .abcd/development/brief, .abcd/development/decisions/adrs
   - tests: record-lint and docs-lint over the amended dependencies chapter (02-constraints/02-dependencies.md), which names srt as the runner route's one requirement that does not degrade, and the ADR's acceptance
2. Probe srt on the real harnesses before the design is built
   - packages: .abcd/development/research/notes, .abcd/development/intents
   - tests: an abcd lab on macOS and on Linux with bubblewrap, each probe named in the research note: `claude --print --bare` and `opencode run` wrapped by srt 0.0.79 with the step 3 policy written by hand; `CLAUDE_CONFIG_DIR` per launch with the person's `~/.claude` and `~/.claude.json` read-only; opencode with fresh XDG folders and a read-only credential link, and a refreshing sign-in; `go build` and `go test ./...` of abcd itself with the per-run caches, and whether module downloads pass TLS under Seatbelt; a commit in a linked worktree under the narrowed main `.git`, and `git gc --auto` with `packed-refs` unwritable; a hard link from the worktree to a file outside it on macOS; SIGTERM cleanup, and whether bubblewrap's session leaves the process group abcd kills; what opencode fetches at start with a seeded cache; a prompt with shell metacharacters. Each limit found becomes a scope condition on the intent or a capture of its own, and a failure on the macOS or Linux half stops the step and goes back to the person, with nono as the named alternative
3. Derive a launch's fence policy (itd-2610161621364075 criteria 1, 2, 3)
   - packages: internal/core/fence
   - tests: fence: for a lane launch, for an interview launch, and for each runner, the write list is exactly the worktree (or the record's folder), the receipt folder, the main `.git`'s `objects/`, the lane's own `worktrees/<name>/` without its `config.worktree`, `gitdir` and `commondir`, and `refs/heads/build/` with its reflogs, the run's Go caches, the launch's temp and host-state folders; `hooks/`, `config` and `packed-refs` are never granted; abcd's home is read-only but for the worktree; the credential stores are denied for reads; a path given through a symlink is refused; nothing read from the repository's `.abcd/config.json` changes the list; the host list is the runner's endpoint, `GOPROXY`/`GOSUMDB` with a `go.mod`, and the machine layer's additions; the settings file round-trips through srt's schema
4. Read the off switch and the host additions from the machine layer only (criterion 5)
   - packages: internal/core/runner, .abcd/development/brief
   - tests: runner: `runner.sandbox.off` and `runner.sandbox.hosts` read from the machine file; the same keys in a repository's `.abcd/config.json` refuse the load, naming the machine's file; a host entry that is `*`, a bare top-level domain or not `host[:port]` is refused; no flag sets either
5. Admit srt and prove the fence runs (criterion 4)
   - packages: internal/core/fence, internal/core/runner
   - tests: fence: srt absent, relative, group- or other-writable, inside the repository, an interpreter or package folder that fails the same checks, or a version other than the pinned one is refused with its reason and the off switch; `fence`-tagged: the trial launch's inside write lands and the outside write does not, on macOS and Linux; a trial whose outside write lands refuses
6. Refuse a run before its first launch, and say once when the fence is off (criteria 4, 5)
   - packages: internal/core/implement/loop, internal/core/interview, internal/surface/cli, commands
   - tests: loop and cli: a `build`, a `drain` lane start and an `intent interview` with a role routed to a runner, on a machine whose fence check fails, exit 2 before the run is created and start no process; a run whose roles all stay on the host checks nothing; with the switch on, the run's output says once that its launches are not fenced and names the machine file
7. Wrap every launch at the launch point (criteria 1, 2)
   - packages: internal/core/runner, internal/core/fence, internal/procgroup, internal/core/implement/loop, ACKNOWLEDGEMENTS.md
   - tests: runner: a fenced claude and opencode launch run `srt --settings <file> -- <harness> …` with the harness's own argv unchanged after `--`, and the environment sets the per-launch temp, Go caches and host-state folders over the parent's; the settings file sits outside every writable path; a timeout sends SIGTERM to srt and kills the group after the grace; a wrap failure is reason `fenced`, records no fallback and runs no other route; loop: a lane whose launch is refused as fenced refuses at the runner stage naming the remedy; a path carrying a quote or a control character is refused before the launch; a launch that moved a ref other than its lane's branch is recorded and its lane refuses to land; an empty untracked placeholder at one of srt's protected names is removed after the launch, and a tracked or non-empty file there is left alone; `fence`-tagged: a prompt carrying every shell metacharacter arrives byte for byte; the run's fence folder is removed when the run finishes
8. Record the fence's refusals in the run record (criteria 1, 3)
   - packages: internal/core/runner, internal/core/implement/loop, internal/surface/cli
   - tests: `fence`-tagged: a stand-in harness that writes outside its worktree, to `~/.abcd.noindex/config.json`, `rules.json`, `trusted-roots` and another lane's worktree is refused by the OS each time, and the lane's run record lists each refusal with its path; `implement status` renders the fence block (mechanism, version, policy digest, refusals) and its `--json` carries it
9. Keep the guard teaching inside the fence (criterion 8; waits for `feat/headless-approvals`)
   - packages: internal/core/runner, internal/surface/cli
   - tests: `fence`-tagged: a stand-in harness speaking the prompt tool's MCP to `abcd guard approve` started inside srt gets a refused command answered `deny` with the guard's reason and advice, and a cleared one answered `allow`; the approver's run leaves no fence refusal of its own
10. Read the load gate outside the fence (criterion 7)
   - packages: internal/core/implement, internal/core/runner, internal/surface/cli, commands
   - tests: implement: a waiter that cannot write its hold refuses naming the fence and never opens; a waiter with a gate directory opens only on the outer waiter's answer; an outer waiter given a loaded snapshot keeps the gate shut while the fenced side's own view is idle; it holds while the inner command runs and releases on the done marker or the launch's end; a second outstanding request from one launch is refused; cli: `implement load --wait --gate` is in help and the generated reference
11. Give each run a fresh Go cache (criterion 6)
   - packages: internal/core/fence, internal/core/runner, internal/core/implement/loop
   - tests: fence: `GOCACHE` and `GOMODCACHE` point into the run's fence folder and `GOPROXY` leads with the person's module cache as a file proxy; `fence`-tagged: a `go build` in a fenced launch fills the run's caches, and the person's `GOCACHE` and `GOMODCACHE` listings and modification times are unchanged
12. Leave the person's host settings untouched (criterion 9)
   - packages: internal/core/fence, internal/core/runner
   - tests: `fence`-tagged: a stand-in claude that writes `~/.claude/settings.json`, `~/.claude.json` and `~/.claude/agents/`, and a stand-in opencode that writes its configuration folder, are refused; the hashes of every host configuration path the policy names are identical before and after a fenced launch
13. State the rule and prove it on a real run
   - packages: .abcd/development/brief, commands, docs
   - tests: a brief invariant says an agent abcd launches runs inside a fence the operating system enforces, unless the person turned it off in their user layer, and the adapters chapter describes the fence; `commands/build.md` and `commands/intent.md` name the refusal and the off switch; a how-to under `docs/` says how to install the fence and turn it off, crediting the package under the docs-lint allow escape; docs-lint and record-lint pass; a fenced `build` of a real intent on the built binary, on macOS and on Linux, records a refused outside write, commits in its worktree, lands its push, and leaves the person's host settings byte-identical
