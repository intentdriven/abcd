---
id: itd-172
slug: every-record-has-a-short-title-and-a-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# Every record has a short title and a one-line summary a non-engineer can read

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

A record's title is also its slug and therefore its filename, so a title written as a paragraph produces a filename ninety characters long and a listing nobody can scan. Several records in this repository are already in that state.

The fix is two fields rather than one. The title stays short and bounded, and carries the handle. A one-line summary carries what the record is about, written so that a reader who is not an engineer can tell whether it concerns them.

The summary belongs in frontmatter rather than in the prose. Frontmatter is what every listing surface already reads, so a summary there renders in the ledger listing, the machine-readable payloads, and the record export without any of them parsing prose and guessing which paragraph was meant. It is also checkable: presence, length, and register are all mechanical, where a prose convention drifts silently, which is how the long titles arrived. The agent prompts in this repository already carry exactly such a field, so the shape is not new here.

The summary is the natural home for product-thinker-addressed language, leaving the body free to address the facilitator. That makes this rule and the addressee register the same mechanism seen from two sides.

The migration is warn then promote, the pattern the writing-style guide already uses: required on new records, advisory on the corpus, swept opportunistically, and armed as a blocker once the corpus is clean. The duplication risk is real and is managed by making the field authoritative rather than repeating it as the body's opening line.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [elicitation interviews SOTA note](../../research/notes/2026-10-23-elicitation-interviews-sota.md).

- GOV.UK's title and summary rules are public and mechanical: titles "65
  characters or less (including spaces)", "unique and informative", no content
  type in the title; summaries "160 characters, including spaces", ending "with a
  full stop", using "as few words from the title as possible".
- Software conventions for the handle agree on short and say little else: git's
  "no more than 50 characters" subject (a different artefact), Conventional
  Commits' unbounded "short summary", MADR's "{short title, representative of
  solved problem and found solution}" with `NNNN-title-with-dashes.md`, RFC
  abstracts "complete in itself" with no citations.
- Two fields with two jobs is an existing split (Hugo's `description` against
  `summary`; the Agent Skills `description` field, with its example of a poor one,
  "Helps with PDFs.").
- A plain-language summary field with mechanical rules has a precedent in
  Cochrane's standards ("Limit sentences to one key point", "Avoid more than two
  hard words in a sentence unless you explain them", a fixed hedging vocabulary).
- Readability formulas are not gates (Lenzner, 2014; Cachola et al., 2025); the
  register rules under itd-168 apply to the summary's words, the length rules
  here to its shape.

**Path 1**: adopt the GOV.UK limits as the rule set (title at most 65
characters with no record-type word, summary at most 160 ending in a full stop,
few words shared with the title, a verb, present tense), with the warn-then-
promote migration already named. GOV.UK's numbers come from search-result
truncation ("Google cuts off the rest of the title at around 65 characters")
and its title limit is soft ("You can use more than 65 characters if it's
essential"); abcd adopts them as a convenient bound, to be checked against the
slug length and an 80-column listing, warning rather than refusing above 65.
No Go dependency.

The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether a title cap re-slugs existing files, and so breaks
  links, or binds new records only; the GOV.UK limit is 65 characters.
- (SOTA 2026-10-23) Whether "no record-type word in the title" (GOV.UK's rule for
  content type) holds for abcd's records, whose folder already says the type.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
