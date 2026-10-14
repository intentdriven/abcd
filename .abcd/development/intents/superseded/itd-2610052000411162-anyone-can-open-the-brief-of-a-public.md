---
id: itd-2610052000411162
slug: anyone-can-open-the-brief-of-a-public
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-14, from: standalone, to: superseded, reason: "folded into the team brief-session plan by the product thinker on 2026-10-14 (Fold into the team plan)" }
builds_on: [itd-2610040754453237]
severity: minor
origin: researcher-authored
production_mode: hand-written
kind_at_supersession: standalone
superseded_by: itd-2610091016262335
---

# Anyone can open, edit on their own device and submit the brief of a public abcd-managed project on abcdesign.app

> **Superseded by itd-2610091016262335** on 2026-10-14: folded into the team brief-session plan by the product thinker on 2026-10-14 (Fold into the team plan)

## Press Release

> Anyone can open the brief of a public abcd-managed project on abcdesign.app, read it laid out as pages, change it on their own device, and submit their changes to the project as a proposal. Nothing they type leaves their device until they submit it, and the project's own team decides what to take.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- The state-of-the-art pass is owed before planning: how current tools let a person edit a document on their own device and hand it back, and keep a public brief readable without an account, recorded as a dated research note. The product thinker's ruling, 2026-10-05: "File now, research later".
- The trust-boundary rule that a submission is only ever a proposal, never a direct write into someone else's record, is likely a decision record and a brief invariant at planning.
- Whether the pre-filled issue on abcdev.app has a size limit, and what a reader is told when a change is too long for a link (Decisions, 2026-10-14).
- How the page reads a public project's brief and stays current with it, and what it shows when the brief is not public or not abcd-managed.

## Decisions

- 2026-10-05, the product thinker: the page lives on a new site, abcdesign.app, not on abcd's own site (abcdev.app).
- 2026-10-05, the product thinker: filed as two intents. This one is one person editing on their own device; working together, notes and a combined export are itd-2610052000424067, which builds on this one.
- 2026-10-14: the product thinker ruled on reach and submission, verbatim: "first, same principle for abcdesign.app: Repo owner specifies whether it can be used; second, it only works if (a) the user is logged in and (b) if the repo can be reached, e.g. the user has access to a private repo or the repo is public; an alternative route is for the user to download formatted json from the browser's local storage (if they're not logged into git), but that only works for public repos. Now, I think it should go through a gh pull request directly; if the user isn't logged into gh, they should get help with a template or something already pre-filled -- possible?". Read as: a project is on abcdesign.app only once its owner switches it on in the repository's abcd configuration, as on abcdev.app; submitting needs a GitHub sign-in and a repository the person can reach (public, or private with access); Submit opens a GitHub pull request; a person not signed in can still download their edits as formatted JSON from their browser's storage, for a public repository only.
- 2026-10-14: asked what Submit offers a person not signed in to abcdesign.app besides the download (told that GitHub accepts a link opening a new issue with its title and body filled in, and that GitHub's web editor can open a pull request from a file changed there, both needing a GitHub account signed in on github.com), the product thinker chose "A pre-filled GitHub issue": the change arrives as text in a new issue on the project's repository, and the project's team applies it by hand.
- 2026-10-14: asked what happens to the routes for people outside the team now that abcdesign.app is team-only, the product thinker ruled, verbatim: "abcdev.app is the public facing site, abdesign.app the one for the team (an optional link connects both if the visitor to abcdev has access to abcdesign); but abcdev should have a simple option for anyone to submit an issue, even if they cannot access abcdesign: if they make a change, they should have the option to submit an issue for that specific change, pre filled template on gh they can submit." Read as: abcdev.app is public and abcdesign.app is the team's; abcdev.app shows a link to abcdesign.app only to a visitor who has access to it; on abcdev.app anyone may make a change and submit it as a pre-filled GitHub issue for that one change. The earlier same-day routes for outsiders on abcdesign.app (editing there, the downloaded file, choosing a route by the size of the change) are not carried forward; this intent's press release, which puts the outsider route on abcdesign.app, is rewritten at planning to put it on abcdev.app.
- 2026-10-14: asked whether the route for proposing a change to a public brief stays its own plan or folds into the abcdev.app project pages, the product thinker rejected both, verbatim: "no, abcdev is read only, abcdesign.app is editing for authorised people; they can submit issues as described, pull requests for large work etc." Read as: abcdev.app offers no way to propose a change, for anyone; editing happens only on abcdesign.app, by the people admitted there, who submit a small change as a pre-filled GitHub issue and larger work as a pull request. This reverses the same day's earlier ruling that abcdev.app offers anyone a pre-filled issue for a change, and narrows the ruling that every change on abcdesign.app becomes a pull request.
- 2026-10-14: asked whether this draft stays its own plan or folds into the team's abcdesign.app plan (itd-2610091016262335), now that its submit step (a small change as a pre-filled issue, larger work as a pull request) is team work there, the product thinker chose "Fold into the team plan": this draft is retired as replaced by itd-2610091016262335, which takes over its submit step.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
