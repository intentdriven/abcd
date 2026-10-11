---
schema_version: 1
id: "iss-2610021446271464"
slug: "the-site-s-landing-page-composes-its"
severity: "major"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-60 (preflight site-render gate)"
origin: researcher-authored
production_mode: hand-written
remedy: "Waits on a ruling by the product thinker: either (a) rewrite the retired word in itd-60's press release and body to the role each sentence means (the product thinker reads the flagged brief sentence; the technical facilitator runs the gate), as the sweep did elsewhere, accepting an edit to a shipped record's prose and checking no gate pins grilled_intent_hash on a shipped intent; or (b) have the composer's hero selection skip, with a named reason in the build summary, a press release the site's banned-token rule would refuse, so the record keeps its words and the page stays publishable; and in either case add the shipped and planned intents' press releases to the docs-lint token's roots or to a site pre-check, so the refusal is met at the record and not at the render. Grounds: itd-2609212137129937's press release 'every place abcd writes names which of the two people it means' and the site chapter's single-source rule, under which the page is fixed at its source."
resolution: "Reworded under the product thinker's ruling R5 (2026-10-02): itd-60's three named sites (the press release sentence the landing page renders as its hero quote, and the two Decisions sites) name the product thinker in place of the retired role word, so abcd lint site passes and the site-render gate exits 0. Option (b), the composer's skip-and-say, was ruled out; the record-time check in the remedy was not asked and is captured separately."
impact: internal
resolved_by:
  commit: "41570f056d3d014903c64c83bfa405cbe6e51c01"
---

The site's landing page composes its hero quote from the press release of the newest shipped intent whose fidelity audit is MET (internal/core/site/compose.go:1251 newestMetIntent, 1153), and the first ingested audit of itd-60 (shipped 2026-09-30, two days after the role-vocabulary sweep b4df4f25d, which rewrote no intent body) makes that intent the hero: its press release says 'flags it for the maintainer to read', so 'abcd lint site' refuses the render with roles/retired-role-word on index.html and the site-render gate of make preflight exits 2 on a branch that only recorded the audit (chore/audit-v012-gaps at 373d8b5f9, over main 7fb52a6b5). The sweep intent itd-2609212137129937 scoped intent bodies out, yet a press release is abcd's own text the moment the composer renders it publicly; the retired word stands nine times in itd-60 and ten in the sweep intent's own record, so any later MET audit of an intent carrying it reproduces the refusal, and the landing page's text is decided by which audit was ingested last rather than by a guarded source.

## Grounds

- pursued: the site renders with itd-60 as the hero quote; a later shipped press release carrying the word would show it wrong
