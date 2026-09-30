---
id: itd-2609292106557115
slug: a-fresh-machine-installs-abcd-with-one-small-starter-script
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

Typed links: `related_issues` [iss-377](../../../work/issues/open/iss-377-bootstrap-trampoline-demotion.md) (the demotion record this draft plans); `builds_on` [itd-130](../shipped/itd-130-abcd-update-completes-a-chosen-update-in-one-verb-it-fetches.md) (the update verb the starter hands over to).

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

## Open Questions

- **Which surfaces become the starter**: the session-start hook and the site's
  one-line installer both, or one starter that the other calls?
- **What "owned" means on the hand-over**: the binary's file owner, its
  directory's owner, or the recorded hash itd-130's store keeps?
- **Where the asset-layout contract lives**: a brief chapter the release gate
  reads, or a test over the release workflow.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
