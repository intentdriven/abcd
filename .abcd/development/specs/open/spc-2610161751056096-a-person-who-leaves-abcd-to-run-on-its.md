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

The spec was written on 16 to 22 October 2026 and reworked on 14 November 2026 against the launch path as it stands on the default branch. The choice of fence, the product thinker's answers of 22 October and the Step 1 evidence are the record and stand as they were written. The approach and the steps describe the build against today's code, and "What changed since 22 October" below lists what moved in between.

Every launch passes through the runner's one launch point, `launcher.run` in `internal/core/runner/proc.go`. The launches that start an agent are fenced:

- the loop's roles (implementer, reviewers, auditor), dispatched from `internal/core/implement/loop/runner_pause.go` for a `build`, an `implement step` and each lane of a `drain`, with or without `--drive`;
- the drain's judge, dispatched from `internal/core/implement/loop/drain_judge.go`, which runs in a directory of its own;
- the planning interviewer started by `interview.Written`;
- a role that falls back to `runner.fallback_host`, which is another runner launch;
- the two short launches a drive makes for itself on the same login, the wall proof (`ProveWall` in `internal/core/runner/wall.go`) and the limit probe (`ProbeLimit` in `internal/core/runner/limit_probe.go`).

Three uses of the launch point start no agent and are not fenced: `runner.Exec`, which runs a program for abcd's own short calls such as `go env` and a host's version; the harness's own login status (`CheckLogin`); and opencode's version (`internal/core/runner/pin.go`).

One command that is not a launch runs what a launch wrote. A drive runs a lane's preflight itself, outside any agent (`drivePreflight` in `internal/surface/cli/drain_drive.go`), and that preflight builds and tests the lane's code. Criteria 6 and 7 speak of a fenced run's lane building and testing, and on a drive that is this preflight. It runs inside the lane's fence, on the run's own fresh cache and its own results folder, by the product thinker's answer to question (g) of 2026-11-15; step 13 builds it.

Building waits on four things:

- **Step 1's remaining probes.** Only the first macOS probes ran. The steps that rest on a fact not yet measured name step 1 in their `needs:` line.
- **The open question** (f) under Questions for the product thinker, which step 14 waits for if step 1's probe finds the drive's sign-in in the keychain. Question (g) is answered, and step 13 builds its answer.
- **The worktree store** (itd-2609091014076309), by decisions 2 and 5. The code the fence stands on is on the default branch: every lane's worktree comes from the store (`laneWorktree` in `internal/core/implement/loop/lane.go`, over `internal/core/ahoy/worktree`). The store's intent is still planned, with the last step of its spec not landed, and decision 2 has this record wait until it ships.
- **The guard's per-launch approver** (iss-2610161612203555), by decision 5. It has landed and its issue is resolved: `approverConfig` in `internal/core/runner/claude.go` starts `abcd guard approve` for every launch that grants the shell. Step 11 no longer waits on it.

### What changed since 22 October

The spec's first text described a launch path with one shape. Today's has more, and several things the spec planned are done another way.

- **A drive and its separate login.** `abcd drain --drive` starts agents with no host session (adr-2610222222326104), on a profile: a directory under abcd's home that the person logged the harness into once (`DriveProfile` in `internal/core/runner/profile.go`). So the claude runner has two launches. The bare one (`--print --bare`, a key from the environment) is the one this spec first described. The walled one (`walledFlags`) runs without `--bare`: its `CLAUDE_CONFIG_DIR` is the profile directory, its hooks are abcd's (`wallSettings`), its shell runs under a home of its own that abcd empties before each launch (`ResetStageHome`), and it carries only a named part of the environment (`stageEnv`), with every credential variable taken out.
- **The wall is a filter, and the fence makes it a boundary.** A drive replaces the profile's settings with its own at each start (`WriteWallSettings`), checks them before each stage (`CheckWallSettings`), proves them in one live launch (`ProveWall`), and tells the guard what a stage may write (`StageWritesEnv`, `StageProtectEnv`). All of that is checked by abcd's own hooks and approver, which adr-42 calls a mistake filter. The same paths become the fence's write list and write denies here.
- **The loop takes the reservations.** A lane's reservation is taken, widened to what the lane changes and released by the process that drives the run (`internal/core/implement/loop/widen.go`), and a drive's stage holds no credential that could push one. A fenced launch therefore needs no write for reservations and no route to the forge.
- **A drive holds what its git reads.** Every git command a drive can run is a row of one table, with the environment it runs in (`driveGitCommands` in `internal/core/implement/loop/gitenvs.go`), and the drive holds the settings and hooks each environment reads before each call. A git command this spec adds to the loop joins that table.
- **The implementer runs a scoped definition of done.** Its brief tells it to run the checks of the packages it changed and never the whole preflight; the landing waits for a preflight receipt naming the lane's head before it pushes (`landPush`), and on a drive the drive runs that preflight. A lane that changes what judges it (`DriveGates` in `internal/core/implement/loop/gates.go`) does not land on a drive.
- **The load gate has three gates and keeps more state.** Beside the first gate there are a race gate and a narrow gate for a test of a few packages (`internal/core/implement/load_narrow.go`), a setting for how many preflights run at once, floors for memory and disk, and a marker for each whole run. All of it is kept under the person's home, which a stage finds through `ABCD_PERSON_HOME` (`abcdhome.MachineHome`). A launch's clock stops while a waiter in its process tree is queued (`internal/core/runner/gatewait.go`).
- **Test stages reuse results.** A preflight's test stages run from one check directory for each repository and account under abcd's home (`internal/testreuse`), so that an unchanged package's result is replayed.
- **The approver judges against the lane's base.** `guard.LoadRepoAt` takes the registry of the working tree only where it is at least as strong as it was at the commit the lane was cut from, so an agent that weakens the registry leaves none in force and every command is refused.
- **The landing is held to the judged head.** `landPrepare` and `landPush` refuse a lane whose branch is not at the head its validators judged, and the push names that commit and not the branch.
- **Each launch is written to a ledger, and the run state is at version 26.** `RecordLaunch` writes one entry for each launch under abcd's home, from outside the launch (`internal/core/runner/ledger.go`). A request also carries a model, a class and an effort, which change the harness's own arguments and nothing the fence reads.

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

**The dependency.** abcd does not vendor srt and `go.mod` does not change. The person installs it once, like the harness it wraps. It is a new external program on the runner route, and on that route it is a requirement, not an adapter that degrades. [adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md) makes every external tool an opt-in adapter over a native default that a missing tool degrades to. A fence has no native default to degrade to: abcd writes no fence of its own (ruling 6), and an unfenced launch is the very thing the intent removes. So srt is framed as adr-22's first declared exception, scoped to the runner route alone. Everywhere else abcd still needs no external program, and in place of degradation the person has the per-machine opt-out, `runner.sandbox.off` (ruling 7). The exception is written as a consequence on adr-22 and as one paragraph in the dependencies chapter (`02-constraints/02-dependencies.md`), both signed off by the product thinker at step 2 on 2026-10-22 (question a, dcn-2610222041417934).

**What the fence brings that part A did not ask for.** srt filters the network to a list of hosts, and it refuses `*` in that list ("Overly broad patterns like "*.com" or "*" are not allowed for security reasons", `sandbox-config.js` in 0.0.79). So a fenced launch reaches only the hosts it is given, whether or not part C has been built. The list here is derived for function, not as part C's promise:

- the runner's model endpoint: `api.anthropic.com` for claude, or the host of `ANTHROPIC_BASE_URL` when it is set; for opencode, the endpoint of the provider its model route names;
- the hosts of `GOPROXY` and `GOSUMDB` when the worktree holds a `go.mod`;
- with a `go.mod`, the hosts named by `GOPRIVATE` and `GONOPROXY`. Go fetches those modules straight from their version-control host, past every proxy, the file proxy below included, so without the host a private module cannot be fetched at all. The host part of each pattern is taken. A pattern whose host srt refuses as too broad is named at the check, and the person adds the concrete host in `runner.sandbox.hosts`;
- the hosts the person adds in `runner.sandbox.hosts`, in their machine layer only.

