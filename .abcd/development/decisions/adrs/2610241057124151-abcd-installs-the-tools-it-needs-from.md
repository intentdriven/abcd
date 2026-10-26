---
id: adr-2610241057124151
slug: abcd-installs-the-tools-it-needs-from
status: accepted
date: 2026-10-24
supersedes: null
superseded_by: null
related_intents: [itd-2610240918594138]
related_rfcs: []
related_adrs: [adr-38, adr-2609151706587280]
---

# ADR-2610241057124151: abcd installs the tools it needs from downloads pinned in its own release

## Context

Setup offered to install gitleaks and gh through Homebrew, one fixed argv per
platform (`brew install <formula>`). On an account that cannot write to the
machine's Homebrew, the step failed on every run and passed on Homebrew's own
advice to change ownership with admin rights (iss-2610240824571059), although
abcd never escalates privileges. A supported Homebrew for one non-admin account
cannot be made by a tool on a Mac: the installer refuses a prefix longer than
`/opt/homebrew`. The product thinker therefore decided that every tool abcd
installs, on macOS and Linux alike, is a checked download of the tool's
official release into the person's own folder, the route abcd already uses for
itself (dcn-2610240845548776, which this record graduates).

Three constraints were already locked. Brief invariant 7: "provisioning fetches
only a manifest-named, pinned, checksum-verified artifact. No version-discovery
request exists anywhere in abcd" ([adr-38](0038-implicit-checks-are-disk-only.md)).
Brief invariant 12 and [adr-2609151706587280](2609151706587280-the-cache-reaches-path-only-through-a.md):
ownership of a file abcd placed is recorded provenance, never content-guessing.
And internal/core/update's rationale for its own binary: a checksums file
published beside an asset proves nothing alone, since whoever can replace one
can replace both.

## Decision

We will install a tool abcd needs only from a release table compiled into abcd
(`internal/core/tools/release.go`): per tool, the pinned version, and per
platform (macOS and Linux on arm64 and amd64) the asset's full GitHub release
download URL, its sha256 copied from the makers' published checksums file and
checked against the asset when it was pinned, and the program's path inside
the archive; for a tool whose makers sign SLSA build provenance, the exact
signer (repository, workflow file and ref); and the range of the tool's
versions abcd works with. A tool updates only when abcd does.

The trust boundary is `tools.Installer.Install`. It downloads, over https and
from GitHub's release hosts alone, bounded in size and time, with the proxy and
CA overrides ignored as the updater ignores them; it refuses the bytes unless
their sha256 equals the pin; for a signed tool it verifies the build record in
process with sigstore-go against the pinned signer, under GitHub Actions'
issuer, from the signer's repository at its ref on a GitHub-hosted runner; it
reads the one archive member by its exact name. Only after every check passes
is anything written: atomically, at the account's own rights, into
`$XDG_BIN_HOME`, else `$XDG_DATA_HOME/../bin`, else `~/.local/bin`, and never
into the repository the verb ran from. The written copy must report the pinned
version, or it is removed. It is then recorded, path and digest, in
`~/.abcd.noindex/tools.json`. abcd runs a recorded copy by its path whatever
comes first on PATH, and uninstall removes it, each only while its bytes still
hash to the record. A copy installed some other way is used when its version
lies in the compatible range, never changed, and never overwritten: when the
bin folder already holds one of that name, nothing is installed. A tool that
publishes no signed build record (gitleaks) is checked by checksum only, and
every explanation and result says so. Nothing abcd shows or runs needs admin
rights, and nothing is installed in CI.

The provenance check does not bind the record to the release tag's commit, as
abcd's own update does: that binding exists there because the digest is learnt
from the release page, so an older genuine build could be replayed under a
newer tag. Here the digest is fixed in abcd's source, and no replay changes the
bytes installed.

## Alternatives Considered

- **Keep Homebrew, add a fallback.** Rejected: Homebrew stays the default only
  where it works, which is exactly the admin-owned prefix the person cannot
  write, and its failure text relays the privilege escalation abcd refuses.
- **Resolve the latest release at install time and check the release's own
  checksums file.** Rejected: it is a version-discovery request (invariant 7),
  and a same-page checksums file proves nothing against whoever can edit the
  release page.
- **Run `gh attestation verify` for the provenance.** Rejected: it needs gh
  installed and signed in before gh itself can be installed, and abcd already
  verifies Sigstore bundles in process for its own binary.
- **Pinned downloads with the in-process check (chosen).** It needs no package
  manager and no admin rights, holds to invariant 7, and reuses the one
  verifier abcd already ships.

## Consequences

- Each abcd release carries a tool's version and digests; moving them is a
  change to `release.go` that re-copies the digests from the makers'
  checksums files and checks them against the assets. The compatible range is
  the seam a release-time check can widen or move (iss-2610240959030370,
  still open).
- A copy installed some other way at an older version is no longer counted as
  installed: setup names it, says why, and offers abcd's own copy beside it.
- GitHub sometimes hands out a build record only by a `bundle_url` into its
  attestation store, snappy-compressed; abcd decodes that format itself rather
  than taking a dependency, and every decoded byte is then verified by
  Sigstore.
- The download transport is a second copy of the updater's policy, because
  update imports ahoy, which imports the tools package. The two are held to
  the same rules by their tests rather than by one function.
- A test binary's default installer has no network, so no test of another
  package can fetch a real release by reaching a yes.
