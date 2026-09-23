# `/abcd:version` — Print the Installed Version

Tell, in one command, whether the abcd you are running is the one you think it
is: its version, how it was installed, how old it is, and whether it has
drifted from the reference it should match. The whole answer is read off disk,
so it costs nothing, works offline, and writes nothing.

The one exception is the opt-in `--check` flag, which fetches the latest
release exactly once, compares, and names the source it consulted. That is this
surface's only network touch, and abcd never fetches implicitly
([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)).

## Behaviour

Bare `abcd version` prints a short block: the version line, then `install:`
(only when an install mode is resolvable), `vintage:` and `staleness:`. It is
the bare-invocation convention the [surfaces index](README.md) sets out, a
read-only render of the verb's own state, and `version` keeps it rather than
sitting among the exceptions that index enumerates. What it is not is a board for
the repository: the state it reports is the binary's. Adding
`--json` emits the same facts as `name`, `version`, `vintage` and `staleness`,
with `install_mode` present only when it resolves and a `check` object present
only when `--check` was passed.

When `--check` finds an update, the answer carries the command that takes it,
so the reader's next move is on screen rather than inferred: `abcd update` for
the one install shape the update verb can swap, and for every other shape the
remedy that shape's owner requires (the host's plugin update for a plugin-root
binary, `ahoy install` for a stranded entry, the package manager's own command
for a Homebrew install). The classification is the disk-only one `abcd update`
itself dispatches on, so `--check` keeps its single sanctioned fetch
([`21-update.md`](21-update.md)).

**`staleness` is prose, not a token enum.** The field carries the same words the
plain render prints, because one derivation serves both and a second spelling
for the machine would be a second thing to keep true. A binary that matches its
reference reads `up to date`; one with no reference to compare against reads
`unknown`; one that has drifted reads `stale — ` followed by the comparison and
the reference. A consumer matches on the `stale` prefix and on `unknown`
verbatim; there is no `fresh` token to match.

## A stale binary names itself

The plugin surface and the binary ship from one release but drift apart: a
plugin update lands a newer surface before the bootstrap re-provisions the
binary, a cached root goes stale, a PATH copy outlives the root it was copied
from. A page then names a verb or flag the binary predates, and the CLI
framework's answer — `unknown command` or `unknown flag` — has the shape of a
typo rather than of a stale install.

So an unknown command or flag carries a second line, derived from what the
binary can prove on disk alone and never from the network (adr-38). Where the
command surface beside the resolved plugin root documents the very verb or flag
that was refused, the line says the binary predates it and names the remedy for
where the binary sits. Failing that evidence, the disk-only vintage this verb
renders stands in. When neither says anything, the framework's line stands
byte-for-byte. The exit code, the stream and the JSON envelope are the
framework's own.

## Where the version comes from

The version is **derived, never hand-authored**: it is read from the shipped
build, not from a literal in the record
([adr-31](../../decisions/adrs/0031-derived-versioning-from-intents.md)).
`/abcd:launch` stamps the derived version into the release artefact;
`/abcd:version` only reports what is installed.

## References

- Plugin command: [`commands/version.md`](../../../../commands/version.md)
- Derived versioning: [`04-launch.md § 3`](04-launch.md#3-versioning--marketplace)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd version`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--check` | bool |

<!-- surface-appendix:end -->