A launch never pushes and never reserves. Its brief says "Never push" (`renderBrief` and `renderIssueBrief` in `internal/core/implement/loop`), the land stage pushes from abcd, outside the fence (`landPush`, `internal/core/implement/loop/land.go`), and the loop takes and widens the lane's reservation from the process that drives the run. So the forge is not on the list: only abcd's own git, outside the fence, reaches it. `abcd implement widen`, which a host session's implementer may call, needs the network and the person's credential, so it does not work inside a fence, as it does not on a drive's stage today. The product thinker chose this derived list at step 2 on 2026-10-22 (question c, dcn-2610222041427241). The record of each launch keeps the list; the run does not print it at its start. Which hosts the harness contacts on a profile's login, beyond its model endpoint, is not in the Step 1 evidence. Step 1 probes it, and a host the probe finds joins the runner's own derived entry.

### The fence is a seam

The fence sits behind a seam, as the brief's rule for every capability taken from an external tool asks ([internals § 7, the paragraph that opens "Each seam is a Go interface"](../../brief/05-internals/04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching)): a Go interface, consumers that depend on the interface and never on a vendor, a backend selected by configuration, and a new backend added by implementing the interface and registering it, with no edit to the consumers. A new package, `internal/core/fence`, holds:

- `Launch`, what the caller says about one launch in plain fields: its working directory, the checkout that directory belongs to, the directories of its brief and receipt, the paths its work declares, the run it belongs to, and on a drive the profile directory and the stage's home. The package imports neither the runner nor the loop, so both import it.
- `Policy`, which is fence-neutral: the paths a launch may write, the paths inside those it may not, the paths it may not read, and the hosts it may reach. `Derive` builds it from a `Launch` (steps 3, 13 and 14), and nothing in it names srt.
- `Fence`, the interface each mechanism implements:
  - `Check` admits the mechanism, reads its version and makes a trial launch, and returns the evidence: the admitted paths, the version, whether that version is one abcd was tested against, and the trial's two writes. The caller hands it the runner's own admission rule (`launcher.admit`), so one rule admits a harness and a fence;
  - `Wrap(policy, argv, env)` returns the argv and the environment to start, and a cleanup to run after the command ends, however it ends. It takes any argument vector, so the same call fences a harness and the preflight a drive runs for a lane;
  - `Refusals(stderr, transcript)` reads the mechanism's refusals back out of a finished launch.
- `internal/core/fence/srt`, the one adapter. It writes srt's settings file from a `Policy`, builds srt's argv, and parses srt's refusals.

The adapter is chosen by `runner.sandbox.backend`, in the machine layer only like the rest of `runner`, and its default is `srt`. Two departures from the rule are deliberate. There is no native default to fall back to, so a missing or failing backend refuses the run (ruling 7) rather than degrading. And the interface lives in `internal/core/fence`, not under `internal/adapter/`, because deriving a policy from a launch is core logic the runner and the front doors share; the adapter sits beneath it. nono, the named alternative, or Codex's sandbox once abcd launches Codex, would be a second adapter, and neither the runner nor the loop would change.

### What a launched agent may write

The write list is derived from the launch itself; nothing in the repository widens it. Every other path is read-only. On Linux srt takes literal paths only, so each entry is resolved to a real absolute path before the settings file is written (`fsutil.EnsureRealDirAll`, so a symlinked component is refused).

- **The working tree and the run's own directories.** These are the three a walled launch's file tools are already told they may write (`stageWriteDirs` in `internal/core/runner/claude.go`): `Request.Dir` and the directories of the brief and the receipt.
  - For a lane, the working tree is `~/.abcd.noindex/worktrees/<root-sha>/<run-id>-<lane-id>` and the other directory is the lane's `.abcd/.work.local/run/<run-id>/<lane-id>/`. The run's `state.json`, its lock and its fingerprint stay readable and are not writable.
  - For the drain's judge, the wall proof and the limit probe, all three are one temporary directory abcd makes for that launch.
  - For the planning interviewer the working tree is the checkout, and the checkout is not granted. The interview names the record paths a turn may change in `Request.Declared`, the policy grants those and the turn directory, and the interview's changed-paths check (`heldToContract`) keeps refusing every other change, as today.
  - A working tree that holds its own `.git` directory, and is no linked worktree, is granted whole, and srt's own denies cover that directory's `hooks` and `config`.
- **The main checkout's git directory**, when the working tree is a linked worktree, narrowed to what a commit in the lane writes:
  - `objects/`;
  - the lane's own `worktrees/<name>/`, with its `config.worktree`, `gitdir` and `commondir` denied;
  - the lane branch namespace `refs/heads/build/` (`BranchPrefix`) and its reflogs under `logs/refs/heads/build/`.

  Nothing grants `hooks/`, `config`, `packed-refs`, `HEAD`, the main index or any other worktree's folder, so what decides what git runs, and every branch outside the loop's namespace, stays out of reach. srt's own `.git` denies do not cover this: they reach from the wrapped process's directory down, and in a linked worktree `.git` is a file pointing at the main one.

  The folder grant covers every lane's branch (Step 1 evidence, probe 2). The spec first planned to compare every ref after each launch. That comparison is not built, for two reasons. Lanes of one run launch side by side, and the person and the loop move refs of their own while a launch runs, so a ref that moved during a launch says nothing about who moved it. And what the comparison was for is held another way: a lane lands only at the head its validators judged (`landPrepare`, `landPush`), so a branch another lane's launch moved either goes through that lane's own validators or refuses that lane's landing. Step 1 probes whether Seatbelt takes one ref file and its lock as a grant; where it does, the macOS policy grants the lane's own ref and not the folder.
