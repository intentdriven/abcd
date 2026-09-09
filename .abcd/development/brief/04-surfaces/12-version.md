# `/abcd:version` — Print the Installed Version

`/abcd:version` reports the installed abcd version, install mode, and vintage
(the verb's own help summary). It performs **zero writes**, and it is
network-silent unless the opt-in `--check` flag is passed — that flag fetches
the latest release exactly once and compares, this command's **only network
touch** (abcd never fetches implicitly —
[adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)), naming its
source and verdict in the render.

## Behaviour

```bash
abcd version --json
```

emits `{ "name": "abcd", "version": "<version>", "vintage": "<revision>",
"staleness": "<verdict>" }`, plus `install_mode` when a PATH-entry
install mode is resolvable (the field is omitted when empty) and a `check`
object (latest, source, verdict) when `--check` was passed. When the verdict is
that an update is available, `check` also carries `next_step` and the plain
render adds a `next:` line: the command that takes the update — `abcd update`
for the one install shape the update verb can swap, and for every other shape
the refusal remedy `abcd update` itself would print (the host's plugin update
for a plugin-root binary, `ahoy install` for a stranded entry, the package
manager's own command for a Homebrew install) — chosen by the same disk-only
classification the update verb dispatches on, so `--check` keeps its single
sanctioned fetch ([`21-update.md`](21-update.md)). The plugin command
(`commands/version.md`) reads the JSON and tells the user the `name`,
`version`, `install_mode`, `vintage`, and `staleness`.

**`staleness` is prose, not a token enum.** The field carries the same words the
plain render prints on its `staleness:` line, because one derivation serves both
and a second spelling for the machine would be a second thing to keep true. A
binary that matches its reference reads `up to date`; a binary with no reference
to compare against reads `unknown`; a binary that has drifted reads `stale — `
followed by the comparison and the reference, either `stale — behind the
checkout tip (<rev>)` for the ancestry-guarded checkout comparison or `stale —
differs from the <source> (<rev>)` for a version or pin comparison, which is
string equality and therefore claims no direction. A consumer matches on the
`stale` prefix and on `unknown` verbatim; there is no `fresh` token to match. Without `--json`, bare
`abcd version` prints a short block — the version line (e.g. `abcd dev` in a
development build) followed by `install:` (when resolvable), `vintage:`, and
`staleness:` lines — not the version string alone, and it does **not** render
a full status board; the [surfaces index](README.md) carries the one
enumeration of where the bare-status convention holds, and `version` is not on it.

## A stale binary names itself

The plugin surface (`commands/*.md`) and the binary ship from one release but
drift: a plugin update lands a newer surface before the bootstrap re-provisions
the binary, a cached root goes stale, a PATH copy outlives the root it was
copied from. A page then names a verb or flag the binary predates, and the CLI
framework's answer — `unknown command` or `unknown flag` — has the shape of a
typo, not of a stale install. So an unknown command or flag carries a second
line derived from what the binary can prove on disk alone, never from the
network ([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)):
when the command surface beside the resolved plugin root documents the very verb
or flag that was refused, the line says the binary predates it and names the
remedy for where the binary sits — `make build` in a source checkout, the plugin
update for a plugin-root binary, `abcd update` for a PATH copy; failing that
evidence, the disk-only vintage this verb renders stands in (behind the checkout
tip, or differing from the release the plugin cache pinned); and when neither
says anything, the framework's line stands byte-for-byte. The exit code, the
stream and the JSON envelope are the framework's own.

## Where the version comes from

The version is **derived, never hand-authored** — it is read from the shipped
build, not from a literal in the record ([adr-31](../../decisions/adrs/0031-derived-versioning-from-intents.md)).
`/abcd:launch` is the surface responsible for stamping the derived version into
the release artefact; `/abcd:version` only reports what is installed.

## References

- Plugin command: [`commands/version.md`](../../../../commands/version.md)
- Derived versioning: [`04-launch.md § 3`](04-launch.md#3-versioning--marketplace)
