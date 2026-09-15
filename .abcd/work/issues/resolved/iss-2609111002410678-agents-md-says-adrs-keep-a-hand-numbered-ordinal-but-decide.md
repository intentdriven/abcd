---
schema_version: 1
id: "iss-2609111002410678"
slug: "agents-md-says-adrs-keep-a-hand-numbered-ordinal-but-decide"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "itd-84 decomposition of the lifecycle-symmetry proposal"
origin: researcher-authored
production_mode: hand-written
found_at: "CLAUDE.md"
resolution: "AGENTS.md no longer carves ADRs out of the record-id rule. The Concurrent sessions bullet now states that ADRs mint through that same seam, that abcd decide allocates the timestamp-numeric adr stamp and files it under that stamp, and that the ordinals 0001-0058 keep their ids and their filenames while every reader admits both vintages through one derivation, so no record family needs a word first. The prose is pinned to the code rather than to another page of prose: TestTheRootRouterDoesNotSendAnAuthorToCoordinateAnADRMint in internal/core/decide mints a stamp in a fixture already holding the ordinals and then refuses a router bullet that still carries the stale sentences. The corpus sweep this record asked for finds no other committed surface asserting the exception: the decide brief chapter, the ADR store charter and the CLI command reference all describe the mint with the ordinals grandfathered. Same defect as iss-2609090636110810, which carried the correction; this record was filed independently out of the itd-84 decomposition and is closed on that fix."
impact: internal
resolved_by:
  commit: "f8541182ddbce6b5314c2af1a7f17949113d15bf"
---

`CLAUDE.md` tells every session that ADRs are the one record family still
hand-numbered. They are not, and have not been since `decide` adopted the mint.

## The claim

Under *Concurrent sessions*, on record ids:

> ADRs keep their hand-numbered filename ordinal, so an ADR is the one record
> family where minting from two checkouts still needs a word first.

## The reality

`core/decide` mints an `adr-<stamp>` through `core/recordid` like every other
family. `internal/README.md:104-112` records the adoption and its date: decide
"is the last record family to reach that seam (the 2026-09-01 ruling, the turn
adr-45 ruling 3 deferred), and it adopts it the way every family before it did
— by holding a `recordid.Minter` and naming its family tag, never by carrying an
allocator of its own."

`internal/core/decide/decide.go:48` holds that `recordid.Minter`. Nine ADRs in
`.abcd/development/decisions/adrs/` already carry minted ids, the newest being
`2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md`
— which `CLAUDE.md` itself cites, two sections above the claim that says such an
id cannot exist.

## Why this is major rather than cosmetic

It is not prose in a chapter somebody may read. It is a rule in the file the
rules loader puts in front of every session, and it tells an agent that a
coordination step is required where none is. The predicted failures are ordinary:
an agent pauses to ask before minting an ADR, or hand-numbers one to obey the
rule and lands a `00NN-` filename beside the minted ones, deepening the very
split the mint was adopted to end.

The sequential ids that remain are not counter-evidence. Minting is forward-only
and nothing renumbers an existing record, because an id is a citation: 50
sequential ADRs and 9 minted ones is what a family mid-adoption looks like, not
a family that hand-numbers.

## Scope of the correction

The rule's surrounding sentence is right and should stay: ids need no
coordination between checkouts because the allocator reads no maximum (adr-45).
What must go is the ADR exception carved out of it. Check the same claim
wherever else it was copied — `.abcd/development/brief/` and the `decide`
surface chapter are the likely carriers, since the exception was true when they
were written.

## Acceptance

- **Given** a session reading the rules, **when** it reaches the record-id rule,
  **then** no family is named as an exception to minting.
- **Given** the corpus, **when** the claim is checked, **then** no committed
  surface states that ADRs are hand-numbered.

## Grounds

- pursued: the router is the surface an agent reads first, so the correction holds only while the test that mints and then reads the bullet stays armed; it would be shown wrong if a rewrite restated the exception in words the containment check does not carry