- **A Go cache for each run**, by ruling 8: `GOPATH`, `GOCACHE` and `GOMODCACHE` point into the run's fence folder, `~/.abcd.noindex/fence/<root-sha>/<run-id>/`, which every launch of the run shares and abcd removes when the run finishes. `GOPATH` moves with the caches because `go` writes its checksum-database cache to `$GOPATH/pkg/sumdb/`, outside `GOMODCACHE`: with the person's `GOPATH` read-only, a build that must verify a module not yet in `go.sum` fails. The move also sends `go install`'s binaries to the run's `$GOPATH/bin`, inside the fence. The module cache is filled from the person's own cache through `GOPROXY=file://<their GOMODCACHE>/cache/download,<their GOPROXY>`, a form `go help goproxy` documents. A cached module is copied in without the network and checked against `go.sum` as usual. The person's caches and `GOPATH` are only read. A drive today hands its stages the person's own caches (`stageGoExports` in `internal/core/runner/wall.go`); under the fence the run's take their place, and `GOBIN` stays under the stage's home. Whether the run's cache starts empty or from a private copy is open question (g).
- **Temp**: a fresh directory for each launch, set as `TMPDIR`, `GOTMPDIR` and `CLAUDE_CODE_TMPDIR`. The system's `/tmp` is not granted, except srt's own built-in paths.
- **The host's state**, kept apart from its configuration:
  - claude, bare: a fresh `CLAUDE_CONFIG_DIR` for each launch. Under `--bare` the harness reads no OAuth credential and no keychain, and takes its key from `ANTHROPIC_API_KEY` or an `apiKeyHelper` passed with `--settings` ([headless](https://code.claude.com/docs/en/headless)), so the person's `~/.claude` and `~/.claude.json` are never needed and stay read-only. That also covers the paths the vendor warns about: "a sandboxed session that can write them can persist hooks, permission rules, or MCP servers that run unsandboxed the next time you launch Claude Code".
  - claude, walled: the login lives in the profile directory, so `CLAUDE_CONFIG_DIR` stays that directory and the harness may write its own state there. Inside it the launch may not write what decides the next launch: `settings.json`, and every name a launch would load beside it (`wallLoaded`: the local settings, `CLAUDE.md`, and the agents, skills, commands, rules, output-styles and hooks folders). The stage's home (`Profile.ShellHome`) is writable, since the shell and `GOBIN` live there, but for its `.gitconfig`, which abcd writes before each launch. The shell file and the git configuration beside that home (`Profile.EnvFile`, `stageGitFile`) are not writable. What the harness writes in a profile during a session, and whether it starts with those denies in place, is not in the Step 1 evidence; step 1 probes it. Where the harness keeps the profile's sign-in on macOS decides open question (f).
  - opencode: fresh `XDG_DATA_HOME` and `XDG_STATE_HOME` for each launch, and an `XDG_CACHE_HOME` for each run in the run's fence folder, because opencode downloads and later runs programs from its data and cache folders. abcd seeds the run's cache once, at the run's first opencode launch and outside the fence, from the person's: by `clonefile` on APFS or a reflink on Btrfs and XFS where the filesystem offers one, and by a plain copy elsewhere. The run's launches share the seed, so what one launch fetches there a later launch of the same run may run; that stays inside the run's fence, and nothing written there flows back to the person's cache. The person's configuration folder stays where it is, read-only, since it names the model and the server (`internal/core/runner/opencode.go`); a pinned run's sealed configuration (`sealPin` in `internal/core/runner/pin.go`) is read-only too. The person's credential file is reached through a read-only link from the fresh data folder. A provider whose sign-in rewrites that file cannot refresh inside the fence: the launch fails, naming it, and the person uses a key for runner launches. abcd sees only the `<provider>/<model>` route, not a base URL set in opencode's configuration, so a provider abcd does not know, or a local model server, is reached by adding its host to `runner.sandbox.hosts`. Step 1 measures all of this.
- **abcd's home** (`~/.abcd.noindex/`) is readable and is not writable, except the launch's own worktree, the run's fence folder and, on a drive, the profile directory and the stage's home as described above. So the next launch's policy (`config.json`, `rules.json`, `trusted-roots`, `load-limits`, the other worktrees, the other profiles, the launch ledger) is the person's alone to change.

**Reads.** Reads stay open, as srt's default leaves them, except for the credential stores a launch never needs. Those are denied:

- `~/.ssh` and `~/.gnupg`;
- `~/.aws`, `~/.config/gh`, `~/.netrc`, `~/.git-credentials` and `~/.docker/config.json`;
- abcd's own credential file under its home;
- every profile directory but the launch's own, and every stage's home but its own, so a drive on one login cannot read another;
- on macOS, the login keychain's files, `~/Library/Keychains`. Step 1's probes added this one: without it the keychain handed a stored GitHub token to a command inside the fence, although `~/.config/gh` was denied (Step 1 evidence, probe 3).

The vendor's guide says why the two layers go together: "Without network isolation, a compromised agent could exfiltrate sensitive files like SSH keys". The model endpoint is reachable, so this narrows what a misled agent can send there. It is not part C's promise.

Writes srt already allows outside the list (the child's standard streams, `/tmp/claude`, `~/.npm/_logs`, `~/.claude/debug`) are named in the launch's fence record as the fence's own, so the list abcd states is the list that holds.

The policy goes into a settings file abcd writes for each launch, in the run's fence folder but outside every path the launch may write. It is never `~/.srt-settings.json`, and `--control-fd` is not used, so nothing can change the policy while the launch runs.

### Commits inside the fence

The agent commits in its own worktree: its brief tells it to (`renderBrief` and `renderIssueBrief`). Two things a commit sets off would otherwise fail inside the fence.

**Automatic gc.** After a commit git may run `git gc --auto`, through `git maintenance run --auto`, and that wants `packed-refs` and `.git/gc.log`, neither of which is on the write list. abcd's own git turns automatic maintenance off on its command line (`workingTreeCommand` in `internal/gitutil/repo.go`), but the agent's git is not abcd's, and the launch's environment drops every inherited `GIT_CONFIG_*` (`gitutil.ScrubbedEnv`). So the launch carries `gc.auto=0` and `maintenance.auto=false` as its own seal: `GIT_CONFIG_COUNT` with its keys and values, set over the scrubbed environment by `sealedEnv` in `internal/core/runner/proc.go`. On a walled launch the stage's shell keeps only named variables (`stageShellNames`), so those names join that list and the wall proof admits them. The spec first planned to write the two values into the lane's `config.worktree` as well. Git reads that file only where `extensions.worktreeConfig` is on, abcd never turns it on, and a drive holds that setting as one a lane may not change, so the file is left alone and stays on the deny list. An agent that unsets the seal in its own shell gets a gc that is refused after its commit has landed, recorded like any other refusal.

**Commit signing.** Where the person's git signs every commit (`commit.gpgsign`), the agent's commit needs the signing key, and the read list denies `~/.gnupg` and `~/.ssh`. On Linux srt's seccomp filter also blocks the Unix socket a signing agent answers on. The key is not opened to the agent: an agent that can read it can sign anything as the person. Fenced lanes commit unsigned (question d, dcn-2610222041439951), with `commit.gpgsign=false` in the launch's seal, and abcd signs outside the fence, where it lands the work.

The landing already makes a commit of its own on top of the judged head, the records commit (`landRecords` in `internal/core/implement/loop/land.go`), and the preflight receipt the push needs is minted for the head that commit leaves. So the signing goes there, before the records commit and before any receipt names the head: abcd rewrites each lane commit with the same tree, message, author and order, signed, compares the rewritten chain tree for tree with the head the validators judged, and refuses the landing on any difference. The rewrite is `git commit-tree`, which runs no hook, in the person's own environment (`personEnv` in `gitenvs.go`), where their signing configuration is; a drive's stage environment carries their identity and filters and nothing else. A commit id the lane's receipt names in `resolves` is mapped to its rewritten commit before `capture resolve` stamps it. Where the person does not sign, nothing is rewritten. A signing program that waits on a person who is not there is bounded as the landing's other commits are (`commitTimeout`), and the lane waits at its landing, naming it.

### A machine without a fence

Ruling 7: the run stops until the person acts. The check (`Fence.Check`) runs in the front doors, before anything is created, wherever a role may be launched:

- `abcd build` and `abcd implement step`, once `loadRunners` has read the runner configuration and a role is routed to a runner;
- `abcd drain`, once at its start for the same condition. The spec first placed this at each lane's start; a drain's lanes all launch on the configuration its command read, so one check before the first move covers them without a change to the loop;
- `abcd drain --drive`, as one of the walls its start proves before its question (`driveWalls.static` in `internal/surface/cli/drain_drive.go`), since every role of a drive is routed (`Config.Walled`);
- `abcd intent interview` (`internal/surface/cli/written_interview.go`).

A run whose roles all stay on the host checks nothing. The front door hands the evidence to the runner configuration, as it hands a drive's profile (`Config.Walled`), and the launch point starts no agent on a configuration that carries neither that evidence nor the off switch. So a front door that missed the check launches nothing unfenced. The check:

- finds `srt` on PATH and admits it as a harness is admitted (`launcher.admit`): an absolute path, not writable by group or other, not inside the repository or the checkout. srt's own README warns that a copy inside a writable path can be rewritten from inside the sandbox. srt is a Node script, so the check also covers the interpreter its first line names, found on PATH, and the package folder the script resolves into, and none of them may sit inside a path any launch may write;
- reads its version. abcd was tested against 0.0.79, and that is a statement of what was tested, not an exact pin: a 0.0.x version promises nothing between releases, so no range can be trusted, and an exact pin would break every machine at each upstream release without saying whether the fence still holds. A version other than the tested one is warned about once, at the run's start, naming the version found and the one tested, and is not refused. The tested version moves in a change that runs the `fence` test lane on the new one, and step 1 moves it first, to the version current when its remaining probes run;
- runs a trial launch, `srt --settings <trial> -- /bin/sh -c ': > "$1"; : > "$2"' sh <inside> <outside>`, and checks from outside that the write inside landed and the write outside did not. The trial, not the version string, is what decides: it makes "fenced" a fact, not an assumption, per [`enforcement-claims-are-facts`](../../principles/enforcement-claims-are-facts.md).

A failed admission or a failed trial refuses the run: exit 2, nothing created and nothing launched. The refusal names the reason (srt not found, refused, bubblewrap or user namespaces missing as srt reports, the trial failing to start, or the trial writing outside), how to fix it, and the off switch.

Where srt is not found, abcd offers to install it (question b, dcn-2610222041424027): it names the exact install it would run (srt's npm package at the tested version, and on Linux the system packages srt needs, through the system's package manager) and asks. It installs only after the person's explicit yes, never by an agent's answer and never by a flag or a config value. The offer is made only where the person typed the command at a terminal with no host session: a drive's start, which already asks its question there, and a plain-terminal interview. Inside a host session an answer would reach abcd through the agent, so there the refusal names the install commands for the person to run themselves. The install runs with everything the person's account can do, which the product thinker accepted. After it, the check runs again from the start, admission and trial included. Anything short of a yes installs nothing, and the run stays refused as above.

The off switch is `runner.sandbox.off: true` in `~/.abcd.noindex/config.json`. The `runner` namespace is already the machine's alone, and a repository's `.abcd/config.json` that sets it is refused, naming the machine's file (`runner.Load`). No flag sets it either, so a project's files and a command an agent can type can never turn the fence off. With it set, the run says once, at its start, that its launches are not fenced and where that was set.

While the steps land one by one, an unset switch leaves launches as they are today and only `runner.sandbox.off: false` puts the fence up, so the default branch never refuses a run for a fence it does not yet wrap every launch in. Step 18 makes an unset switch mean fenced.

A wrap that fails during a run (srt removed, its settings refused) is a new runner reason, `fenced`. It is not a fallback: like a rate limit, `Dispatch` hands the role to no other route and writes no fallback receipt. The step that launched it is refused with the same remedy and the lane keeps its await; a drive ends there and says so.

### The guard inside the fence

The approver runs **inside** the fence. The harness starts `abcd guard approve` itself, as the one MCP server on stdio its `--mcp-config` names (`approverConfig` in `internal/core/runner/claude.go`), and a child of the wrapped process is inside the wrapper. On a walled launch the hooks abcd's settings name (`wallHooks`: the guard's hook, the rules loader and the session hooks) are children of the harness too. Keeping them inside costs nothing and is the safer side:

- the approver only reads: abcd's own binary, the registry of the working tree and of the lane's base commit, and the bundled registry, all readable in the fence. It writes nothing;
- its answer can only narrow what the agent does. An allow it gives still runs inside the fence, so an approver that is wrong, or fooled by the input it judges, can never let a command past the boundary;
- putting it outside would mean a channel crossing the fence, carrying input the agent controls, to a process with the person's full reach. That channel would be a way out, and abcd would have to build it, which ruling 6 rules out. On Linux, srt's seccomp filter blocks exactly that kind of Unix-socket bridge.

A hook that keeps a record writes it under the home it finds. On a stage that is the stage's home, which the launch may write and abcd empties; a write a hook aims at the person's home is refused by the fence. Step 11 runs each hook under the real fence and lists what it writes, so none fails where the agent would see only a broken hook.

The command the `--mcp-config` names is abcd's own executable (`selfBinary`, the `os.Executable()` of the abcd that starts the run), which a drive also pins by content (`CheckPinnedBinaries`). It is never a name looked up again on PATH, and never `go run ./cmd/abcd` from the worktree, whose source the agent may edit. Under the fence it is admitted by the same rule as srt: an absolute path, not writable by group or other, and never inside a path the launch may write. An executable that fails the rule refuses the launch, naming it.

So the guard keeps its place as adr-42's mistake filter, teaching inside the boundary the OS enforces. The limit the spec first stated here, that an agent could edit the registry it is judged against, no longer holds: a registry weakened since the lane's base leaves none in force, and every command is then refused (`registry-not-in-force`).

### Checks that read the machine run outside

The peers scan, the reservation's renewals and widenings, the landing's push and the drive's own holds already run in abcd, outside any launch. The load gate is the one check an agent reaches from inside: the implementer's brief has it run `abcd implement load --wait --narrow -- go test …` on the packages it changed, and a preflight run for a lane waits at the first gate and the race gate. Inside the fence such a waiter would read the fence's view of the process table. On Linux the process namespace shows it only its own processes. On macOS it sees none at all: `/bin/ps` cannot be started inside the fence and `pgrep` finds nothing, so `implement load` reports the check unavailable and carries on (Step 1 evidence, probe 5). It also could not write its hold, its narrow budget or a whole run's marker, which are kept under the person's home (`abcdhome.MachineHome`), and on macOS today's waiter already refuses for that reason rather than opening.

So a waiter never judges the machine from inside a fence:

- a waiter that finds its launch's gate directory (an `ABCD_STAGE_GATE` the launch sets, a name a stage's shell already keeps by its prefix) asks through it, and waits for an answer. Its request is a small file of closed fields: which gate (the first, the race or the narrow one), its timeout, and for a narrow run the words of its command, which the outer side reads to key the budget and never runs;
- a waiter that cannot write its hold refuses, naming the fence, rather than opening. That covers an agent that unsets the variable;
- whoever started the fenced command serves the directory from outside the fence: the launch point for a launch, and the drive for a lane's preflight as the answer to question (g) has it run. For each request it starts an ordinary `abcd implement load` waiter of the same gate, from abcd's own admitted binary, with `--gate <dir>`. That waiter judges the whole machine as every waiter does, is visible in the process table to every other account's waiters, takes the hold when the gate opens, and writes its answer where the fenced side can read it and cannot write;
- that outer waiter then holds the gate for the inner command's length, as `--wait -- <command>` does, until the inner side writes its done marker or the fenced command ends;
- one request may be outstanding for each fenced command, the outer waiter's own timeout bounds it, and a request that does not parse is refused and answered as shut;
- the launch's clock stands still while its outer waiter is queued, as it does today for a waiter in the harness's own tree (`queuedAtGate` in `internal/core/runner/gatewait.go`).

A preflight run inside the fence runs its commands inside the fence, under gates read outside it. Its receipt lands in the worktree's local tier, which the launch may write. None of this needs the fence to exist: a waiter that can write its hold behaves as today, so step 12 can land before any other step.

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

The list above stands as it was written on 22 October. Steps 3 to 14 keep their numbers and their subjects in the rework of 14 November, so its step numbers still point at the right work, with four differences. The run record's half of step 10 is step 16: step 10 records the refusals on the launch's ledger entry, and step 16 carries them into the run's state. The signing that criterion 2 rests on is step 15. The live run that was step 15's is step 18's. And the ref comparison of step 9 is not built, for the reasons under "What a launched agent may write".

### Proof

The tests that matter run against the real fence. A test lane under the build tag `fence` runs real srt on macOS and on a Linux runner with bubblewrap installed, wrapping a stand-in harness: a script named `claude` or `opencode` on a PATH the test controls, which tries the writes, commits and gates each criterion names. Fakes stand in only for what a test cannot cause, such as a missing srt or an untested version. Before this spec closes, one `build` of a real intent and one lane of a real `drain --drive` run fenced on the built binary, on macOS and on Linux, and their records and the before-and-after hashes of the person's host settings are the evidence (step 18), per the product thinker's rule that real runs are the proof.

The `fence` lane is a Make target that joins the prerequisites of `preflight-stages` beside `smoke` and `evals-cold-reading`. On a machine where srt does not pass `Check`, preflight says by name that it skipped the lane: a skip, never a pass. CI runs the lane where it cannot skip, as a job of its own in `.github/workflows/ci.yml` as `smoke` is: on a Linux runner with unprivileged user namespaces, where ubuntu-24.04 needs the AppArmor sysctl `kernel.apparmor_restrict_unprivileged_userns=0` set first, with bubblewrap, `socat`, `ripgrep`, Node and srt installed. The job stands down where `smoke` does, on a pull request confined to documents and records, and runs on every other event, so every merge-queue entry runs the real fence on Linux. The nightly macOS suite runs the lane in one of its shards.

### Known limits, stated, not claimed

- On macOS srt's README gives "no guarantee" for a hard link, inside an allowed path, to a file outside it.
- On Linux srt's own deny scan is fixed at launch. abcd's explicit denies cover the main git directory, which exists before the launch, and not a repository the agent creates later inside its worktree.
- Go programs that check TLS through the system's trust service fail under Seatbelt (the vendor's guide names `gh`, `gcloud` and `terraform`), and `go` itself is one of them. Measured on macOS (Step 1 evidence, probe 1): every HTTPS request `go` makes inside the fence fails with `x509: OSStatus -26276`, both a module download from `proxy.golang.org` and the checksum lookup it routes through that proxy, so a module missing from the person's cache, or from `go.sum`, cannot be fetched or verified. A build whose modules are all in `go.sum` and in the person's cache works, through the file proxy, with no network. With `SSL_CERT_FILE` naming the system's certificate file (`/etc/ssl/cert.pem`), Go reads its roots from that file instead of the trust service, and the download and the checksum verification both pass, with nothing in the fence widened. srt's own remedy, `enableWeakerNetworkIsolation`, opens the trust service and is what its README warns is an exfiltration path; it is not used. So on macOS the launch also sets `SSL_CERT_FILE` (step 7). A program that ignores that variable and asks the trust service still fails, and is named where it does.
- A private module (`GOPRIVATE`) is fetched from its version-control host with the person's own credentials. One held in a store the read list denies (`~/.ssh`, `~/.netrc`, `~/.git-credentials`, `~/.config/gh`) cannot be read, and the fetch fails naming the module. Measured on macOS (probe 3): git's keychain helper gave nothing inside the fence (`failed to get: -128`, as it does outside in a session nobody can answer a prompt in), and the GitHub helper `gh auth git-credential` could not start, so no git credential reached the fence. The keychain itself did answer: a token stored there was returned in full until `~/Library/Keychains` joined the read denies (Reads, above). The ssh agent's socket is refused inside the fence (`Error connecting to agent: Operation not permitted`), so neither a fetch over ssh nor an ssh signature can use the agent.
- srt's CLI reports no refusal of its own on macOS, with or without `--debug`: the refusal reaches the launch's stderr only as the refused program's own error, `Operation not permitted` with the path (probe 4). The kernel logs each refusal, tagged by srt (`Sandbox: <program>(<pid>) deny(1) file-write-create <path>`, then `CMD64_<command in base64>_END_<session>_SBX`), to the unified log, which a `log stream` running during the launch sees and a later `log show` does not. That stream also carries refusals nothing asked about (`sysctl-read kern.iossupportversion` from every shell). `Fence.Refusals` reads the program errors (step 10); reading the log stream as a second source is left to step 10 to weigh.
- Every commit inside the fence, with git 2.52, prints `error: Unable to create '<git dir>/packed-refs.lock': Operation not permitted` during its ref update, and still exits 0 with its commit, ref and reflog written (probe 2). `packed-refs` stays off the write list. `Fence.Refusals` records the line like any other refusal, and the lane does not read it as a failed commit.
- On Linux, srt covers a protected path that does not yet exist with a placeholder on the host, and a SIGKILL skips its cleanup. abcd's timeout kill is the group's SIGKILL (`procgroup.Kill`), so a timeout first sends SIGTERM to srt and kills the group only after a grace period (step 8). After every launch abcd also removes, outside the fence, any empty placeholder that appeared at one of srt's protected names and that git does not track, so none is committed or left in the person's worktree (step 9). Step 1 checks whether bubblewrap's own session takes descendants out of the group abcd kills.
- srt quotes the argument vector into a shell string on macOS and Linux, which amends `proc.go`'s rule that the argv is never a shell line. abcd's wrapped vector is fixed words, a role name from a closed list, and paths abcd made. A path carrying a quote or a control character is refused before the launch, and a test sends a prompt through with every shell metacharacter (step 7).
- A fenced agent can still read most of what the person can read. The intent's press release was narrowed to what holds at step 2 (question e, dcn-2610222041434056): the agent cannot change anything outside its own work, and cannot read the person's passwords, keys and sign-ins (the read denies above, the macOS keychain included).
- The credential a launch runs on is inside the fence with it: the harness must read its key or its profile's sign-in to reach its model, and everything the harness starts is inside the same fence. The read denies cover every other sign-in, the other profiles' among them. The intent's press release says the agent "cannot read your passwords, keys and sign-ins"; by the answer to question (e) those words are held to what is true, so the guide written at step 18 names this one exception.
- One lane's launch can move another lane's branch, since the grant on `refs/heads/build/` is a folder's. What lands is still only a head that lane's own validators judged; a branch moved back or deleted refuses that lane's landing until it is restored. Where step 1 finds the macOS mechanism takes a single ref as a grant, the folder grant is narrowed there.
- Between a lane's launches the drive runs git for the lane outside the fence (the add, commit, merge and reset rows of `driveGitCommands`), with the repository's hooks, under the holds a drive already keeps on them. The fence neither adds to those holds nor replaces them.

