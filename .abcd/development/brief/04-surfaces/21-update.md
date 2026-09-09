# `/abcd:update` — Complete a Chosen Update

`/abcd:update` completes what `version --check` reports: it fetches the named
release (or resolves the latest, naming the tag before acting), verifies the
platform binary against the same release's `checksums.txt`, and swaps the
PATH-installed copy atomically. The verb is the explicit ask — abcd never
checks for or applies updates on its own
([adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md)); this
verb and `version --check` are the only two paths to the release origin, each
only when invoked ([itd-130](../../intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md), spc-32).

`version --check` hands over to this verb: when an update is available, its
`next:` line names the command to type, chosen by the same on-disk
classification this verb dispatches on ([`12-version.md`](12-version.md)).

## Behaviour

```bash
abcd update [tag] [--yes] [--json]
```

The dispatch is keyed on what actually runs — the first `abcd` PATH occupant,
classified by the same ownership predicate detection and install use. Only a
regular file proven to be abcd's own is ever swapped, and there are **three
proofs**, tried in that order:

| Proof | What establishes it |
|---|---|
| `release-manifest` | the file's digest appears in a published release's `checksums.txt`. The strongest, and the only one that also DATES the file, so it is tried first |
| `running-executable` | the file IS the executable this process runs from. Nothing else can be — the code asking the question was loaded out of those very bytes — so no forge object is consulted |
| `path-entry-record` | `~/.abcd/path-entry` records this exact file as the copy abcd installed, and the bytes still hash to what was recorded. Written at install time by every install route |

The last two exist because the first one dies with the release object
(iss-2609012000222546): release assets are deletable, and an ownership proof
resting on them stops proving the day they are deleted, stranding an install
with no way forward. Both replacements are independent of what the forge still
serves — one is a property of the running process, the other a claim abcd wrote
on this machine — so a deleted release costs the receipt its VINTAGE and never
its ownership. `abcd update --help` states all three.

Every other shape is a loud refusal naming its remedy: a plugin-root binary
(the plugin update owns it — itd-108's one-cut coherence), the track-latest dev
shim, a stranded owned entry (`ahoy install` heals it), a Homebrew
Cellar-resolved install (`brew upgrade abcd`), a foreign occupant, or an empty
PATH.

The transport is pinned: no proxy or CA overrides from the environment (set
ones are ignored and named in the receipt), redirects only onto the release
origin's own hosts, every hop re-checked under the urlguard policy. The swap
is atomic in the target's directory; a failed download or verification
leaves no partial file. Progress renders on a TTY only; the receipt (origin,
tag, digest, old→new) prints in both modes.

## References

- Plugin command: [`commands/update.md`](../../../../commands/update.md)
- Intent / spec: [itd-130](../../intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md) / [spc-32](../../specs/closed/spc-32-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md)
- Staleness check it completes: [`12-version.md`](12-version.md)
