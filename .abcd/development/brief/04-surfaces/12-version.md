# `/abcd:version` — Print the Installed Version

Tell, in one command, whether the abcd you are running is the one you think it
is: its version, how it was installed, how old it is, and whether it has
drifted from the reference it should match. The whole answer is read off disk,
so it costs nothing, works offline, and writes nothing.

The answer is a flag on the root, where every tool keeps its version, and not a
verb (itd-2609212130136102); the root's appendix in
[`08-abcd.md`](08-abcd.md) lists it. The opt-in online check is a flag of the
update verb, which fetches the latest release exactly once, compares, and names
the source it consulted: it lives with the verb that takes the update it finds
([`21-update.md`](21-update.md)), and it is the only network touch
either answer makes, because abcd never fetches implicitly
([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)). For one
release the retired verb answers with the flag, or with the check when asked for
it, and exits non-zero.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|

The table is empty: the surface is a root flag and registers no sub-command.

## Behaviour

The version flag prints a short block: the version line, then `install:`
(only when an install mode is resolvable), `vintage:` and `staleness:`. It is a
read-only render of the binary's own state, not a board for the repository, and
it answers alone: a record id beside the flag is refused rather than silently
dropped. The JSON form emits the same facts as `name`, `version`, `vintage` and
`staleness`, with `install_mode` present only when it resolves. The update
verb's check prints the same report with a `check` object added.

When the online check finds an update, the answer carries the command that takes it,
so the reader's next move is on screen rather than inferred: `abcd update` for
the one install shape the update verb can swap, and for every other shape the
remedy that shape's owner requires (the host's plugin update for a plugin-root
binary, a fresh ahoy installation for a stranded entry, the package manager's own command
for a Homebrew install). The classification is the disk-only one `abcd update`
itself dispatches on, so the online check keeps its single sanctioned fetch
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

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd version`

It moved to `abcd --version`.

<!-- surface-appendix:end -->