### Coordination

- Steps 1 to 14 change no file of `internal/core/implement/loop`, where another session is working, and come first. Steps 15 to 17 change it and land after that session's work, each on a reservation of the files its `packages:` line names. Step 18 comes last because it puts the fence up by default.
- Steps 3 to 11, 13 and 14 change `internal/core/runner` and the new `internal/core/fence`. Their `needs:` lines say which may run side by side; two that change the same file of the runner land one after the other.
- Step 7 changes `internal/core/runner/claude.go`, `opencode.go`, `wall.go` and `limit_probe.go`, which every launch passes through, so it lands alone.
- Step 8 also changes `internal/procgroup`, which every launch and every waiter kills through, so it reserves that package too.
- Step 12 changes `internal/core/implement/load_wait.go`, which every preflight runs through, so it lands alone, after a reservation. It does not depend on the fence, and can land before step 3.
- Step 5 changes `Makefile` and `.github/workflows/ci.yml`, which every preflight and every CI run read, so it reserves both. A lane of a drive that changes `Makefile` does not land on that drive, so step 5 is built by a person's own session.
- Step 15 adds a git command to the loop, so it adds that command's row to `driveGitCommands`.
- Until step 18, an unset `runner.sandbox.off` leaves launches unfenced, so a release cut between two steps ships nothing half built.
- Every step reserves its footprint with `abcd reserve` before it starts.

