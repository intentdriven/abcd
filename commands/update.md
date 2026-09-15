---
name: update
description: Complete a chosen update of the PATH-installed abcd binary — fetch the named (or resolved) release, verify it against the release's own checksums, swap atomically. The verb is the explicit ask; abcd never updates on its own.
argument-hint: "[tag]"
---

# `/abcd:update`

Complete a chosen update of the PATH-installed binary. The verb's documented
meaning IS the fetch: it resolves the latest release (or takes an explicit
tag), verifies the platform binary against the same release's
`checksums.txt`, and swaps the PATH copy atomically, printing a receipt with
the origin, tag, digest, and old→new versions. abcd never checks for or
applies updates on its own — this verb, and `version --check`, are the only
two commands that reach the release origin, and each only when invoked.

Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update --yes --json
```

An explicit tag pins the release; the bare form resolves the latest and the
receipt names what it resolved (`--yes` skips the terminal confirmation,
which cannot be answered here):

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" update v0.7.0 --json
```

Report the receipt's `action`, `tag`, `digest`, `target_path`, and
`old_version` → `new_version` from the JSON. If `env_ignored` is present,
relay it — it names proxy/CA environment overrides the fetch deliberately
refused to honour.

`ownership` names the proof that let abcd replace the file: `release-manifest`
(its digest is published), `running-executable` (the file is the binary that
ran the command), or `path-entry-record` (`~/.abcd/path-entry` records it as
this machine's install). The last two carry no `old_version` — no published
release names those bytes any more — and the receipt reports `old_digest`
instead; relay it as an unpublished build with its digest, not as a missing
value.

**Expect a refusal in a plugin session, and relay it as the answer, not an
error.** Every refusal is a named shape with a remedy in `refusal`:

- `plugin-root` — the binary belongs to the plugin install, and `abcd update`
  never touches a plugin root. Tell the user to take a plugin update in the
  host.
- `dev-shim` — the PATH entry is the track-latest dev shim; `abcd ahoy
  install` switches modes first.
- `owned-dangling` — a plugin update stranded the entry; `abcd ahoy install`
  repoints it.
- `package-manager` — the binary resolves into a Homebrew Cellar; relay the
  printed `brew upgrade abcd`.
- `foreign` / `unprovenanced-file` — abcd never clobbers a binary it cannot
  prove is its own; relay the described occupant and remedy. The remedy
  reinstalls OVER the file: never suggest deleting it first, because the verb
  cannot run once it is gone.
- `absent` — nothing on PATH to update; point at the install remedy.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
