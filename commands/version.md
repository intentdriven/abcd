---
name: version
description: Print the installed abcd version, install mode, and vintage by invoking the abcd binary. Read-only unless --check is passed.
---

# `/abcd:version`

Report the installed abcd version, install mode, and vintage — the running
binary's build revision (in a source checkout) or pinned version, and whether it
is up to date, stale, or of an undeterminable vintage relative to the on-disk
reference. This command performs **zero writes** and touches no network.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" version --json
```

Then tell the user the `name`, `version`, `vintage`, and `staleness` from the
JSON, plus `install_mode` when it is present — the key is omitted entirely when
no abcd-owned `PATH` entry is resolvable (nothing installed yet, a foreign or
dangling entry, or an unresolved plugin root). In that case say abcd is not on
`PATH` yet and point at `ahoy install` below, rather than inventing a mode.

**Checking for a newer release.** Only when the user explicitly asks whether a
newer version exists, add `--check`:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" version --check --json
```

`--check` reaches the network — it fetches the latest release once, compares,
and reports under `check` (with its `source` named). When an update is
available, `check.next_step` names the command that takes it — `abcd update`
for a binary the update verb can swap, the host's plugin update for a
plugin-root binary, or the package manager's own command for a Homebrew
install — chosen by the same on-disk classification `abcd update` dispatches
on. Relay it verbatim rather than paraphrasing; it is the one line the user
types next. abcd never fetches implicitly (adr-38): the network is only ever
touched by a verb whose documented job is that fetch — `version --check`,
`update`, `docs cite refresh`, and `memory ingest <url>`; every other path
reads only what is on disk.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**When no binary resolves.** If every rung fails, the fix is **not** to install
Go. A compiler is a dependency of neither supported install route, which both
provision a prebuilt, checksum-verified binary. Tell the user to recover in this
order, no toolchain needed:

1. Restart a session with network access. `hooks/bootstrap.sh` re-provisions the
   plugin-root binary at the start of every session that can reach the release
   origin; an empty `.bootstrap.attempt` marker with no binary beside it means a
   previous provisioning began and did not finish, so a networked restart lands it.
2. Reinstall the plugin from its marketplace when its remote is stale (for example
   one predating an organisation rename). Re-adding the marketplace re-points it at
   the live release origin; the install guide gives the exact steps.
3. Install the CLI binary with the one-liner in the README, which downloads and
   SHA-256-verifies the same prebuilt binary into `~/.local/bin`.

`go run ./cmd/abcd` and `go build ./cmd/abcd` serve only a source checkout of this
repo (contributors) or a platform carrying no released binary; they are never a
prerequisite for a plugin or CLI user.

**User input:** $ARGUMENTS