### Out of scope

- Part B: the isolation report for drains and builds a person's own session drives. Part C: the network promise and its claim (the host list here is functional, not a promise).
- Confining reads. Secrets the account can read stay readable to a fenced agent.
- Windows, cloud sessions, and more than one project per run (the intent's scope conditions).
- Caches of other toolchains (npm, cargo, pip): writes to them are refused, and a run that needs one shows up in step 18's evidence before anything is added for it.
- Letting a command the guard cannot read run inside the fence instead of refusing it (iss-2610171200549757). The fence bounds what such a command could touch, and that issue stays open until the fence has run on real work.
- Choosing among several profiles, and a key route with a budget (itd-2611040344170206, itd-2611011230226453). The fence takes whichever profile or key a launch runs on from the launch.

## Questions for the product thinker

Put at step 2, after step 1's probes. Each question is written for the product thinker, in plain words, and each answer says what it leads to. The mechanism behind each one is in the sections above.

**Answered by the product thinker on 2026-10-22**, each recorded as a decision and folded into the sections above: (a) require srt (dcn-2610222041417934); (b) offer to install it, only on an explicit yes (dcn-2610222041424027); (c) the derived list, not announced per run (dcn-2610222041427241); (d) abcd signs at hand-in (dcn-2610222041439951); (e) narrow the words to what holds (dcn-2610222041434056). (g) was put on 2026-11-14 and answered on 2026-11-15, below. The questions as they were put stay below for the record.

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
- **(g) The full check a drive runs on each piece of work.** Answered by the product thinker on 2026-11-15: the check runs inside the fence, on the run's own fresh cache and its own results folder. The product thinker accepted what that leads to: each piece of work is built and tested from nothing, so a drive hands in fewer pieces in a night, and the reuse of test results never applies to a drive. When a drive's agent has finished a piece of work, abcd itself runs the project's full check on it, the build and every test, before handing it in. Those tests may be ones the agent wrote or changed. Today that check runs outside any fence, on the person's own build cache and on a shared folder that lets one check reuse another's test results (itd-2611011325421304). Ruling 8 gave each fenced run a fresh cache "at the cost of cold builds and race runs"; it was made before a drive ran this check itself and before test results were reused.
  - Run the check inside the fence, on the run's own fresh cache and its own results folder. Leads to: what the agent wrote cannot touch anything outside its work, even while it is being tested; every piece of work is built and tested from nothing, so a drive hands in fewer pieces in a night, and the reuse of test results never applies to a drive.
  - Run the check inside the fence, starting from a private copy of the person's cache and of the shared results, taken when the run starts. Leads to: the same protection, and checks nearly as fast as today where the disk can copy without duplicating (a Mac's own disk format, and some Linux ones); elsewhere a slow first copy and more disk used. Nothing flows back, so the next run starts from the person's cache again. It replaces "fresh" in ruling 8 with "private", for the agents' own builds as well.
  - Leave the check outside the fence, as it is today. Leads to: no slowdown; tests an agent wrote run with everything the person's account can do and can write the person's caches, so "can change their own work and nothing else" and the sixth acceptance criterion do not hold while a drive checks the work.

