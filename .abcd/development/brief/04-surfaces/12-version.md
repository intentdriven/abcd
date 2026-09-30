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
`staleness`, with `install_mode` present only when it resolves, and
`superseded_root` present only when the answering binary sits in a plugin root
other than the one this session resolves (see *A superseded plugin root names
itself* below); the plain render prints that note on a `note:` line. The update
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
where the binary sits. A page that documents no verb is not that evidence: the
dispatcher page `abcd.md` documents the bare call, and the host-delegated pages
(`consult`, `ingest`, `prepare-this-repo`) run in the host agent, so the line
for one of those tokens says what it is instead — `abcd <record-id>` for the
first, the `/abcd:<page>` invocation for the rest — and never sends the reader
to rebuild or update. Neither is a `status` or `show` sub-verb under a record
verb (`capture`, `intent`, `spec`): the refusal names the record dispatcher,
`abcd <record-id>`, which answers that question. Failing that evidence, the disk-only vintage this verb
renders stands in. When neither says anything, the framework's line stands
byte-for-byte. The exit code, the stream and the JSON envelope are the
framework's own.

## A superseded plugin root names itself

The two shapes above are loud: the binary is asked for something it does not
have, so there is an error to hang a line on. A third shape has none. A plugin
root is named for the commit it was installed from, so every update mints a new
root and nothing prunes the old ones; a command page interpolates an absolute,
hash-pinned binary path into its own prose, and that path is designed to
expire. Between an update and the reload that re-interpolates it, following the
page runs a superseded binary that is still on disk, answers normally — exit 0,
no diagnostic — and reports a version that is true of that root and false of
this machine (iss-2609020113012227, refining iss-2608230943088357).

What the disk proves, with no network and no heuristic, is the divergence: the
plugin root this session resolves — through the same ladder every other surface
uses, which prefers the environment's own plugin-root variable over the
executable's ancestors — against the plugin root the running binary sits in,
found by that ladder's own executable-ancestor walk and layout check. When those
are two different roots, the version flag's report (and the update verb's check,
which extends it) and bare `ahoy` add a `superseded_root` note naming both roots
by the commit each was installed from, in the plain render as well as in the
JSON form. It is a note beside the answer: the reported version, vintage and
staleness are unchanged, and nothing refuses.

The two names are directory names read off the disk, so each passes through the
terminal sanitiser before it is printed: a control or bidirectional character in
one is replaced, never rendered. The command pages tell the agent to relay the
note as abcd printed it and never to rebuild the names from a path, which would
undo that.

The note is silent in three cases. A binary inside no plugin root at all — a
PATH copy, a `go run` build — has no superseded root to name, and the vintage
comparison already covers it. A binary in the root this session resolves has
nothing to disclose. And a binary served from a source checkout of abcd says
nothing: a checkout is a valid plugin root (`hooks/` sits at its top), so a
developer running the `make build` artefact while a harness session resolves its
own cache root satisfies the divergence test, but a checkout is not named for a
commit it was installed from, and the note's remedy would point at the
plugin-root binary the dogfooding rule calls the stale one. That case is the
vintage comparison's, and the stale-binary line above keys its rebuild remedy on
the same source-checkout test. The guard is keyed on the root that served the
answer: a provisioned root answering into a session whose own root is a source
checkout still names itself.

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

It ships as a host-delegated command page: the command tree registers no `abcd version` verb, so there are no flags and no sub-verbs to list.

<!-- surface-appendix:end -->
