---
id: itd-2610091010348892
slug: anyone-who-opens-abcdev-app-finds-the-landing-page-and
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-135]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2610052000411162]
related_adrs: [adr-47, adr-2610091010387033]
---

# abcdev.app shows the record of any public abcd-managed project, with abcd as the showcase

## Press Release

> Anyone who opens abcdev.app finds the landing page and development record of any public abcd-managed project on GitHub, each at abcdev.app/<owner>/<repo>/, built from that repository. abcd's own pages become the showcase at abcdev.app/[redacted-user]/abcd/, built from its GitHub repository like any other, so a visitor sees what abcd makes of a real project and how to get the same for theirs.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Which repositories are rendered, and how does one join or leave? Every public repository carrying abcd's managed marker, or only those whose owners opt in, and how an owner withdraws one. Decided in adr-2610091010387033, which reverses adr-47's decision that the site renders abcd's repository and nothing else.
- Since abcdev.app renders private repositories too (Decisions, 2026-10-09), what does a private project's page show to someone not signed in: nothing, or that it exists?
- What does a page say about who wrote it? The site renders spans of a repository's own files; a third party's files are not abcd's, under abcd's domain.
- How are pages built: on a schedule, on the owner's request, or from the owner's own release, within the GitHub API's limits? The research is owed first (iss-2610091010558682).
- What sits at abcdev.app/ itself once abcd's pages move to /intentdriven/abcd/: a general landing page, a directory of projects, or abcd's showcase again?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-09: the product thinker ruled that abcdev.app and abcdesign.app are both kept: abcdev.app shows a project's landing page and record, abcdesign.app is where anyone opens and edits its brief (itd-2610052000411162). Answer verbatim: "Keep both sites".
- 2026-10-09: the product thinker ruled that the reach and load rules set for abcdesign.app apply to abcdev.app too. Answer verbatim: "same applies to abcdev.app, too (not just abcdesign.app)". So public and private repositories are both supported, a private one only through the person's own GitHub sign-in; read-only builds without a sign-in are rate-limited so bots cannot exhaust the site's GitHub allowance; a signed-in build spends the person's own allowance; and a shared cache serves public pages to everyone but a private page only to people the repository admits (adr-2610091010387033).