### Open questions

Raised by the rework of 2026-11-14. Question (f) comes from something built after the answers above, the unattended drive and its separate sign-in. It is written for the product thinker, in plain words, and each answer says what it leads to. Nothing here is decided, and it stands unanswered: it is put only if step 1's probe finds the drive's sign-in in the keychain. Until that probe has run, it is not put. Question (g), the other raised by the rework, is answered above.

- **(f) The sign-in an unattended drive runs on, on a Mac.** `abcd drain --drive` starts its agents on a separate sign-in the person made once, kept in a folder of its own. The answer to question (e) hides the Mac's keychain from a fenced agent, because the first probes showed that a command inside the fence could otherwise read a GitHub sign-in stored there. Step 1 has not yet measured where the agent program keeps that separate sign-in. This question is put only if the probe finds it in the keychain, because hiding the keychain would then also hide the sign-in the agent needs in order to work at all.
  - Let a drive's agents read the keychain. Leads to: a drive works on a Mac as it does today; a misled agent in a drive can read any sign-in the keychain hands over without asking, so the promise in (e) does not hold for drives on a Mac, and the guide says so.
  - Keep the keychain hidden, and refuse a fenced drive on a Mac until its agents can sign in another way, such as a key the person sets a budget for (itd-2611011230226453, not built). Leads to: the promise holds everywhere; until then a person on a Mac runs drives with the fence switched off for that machine, or does not run them.
  - abcd reads that one sign-in itself, outside the fence, and hands it to each agent as it starts. Leads to: the keychain stays hidden and drives work; abcd handles a sign-in for the first time, which adr-2610222222326104 says it never does.

## Footprint

- packages: internal/core/fence, internal/core/fence/srt, internal/core/runner, internal/core/implement, internal/core/implement/loop, internal/core/interview, internal/core/ahoy/worktree, internal/procgroup, internal/gitutil, internal/testreuse, internal/surface/cli, commands, docs, ACKNOWLEDGEMENTS.md, Makefile, .github/workflows/ci.yml, .abcd/development/brief, .abcd/development/research/notes, .abcd/development/intents, .abcd/development/decisions/adrs
- tests: internal/core/fence (policy derivation for each kind of launch, the seam's test double, admission and trial tables), internal/core/fence/srt (settings file, argv, refusal parsing and the placeholder sweep), internal/core/runner (the sandbox keys, fenced argv, environment and seal for the bare and the walled launch, the fenced reason, the ledger's fence block), internal/procgroup (the kill order), internal/core/implement (the fenced waiter and its outer gate), internal/core/interview (the declared record paths), internal/surface/cli (the check before a run starts in build, implement step, drain, drain --drive and intent interview, the drive's preflight inside the fence, the agents and status surfaces), internal/core/implement/loop and internal/gitutil (signing at the landing, the fence block of the run record, the run's fence folder removed), the `fence`-tagged real-srt lane on macOS and Linux in preflight and CI, and step 18's live fenced build and drive

## Steps

1. Probe srt on the real harnesses before anything is signed off or built
   - needs: none
   - packages: .abcd/development/research/notes, .abcd/development/intents
   - tests: an abcd lab on macOS and on Linux with bubblewrap, each probe named in the research note, on the srt version current when it runs, with the step 3 policy written by hand. Still to run from the first list (Step 1 evidence, "Not yet probed"): the Linux half, all of it; `claude --print --bare` and `opencode run` as real sessions, with a fresh `CLAUDE_CONFIG_DIR` and the person's `~/.claude` and `~/.claude.json` read-only; opencode with fresh XDG folders, a cache seeded by clone, a read-only credential link and a refreshing sign-in, and what it fetches at start; `go test ./...` of abcd itself on the run's `GOPATH` and caches; a `GOPRIVATE` module fetched from its host; a commit without the gc seal and one with `commit.gpgsign` on; a hard link from the worktree to a file outside it on macOS; SIGTERM cleanup, and whether bubblewrap's session leaves the process group abcd kills; a prompt with shell metacharacters. New since the rework: a walled launch as a drive starts it (`claude --print` without `--bare`, `CLAUDE_CONFIG_DIR` naming a profile directory, the stage's home as its shell's home, abcd's hooks on), recording where the harness keeps that profile's sign-in on macOS and on Linux, what it writes under the profile directory in a session, whether it starts with `settings.json` and the names of `wallLoaded` denied for writes, and which hosts it contacts; each hook the walled settings name run inside the fence, with what it writes; the approver started by the harness inside the fence, reading the registry at a base commit; whether Seatbelt takes one ref file and its lock as a write grant; and srt's version, flags, settings schema, built-in write paths and refusal reporting against what 0.0.79 showed. Probing builds nothing. Each limit found becomes a scope condition on the intent or a capture of its own, and a failure on the macOS or Linux half goes back to the person, with nono as the named alternative
   - evidence: the macOS half's first probes ran on 2026-10-17 and are recorded under Step 1 evidence, after the steps. The Linux half, the agent sessions and the probes listed there as not yet run are still to do
2. Sign-off: the person answers the five questions for the product thinker (done 2026-10-22; see the answers under Questions for the product thinker)
   - packages: .abcd/development/brief, .abcd/development/decisions/adrs, .abcd/development/specs
   - tests: each answer recorded as a decision and folded into this spec; on a yes to question (a), the proposed marks come off adr-22's consequence and the dependencies chapter's paragraph (02-constraints/02-dependencies.md), and on a no both are withdrawn; record-lint and docs-lint pass over both and over the ADR's acceptance
   - landed: 6d93172a6
