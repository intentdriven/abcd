---
id: itd-2609292108089653
slug: ingesting-a-source-and-consulting-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-76]
related_issues: [iss-27, iss-55]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Ingesting and consulting sources are abcd verbs, and no corpus entry is a keyword stub

Typed links: `related_issues` [iss-27](../../../work/issues/resolved/iss-27-corpus-tooling-to-go-core.md) (sources tooling into the Go core; resolved by itd-76 for the corpus contract, and the ruling plans what remains) and [iss-55](../../../work/issues/open/iss-55-corpus-stub-backfill.md) (the placeholder entries to backfill); `builds_on` [itd-76](../shipped/itd-76-source-provenance-ledger.md) (the `abcd source` verb family this extends).

## Press Release

> Ingesting a source and consulting the sources corpus are abcd verbs, and every entry the corpus holds carries the text it claims to hold. The deterministic half of ingest and consult moves out of the command pages into the binary: a search over the corpus that answers by key, a text-quality check that refuses a keyword stub whose entry claims a fetchable source, and the backfill of the placeholder entries registered before the text was stored. "I asked the corpus a question and got back six entries that were only keywords," said Maya, a researcher-developer. "Now a stub is refused at ingest, the old ones carry their text, and the search is the same every time I run it."

## Why This Matters

itd-76 moved the corpus contract into the binary as the `abcd source` verb
family (init, add, ledger, declassify, sync-banlist, cite-check), and iss-27 was
resolved on that delivery. Ingest and consult themselves are still command pages
that an agent follows by hand: the page says "abcd fetches and converts
nothing", consulting is "plain search", and the text-quality check is a step in
prose. Six public entries (iss-55) are keyword stubs with no document text,
which yielded nothing when the corpus was searched, so the corpus claims
coverage it cannot deliver. The product thinker ruled on 2026-09-29 to plan the
sources tooling into abcd's core with ingest and consult as abcd verbs (ruling
J5), with the backfill of the placeholder entries as part of the same intent
(ruling J6).

## What's In Scope

- **A consult verb**: a read-only search over the corpus that answers by key and
  class, under the confidentiality hard rule (a confidential entry is named by
  key only).
- **The ingest quality check as a verb**: a stored text that is only keywords,
  for an entry that claims a fetchable source, is refused at `source add` and
  reported by a corpus lint.
- **The backfill** of the six stub entries iss-55 names, and the on-disk PDF
  attached to the research-integrity paper entry, through the ingest path.

## What's Out of Scope

- A pluggable retrieval back end (iss-26, ruled "decide later" the same day).
- Fetching and converting inside the binary, unless the planning interview
  decides it (see Open Questions).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from the two records and the ruling, unconfirmed: the
planning interview walks each one.

- **Given** a corpus, **when** the consult verb runs with a query, **then** it
  lists the matching entries by key and class, names a confidential entry by key
  only, and writes nothing.
- **Given** a text that is only keywords, for an entry that claims a fetchable
  URL, **when** `source add` is given it, **then** the add is refused, naming the
  entry.
- **Given** the corpus after the backfill, **when** the corpus lint runs,
  **then** it reports no stub entry, and a consult over the six entries iss-55
  names returns document text rather than keywords.

## Open Questions

- **Does ingest fetch and convert in the binary**, or does the agent keep the
  fetch and the conversion while the binary does everything deterministic? The
  ingest page states "abcd fetches and converts nothing" today, and adr-38 bars
  an implicit fetch.
- **The consult verb's name**: a sub-verb of `abcd source`, or its own top-level
  verb beside the `/abcd:consult` page.
- **The backfill's home**: the six entries live in a person's own corpus, not in
  this repository, so the backfill may be a documented act rather than a
  criterion the build can meet.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
