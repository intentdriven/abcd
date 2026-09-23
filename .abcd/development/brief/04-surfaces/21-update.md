# `/abcd:update` — Complete a Chosen Update

`abcd version --check` says a newer release exists. `/abcd:update` is the one
command that acts on that: it fetches the release, verifies the platform binary
against that release's own checksums, and swaps the installed copy atomically. A
person types one verb and either has the new version or has a refusal that names
the command that owns the file instead.

The verb is the explicit ask. abcd never checks for or applies updates on its own
([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)); this verb
and `version --check` are the only two **commands** that reach the release origin,
each only when invoked
([itd-130](../../intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md),
spc-32).

They are not the only code paths to that origin. `hooks/bootstrap.sh` pins the
same repository, resolves its latest release over the network, and downloads the
checksum-verified asset — and the hook configuration fires it whenever the plugin
root holds no binary, so it reaches the origin without the user naming a network
action in that moment. adr-38 admits it as a tier of its own: provisioning
completes a chosen update, it never discovers one.

`version --check` hands over to this verb: when an update is available, its
`next:` line names the command to type, chosen by the same on-disk classification
this verb dispatches on ([`12-version.md`](12-version.md)).

## Behaviour

```bash
abcd update [tag] [--yes] [--json]
```

The dispatch is keyed on what actually runs: the first `abcd` on `PATH`. Only a
regular file proven to be abcd's own is ever swapped, and there are three proofs,
tried in order:

| Proof | What establishes it |
|---|---|
| `release-manifest` | the file's digest appears in a published release's checksums. The strongest, and the only one that also DATES the file, so it is tried first |
| `running-executable` | the file IS the executable this process runs from. Nothing else can be, so no forge object is consulted |
| `path-entry-record` | `~/.abcd/path-entry` records this exact file as the copy abcd installed, and the bytes still hash to what was recorded |

The last two exist because the first one dies with the release object
(iss-2609012000222546). Release assets are deletable, and an ownership proof
resting on them stops proving the day they are deleted, stranding an install with
no way forward. Both replacements are independent of what the forge still serves —
one is a property of the running process, the other a claim abcd wrote on this
machine — so a deleted release costs the receipt its vintage and never its
ownership.

Everything else on `PATH` is a loud refusal naming its remedy rather than a swap:
a plugin-root binary belongs to the plugin update, a Homebrew-resolved install to
`brew upgrade`, a stranded owned entry to `ahoy install`, an owned pin into a
superseded plugin vintage to that same `ahoy install`, a track-latest dev shim
to a mode switch first, and a foreign occupant to whoever put it there, its
remedy asking for that occupant to be removed or renamed. Two more answer the
cases where there is nothing to act on at all: no `abcd` anywhere on `PATH`, and
an entry whose install shape abcd cannot classify, which fails closed rather than
fetching.

One further refusal arrives after the dispatch, once the release checksums are in
hand: a regular file that none of the three proofs vouches for. That one
deliberately does not ask for a deletion — removing the file would destroy the
only tool able to fetch a replacement — and names the reinstall route that
overwrites the entry in place instead.

A tag abcd resolved rather than one the caller typed is confirmed before the
fetch, unless `--yes` is passed. The question is only put where somebody is there
to answer it: the input the answer is read from and the stream the question is
written to both have to be a terminal, so a hooked or scripted run is never left
blocking on a read. Under the invocation the plugin command issues (`abcd update
--yes --json`) nothing is emitted until the receipt, so there the resolved tag is
first named in the receipt itself.

The transport is pinned: no proxy or CA overrides from the environment (set ones
are ignored and named in the receipt), redirects only onto the release origin's
own hosts, every hop re-checked. The swap is atomic in the target's directory, so
a failed download or verification leaves no partial file.

The target file is not quite the only thing the verb touches. Where this machine's
install record names the very entry just refreshed, abcd re-stamps that record
with the digest the release checksums proved, so the file it updated does not read
as somebody else's binary the next time an install shape is judged. The re-stamp
runs on an already-current outcome too, and it does nothing at all where no record
names that path.

## The receipt

Three terminal outcomes ship, and the receipt's `action` field names which one
happened:

| `action` | What it means |
|---|---|
| `swapped` | the file was replaced, and the render reads `updated <path>: <old> -> <tag>` |
| `already-current` | the target's digest already equals the release's, so the binary is left untouched |
| `refused` | a dispatch or ownership refusal, naming its shape and its remedy |

On a run that reached the release origin, the receipt carries beside `action` the
origin, the tag, the asset and its digest, the target path (redacted to `~`), and
the ownership proof that allowed the swap. It carries `env_ignored` when proxy or
CA overrides were scrubbed. A refusal receipt is deliberately thinner: a refusal
raised before any fetch carries the target path and a block naming shape, detail
and remedy, and nothing else, because there is no release it could name.

An old version number is only derivable when a release manifest dated the file it
replaced. A file swapped under either local proof has no published release naming
those bytes, so the receipt reports the old digest instead and reads `updated
<path>: an unpublished build -> <tag>`. An absent old version is therefore the
documented shape, never a broken receipt.

## References

- Plugin command: [`commands/update.md`](../../../../commands/update.md)
- Intent / spec: [itd-130](../../intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md) / [spc-32](../../specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md)
- Staleness check it completes: [`12-version.md`](12-version.md)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails the build when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd update`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--yes` | bool |

<!-- surface-appendix:end -->
