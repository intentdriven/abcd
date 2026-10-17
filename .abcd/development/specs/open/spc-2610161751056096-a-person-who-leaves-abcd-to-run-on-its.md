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

Building waits on two records, by decision 5: the worktree store (itd-2609091014076309) and the guard's per-launch approver (iss-2610161612203555, on branch `feat/headless-approvals` with its decision record). Steps 3 to 10 and 12 to 14 can be built before the approver lands. Step 11 cannot. Step 12, the load gate read outside the fence, does not depend on the fence at all and can land first. Nothing is built before step 2, where the product thinker answers the questions listed under Questions for the product thinker below.

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

- **nono** ([always-further/nono](https://github.com/always-further/nono)): Apache-2.0, a single Rust binary, Seatbelt on macOS and Landlock on Linux, with profiles for Claude Code and OpenCode. It is the nearest rival and needs no user namespaces. It is pre-1.0 ("APIs are stabilizing"), it is not Anthropic's, and ruling 6 asks for Anthropic's while abcd launches Claude. Two fences for two runners would be two policies to keep in step ([`one-canonical-primitive`](../../principles/one-canonical-primitive.md)). It stays the first alternative if srt fails the probes in step 1, and the fence seam below makes it a second adapter rather than a rewrite.
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

It also named two things to probe rather than assert, the macOS hard-link gap and the harnesses' behaviour under a fresh state folder (step 1). nono is reopened only if a probe finds a gap in srt that cannot be worked around.

An advisor's fit review the same day kept the pick again and changed the design in eight more places, each written in below: Go's checksum database and private modules, commit signing, automatic gc, the fence as a seam, the version policy, the approver's executable, srt as adr-22's first declared exception, and the order and split of the steps.

**The dependency.** abcd does not vendor srt and `go.mod` does not change. The person installs it once, like the harness it wraps. It is a new external program on the runner route, and on that route it is a requirement, not an adapter that degrades. [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) makes every external tool an opt-in adapter over a native default that a missing tool degrades to. A fence has no native default to degrade to: abcd writes no fence of its own (ruling 6), and an unfenced launch is the very thing the intent removes. So srt is framed as adr-22's first declared exception, scoped to the runner route alone. Everywhere else abcd still needs no external program, and in place of degradation the person has the per-machine opt-out, `runner.sandbox.off` (ruling 7). The exception is written as a consequence on adr-22 and as one paragraph in the dependencies chapter (`02-constraints/02-dependencies.md`), both marked proposed until the product thinker signs off at step 2 (question a).

**What the fence brings that part A did not ask for.** srt filters the network to a list of hosts, and it refuses `*` in that list ("Overly broad patterns like "*.com" or "*" are not allowed for security reasons", `sandbox-config.js` in 0.0.79). So a fenced launch reaches only the hosts it is given, whether or not part C has been built. The list here is derived for function, not as part C's promise:

- the runner's model endpoint: `api.anthropic.com` for claude, or the host of `ANTHROPIC_BASE_URL` when it is set; for opencode, the endpoint of the provider its model route names;
- the hosts of `GOPROXY` and `GOSUMDB` when the worktree holds a `go.mod`;
- with a `go.mod`, the hosts named by `GOPRIVATE` and `GONOPROXY`. Go fetches those modules straight from their version-control host, past every proxy, the file proxy below included, so without the host a private module cannot be fetched at all. The host part of each pattern is taken. A pattern whose host srt refuses as too broad is named at the check, and the person adds the concrete host in `runner.sandbox.hosts`;
- the hosts the person adds in `runner.sandbox.hosts`, in their machine layer only.

A lane's agent never pushes: its brief says "Never push", and the land stage pushes from abcd, outside the fence (`landPush`, `internal/core/implement/loop/land.go`). So the forge is not on the list: only abcd's own push, outside the fence, reaches it. Whether a fenced launch may reach the network only through a host list, and whether this is the list, is question (c), answered at step 2.

### The fence is a seam

The fence sits behind a seam, as the brief's rule for every capability taken from an external tool asks ([internals § 7, the paragraph that opens "Each seam is a Go interface"](../../brief/05-internals/04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching)): a Go interface, consumers that depend on the interface and never on a vendor, a backend selected by configuration, and a new backend added by implementing the interface and registering it, with no edit to the consumers. A new package, `internal/core/fence`, holds:

- `Policy`, which is fence-neutral: the paths a launch may write, the paths it may not read, and the hosts it may reach. Steps 3 and 13 derive it, and nothing in it names srt.
- `Fence`, the interface each mechanism implements:
  - `Check` admits the mechanism, reads its version and makes a trial launch, and returns the evidence: the admitted paths, the version, whether that version is one abcd was tested against, and the trial's two writes;
  - `Wrap(policy, argv, env)` returns the argv and the environment to start, and a cleanup to run after the launch ends, however it ends;
  - `Refusals(stderr, transcript)` reads the mechanism's refusals back out of a finished launch.
- `internal/core/fence/srt`, the one adapter. It writes srt's settings file from a `Policy`, builds srt's argv, and parses srt's refusals.

The adapter is chosen by `runner.sandbox.backend`, in the machine layer only like the rest of `runner`, and its default is `srt`. Two departures from the rule are deliberate. There is no native default to fall back to, so a missing or failing backend refuses the run (ruling 7) rather than degrading. And the interface lives in `internal/core/fence`, not under `internal/adapter/`, because deriving a policy from a launch is core logic the runner and the loop share; the adapter sits beneath it. nono, the named alternative, or Codex's sandbox once abcd launches Codex, would be a second adapter, and neither the runner nor the loop would change.

### What a launched agent may write

The write list is derived from the launch itself; nothing in the repository widens it. Every other path is read-only. On Linux srt takes literal paths only, so each entry is resolved to a real absolute path before the settings file is written (`fsutil.EnsureRealDirAll`, so a symlinked component is refused).

- **The worktree** the role runs in (`Request.Dir`). For a lane that is `~/.abcd.noindex/worktrees/<root-sha>/<run-id>-<lane-id>`. For the planning interviewer, which runs in the checkout, it is not the checkout: the interviewer gets the folder holding the intent record it is editing, and the interview's changed-paths check keeps refusing every other change, as today.
- **The receipt folder**: the lane's directory `.abcd/.work.local/run/<run-id>/<lane-id>/` (its `brief.md` and `receipt.json`), or the interview's turn directory. The run's `state.json`, its lock and its fingerprint stay readable and are not writable.
- **The main checkout's git directory**, narrowed to what a commit in the lane writes:
  - `objects/`;
  - the lane's own `worktrees/<name>/`, with its `config.worktree`, `gitdir` and `commondir` denied;
  - the lane branch namespace `refs/heads/build/` and its reflogs under `logs/refs/heads/build/`.

  Nothing grants `hooks/`, `config`, `packed-refs`, `HEAD`, the main index or any other worktree's folder, so what decides what git runs, and every branch outside the loop's namespace, stays out of reach. srt's own `.git` denies do not cover this: they reach from the wrapped process's directory down, and in a linked worktree `.git` is a file pointing at the main one.

  On Linux srt grants folders, not single files, and `build/` holds every lane's branch. So after each launch abcd compares every ref, read outside the fence, with the list it took before. A launch that moved any ref but its lane's own branch is recorded, and its lane refuses to land (step 9).
- **A fresh Go cache per run**, by ruling 8: `GOPATH`, `GOCACHE` and `GOMODCACHE` point into the run's fence directory. `GOPATH` moves with the caches because `go` writes its checksum-database cache to `$GOPATH/pkg/sumdb/`, outside `GOMODCACHE`: with the person's `GOPATH` read-only, a build that must verify a module not yet in `go.sum` fails. The move also sends `go install`'s binaries to the run's `$GOPATH/bin`, inside the fence. The module cache is filled from the person's own cache through `GOPROXY=file://<their GOMODCACHE>/cache/download,<their GOPROXY>`, a form `go help goproxy` documents. A cached module is copied in without the network and checked against `go.sum` as usual. The person's caches and `GOPATH` are only read.
- **Temp**: a fresh directory per launch, set as `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR`. The system's `/tmp` is not granted, except srt's own built-in paths.
- **The host's state**, kept apart from its configuration:
  - claude: a fresh `CLAUDE_CONFIG_DIR` per launch. Under `--bare` the harness reads no OAuth credential and no keychain, and takes its key from `ANTHROPIC_API_KEY` or an `apiKeyHelper` passed with `--settings` ([headless](https://code.claude.com/docs/en/headless)), so the person's `~/.claude` and `~/.claude.json` are never needed and stay read-only. That also covers the paths the vendor warns about: "a sandboxed session that can write them can persist hooks, permission rules, or MCP servers that run unsandboxed the next time you launch Claude Code".
  - opencode: fresh `XDG_DATA_HOME` and `XDG_STATE_HOME` per launch, and an `XDG_CACHE_HOME` per run, because opencode downloads and later runs programs from its data and cache folders. abcd seeds the run's cache once, at the run's start and outside the fence, from the person's: by `clonefile` on APFS or a reflink on Btrfs and XFS where the filesystem offers one, and by a plain copy elsewhere. Seeding per launch would copy the cache again for every role. The run's launches share the seed, so what one launch fetches there a later launch of the same run may run; that stays inside the run's fence, and nothing written there flows back to the person's cache. The person's configuration folder stays where it is, read-only, since it names the model and the server (`opencode.go`). The person's credential file is reached through a read-only link from the fresh data folder. A provider whose sign-in rewrites that file cannot refresh inside the fence: the launch fails, naming it, and the person uses a key for runner launches. abcd sees only the `<provider>/<model>` route, not a base URL set in opencode's configuration, so a provider abcd does not know, or a local model server, is reached by adding its host to `runner.sandbox.hosts`. Step 1 measures all of this.
- **abcd's home** (`~/.abcd.noindex/`) is readable and is not writable, except the launch's own worktree inside it. So the next launch's policy (`config.json`, `rules.json`, `trusted-roots`, `load-limits`, the other worktrees) is the person's alone to change.

**Reads.** Reads stay open, as srt's default leaves them, except for the credential stores a launch never needs. Those are denied:

- `~/.ssh` and `~/.gnupg`;
- `~/.aws`, `~/.config/gh`, `~/.netrc`, `~/.git-credentials` and `~/.docker/config.json`;
- abcd's own credential file under its home;
- on macOS, the login keychain's files, `~/Library/Keychains`. Step 1's probes added this one: without it the keychain handed a stored GitHub token to a command inside the fence, although `~/.config/gh` was denied (Step 1 evidence, probe 3).

The vendor's guide says why the two layers go together: "Without network isolation, a compromised agent could exfiltrate sensitive files like SSH keys". The model endpoint is reachable, so this narrows what a misled agent can send there. It is not part C's promise.

Writes srt already allows outside the list (the child's standard streams, `/tmp/claude`, `~/.npm/_logs`, `~/.claude/debug`) are named in the run record's fence block as the fence's own, so the list abcd states is the list that holds.

The policy goes into a settings file abcd writes for each launch, inside the run's fence directory but outside every path the launch may write. It is never `~/.srt-settings.json`, and `--control-fd` is not used, so nothing can change the policy while the launch runs.

### Commits inside the fence

The agent commits in its own worktree: its brief tells it to (`internal/core/implement/loop/brief.go:583`). Two things a commit sets off would otherwise fail inside the fence.

**Automatic gc.** After a commit git may run `git gc --auto`, through `git maintenance run --auto`, and that wants `packed-refs` and `.git/gc.log`, neither of which is on the write list. abcd's own git turns automatic maintenance off on its command line, but the agent's git is not abcd's, and the launch's environment drops every inherited `GIT_CONFIG_*` (`ScrubbedEnv`, `internal/gitutil/net.go:37`). So before each launch abcd writes `gc.auto=0` and `maintenance.auto=false` into the lane's `worktrees/<name>/config.worktree`, outside the fence. That file is on the deny list, so the agent cannot take the values out. Git reads `config.worktree` only in a repository with `extensions.worktreeConfig` on, and abcd never turns that on, since doing so rewrites the main `.git/config` and changes how the repository works. So the launch also carries both values as its own seal (`GIT_CONFIG_COUNT` with its keys and values, set over the scrubbed environment, `internal/core/runner/proc.go:168`), which holds in every repository. A gc the agent forces anyway is refused after its commit has landed, and is recorded like any other refusal.

**Commit signing.** Where the person's git signs every commit (`commit.gpgsign`), the agent's commit needs the signing key, and the read list denies `~/.gnupg` and `~/.ssh`. On Linux srt's seccomp filter also blocks the Unix socket a signing agent answers on. abcd's own commits sign as the person's commits do (`internal/core/implement/loop/pickcommit.go:19-21`), and that does not change. The key is not opened to the agent: an agent that can read it can sign anything as the person. The proposal is that fenced lanes commit unsigned, with `commit.gpgsign=false` in the launch's seal, and that abcd signs outside the fence, where it lands the work. Before its push, the land stage rewrites each lane commit with the same tree and message, in the same order, signed under the person's own configuration. Its judged-head check (`landPrepare`, `internal/core/implement/loop/land.go:198`) then compares the rewritten chain, tree for tree, with the head its validators judged. This is question (d), with its other answers, and none of it is built before the answer.

### A machine without a fence

Ruling 7: the run stops until the person acts. When a run is about to start and any role it may launch is routed to a runner, abcd checks the fence (`Fence.Check`) before creating the run. That covers a `build` start, each `drain` lane's start, and an `intent interview` start. The check:

- finds `srt` on PATH and admits it as a harness is admitted (`launcher.admit`): an absolute path, not writable by group or other, not inside the repository or the checkout. srt's own README warns that a copy inside a writable path can be rewritten from inside the sandbox. srt is a Node script, so the check also covers the interpreter its first line names, found on PATH, and the package folder the script resolves into, and none of them may sit inside a path any launch may write;
- reads its version. abcd was tested against 0.0.79, and that is a statement of what was tested, not an exact pin: a 0.0.x version promises nothing between releases, so no range can be trusted, and an exact pin would break every machine at each upstream release without saying whether the fence still holds. A version other than the tested one is warned about once, at the run's start, naming the version found and the one tested, and is not refused. The tested version moves in a change that runs the `fence` test lane on the new one;
- runs a trial launch, `srt --settings <trial> -- /bin/sh -c ': > "$1"; : > "$2"' sh <inside> <outside>`, and checks from outside that the write inside landed and the write outside did not. The trial, not the version string, is what decides: it makes "fenced" a fact, not an assumption, per [`enforcement-claims-are-facts`](../../principles/enforcement-claims-are-facts.md).

A failed admission or a failed trial refuses the run: exit 2, nothing created and nothing launched. The refusal names the reason (srt not found, refused, bubblewrap or user namespaces missing as srt reports, the trial failing to start, or the trial writing outside), how to fix it, and the off switch.

The off switch is `runner.sandbox.off: true` in `~/.abcd.noindex/config.json`. The `runner` namespace is already the machine's alone, and a repository's `.abcd/config.json` that sets it is refused, naming the machine's file (`runner.Load`). No flag sets it either, so a project's files and a command an agent can type can never turn the fence off. With it set, the run says once, at its start, that its launches are not fenced and where that was set.

A wrap that fails during a run (srt removed, its settings refused) is a new runner reason, `fenced`. It is not a fallback: like a rate limit, it hands the role to no other route. The lane refuses at its runner stage with the same remedy.

### The guard inside the fence

The approver runs **inside** the fence. The harness starts `abcd guard approve` itself, as the one MCP server on stdio its `--mcp-config` names (`approverConfig` on `feat/headless-approvals`), and a child of the wrapped process is inside the wrapper. Keeping it inside costs nothing and is the safer side:

- it only reads: abcd's own binary, the worktree's `.abcd/guard.json` and the bundled registry, all readable in the fence. It writes nothing;
- its answer can only narrow what the agent does. An allow it gives still runs inside the fence, so an approver that is wrong, or fooled by the input it judges, can never let a command past the boundary;
- putting it outside would mean a channel crossing the fence, carrying input the agent controls, to a process with the person's full reach. That channel would be a way out, and abcd would have to build it, which ruling 6 rules out. On Linux, srt's seccomp filter blocks exactly that kind of Unix-socket bridge.

The command the `--mcp-config` names is abcd's own executable, `os.Executable()` of the abcd that starts the run. It is never a name looked up again on PATH, and never `go run ./cmd/abcd` from the worktree, whose source the agent may edit. It is admitted by the same rule as srt: an absolute path, not writable by group or other, and never inside a path the launch may write. An executable that fails the rule refuses the launch, naming it.

So the guard keeps its place as adr-42's mistake filter, teaching inside the boundary the OS enforces. One limit is stated, not fixed: the approver judges against the registry of the working tree, which the agent can edit. A registry switched off is refused (`registry-not-in-force`). An entry the agent deletes stops teaching for the rest of that launch, and the fence still bounds what follows.

### Checks that read the machine run outside

The load gate, the peers scan, and the reservation renewals and fences the loop takes between stages already run in abcd, outside any launch. What changes is what happens to a gate an agent reaches inside the fence: `make preflight` runs its test stages through `abcd implement load --wait`, and inside the fence that waiter would read the fence's view of the process table. On Linux the process namespace shows it only its own processes. On macOS it sees none at all: `/bin/ps` cannot be started inside the fence and `pgrep` finds nothing, so `implement load` reports the check unavailable and carries on (Step 1 evidence, probe 5). It also could not write its hold under abcd's home, and on macOS today's waiter already refuses for that reason rather than opening.

So a waiter never judges the machine from inside a fence:

- a waiter that finds its launch's gate directory (an `ABCD_FENCE_GATE` the launch sets) asks through it, and waits for an answer;
- a waiter that cannot write its hold refuses, naming the fence, rather than opening. That covers an agent that unsets the variable;
- for each request, the runner starts, outside the fence, an ordinary `abcd implement load --wait --gate <dir>` process. It judges the whole machine as every waiter does, visible in the process table to every other account's waiters. When the gate opens it takes the hold, and writes its answer where the fenced side can read it and cannot write.
- that outer waiter then holds the gate for the inner command's length, as `--wait -- <command>` does, until the inner side writes its done marker or the launch ends;
- one request may be outstanding per launch, and the outer waiter's own timeout bounds it.

A preflight run inside the fence therefore runs its commands inside the fence, under a gate read outside it. Its receipt lands in the worktree's local tier, which the agent may write. None of this needs the fence to exist: a waiter that can write its hold behaves as today, so step 12 can land before any other step.

### How each criterion is met

1. **A write outside is refused and recorded.** The derived list (step 3) is what the wrapped launch can write (step 7). The fence's refusals are read back from the launch's stderr and the tool errors in its transcript (`Fence.Refusals`), and written to the lane's run record under `fence.refusals`, each with its path and the tool that tried (step 10).
2. **Edit, commit and push succeed.** The worktree, and the parts of the main git directory a lane's commit writes, are writable (step 3), and the commit's gc and signing are settled before the launch (step 7), so the edit and the commit land. The push is the land stage's, outside the fence, as it is today. Step 15's live run shows all three.
3. **abcd's settings, rules, trusted roots and other worktrees are refused.** abcd's home is read-only except the launch's own worktree (step 3). Step 10's real-fence test writes each of them and asserts the refusal.
4. **No fence: the run is refused before its first launch.** Steps 5 and 6.
5. **The off switch is the person's alone and said once.** Steps 4 and 6.
6. **A fresh Go cache of its own.** Step 13.
7. **The load gate reads the whole machine from outside.** Step 12.
8. **The guard's refusal reaches the agent inside.** Step 11.
9. **Host settings byte-identical.** Step 14.

### Proof

The tests that matter run against the real fence. A test lane under the build tag `fence` runs real srt on macOS and on a Linux runner with bubblewrap installed, wrapping a stand-in harness: a script named `claude` or `opencode` on a PATH the test controls, which tries the writes, commits and gates each criterion names. Fakes stand in only for what a test cannot cause, such as a missing srt or an untested version. Before this spec closes, one `build` of a real intent runs fenced on the built binary with the real claude runner (and the real opencode runner where routed), on macOS and on Linux, and its run record and the before-and-after hashes of the person's host settings are the evidence (step 15), per the product thinker's rule that real runs are the proof.

The `fence` lane is a Make target that joins `make preflight` beside `smoke` and `evals-cold-reading` (`Makefile:297`). On a machine where srt does not pass `Check`, preflight says by name that it skipped the lane: a skip, never a pass. CI runs the lane where it cannot skip: in `.github/workflows/ci.yml`, on a Linux runner with unprivileged user namespaces, where ubuntu-24.04 needs the AppArmor sysctl `kernel.apparmor_restrict_unprivileged_userns=0` set first, with bubblewrap, `socat`, `ripgrep`, Node and srt installed; and in the macOS job where it runs. So every merge-queue entry runs the real fence on Linux.

### Known limits, stated, not claimed

- On macOS srt's README gives "no guarantee" for a hard link, inside an allowed path, to a file outside it.
- On Linux srt's own deny scan is fixed at launch. abcd's explicit denies cover the main git directory, which exists before the launch, and not a repository the agent creates later inside its worktree.
- Go programs that check TLS through the system's trust service fail under Seatbelt (the vendor's guide names `gh`, `gcloud` and `terraform`), and `go` itself is one of them. Measured on macOS (Step 1 evidence, probe 1): every HTTPS request `go` makes inside the fence fails with `x509: OSStatus -26276`, both a module download from `proxy.golang.org` and the checksum lookup it routes through that proxy, so a module missing from the person's cache, or from `go.sum`, cannot be fetched or verified. A build whose modules are all in `go.sum` and in the person's cache works, through the file proxy, with no network. With `SSL_CERT_FILE` naming the system's certificate file (`/etc/ssl/cert.pem`), Go reads its roots from that file instead of the trust service, and the download and the checksum verification both pass, with nothing in the fence widened. srt's own remedy, `enableWeakerNetworkIsolation`, opens the trust service and is what its README warns is an exfiltration path; it is not used. So on macOS the launch also sets `SSL_CERT_FILE` (step 7). A program that ignores that variable and asks the trust service still fails, and is named where it does.
- A private module (`GOPRIVATE`) is fetched from its version-control host with the person's own credentials. One held in a store the read list denies (`~/.ssh`, `~/.netrc`, `~/.git-credentials`, `~/.config/gh`) cannot be read, and the fetch fails naming the module. Measured on macOS (probe 3): git's keychain helper gave nothing inside the fence (`failed to get: -128`, as it does outside in a session nobody can answer a prompt in), and the GitHub helper `gh auth git-credential` could not start, so no git credential reached the fence. The keychain itself did answer: a token stored there was returned in full until `~/Library/Keychains` joined the read denies (Reads, above). The ssh agent's socket is refused inside the fence (`Error connecting to agent: Operation not permitted`), so neither a fetch over ssh nor an ssh signature can use the agent.
- srt's CLI reports no refusal of its own on macOS, with or without `--debug`: the refusal reaches the launch's stderr only as the refused program's own error, `Operation not permitted` with the path (probe 4). The kernel logs each refusal, tagged by srt (`Sandbox: <program>(<pid>) deny(1) file-write-create <path>`, then `CMD64_<command in base64>_END_<session>_SBX`), to the unified log, which a `log stream` running during the launch sees and a later `log show` does not. That stream also carries refusals nothing asked about (`sysctl-read kern.iossupportversion` from every shell). `Fence.Refusals` reads the program errors (step 10); reading the log stream as a second source is left to step 10 to weigh.
- Every commit inside the fence, with git 2.52, prints `error: Unable to create '<git dir>/packed-refs.lock': Operation not permitted` during its ref update, and still exits 0 with its commit, ref and reflog written (probe 2). `packed-refs` stays off the write list. `Fence.Refusals` records the line like any other refusal, and the lane does not read it as a failed commit.
- On Linux, srt covers a protected path that does not yet exist with a placeholder on the host, and a SIGKILL skips its cleanup. abcd's timeout kill is the group's SIGKILL (`procgroup.Kill`), so a timeout first sends SIGTERM to srt and kills the group only after a grace period (step 8). After every launch abcd also removes, outside the fence, any empty placeholder that appeared at one of srt's protected names and that git does not track, so none is committed or left in the person's worktree (step 9). Step 1 checks whether bubblewrap's own session takes descendants out of the group abcd kills.
- srt quotes the argument vector into a shell string on macOS and Linux, which amends `proc.go`'s rule that the argv is never a shell line. abcd's wrapped vector is fixed words, a role name from a closed list, and paths abcd made. A path carrying a quote or a control character is refused before the launch, and a test sends a prompt through with every shell metacharacter (step 7).
- The intent's press release says the person's other projects and settings "stay out of its reach". With this design that holds for changes. It holds for reads only where the read denies above reach. The wording is question (e).

### Coordination

- Steps 3 to 10, 13 and 14 change `internal/core/runner` and the new `internal/core/fence`, so they land in order. Step 8 also changes `internal/procgroup`, which every launch and every waiter kills through, so it reserves that package too.
- Step 11 waits for `feat/headless-approvals` to merge, and reserves `internal/core/runner/claude.go` after it.
- Step 12 changes `internal/core/implement/load_wait.go`, which every preflight runs through, so it lands alone, after a reservation. It does not depend on the fence, and can land before step 3.
- Step 5 changes `Makefile` and `.github/workflows/ci.yml`, which every preflight and every CI run read, so it reserves both.
- Every step reserves its footprint with `abcd reserve` before it starts.

### Out of scope

- Part B: the isolation report for drains and builds a person's own session drives. Part C: the network promise and its claim (the host list here is functional, not a promise).
- Confining reads. Secrets the account can read stay readable to a fenced agent.
- Windows, cloud sessions, and more than one project per run (the intent's scope conditions).
- Caches of other toolchains (npm, cargo, pip): writes to them are refused, and a run that needs one shows up in step 15's evidence before anything is added for it.

## Questions for the product thinker

Put at step 2, after step 1's probes. Each question is written for the product thinker, in plain words, and each answer says what it leads to. The mechanism behind each one is in the sections above. Nothing is built before all five are answered.

- **(a) One extra program to install.** Fencing the agents abcd starts needs one program abcd does not ship: Anthropic's sandbox. The person installs it once, with the small runtime it needs. Everything else in abcd keeps needing nothing extra.
  - Require it for the agents abcd starts. Leads to: on a machine without it, abcd will not start such an agent until the person installs it or switches the fence off for that machine.
  - Use another open-source fence instead, a single file with nothing else to install. Leads to: less to install, but the fence around Claude is not Anthropic's.
  - Wait for a fence that needs nothing extra. Leads to: the agents abcd starts keep running unfenced until then.
- **(b) When the program is missing.**
  - abcd offers to install it, and installs only after the person says yes. Leads to: one step for the person; the install runs with everything the person's account can do.
  - abcd says how to install it and how to switch the fence off, and installs nothing. Leads to: the person installs it by hand.
- **(c) Which internet addresses a fenced agent may reach.** The fence lets an agent reach only the addresses it is given.
  - The AI service, what the project's build needs to download, and any the person adds. Leads to: builds that download code keep working, and anything more is the person's to add.
  - The AI service and any the person adds, nothing else. Leads to: a build that needs a download fails until the person names where it downloads from.
  - The first, with each run listing every address it allows when it starts. Leads to: the same reach, said out loud at every run.
- **(d) Signed commits.** Where the person signs every commit, an agent inside the fence cannot reach the signing key.
  - The agent's commits are unsigned inside the fence, and abcd signs them when it hands the work in. Leads to: the work arrives signed, and the agent never holds the key.
  - The agent may use the signing key. Leads to: commits are signed as today, and a misled agent could sign anything in the person's name.
  - abcd refuses a fenced run while signing is on. Leads to: the person turns signing off for these runs, or turns the fence off.
- **(e) The promise that the person's other projects and settings "stay out of its reach".** The probes show the agent cannot change anything outside its own work. It can still read most of what the person can read; only the places that hold passwords, keys and sign-ins are hidden from it.
  - Keep the words, and hide more from reading. Leads to: more work now, and some tools may stop working inside the fence.
  - Narrow the words to what holds: it cannot change anything outside its own work, and cannot read the person's passwords, keys and sign-ins. Leads to: a smaller promise that is true as written.
  - Keep the words, and state the reading limit in the guide. Leads to: the promise reads wider than what it does.

## Footprint

- packages: internal/core/fence, internal/core/fence/srt, internal/core/runner, internal/core/implement, internal/core/implement/loop, internal/core/interview, internal/procgroup, internal/surface/cli, commands, docs, ACKNOWLEDGEMENTS.md, Makefile, .github/workflows/ci.yml, .abcd/development/brief, .abcd/development/research/notes, .abcd/development/decisions/adrs
- tests: internal/core/fence (policy derivation, the seam's test double, admission and trial tables), internal/core/fence/srt (settings file, argv and refusal parsing), internal/core/runner (fenced argv, environment and seal, the fenced reason, config tests), internal/procgroup (the kill order), internal/core/implement (the fenced waiter and its outer gate), internal/core/implement/loop and internal/core/interview (refusal before a run starts, the ref diff and placeholder sweep, the fence block of the run record), internal/surface/cli (build, drain, interview and status surface tests), the `fence`-tagged real-srt lane on macOS and Linux in preflight and CI, and step 15's live fenced build

## Steps

1. Probe srt on the real harnesses before anything is signed off or built
   - packages: .abcd/development/research/notes, .abcd/development/intents
   - tests: an abcd lab on macOS and on Linux with bubblewrap, each probe named in the research note: `claude --print --bare` and `opencode run` wrapped by srt 0.0.79 with the step 3 policy written by hand; `CLAUDE_CONFIG_DIR` per launch with the person's `~/.claude` and `~/.claude.json` read-only; opencode with fresh XDG folders, a per-run cache seeded by clone, a read-only credential link, and a refreshing sign-in; `go build` and `go test ./...` of abcd itself with the per-run `GOPATH` and caches, a module missing from `go.sum` verified through the checksum database, a `GOPRIVATE` module fetched from its host, and whether module downloads pass TLS under Seatbelt; whether a keychain credential helper answers git inside the fence; a commit in a linked worktree under the narrowed main `.git`, with and without `gc.auto=0`, with `packed-refs` unwritable, and with `commit.gpgsign` on; a hard link from the worktree to a file outside it on macOS; SIGTERM cleanup, and whether bubblewrap's session leaves the process group abcd kills; what opencode fetches at start with a seeded cache; a prompt with shell metacharacters. Probing builds nothing, and its findings are what step 2's questions read. Each limit found becomes a scope condition on the intent or a capture of its own, and a failure on the macOS or Linux half goes back to the person at step 2, with nono as the named alternative
   - evidence: the macOS half's first probes ran on 2026-10-17 and are recorded under Step 1 evidence, after the steps. The Linux half, the agent sessions and the probes listed there as not yet run are still to do
2. Sign-off: the person answers the five questions for the product thinker (waits for the person; nothing is built before it)
   - packages: .abcd/development/brief, .abcd/development/decisions/adrs, .abcd/development/specs
   - tests: each answer recorded as a decision and folded into this spec; on a yes to question (a), the proposed marks come off adr-22's consequence and the dependencies chapter's paragraph (02-constraints/02-dependencies.md), and on a no both are withdrawn; record-lint and docs-lint pass over both and over the ADR's acceptance
3. Declare the fence seam and derive a launch's fence-neutral policy (itd-2610161621364075 criteria 1, 2, 3)
   - packages: internal/core/fence
   - tests: fence: `Policy` names no mechanism, and `Fence` is met by a test double the runner's tests use; for a lane launch, for an interview launch, and for each runner, the write list is exactly the worktree (or the record's folder), the receipt folder, the main `.git`'s `objects/`, the lane's own `worktrees/<name>/` without its `config.worktree`, `gitdir` and `commondir`, and `refs/heads/build/` with its reflogs, the run's `GOPATH` and Go caches, the launch's temp and host-state folders and the run's opencode cache; `hooks/`, `config` and `packed-refs` are never granted; abcd's home is read-only but for the worktree; the credential stores are denied for reads, `~/Library/Keychains` among them on macOS; a path given through a symlink is refused; nothing read from the repository's `.abcd/config.json` changes the list; the host list is the runner's endpoint, with a `go.mod` the `GOPROXY`/`GOSUMDB` hosts and the `GOPRIVATE`/`GONOPROXY` hosts, and the machine layer's additions, and a pattern too broad to pass is named
4. Read the off switch, the backend and the host additions from the machine layer only (criterion 5)
   - packages: internal/core/runner, .abcd/development/brief
   - tests: runner: `runner.sandbox.off`, `runner.sandbox.backend` and `runner.sandbox.hosts` read from the machine file; the backend defaults to `srt` and an unknown one is refused, naming the known ones; the same keys in a repository's `.abcd/config.json` refuse the load, naming the machine's file; a host entry that is `*`, a bare top-level domain or not `host[:port]` is refused; no flag sets any of them
5. Admit srt and prove the fence runs (criterion 4)
   - packages: internal/core/fence, internal/core/fence/srt, internal/core/runner, Makefile, .github/workflows/ci.yml
   - tests: srt's `Check`: srt absent, relative, group- or other-writable, inside the repository, or an interpreter or package folder that fails the same checks is refused with its reason and the off switch; a version other than the tested 0.0.79 is admitted with one warning naming both; the evidence carries the admitted paths, the version and the trial's result; `fence`-tagged: the trial launch's inside write lands and the outside write does not, on macOS and Linux; a trial whose outside write lands, or that fails to start, refuses; `make fence` runs the tagged lane and preflight names it as skipped where srt fails `Check`; CI's Linux job runs it on a runner with user namespaces, setting the AppArmor sysctl on ubuntu-24.04, and fails rather than skips
6. Refuse a run before its first launch, and say once when the fence is off (criteria 4, 5)
   - packages: internal/core/implement/loop, internal/core/interview, internal/surface/cli, commands
   - tests: loop and cli: a `build`, a `drain` lane start and an `intent interview` with a role routed to a runner, on a machine whose fence check fails, exit 2 before the run is created and start no process; a run whose roles all stay on the host checks nothing; with the switch on, the run's output says once that its launches are not fenced and names the machine file; an untested srt version is said once at the run's start
7. Wrap every launch at the launch point, with its environment (criteria 1, 2)
   - packages: internal/core/runner, internal/core/fence, internal/core/fence/srt, ACKNOWLEDGEMENTS.md
   - tests: runner: a fenced claude and opencode launch run `srt --settings <file> -- <harness> …` with the harness's own argv unchanged after `--`; the environment sets the per-launch temp, the run's `GOPATH` and Go caches and the host-state folders over the parent's, on macOS sets `SSL_CERT_FILE` to the system's certificate file, and seals `gc.auto=0` and `maintenance.auto=false`, and `commit.gpgsign=false` unless question (d) is answered otherwise; the lane's `config.worktree` takes the gc values when the repository has `extensions.worktreeConfig` on, and the main `.git/config` is unchanged either way; opencode's cache is seeded once per run, by clone where the filesystem offers one; the settings file sits outside every writable path and round-trips through srt's schema; a wrap failure is reason `fenced`, records no fallback and runs no other route; `Wrap`'s cleanup runs on every way a launch ends; loop: a lane whose launch is refused as fenced refuses at the runner stage naming the remedy; a path carrying a quote or a control character is refused before the launch; `fence`-tagged: a prompt carrying every shell metacharacter arrives byte for byte; a commit in the lane leaves no gc refusal; the run's fence folder is removed when the run finishes
8. Kill a fenced launch in order (criteria 1, 2)
   - packages: internal/procgroup, internal/core/runner
   - tests: procgroup: a timeout sends SIGTERM to srt first and kills the group only after the grace; an srt that exits on SIGTERM within the grace is not sent SIGKILL; the kill still goes through the pid the handle holds, never a pattern; `fence`-tagged: after a timeout no descendant of the launch survives, including any bubblewrap session step 1 found outside the group
9. Check the refs and sweep srt's placeholders after each launch (criteria 1, 2)
   - packages: internal/core/runner, internal/core/fence, internal/core/implement/loop
   - tests: loop: a launch that moved a ref other than its lane's branch is recorded and its lane refuses to land; a launch that moved only its own branch lands; an empty untracked placeholder at one of srt's protected names is removed after the launch, and a tracked or non-empty file there is left alone; both run after a killed launch as after a clean one
10. Record the fence's refusals in the run record (criteria 1, 3)
   - packages: internal/core/fence/srt, internal/core/runner, internal/core/implement/loop, internal/surface/cli
   - tests: srt: `Refusals` reads each refusal out of srt's stderr and the transcript's tool errors, with its path and tool; `fence`-tagged: a stand-in harness that writes outside its worktree, to `~/.abcd.noindex/config.json`, `rules.json`, `trusted-roots` and another lane's worktree is refused by the OS each time, and the lane's run record lists each refusal with its path; `implement status` renders the fence block (mechanism, version, whether it was the tested one, policy digest, refusals) and its `--json` carries it
11. Keep the guard teaching inside the fence (criterion 8; waits for `feat/headless-approvals`)
   - packages: internal/core/runner, internal/surface/cli
   - tests: runner: the command `--mcp-config` names is `os.Executable()`, admitted by srt's rule, and a launch whose executable sits inside a path the launch may write, or is `go run ./cmd/abcd` from the worktree, is refused naming it; `fence`-tagged: a stand-in harness speaking the prompt tool's MCP to `abcd guard approve` started inside srt gets a refused command answered `deny` with the guard's reason and advice, and a cleared one answered `allow`; the approver's run leaves no fence refusal of its own; a real `claude --print --bare` launch honours `--mcp-config` under `--strict-mcp-config`, starting the approver and no other server
12. Read the load gate outside the fence (criterion 7; does not depend on the fence and can land first)
   - packages: internal/core/implement, internal/core/runner, internal/surface/cli, commands
   - tests: implement: a waiter that cannot write its hold refuses naming the fence and never opens; a waiter with a gate directory opens only on the outer waiter's answer; an outer waiter given a loaded snapshot keeps the gate shut while the fenced side's own view is idle; it holds while the inner command runs and releases on the done marker or the launch's end; a second outstanding request from one launch is refused; a waiter that can write its hold behaves as today; cli: `implement load --wait --gate` is in help and the generated reference
13. Give each run a fresh Go cache (criterion 6)
   - packages: internal/core/fence, internal/core/runner, internal/core/implement/loop
   - tests: fence: `GOPATH`, `GOCACHE` and `GOMODCACHE` point into the run's fence folder and `GOPROXY` leads with the person's module cache as a file proxy; `fence`-tagged: a `go build` in a fenced launch fills the run's caches; a build that verifies a module missing from `go.sum` writes `pkg/sumdb` under the run's `GOPATH`; a `go install` lands in the run's `GOPATH/bin`; the person's `GOPATH`, `GOCACHE` and `GOMODCACHE` listings and modification times are unchanged
14. Leave the person's host settings untouched (criterion 9)
   - packages: internal/core/fence, internal/core/runner
   - tests: `fence`-tagged: a stand-in claude that writes `~/.claude/settings.json`, `~/.claude.json` and `~/.claude/agents/`, and a stand-in opencode that writes its configuration folder, are refused; the hashes of every host configuration path the policy names are identical before and after a fenced launch
15. State the rule and prove it on a real run
   - packages: .abcd/development/brief, commands, docs
   - tests: a brief invariant says an agent abcd launches runs inside a fence the operating system enforces, unless the person turned it off in their user layer, and the adapters chapter describes the fence seam; `commands/build.md` and `commands/intent.md` name the refusal and the off switch; a how-to under `docs/` says how to install the fence and turn it off, crediting the package under the docs-lint allow escape; docs-lint and record-lint pass; a fenced `build` of a real intent on the built binary, on macOS and on Linux, records a refused outside write, commits in its worktree, lands its push, and leaves the person's host settings byte-identical

## Step 1 evidence

### macOS, 2026-10-17

Run on macOS 27.0 (arm64) with srt 0.0.79 from npm, Node 25.1.0, go 1.27.1, git 2.52.0, Claude Code 2.1.302 and opencode 1.18.31. Each probe ran `srt --settings <run>/settings.json -- <command>` from a scratch lane worktree of abcd, `~/.abcd.noindex/worktrees/<root-sha>/probe-lane` on a branch `build/probe-srt-lane` cut from `origin/main`, whose git directory is the main checkout's (`<checkout>/.git`). `<run>` is a fresh folder under `$TMPDIR`. The worktree, both branches it made, its nine loose objects and its receipt folder were removed afterwards.

Nothing in the settings was widened to make a probe pass. One variant was tighter: probe 3 ran again with `~/Library/Keychains` added to `denyRead`.

**The settings file.** srt was given real absolute paths; they are written here with `~`, `<checkout>` and `<run>`.

```json
{
  "network": {
    "allowedDomains": ["api.anthropic.com", "proxy.golang.org", "sum.golang.org"],
    "deniedDomains": []
  },
  "filesystem": {
    "denyRead": [
      "~/.ssh", "~/.gnupg", "~/.aws", "~/.config/gh", "~/.netrc",
      "~/.git-credentials", "~/.docker/config.json", "~/.abcd.noindex/credentials.json"
    ],
    "allowRead": [],
    "allowWrite": [
      "~/.abcd.noindex/worktrees/<root-sha>/probe-lane",
      "<checkout>/.abcd/.work.local/run/<run-id>/lane-1",
      "<checkout>/.git/objects",
      "<checkout>/.git/worktrees/probe-lane",
      "<checkout>/.git/refs/heads/build",
      "<checkout>/.git/logs/refs/heads/build",
      "<run>/gopath", "<run>/gocache", "<run>/gomodcache", "<run>/tmp",
      "<run>/claude-config", "<run>/xdg-data", "<run>/xdg-state", "<run>/xdg-cache"
    ],
    "denyWrite": [
      "<checkout>/.git/worktrees/probe-lane/config.worktree",
      "<checkout>/.git/worktrees/probe-lane/gitdir",
      "<checkout>/.git/worktrees/probe-lane/commondir"
    ]
  }
}
```

**The environment.** Each launch started from an empty environment (`env -i`), as the runner's scrubbed one is, with `HOME`, `USER`, `PATH` and `LANG`; `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR` set to `<run>/tmp`; `GOPATH`, `GOCACHE` and `GOMODCACHE` set to the run's folders; `GOPROXY=file://~/go/pkg/mod/cache/download,https://proxy.golang.org,direct`; `CLAUDE_CONFIG_DIR` and the three `XDG_*` folders set to the run's; `GIT_TERMINAL_PROMPT=0`; and the seal as `GIT_CONFIG_COUNT=3` with `gc.auto=0`, `maintenance.auto=false` and `commit.gpgsign=false`. `GOFLAGS=-p=4` kept the cold build's share of a machine running three other preflights small; it is not part of the policy.

**Trial launch.** The spec's trial (step 5): the write inside `<run>/tmp` landed, and the write outside was refused, `Operation not permitted`, and was not on disk afterwards. `sh` exits 1 at the refused redirection, so the trial is judged from outside, by the two files, as step 5 says, and not by its exit status.

| Probe | What ran inside the fence | Outcome |
|---|---|---|
| 1. Build and test | `go build ./...` with every cache empty | Passed in 16 s. 75 modules came from the person's cache through the file proxy, with no network. |
| | `go test ./internal/fsutil/` | Passed. |
| | `go get rsc.io/quote/v4@v4.0.1` in a scratch module, a module not in the person's cache | **Failed**: `tls: failed to verify certificate: x509: OSStatus -26276`. Go asks the system's trust service, which the fence closes. |
| | `go get github.com/google/go-cmp@v0.7.0`, in the person's cache but not in the scratch `go.sum` | **Failed** at verification: Go asked the checksum database through `https://proxy.golang.org/sumdb/sum.golang.org/supported`, with the same TLS error. |
| | both again with `SSL_CERT_FILE=/etc/ssl/cert.pem` | Passed. The checksum database's cache was written to `<run>/gopath/pkg/sumdb/sum.golang.org/`, under the run's `GOPATH` as the spec expects. Whether `sum.golang.org` is dialled directly, or only through the proxy, was not measured. |
| 2. Commit | `git add` and `git commit` in the lane, through the repository's own hooks (`core.hooksPath` names the main checkout's `.githooks`) | Committed, exit 0. `pre-commit`, `prepare-commit-msg` and `commit-msg` all ran inside the fence, and `commit-msg` built abcd with the run's caches. The commit's objects, the lane branch and its reflog landed in `<checkout>/.git`. The seal read back as `gc.auto=0`, `maintenance.auto=false`, `commit.gpgsign=false`. Each commit also printed `error: Unable to create '<checkout>/.git/packed-refs.lock': Operation not permitted`, from git's own ref update, without failing. |
| | writes to `.git/hooks/`, `.git/config`, `.git/packed-refs`, `.git/HEAD`, the lane's `config.worktree` (create), `gitdir` and `commondir`, and the main checkout's `.githooks/pre-commit` | All refused, `Operation not permitted`. Their checksums matched before and after. |
| | `git config --local srtprobe.x 1` | Refused: `could not lock config file <checkout>/.git/config: Operation not permitted`. |
| | `git update-ref refs/heads/srt-probe-outside HEAD` | Refused: the ref's lock file could not be created. |
| | `git update-ref refs/heads/build/srt-probe-other HEAD` | **Allowed**, as the spec expects: the folder grant covers every lane's branch, and only the ref check after the launch (step 9) catches it. |
| | write to the lane's receipt folder; open the run's `state.json` for writing | Receipt written; `state.json` refused. |
| 3. Credentials | `git credential-osxkeychain get` for github.com | `failed to get: -128`, the same as outside in this session: the item needs a prompt nobody can answer. Nothing returned. |
| | `git credential fill` for github.com (helper: `gh auth git-credential`) | Nothing returned: `gh` could not start, `open ~/.config/gh/config.yml: operation not permitted`. |
| | `gh auth status` | Same refusal; `gh` does not start. |
| | `security find-generic-password -s gh:github.com -w`, counting the bytes only | **The keychain answered**: 75 bytes, the same as outside. The GitHub token gh keeps in the login keychain was readable inside the fence. |
| | the same, with `~/Library/Keychains` added to `denyRead` | Not found (exit 44), 0 bytes. |
| | `ssh-add -l` with the person's `SSH_AUTH_SOCK` | Refused: `Error connecting to agent: Operation not permitted`. |
| 4. Outside writes | create `~/probe-outside`, `~/.ssh/x`, `~/.abcd.noindex/trusted-roots.probe`, a file in another worktree, `~/.claude/agents/srt-probe.md`; open `~/.abcd.noindex/config.json` and `~/.claude.json` for writing; `touch`; `mkdir` | All refused, and none was on disk afterwards; `config.json` and `~/.claude.json` matched their checksums before and after. |
| | reads of `~/.ssh/known_hosts`, `~/.config/gh`, `~/.docker/config.json`; and of `~/.claude.json`, which is not denied | The three denied reads were refused; `~/.claude.json` was read. |
| | how srt reports a refusal | srt adds nothing to stderr, with or without `--debug`. The only report is the program's own: `<program>: <path>: Operation not permitted`. The kernel logs `Sandbox: <program>(<pid>) deny(1) file-write-create <path>` with srt's tag `CMD64_<command in base64>_END_<session>_SBX` to the unified log: a `log stream` running during the launch sees it, and a later `log show` does not. |
| 5. Process table | `ps -A`, `ps -p 1`, `pgrep -f 'make preflight'`, `sysctl -n vm.loadavg` | `/bin/ps` cannot be started (`Operation not permitted`, exit 126); `pgrep` found 0 of the 9 preflight processes visible outside; the `sysctl` command was refused. Outside, `ps -A` counted 1444 processes. |
| | `go run ./cmd/abcd implement load --site preflight` | `LOAD CHECK UNAVAILABLE (preflight): could not read the process table (/bin/ps: fork/exec /bin/ps: operation not permitted); carrying on without checking for stray programs`. The load average was still read (13.9 on 16 cores). |
| | `go run ./cmd/abcd implement load --wait --timeout 20s -- /usr/bin/true` | Refused, exit 1: `the hold could not be judged or written: openat .load-wait.lock: operation not permitted`. Today's waiter does not open inside the fence. |
| 6. Harnesses | `claude --version` and `opencode --version`, with fresh `CLAUDE_CONFIG_DIR` and `XDG_*` folders; then again with the keychain denied | Both started and printed their version under both settings, with nothing refused. opencode made `opencode/repos` and `opencode/log` in the data folder, `opencode` in the state folder and `opencode/bin` in the cache folder. No agent session ran and no model was used. |

**What this changes in the spec.**

- The read denies gain `~/Library/Keychains` on macOS (Reads; step 3's tests).
- On macOS the launch sets `SSL_CERT_FILE` to the system's certificate file, so `go` can fetch and verify modules without srt's weaker trust-service option (Known limits; step 7's tests).
- The Known limits now state how srt reports a refusal on macOS, the `packed-refs.lock` line each commit prints, and what the keychain and the ssh agent answered.
- The checks that read the machine: on macOS a fenced command sees no processes at all rather than only its own, and today's waiter already refuses rather than opening. The conclusion, that the gate is read outside, is unchanged.

**Not yet probed.** The Linux half, all of it. `claude --print --bare` and `opencode run` as real sessions, with the person's `~/.claude` and `~/.claude.json` read-only, and opencode's seeded cache, read-only credential link and refreshing sign-in. `go test ./...` over the whole module (only `./internal/fsutil/` ran). A `GOPRIVATE` module. A commit without the gc seal, and one with `commit.gpgsign` on (the person's git does not sign). The hard link from the worktree to a file outside it. SIGTERM cleanup. A prompt carrying shell metacharacters.