3. Declare the fence seam and derive a launch's fence-neutral policy (itd-2610161621364075 criteria 1, 2, 3)
   - needs: none
   - packages: internal/core/fence
   - tests: fence: the package imports neither `internal/core/runner` nor `internal/core/implement/loop`; `Policy` names no mechanism, and `Fence` is met by a test double the runner's tests use; `Derive` over a `Launch` for a lane in the worktree store gives exactly the worktree, the directories of its brief and receipt, the main `.git`'s `objects/`, the lane's own `worktrees/<name>/` without its `config.worktree`, `gitdir` and `commondir`, `refs/heads/build/` with its reflogs, and the launch's temp; `hooks/`, `config`, `packed-refs` and `HEAD` are never granted; for a judge, a wall proof or a limit probe it gives that launch's one directory and its temp; for an interview it gives the declared record paths and the turn directory and never the checkout; a working tree with its own `.git` directory is granted whole; abcd's home is read-only but for the worktree; the credential stores are denied for reads, `~/Library/Keychains` among them on macOS; a path given through a symlink is refused, and so is one carrying a quote or a control character; nothing read from the repository's `.abcd/config.json` changes the list; the host list is the runner's endpoint, with a `go.mod` the `GOPROXY`/`GOSUMDB` hosts and the `GOPRIVATE`/`GONOPROXY` hosts, and the machine layer's additions, and a pattern too broad to pass is named
4. Read the off switch, the backend and the host additions from the machine layer only (criterion 5)
   - needs: none
   - packages: internal/core/runner, .abcd/development/brief
   - tests: runner: `runner.Load` claims `runner.sandbox` beside the runner names and reads `off`, `backend` and `hosts` from the machine file (`internal/core/runner/config.go`); the backend defaults to `srt` and an unknown one is refused, naming the known ones; the same keys in a repository's `.abcd/config.json` refuse the load, naming the machine's file, as the rest of the namespace does; a host entry that is `*`, a bare top-level domain or not `host[:port]` is refused; no flag sets any of them; an unset `off` reads as its own third value, apart from true and false; the configuration chapter of the brief (`05-internals/03-configuration.md`) lists the three keys
5. Admit srt and prove the fence runs (criterion 4)
   - needs: 3
   - packages: internal/core/fence, internal/core/fence/srt, internal/core/runner, Makefile, .github/workflows/ci.yml
   - tests: runner: `launcher.admit` is handed to `Check` as the one admission rule; srt's `Check`: srt absent, relative, group- or other-writable, inside the repository, or an interpreter or package folder that fails the same checks is refused with its reason and the off switch; a version other than the tested one is admitted with one warning naming both; the evidence carries the admitted paths, the version and the trial's result; `fence`-tagged: the trial launch's inside write lands and the outside write does not, on macOS and Linux; a trial whose outside write lands, or that fails to start, refuses; `make fence` runs the tagged lane, `preflight-stages` lists it beside `smoke`, and preflight names it as skipped where srt fails `Check`; CI runs it as a job of its own on a Linux runner with user namespaces, setting the AppArmor sysctl on ubuntu-24.04, and fails where it would skip
6. Refuse a run before its first launch, and say once when the fence is off (criteria 4, 5)
   - needs: 4, 5
   - packages: internal/surface/cli, internal/core/runner, commands
   - tests: cli: a `build`, an `implement step`, a `drain`, a `drain --drive` and an `intent interview` with a role routed to a runner, on a machine that set `runner.sandbox.off: false` and whose fence check fails, exit 2 before anything is created and start no process; a drive refuses among its start's walls, before its question; a run whose roles all stay on the host checks nothing; with the switch true, the run's output says once that its launches are not fenced and names the machine file; with it unset nothing is checked and nothing is said, until step 18; an untested srt version is said once at the run's start; at a terminal with no host session the refusal offers the install and installs only on a typed yes, and in a host session it names the commands and asks nothing; runner: `Config` carries the check's evidence as it carries a drive's profile, the launch point starts no agent on a configuration told to fence that carries none, with reason `fenced`, and `Dispatch` hands a `fenced` failure to no other route and writes no fallback receipt; `commands/build.md`, `commands/drain.md`, `commands/implement.md` and `commands/intent.md` name the refusal
7. Wrap every agent launch at the launch point, with its environment (criteria 1, 2)
   - needs: 1, 6
   - packages: internal/core/runner, internal/core/fence, internal/core/fence/srt, internal/core/interview, ACKNOWLEDGEMENTS.md
   - tests: runner: the four places that start an agent (`ClaudeCLI.Run` bare and walled, `OpenCodeCLI.Run`, `ProveWall`, `probeLimit`) build their `fence.Launch` from the `Request` and the `Profile` and start `srt --settings <file> -- <harness> …` with the harness's own argv unchanged after `--`; `Exec`, `CheckLogin` and opencode's version are started as before; the harness is still admitted, and on a drive still pinned (`pinnedTo`), before it is wrapped; the environment sets the launch's temp over the parent's and on macOS sets `SSL_CERT_FILE` to the system's certificate file, and the seal carries `gc.auto=0`, `maintenance.auto=false` and `commit.gpgsign=false` as `GIT_CONFIG_COUNT` with its keys and values; on a walled launch those names are among `stageShellNames` and the wall proof passes with them; the lane's `config.worktree` and the main `.git/config` are unchanged; the settings file sits outside every writable path and round-trips through srt's schema; a wrap failure is reason `fenced`; `Wrap`'s cleanup runs on every way a launch ends; the process the ledger records is the wrapper, whose tree holds the harness, and `queuedAtGate` still finds a waiter under it; a path carrying a quote or a control character is refused before the launch; interview: a turn's request names the record paths its contract lets it change in `Request.Declared`; `fence`-tagged: a prompt carrying every shell metacharacter arrives byte for byte; a commit in the lane leaves no gc refusal; `ACKNOWLEDGEMENTS.md` credits the package
8. Kill a fenced launch in order (criteria 1, 2)
   - needs: 7
   - packages: internal/procgroup, internal/core/runner
   - tests: procgroup: a timeout, and a context that ends, send SIGTERM to the wrapper first and kill the group (`Kill`) only after the grace; a wrapper that exits on SIGTERM within the grace is not sent SIGKILL, and what it left is still ended (`KillLeft`); the kill still goes through the pid the handle holds, never a pattern; an unfenced launch is killed as today; `fence`-tagged: after a timeout no descendant of the launch survives, including any bubblewrap session step 1 found outside the group
9. Sweep srt's placeholders after each launch (criteria 1, 2)
   - needs: 7
   - packages: internal/core/fence, internal/core/fence/srt, internal/core/runner
   - tests: srt: `Wrap`'s cleanup removes an empty untracked placeholder at one of srt's protected names, in the worktree and in a profile directory, and leaves a tracked or non-empty file there alone; it runs after a killed launch as after a clean one; a drive's check of the profile's loaded names (`wallLoaded`) passes after a fenced launch. The comparison of every ref after each launch, which this step first held, is not built: lanes launch side by side, so a moved ref names no mover, and the landing already refuses a lane whose branch is not at its judged head (`landPrepare`, `landPush`)
10. Read the fence's refusals and record them on the launch (criteria 1, 3)
   - needs: 7
   - packages: internal/core/fence/srt, internal/core/runner, internal/surface/cli
   - tests: srt: `Refusals` reads each refusal out of the launch's stderr and the transcript's tool errors, with its path and tool, and reads the `packed-refs.lock` line of a commit as a refusal that failed nothing; runner: an `Answer` and a `Failure` carry the launch's fence record (mechanism, version, whether it was the tested one, policy digest, host list, the fence's own write paths, refusals), `RecordLaunch` writes it on the ledger entry and `LedgerSchema` moves to 2, and an entry of schema 1 still reads; cli: `abcd agents` renders the block and its `--json` carries it; `fence`-tagged: a stand-in harness that writes outside its worktree, to `~/.abcd.noindex/config.json`, `rules.json`, `trusted-roots`, another lane's worktree and another profile's directory is refused by the OS each time, and the launch's ledger entry lists each refusal with its path
