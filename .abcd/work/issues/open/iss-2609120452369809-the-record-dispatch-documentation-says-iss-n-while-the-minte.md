---
schema_version: 1
id: "iss-2609120452369809"
slug: "the-record-dispatch-documentation-says-iss-n-while-the-minte"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "peer session report from a downstream repo, 2026-09-12"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/abcd.md"
---

Reported from a downstream repository using abcd: `abcd capture "<text>"` mints
ids like `iss-2609120433096312` while every pre-existing record there uses the
short form (`iss-188`, `iss-227`). That ledger now holds two id shapes — about
48 records in the old shape and 8 in the new — and the record-dispatch
documentation says `iss-N`.

## What is and is not wrong here

**The two shapes are not a defect.** Minting is forward-only by design and
nothing renumbers an existing record, because an id is a citation. A ledger
part-way through adoption holds both shapes and that is what adoption looks
like. This repository is in the same state for every family: 301 sequential ids
against 35 minted across intents, specs and ADRs, and the issue store is
overwhelmingly minted.

**The documentation is wrong**, and that is the reportable part. Surfaces that
spell the id as `iss-N` tell a reader the short form is the form, when what they
will actually be handed is a sixteen-digit stamp. A downstream adopter reads the
spelling as a contract.

This is the same class as `iss-2609111002410678` — `AGENTS.md` asserting ADRs
keep a hand-numbered ordinal when `decide` has minted them since the 2026-09-01
ruling. Both are surfaces describing the pre-mint world. They should be swept
together rather than one at a time, since the sweep is the same grep and the
same judgement about which `<family>-N` spellings are illustrative and which are
claims.

## The care the sweep needs

`iss-N` is not uniformly wrong. It is the right spelling in a **placeholder**
(`abcd capture resolve <iss-N> …`), where `N` stands for whatever the id is, and
the wrong spelling in a **claim** about what ids look like. A blanket
search-and-replace would damage the first to fix the second.

The honest fix is to say the shape once, where a reader can find it, and keep
placeholders as placeholders: an id is `iss-`, `itd-`, `spc-` or `adr-` followed
by digits, either a short ordinal from before the mint or a sixteen-digit stamp
from after it, and both resolve.

## Acceptance

- **Given** a reader of the record-dispatch surface, **when** they look up what
  a record id is, **then** they are told both shapes resolve and neither is
  being migrated away.
- **Given** the surfaces that spell an id, **when** the sweep runs, **then**
  placeholders keep their `<iss-N>` form and claims about the shape are
  corrected.
