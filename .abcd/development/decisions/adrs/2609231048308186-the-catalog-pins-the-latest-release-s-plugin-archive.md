---
id: adr-2609231048308186
slug: the-catalog-pins-the-latest-release-s-plugin-archive
status: accepted
date: 2026-09-23
supersedes: null
superseded_by: null
related_intents: [itd-67, itd-108]
related_rfcs: []
related_adrs: [adr-19, adr-20, adr-28]
---

# ADR-2609231048308186: The catalog pins the latest release's plugin archive

## Context

[itd-67](../../intents/shipped/itd-67-installable-versioned-plugin.md) promised
that every release publishes a version-stamped plugin and that a plugin update
pulls it. What shipped stamped the version only into a payload staged outside the
repository behind `launch ship --payload-dir`, which no workflow published, while
the catalog (`.claude-plugin/marketplace.json`) sourced the plugin from `./` — the
unversioned git tree. An install therefore took whatever `main` held, with no
version and no fingerprint (iss-2609230733497536, found by the itd-67 fidelity
audit).

[adr-19](0019-plugin-json-version-carve-out.md) keeps every version out of the
working tree, and [adr-20](0020-manifest-version-lockstep.md) pins the catalog's
version keys ABSENT in the source view. Both were written for a catalog that
points into the tree. The host harness reads a published release only through an
`archive` source (`{"source": "archive", "url": …, "sha256": …}`, Claude Code
v2.1.224 or later), so every way of making "an install at vX gets vX's stamped
payload" true had to change what the committed catalog says.

The product thinker ruled on 2026-09-23 (E1, 09:50Z): a **pinned** archive,
because "the user should get the latest cut release from github; verified as
such and fingerprinted through a hash"; and (E2, 09:55Z) this amendment to
adr-19 and adr-20. The harness floor was accepted and is documented (E3,
10:13Z); contributors load the plugin from their own checkout, with no second
catalog entry (E4, 10:23Z).

## Decision

We will source the plugin from the latest release's pinned archive, and this
decision amends adr-19 and adr-20 accordingly.

1. **What main's catalog carries.** The plugin listing's `source` is
   `{"source": "archive", "url": "<repository>/releases/download/v<version>/<plugin>-plugin-v<version>.zip", "sha256": "<digest>"}`
   for the newest dated release. The address derives from `plugin.json`'s
   `repository`. The catalog still carries no version key, and `plugin.json`'s
   version stays out of the working tree: adr-19's polarity and adr-20's source
   view hold unchanged for every version key. What changes is that the catalog
   is no longer release-neutral — it names one release by address and digest,
   and the ship rewrites it every cut.
2. **Who writes it.** `launch ship` renders the release's archive from its tree
   after writing the dated heading, and rewrites the listing's source (and the
   surface snapshot beside it) in the same working tree, so the pin is part of
   the release-content commit. It refuses beforehand on a payload with
   uncommitted changes, whose archive the tagged commit could not reproduce.
3. **Where it pins.** The ship pins only in a repository whose version-location
   contract (`.abcd/config/version-location.json`) carries
   `"publishes_plugin_archive": true`, the statement that its release uploads the
   archive. Without that declaration it leaves the catalog untouched and reports
   `archive: not pinned` (`archive_unpinned` in `--json`): the contract alone
   says where the version lives, not that a release publishes an archive, and a
   managed repository's scaffolded workflows upload none, so a pin there would
   name an asset every install fails to fetch. A key that is not a boolean is
   refused rather than read as either answer.
4. **What proves it.** The archive is reproducible: sorted entries, stored
   uncompressed, one fixed timestamp, modes normalised to 0644 or 0755, and the
   catalog left out (it names the digest, so it cannot be inside what it
   hashes). `release.yml` re-renders it from the tagged commit with
   `abcd launch archive --tag <tag> --verify --repository <owner/name>` in
   `verify`, before anything is built, and again in the publish job, and publishes nothing unless the digest
   is the pinned one. The published archive joins `checksums.txt`, the
   build-provenance attestation and the upload, and is downloaded fresh after
   publication, attestation-verified and byte-compared.
5. **The bootstrap.** Releases up to and including v0.9.0 were published without
   an archive. While the newest dated release is one of those, the catalog keeps
   `./`; the first ship past it writes the pin, and
   `TestCommittedTreeSatisfiesDevPolarity` refuses an unpinned catalog from then
   on.

## Alternatives Considered

- **A′ — an unpinned archive at `releases/latest/download/`.** No per-release
  catalog edit, but a replaced asset installs silently: the product thinker's
  "fingerprinted through a hash" is exactly what it gives up. This is also the
  form draft [itd-108](../../intents/drafts/itd-108-the-plugin-installs-from-the-curated-release-artifact-not-th.md)
  chose in its 2026-08-11 grill; the ruling overtakes that choice.
- **B — tag a stamped release commit off `main`.** Keeps the tree unversioned,
  but only `@vX` installs see the stamp, tags leave `main` (breaking the
  changelog anchor and the RS002/RS003 reachability checks), and the tag commit
  would be bot-authored, which the attribution gate refuses.
- **C — commit the stamped version into `main`.** adr-19's rejected alternative:
  between releases, `main`'s HEAD would be labelled with a version it is not.
- **D — publish the archive only, and amend the criterion's "update pulls it"
  half.** Honest, and it leaves the finding unbuilt.
- **A — the pinned archive (chosen).** The harness itself refuses a tampered
  asset, and the release refuses a pin it cannot reproduce.

## Consequences

- An install or update receives a cut release, stamped with its version,
  fingerprinted by the digest the harness enforces. Installs stop following
  `main`'s tip.
- **The window.** From the ship's merge until the publish job uploads the
  archive, the catalog names a zip that is not there yet. An install or update in
  that window fails closed and leaves an installed plugin on its previous
  release, and nothing else can install because the digest refuses other bytes.
  It cannot be closed from the release side (the pin must be in the tagged tree),
  so it is documented in `release.yml` and the release-day guide and kept short
  by approving the release deployment promptly.
- **A pin the release cannot reproduce is refused before the tag** on the
  auto-release path: `auto-release.yml`'s `detect` job renders the archive from
  the pushed commit and verifies the pin, and that the pinned address is the
  repository's own release, before its `tag` job may run. The pushed commit is
  not always the one the ship rendered — a merge-queue batch carries the ship
  with any other queued pull request — so the ship's dirty-tree refusal cannot
  stand in for it. A refusal there tags nothing; a follow-up pull request that
  re-pins or reverts retries on its push. `verify` repeats the proof after the
  tag, where a refusal consumes the version: that is the only proof a
  hand-pushed tag gets, and a local
  `abcd launch archive --tag <tag> --verify --repository <owner/name>` on the
  release branch is what catches it first. The `--repository` is not optional
  there: without it a pin whose address names another repository (a fork, a
  rename, a transfer) passes locally and is refused after the tag, consuming the
  version.
- **The harness floor** is v2.1.224; older harnesses cannot install the plugin,
  and very old ones fail to load the catalog. The install instructions and the
  release notes state it.
- The stamped marketplace version and changelog entry (adr-20 R2/R3) live in the
  staged payload, where the render's lockstep proves them, and not in the
  published archive, which leaves the catalog out.
- The same-release trust root remains: the digest is committed by the ship and
  checked against a render of the same tree, so it proves integrity from commit
  to user, not the commit's own honesty — branch protection and the identity
  gate bound that, as they bound everything else the release publishes.