11. Keep the guard teaching inside the fence (criterion 8)
   - needs: 7
   - packages: internal/core/runner, internal/surface/cli
   - tests: runner: the command `--mcp-config` names is `selfBinary`, admitted under the fence by srt's rule, and a launch whose executable sits inside a path the launch may write, a lane worktree's own build among them, is refused naming it; `fence`-tagged: a stand-in harness speaking the prompt tool's MCP to `abcd guard approve` started inside srt gets a refused command answered `deny` with the guard's reason and advice, and a cleared one answered `allow`; the approver reads the registry at the lane's base inside the fence and leaves no fence refusal of its own; each hook of `wallHooks` run inside the fence on a stage's home completes, and what each writes is listed in the test; a real `claude --print --bare` launch honours `--mcp-config` under `--strict-mcp-config`, starting the approver and no other server, and a real walled launch does the same with its hooks on. The approver this step first waited for is on the default branch
12. Read the load gate outside the fence (criterion 7; does not depend on the fence and can land first)
   - needs: none
   - packages: internal/core/implement, internal/core/runner, internal/surface/cli, commands
   - tests: implement: a waiter that cannot write its hold refuses naming the fence and never opens, at the first gate, the race gate and the narrow gate; a waiter with a gate directory writes a request of closed fields and opens only on the outer waiter's answer; a request that does not parse, names an unknown gate or carries a timeout past the bound is answered as shut; an outer waiter given a loaded snapshot keeps the gate shut while the fenced side's own view is idle; it holds while the inner command runs and releases on the done marker or the fenced command's end; a narrow request's words key its budget and are never run; a second outstanding request from one fenced command is refused; a waiter that can write its hold behaves as today; runner: the launch point serves a launch's gate directory from abcd's own admitted binary and stops serving when the launch ends, and the launch's clock stands still while its outer waiter is queued; cli: `implement load --gate` is in help, `commands/implement.md` and the generated reference
13. Give each run a cache of its own (criterion 6)
   - needs: 7, 12
   - waits on: nothing now. Question (g) is answered: the run's cache and its results folder start empty, and a drive's preflight runs inside the fence
   - packages: internal/core/fence, internal/core/runner, internal/surface/cli, internal/testreuse
   - tests: fence: `GOPATH`, `GOCACHE` and `GOMODCACHE` point into the run's fence folder under abcd's home, `GOPROXY` leads with the person's module cache as a file proxy, and on a drive they take the place of what `stageGoExports` exports while `GOBIN` stays under the stage's home; opencode's run cache is seeded once for each run, by clone where the filesystem offers one; cli: a drive's lane preflight (`drivePreflight`) is started through `Fence.Wrap` on the lane's policy, with the environment `PreflightEnv` gives it, its gate directory served by the drive, and its test stages' check directory (`internal/testreuse`) inside the run's fence folder; `fence`-tagged: a `go build` in a fenced launch fills the run's caches; a build that verifies a module missing from `go.sum` writes `pkg/sumdb` under the run's `GOPATH`; a `go install` lands in the run's `GOPATH/bin`; a fenced preflight mints its receipt in the worktree's local tier; the person's `GOPATH`, `GOCACHE` and `GOMODCACHE` listings and modification times, and the shared check directory, are unchanged
14. Leave the person's host settings untouched (criterion 9)
   - needs: 7
   - waits on: the answer to open question (f) for a walled launch on macOS, only if step 1 finds the profile's sign-in in the keychain; question (f) is put only then, and until then it stands open
   - packages: internal/core/fence, internal/core/runner
   - tests: fence: a bare claude launch gets a fresh `CLAUDE_CONFIG_DIR`; a walled launch keeps the profile directory as its `CLAUDE_CONFIG_DIR`, writable but for `settings.json` and each name of `wallLoaded`, with the stage's home writable but for its `.gitconfig`, the stage's shell file and git configuration not writable, and every other profile and stage home denied for reads; an opencode launch gets fresh data and state folders, the run's cache, and the person's credential file through a read-only link; `fence`-tagged: a stand-in claude that writes `~/.claude/settings.json`, `~/.claude.json` and `~/.claude/agents/`, one on a profile that writes the profile's `settings.json` and a file under its `hooks`, and a stand-in opencode that writes its configuration folder, are refused; `CheckWallSettings` passes after a fenced walled launch; the hashes of every host configuration path the policy names are identical before and after a fenced launch
15. Sign a fenced lane's commits at its landing (criterion 2; question d)
   - needs: 7
   - packages: internal/core/implement/loop, internal/gitutil
   - tests: loop: where the person's git signs, `landRecords` rewrites each lane commit past the base with the same tree, message, author and order, signed in the person's own environment, before its records commit, and the records commit sits on the rewritten head; the rewritten chain is compared tree for tree with the judged head and a difference refuses the landing; a commit a receipt's `resolves` names is stamped by its rewritten id, and the records round judges from the rewritten head (`recordsBase`); where the person does not sign, or the launch was not fenced, no commit is rewritten; a signing program that does not answer within `commitTimeout` leaves the lane waiting at its landing, naming it; the rewrite's git command is a row of `driveGitCommands` in the person's environment with hooks off, and `TestEveryGitCommandADriveRunsIsInTheTable` passes; gitutil: the rewrite runs no hook and starts no program but the signer
16. Keep the fence's record in the run's state (criteria 1, 3)
   - needs: 10
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: loop: a verified receipt's route record, a validator's and a fallback receipt carry the launch's fence record, the run-state schema moves from 26 to 27, and a state at 26 reads with none; a run the off switch left unfenced records that once; cli: `implement status` renders the fence block (mechanism, version, whether it was the tested one, policy digest, refusals) for each lane and its `--json` carries it; `fence`-tagged: the refusals of step 10's stand-in harness appear in the lane's run record with their paths
17. Remove a run's fence folder, and stop a drive on a fence that failed (criteria 4, 6)
   - needs: 7, 13
   - packages: internal/core/implement/loop, internal/core/ahoy/worktree
   - tests: loop: a run that finishes, is discarded or is handed back whole removes its fence folder, and one still in flight keeps it across a resume; the loop names the fence folders its live runs hold, as it names their worktrees (`HeldWorktrees`), and `ahoy worktree prune` removes a fence folder no live run holds; a `fenced` failure refuses the step with the lane still awaiting its receipt, is never counted as a runner outage, and ends a drive with the remedy and the off switch named
18. State the rule, put the fence up by default, and prove it on a real run (was step 15)
   - packages: internal/core/runner, internal/surface/cli, .abcd/development/brief, commands, docs
   - tests: runner: an unset `runner.sandbox.off` means fenced; a brief invariant says an agent abcd launches runs inside a fence the operating system enforces, unless the person turned it off in their user layer, the adapters chapter describes the fence seam, and the dependencies chapter no longer says nothing of it is built; `commands/build.md`, `commands/drain.md`, `commands/intent.md` and `commands/guard.md` name the refusal and the off switch, and `guard approve --help` no longer calls the fence planned; a how-to under `docs/` says how to install the fence and turn it off, names the one sign-in a fenced agent can read, the one it runs on, and credits the package under the docs-lint allow escape; docs-lint and record-lint pass; a fenced `build` of a real intent and one lane of a real `drain --drive` on the built binary, on macOS and on Linux, each record a refused outside write, commit in the worktree, land the push signed where the person signs, and leave the person's host settings and the profile's settings byte-identical

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
