---
id: itd-2609292106557115
slug: a-fresh-machine-installs-abcd-with-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-130]
related_issues: [iss-377]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A fresh machine installs abcd through one small starter that hands over to a trusted abcd binary

Typed links: `related_issues` [iss-377](../../../work/issues/open/iss-377-bootstrap-trampoline-demotion.md) (the demotion record this draft plans); `builds_on` [itd-130](../shipped/itd-130-abcd-update-completes-a-chosen-update-in.md) (the update verb the starter hands over to).

## Press Release

> A fresh machine installs abcd with one small starter script that hands over to the abcd binary the moment one it trusts exists. The session-start bootstrap, the one-line installer and the update verb stop being three downloaders: the starter fetches a binary only when no abcd exists anywhere on the machine, checks that any binary it hands over to belongs to the person running it, and falls back to its own fetch when the hand-over fails. "I used to wonder which of three scripts had put abcd on my laptop," said Frank, a DevOps engineer. "Now there is one path that fetches, and the starter only runs on a machine that has nothing yet."

## Why This Matters

abcd reaches a machine through three downloaders: the session-start hook
`hooks/bootstrap.sh`, the one-line installer the site serves from
`site-src/install.sh.tmpl`, and the `abcd update` verb itd-130 shipped. Each
fetches and verifies a release on its own, so a fix to one (a checksum rule, the
release-asset layout, an ownership check) has to be made three times, and a
person cannot tell which of them put the binary they are running in place.
iss-377 asked for the demotion once `abcd update` shipped; that trigger is met.
The product thinker ruled on 2026-09-29 (ruling J2) to plan it: the installer
becomes a minimal starter with an ownership-checked hand-over.

## What's In Scope

- **One starter** that fetches a binary only when no abcd binary is resolvable
  on the machine: the cold start, the one state a Go updater cannot serve.
- **An ownership-checked hand-over**: when a binary is resolvable, the starter
  delegates provisioning to it (`abcd update` or its successor), and only to a
  binary that passes an ownership check.
- **A fallback**: when the hand-over fails, the starter's own fetch still runs,
  loudly.
- **A frozen release-asset layout**, written down as a contract, so an older
  binary provisioning a newer plugin root keeps working across cuts (carried
  from itd-130's Decisions).

## What's Out of Scope

- Distribution channels beyond the release assets (a Homebrew tap is iss-380,
  ruled "decide later" the same day).
- The curated plugin payload itd-108 drafts: the starter provisions the binary,
  not the plugin surface.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from iss-377 and the ruling, unconfirmed: the planning
interview walks each one.

- **Given** a machine with no abcd binary anywhere, **when** the starter runs,
  **then** it fetches and verifies the release binary itself and says so.
- **Given** a machine where an abcd binary owned by the person running the
  starter is resolvable, **when** the starter runs, **then** it hands
  provisioning to that binary and performs no download of its own.
- **Given** a resolvable abcd binary the person running the starter does not
  own, **when** the starter runs, **then** it refuses the hand-over, names the
  binary and its owner, and falls back to its own fetch.
- **Given** a hand-over that fails, **when** the starter runs, **then** it falls
  back to its own fetch and says the hand-over failed.
- **Given** the release-asset layout contract, **when** a cut changes an asset
  name or path the contract names, **then** the release gate refuses the cut.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).

- **The field has settled on installer + receipt + self-update that refuses
  installs it did not make.** uv's standalone installer is the only route with
  `uv self update` ("will re-run the installer"); every other route names its
  package manager's upgrade, and `UV_UNMANAGED_INSTALL` disables self-update.
  cargo-dist's updater reads an install receipt in `~/.config/<app>` (Linux,
  macOS) or `%LOCALAPPDATA%\<app>` (Windows) before any check. abcd already
  holds the same device: `path-entry`, `release-mark`, and the three ownership
  proofs in the update package.
- **Checksums travel with the script, never from the vendor's domain.**
  mise: "Checksums never come from mise.jdx.dev"; the script embeds them and
  stops "if it cannot get that checksum or if the archive does not match it".
  `install.sh.tmpl` already reads `checksums.txt` from the same release and
  runs `gh attestation verify` when `gh` is present, one step past the field.
- **Ownership checks in the wild take both halves, one precedent each.**
  Homebrew's installer is the uid-and-directory precedent: it refuses root
  and inspects each target directory's owner before queuing `chown`. The
  harness's native installer is the "did I make it" precedent: it leaves a
  launcher it did not create in place, keeps installing new versions beside
  it, and reports it in its doctor output rather than overwriting it, a
  provenance rule closer to the receipt-and-digest half than to a uid check
  (the harness's setup page).
- **A minimum release age is a cheap defence against a bad cut**: mise skips
  releases younger than 24 hours by default; the harness offers a `stable`
  channel about a week behind.
- **The `curl | sh` objections have specific answers**: wrap everything in
  functions so a truncated download runs nothing (which `install.sh.tmpl`
  does), and do not trust a browser view of the script, since a server can
  serve curl a different file (a 2012 proof of concept).
- **The swap primitive is understood**: minio/selfupdate (already a
  dependency) writes `.new`, renames the running file to `.old`, and restores
  it if the final rename fails; on Windows the `.old` file is hidden, not
  removed.

**Path 2 — native starter shaped to the cargo-dist and uv installer-plus-receipt
pattern, with the receipt as the seam.** One starter rendered to both surfaces
(open question 1, as `install.sh.tmpl` is already rendered to `/install.sh`);
"owned" means both halves (open question 2): `path-entry` records the file's
digest and the file and its directory belong to the invoking uid; the
asset-layout contract is a Go test over `release.yml` plus one shared constant,
the way `TestBootstrapUpdateLineIsTheSharedWording` already binds the script
to the core (open question 3). No new dependency. The independent
fit-challenge runs at plan time.

## Open Questions

- **Which surfaces become the starter**: the session-start hook and the site's
  one-line installer both, or one starter that the other calls?
- **What "owned" means on the hand-over**: the binary's file owner, its
  directory's owner, or the recorded hash itd-130's store keeps?
- **Where the asset-layout contract lives**: a brief chapter the release gate
  reads, or a test over the release workflow.
- (SOTA 2026-10-23) Does the starter adopt a minimum release age for the
  cold-start fetch, as mise does, or does the release mark's "newest" stay the
  only notion of latest? Offline either way; a second notion of latest is the
  cost.
- (SOTA 2026-10-23) Does the cold-start fetch resolve the tag from the pinned
  catalog or the release mark rather than `releases/latest`, which the forge
  picks by commit date and so skips a hotfix cut from an older commit?
- (SOTA 2026-10-23) On a Windows machine there is no uid to check; does the
  hand-over there rest on the receipt and digest alone, and does the record
  say so?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
