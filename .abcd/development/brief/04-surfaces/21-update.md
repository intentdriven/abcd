# `/abcd:update` — Complete a Chosen Update

The verb's check says whether a newer release exists, and `/abcd:update` is the one
command that acts on that: it fetches the release, verifies the platform binary
against that release's own checksums and against the build provenance the
release workflow signed for it, and swaps the installed copy atomically. A
person types one verb and either has the new version or has a refusal that names
the way forward instead, such as the command that owns the file.

The two checks prove different things. The checksums are published on the same
release page as the binary, so they prove only that the download arrived
intact: whoever could replace the binary could replace them with it. The build
provenance does not trust the page. It must be signed for this repository's
release workflow as it stands on the main branch, run on a GitHub-hosted
runner, and it must name both the exact bytes fetched and the commit the release
tag points at, so an older release's genuine build uploaded under a newer tag
fails it. A file that already matches the release's checksums is held to the
same check before it is called current. Where the provenance cannot be verified,
offline included, the verb refuses and nothing is swapped.

The verb is the explicit ask. abcd never checks for or applies updates on its own
([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)); this verb,
with its check, reaches the release origin only when invoked
([itd-130](../../intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in.md),
spc-32). It is not the only **command** that reaches it. The launch preview and
the release cut reach it too, and only when asked to read the parity baseline
from a release's published plugin archive rather than render it fresh: they
fetch that archive and the release's checksums, hold the one to the other, and
announce each fetch on stderr as they make it. The release origin is not this
verb's only destination either: to verify a fetched
binary's build provenance, the verb also reaches GitHub's API, for the binary's
attestations and for the commit the release tag names, and Sigstore's TUF
repository, for the trusted root the signature is checked against. Each is
reached only after a release has been chosen, so none of them discovers an
update.

Nor are commands the only code paths to that origin. `hooks/bootstrap.sh` pins the
same repository, resolves its latest release over the network, and downloads the
checksum-verified asset — and the hook configuration fires it whenever the plugin
root holds no binary, so it reaches the origin without the user naming a network
action in that moment. adr-38 admits it as a tier of its own: provisioning
completes a chosen update, it never discovers one.

Nor is this verb the only command that reaches GitHub's API and Sigstore's TUF
repository. An ahoy installation reaches both: before it promotes the plugin's
cached release binary to the owned copy on `PATH`, or adopts an owned copy
already there, it holds that binary to this verb's own provenance check, for the
release the cache's record or the copy's build metadata names. Like this verb's,
that reach is keyed by a release already chosen, so it discovers nothing.

