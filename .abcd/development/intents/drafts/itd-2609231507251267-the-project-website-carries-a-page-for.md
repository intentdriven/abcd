---
id: itd-2609231507251267
slug: the-project-website-carries-a-page-for
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609231013154443]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Release pages on the project website

## Press Release

> The project website carries a page for every feature release, rendered from RELEASE.md and the release archive, so a person who never opens the repository reads what each release was for in the same words the repository shows. It builds on itd-2609231013154443, which writes RELEASE.md at each feature cut and moves the page it replaces to .abcd/development/releases/; this intent is the website half that intent's decomposition split off, ruled by the product thinker on 2026-09-23 (planning decision 7) as its own intent for the next cycle. The site renders the pages it is given and composes nothing: no page for a release that has none, and nothing forward-looking.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md).

- **The changelog convention:** "There should be an entry for every single version", "Versions and sections should be linkable", "The latest version comes first", "The release date of each version is displayed"; a pulled release is marked `[YANKED]`; "using commit log diffs as changelogs is a bad idea: they're full of noise" (Keep a Changelog 1.1.0).
- **The forge's release page is per tag, with a stable latest link and a feed:** every release has "a unique URL", `releases/latest` links the newest, and `releases.atom` is an Atom feed with one entry per release linking its tag page (GitHub docs; a public repository's feed opened). Automatically generated notes list merged pull requests by label: the thing this page deliberately does not do.
- **The site already links release tags** (`/releases/tag/v<version>` on the timeline, `/releases/latest` on the install surface); a page per `RELEASE.md` and archive entry slots under the same addressing.
- **Research touches one ruling and the ruling stands:** "an entry for every single version" is about the changelog, which abcd derives for every cut; the parent's rule that a fixes-only release writes no page concerns a second document, compatible as long as the release index links the changelog section for the releases it skips.

**Path 2 — native, shaped to Keep a Changelog's ordering rules and the forge's per-tag page and feed: one page per archived release at a stable path keyed on the version, an index latest-first with dates, each page linking its tag, and an optional Atom feed from the same data.** The page composes nothing, as ruled. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Whether the site publishes an Atom feed of release pages beside the index, and if so how a feed entry carries the "fictional persona" note the quotes need when read outside the page.
- (SOTA 2026-10-23) Whether the release index lists a fixes-only release as a line linking its changelog section, or omits it entirely as the parent's "no page" rule reads.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
