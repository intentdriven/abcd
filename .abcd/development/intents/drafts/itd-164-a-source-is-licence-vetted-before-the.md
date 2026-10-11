---
id: itd-164
slug: a-source-is-licence-vetted-before-the
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-163]
severity: minor
impact: additive
---

# A source is licence-vetted before the record may lean on it: cite refresh records the verdict, the gate reads the committed baseline

## Press Release

> **No source enters abcd's record without its licence having been looked at: `abcd docs cite refresh` extends to fetch each declared source and record a licence verdict — an SPDX identifier, or an explicit *unknown* with the reason — into the committed citation baseline, and the zero-network gate refuses a new reference or influence whose baseline entry carries no verdict.** The split is the one the citation family already lives by: the network work happens in an explicit verb whose output is a reviewable committed record, and the gate that runs on every push reads only that record. An unreachable source or an unlicensed page is not a silent pass and not a hard stop; it is a loud, first-class *unknown* that the entry must carry as a caveat, the way the record already annotates a source whose advisory host had vanished.

> "Attribution archaeology is the expensive version of a question that is nearly free at admission time," said Dave, a compliance lead who signs off on what the project redistributes. "When a source arrives I can see its licence in thirty seconds. Two years later I may not be able to find the source at all — we have one credit in this record whose archive is already unreachable. I want the thirty-second check made mandatory while it is still cheap, and I want the verdict written down where a reviewer sees it."

## Why This Matters

Of the four elements good attribution carries — title, author, source, licence — the 2026-08-28 review found licence to be the one this record handles weakest: a CC-BY-SA framework credited without its licence named, an adapted archive recorded as "MIT-spirit", an adopted gist with no licence at all, and one influence whose source is now unreachable so its licence can never be verified. None of these rose to a violation, but every one of them is a question that was cheap on the day of adoption and is costly or unanswerable now.

The deterministic half of the machinery (itd-163) proves the declared sets are closed and mirrored; it deliberately says nothing about whether a licence field is filled truthfully. This intent adds the vetting: the moment a source is admitted, its licence is checked at the URL the entry itself cites, and the verdict — including the honest failure — becomes part of the committed record a reviewer can challenge.

## What's In Scope

- A licence verdict per source in the committed citation baseline: an SPDX identifier or the explicit unknown form, plus the URL consulted and the date checked.
- `abcd docs cite refresh` as the sole network path: the refresh fetches each declared source, extracts or fails to extract a licence, and writes the verdict to the baseline for review.
- A gate rule in the citation family: a new CSL or influence-registry entry whose baseline entry carries no licence verdict refuses; existing entries are grandfathered until the licence backfill lands.
- Loud staging: an unknown verdict renders as unknown wherever the entry renders, and an entry resting on an unknown must carry an in-entry caveat naming why.
- Staleness: licence verdicts age under the same warn and block windows the citation baseline already applies to link health.

## What's Out of Scope

- **Licence compatibility analysis or legal advice.** The gate records what a source declares; whether that licence permits a given use stays a human judgement.
- **Detecting licences the source does not declare.** The refresh consults the cited URL and obvious adjacent conventions; it does not go hunting.
- **Blocking on unknown.** An unreachable or unlicensed source with its caveat stated is a legal state of the record; the refusal is reserved for the missing verdict, not the honest one.
- **Runtime dependencies.** `go.mod` licences are the module ecosystem's concern and stay outside the acknowledgements machinery, as the acknowledgements header already states.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a new reference or influence-registry entry with no licence verdict in the baseline, **when** the gate runs, **then** it refuses, naming the entry.
- **Given** `abcd docs cite refresh` runs against a source whose licence is determinable, **when** it completes, **then** the baseline records the SPDX identifier, the URL consulted, and the check date, and the diff is reviewable before commit.
- **Given** a source that is unreachable or declares no licence, **when** the refresh records it, **then** the verdict is the explicit unknown form with the reason, and the gate accepts the entry only when its in-entry caveat is present.
- **Given** any run of the gate rules, **when** they evaluate, **then** no network access occurs; the network work exists only in the refresh verb.
- **Given** a licence verdict older than the staleness policy, **when** lint runs, **then** the entry warns and eventually blocks on the same windows the citation baseline already enforces.

## Open Questions

- Where the verdict canonically lives: the citation baseline (keeping `references.csl.json` pure CSL) or the CSL `custom` field (keeping one file per source, the pattern the confidential-sources design already uses)?
- SPDX normalisation: accept free text and normalise on write, or refuse anything that is not an SPDX identifier or the unknown form?
- Manual override: when a human has verified a licence the fetcher cannot see (a book, a paywalled paper), what is the attested-by-hand verdict form, and does it age differently?
- Does grandfathering end at a dated deadline, or only when the licence backfill for the existing thirty-odd entries lands?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