The check is a flag of this verb (itd-2609212130136102): it prints the
version report [`08-abcd.md`](08-abcd.md#what-the-version-flag-reports) describes, fetches the latest
release's tag once, compares, and names its source, swapping nothing and taking
no tag. When an update is available, its `next:` line names the command to type,
chosen by the same on-disk classification this verb dispatches on, so the check
never constructs the updater that swaps a binary.

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
| `news` | — | shipped |

The verb's one argument is an optional release tag, and the check is a flag.
A lowercase word that is not a release tag (every release tag starts with `v`
and a digit) is a mistyped sub-verb, refused as an unknown command with exit 2
before anything is resolved or fetched.

- **News** is the moment after an abcd update
  (itd-2610161617096084, itd-2610161616086564). Bare, it prints what the agent
  relays at the start of its first reply: one line per release this computer
  has not been shown, from the digest of changes the binary carries
  (`internal/core/releasedigest`), naming how many changes each made and the
  records that need the person to act, a line naming any release the digest
  cannot describe with where its notes are, and, where the update brings this
  repository setup it never answered, the offer to run setup again. Its full
  form prints every change, those needing action first, with the record each
  cites, then what waits here. Neither writes. Two forms write, and only
  under `~/.abcd.noindex/after-update/`, never in the repository: one records
  the person's answer to the offer, yes or not now, once per repository and
  release, and one records that the news was read where no offer was made.
  The plugin page runs them only with a person's reply. A yes is followed by
  ahoy's only-waiting install, the setup that waits. Its words are numbers,
  checked release tags, core's titles and the digest, never the repository's
  text, and nothing is fetched.

## Behaviour

The verb takes an optional release tag; with none, it resolves the latest. The dispatch
 is keyed on what actually runs: the first `abcd` on `PATH`. Only a
regular file proven to be abcd's own is ever swapped, and there are three proofs,
tried in order:

| Proof | What establishes it |
|---|---|
| `release-manifest` | the file's digest appears in a published release's checksums. The strongest, and the only one that also DATES the file, so it is tried first |
| `running-executable` | the file IS the executable this process runs from. Nothing else can be, so no forge object is consulted |
| `path-entry-record` | `~/.abcd.noindex/path-entry` records this exact file as the copy abcd installed, and the bytes still hash to what was recorded |

The last two exist because the first one dies with the release object
(iss-2609012000222546). Release assets are deletable, and an ownership proof
resting on them stops proving the day they are deleted, stranding an install with
no way forward. Both replacements are independent of what the forge still serves —
one is a property of the running process, the other a claim abcd wrote on this
machine — so a deleted release costs the receipt its vintage and never its
ownership.

Everything else on `PATH` is a loud refusal naming its remedy rather than a swap:
a plugin-root binary belongs to the plugin update, a Homebrew-resolved install to
`brew upgrade`, a stranded owned entry to a fresh ahoy installation, an owned pin into a
superseded plugin vintage to that same ahoy installation, a track-latest dev shim
to a mode switch first, and a foreign occupant to whoever put it there, its
remedy asking for that occupant to be removed or renamed. Two more answer the
cases where there is nothing to act on at all: no `abcd` anywhere on `PATH`, and
an entry whose install shape abcd cannot classify, which fails closed rather than
fetching.

Three further refusals arrive after the dispatch, each needing something only
the release origin can supply. A bare update whose resolved latest release is
older than the newest release this machine has installed or verified is refused
before anything is fetched, because the publisher may have withdrawn the newer
one; naming the older release is the way back to it on purpose. Once the release
checksums are in hand, a regular file that none of the three proofs vouches for
is refused. That one deliberately does not ask for a deletion — removing the
file would destroy the only tool able to fetch a replacement — and names the
reinstall route that overwrites the entry in place instead. And a binary whose
build provenance cannot be verified is refused, offline included, whether it is
the downloaded release or a file that already matches the release's checksums
and would otherwise be called current; its remedy is to retry once GitHub's API
and the TUF repository are reachable, or to check the release by hand against
the same policy and copy it over the installed one.

A tag abcd resolved rather than one the caller typed is confirmed before the
fetch, unless the caller has already said yes. The question is only put where somebody is there
to answer it: the input the answer is read from and the stream the question is
written to both have to be a terminal, so a hooked or scripted run is never left
blocking on a read. Under the invocation the plugin command issues, pre-confirmed
and in JSON, nothing is emitted until the receipt, so there the resolved tag is
first named in the receipt itself.

The transport is pinned: no proxy or CA overrides from the environment (set ones
are ignored and named in the receipt), redirects only onto the release origin's
own hosts, every hop re-checked. The redirect rule holds for every destination:
the API and the TUF repository are reached directly over the same transport, and
a redirect from either is followed only onto those same release hosts. The
release is verified in full, against its checksums and its build provenance,
before anything is written, then staged beside the target and swapped in
atomically, so a failed download, verification, staging write or swap leaves no
partial file. Download progress goes to stderr, and only when stderr is a
terminal: stdout carries the receipt and nothing else, and a piped or hooked run
shows no progress.

The target file is not quite the only thing the verb touches. Where this machine's
install record names the very entry just refreshed, abcd re-stamps that record
with the digest the release checksums proved, so the file it updated does not read
as somebody else's binary the next time an install shape is judged. The re-stamp
runs on an already-current outcome too, and it does nothing at all where no record
names that path.

The verb also records the release it installed or verified, on a swap and on an
already-current answer alike, as this machine's newest release: the mark the
newest-release refusal above is judged against, kept in the user-level abcd home
beside the install record. A bare update only ever raises the mark, so a release
at or below it changes nothing. A release the person named is recorded whatever
the mark said, so naming an older release lowers the mark to it, and that is the
one way the mark goes down. The plugin bootstrap raises the same mark whenever it
provisions a release it verified against the published checksums. A mark that
cannot be written leaves the update in place and says so in a note on stderr.

## An update says so once

Every swap of the abcd binary announces itself once, when it completes, in one
wording: `abcd updated from <old> to <new>` (`update.UpdatedFormat`, the ruling
CJ1b of 2026-09-29). `abcd update` opens its receipt with that line, and the
plugin bootstrap (`hooks/bootstrap.sh`) opens its success notice with it when
the release it installs replaces an earlier one, and records the replaced
release as `previous_tag` in the `binary-meta` it writes at the swap (the ruling
CJ1). A first install, a copy out of the cache at the same release and the
per-session fast path replace no release and print no such line. The session
start check never prints that line, with one exception: the bootstrap salvage
that the per-prompt, per-command and pre-compaction hooks run discards its output, so a swap made there adds
`transition_unseen=yes` to the `binary-meta` it writes, and the next session
start shows the line once. Once is a claim per release: the session check
creates `cache/update-shown-<new>` in the plugin data directory, or
`.update-shown-<new>` in the plugin root when the bootstrap ran in its degraded
per-root mode and wrote the root's `.binary-meta`, with one exclusive create
(`O_CREATE|O_EXCL`). Of any number of sessions starting together, the one whose
create succeeds shows the line; a claim path that already exists, or a create
that fails for any other reason, shows nothing, so the line never repeats. That
empty claim file is the session check's single write, and claims for earlier
releases stay where they are. A directory failing the shape check every reader
of the data directory applies, or a tag outside the release-tag alphabet, shows
nothing and writes nothing.

What the update changed is said apart from that line, after the person's
first message (spc-2610170635097509). When this computer has not been shown
the news up to the running release, or the update brings the repository setup
it never answered and the offer for the running release was not answered
here, session start prints one constant sentence to the session's context,
with no version and no count, telling the agent to run this verb's news
sub-verb and relay what it prints at the start of its first reply. Working
that out reads only what is on this computer: the binary's version and the
digest it carries, the repository's `setup_version` checked to be exactly a
release tag, the binary record's `previous_tag`, the setup answers records,
the store under `~/.abcd.noindex/after-update/`, and, where an offer could be
due, the repository files setup reads to tell which of its questions apply;
nothing is fetched or written. The
sub-verb is described under [Sub-verbs](#sub-verbs); the setup a yes runs is
the ahoy chapter's.

## The receipt

Three terminal outcomes ship, and the receipt's `action` field names which one
happened:

| `action` | What it means |
|---|---|
| `swapped` | the file was replaced, and the render opens `abcd updated from <old> to <tag>`, with the path on the line below |
| `already-current` | the target's digest already equals the release's and its build provenance verifies as that release's, so the binary is left untouched |
| `refused` | a dispatch, ownership, newest-release or provenance refusal, naming its shape and its remedy |

On a run that reached the release origin, the receipt carries beside `action` the
origin, the tag, the asset and its digest, the target path (redacted to `~`), and
the ownership proof that allowed the swap. A swap or an already-current answer
also carries `attested_by`, the signer identity of the build provenance the
binary verified against: the release workflow on the main branch. It carries
`env_ignored` when proxy or CA overrides were scrubbed. A refusal receipt is
deliberately thinner: a refusal raised before any fetch carries the target path
and a block naming shape, detail and remedy, and nothing else, because there is
no release it could name. In the JSON form a refusal of either kind is one document on stdout: the receipt,
carrying the three fields of the global refusal envelope (`"abcd": "error"`,
`error`, `exit_code`) beside its own, so a reader expecting one document gets
the refusal whole rather than a receipt followed by a second envelope.

An old version number is only derivable when a release manifest dated the file it
replaced. A file swapped under either local proof has no published release naming
those bytes, so the receipt opens `abcd updated from an unpublished build to
<tag>` and reports the old digest on its `replaced:` line instead. An absent old version is therefore the
documented shape, never a broken receipt.

## References

- Plugin command: [`commands/update.md`](../../../../commands/update.md)
- Intent / spec: [itd-130](../../intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in.md) / [spc-32](../../specs/closed/spc-32-abcd-update-completes-a-chosen-update-in.md)
- Staleness check it completes: [`08-abcd.md` § The version](08-abcd.md#the-version)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd update`

Sub-verbs: `abcd update news`.

| Flag | Type |
|---|---|
| `--check` | bool |
| `--yes` | bool |

### `abcd update news`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--answer` | string |
| `--full` | bool |
| `--seen` | bool |

<!-- surface-appendix:end -->
