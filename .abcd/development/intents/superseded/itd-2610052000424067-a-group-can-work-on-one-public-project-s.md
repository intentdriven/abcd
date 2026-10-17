---
id: itd-2610052000424067
slug: a-group-can-work-on-one-public-project-s
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history:
  - { date: 2026-10-14, from: standalone, to: superseded, reason: "folded into the team brief-session plan by the product thinker on 2026-10-14 (Fold plan 2 into 1)" }
builds_on: [itd-2610052000411162, itd-2610040754453237]
severity: minor
origin: researcher-authored
production_mode: hand-written
kind_at_supersession: standalone
superseded_by: itd-2610091016262335
---

# A group works on one public project's brief together on abcdesign.app, adds notes, and exports the combined work

> **Superseded by itd-2610091016262335** on 2026-10-14: folded into the team brief-session plan by the product thinker on 2026-10-14 (Fold plan 2 into 1)

## Press Release

> A group can work on one public project's brief together on abcdesign.app. Each person sees the others' edits and notes as they make them, can leave a note on any passage, and keeps working when they lose their connection. When they are done, they export their combined work, as one proposal for the project or as a file to keep.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- The state-of-the-art pass is owed before planning: how current tools let several people edit one document together, on and off line, merge their work and carry notes, and how they export the result, recorded as a dated research note. The product thinker's ruling, 2026-10-05: "File now, research later".
- Where a group's shared copy lives while they work, given that each person's edits stay on their own device until submitted (itd-2610052000411162), and who may join a group.

## Decisions

- 2026-10-05, the product thinker: the page lives on a new site, abcdesign.app.
- 2026-10-05, the product thinker: filed as two intents. This one is working together; one person's editing and submitting is itd-2610052000411162, which this builds on.
- 2026-10-14: asked who may join a group editing one project's brief (anyone with the link; GitHub users who can reach the repository; people the group's starter invites), the product thinker answered "2 & 3": a person joins only when the group's starter invites them and they are signed in with GitHub with access to the repository.
- 2026-10-14: asked what a group's combined work becomes, the product thinker chose "One pull request, all named": whoever presses Submit opens one pull request on the project's repository naming every member, and needs a GitHub sign-in with access to the repository.
- 2026-10-14: asked whether live group sessions are allowed on private projects, told that the shared copy is held on a small server abcd runs which can read it, and that end-to-end encryption exists only as beta software (research note 2026-10-14-live-co-editing-to-one-pull-request.md), the product thinker chose "Public projects only": no private brief is held on abcd's server, and private teams wait for encryption. The product thinker then asked to think the research findings over further before the interview continues.
- 2026-10-14: after the research findings, the product thinker ruled, verbatim: "for now, public projects only in abcd, but private ones must be an option later". Read as: abcdev.app and abcdesign.app serve public repositories only in their first release; a private repository is shown or edited nowhere until a later intent adds it, and the design must leave room for that (the 2026-10-09 rules for a private repository, reached only through the person's own GitHub sign-in and never cached beyond those with access, become that later intent's requirements).
- 2026-10-14: asked how a group edits one brief together, told that no existing tool turns a live session into one pull request and that live editing needs a server abcd runs to hold the group's copy, the product thinker chose "Shared branch first": the group shares one GitHub branch, each member's saves land on it, the others see them on reopening the page, and Submit turns the branch into the pull request, each member credited by their own saves; abcd's server holds nothing. Live editing (each sees the others type, notes appear at once) is deferred to a later intent, built only if teams ask. This supersedes, for the first release, the press release's live-editing and offline wording.
- 2026-10-14: on the shared branch, the product thinker added, verbatim: "agreed, good idea -- site must auto refresh frequenlty and deal with clashes". So the page refreshes from the shared branch often enough that members see each other's saves without reopening it, and handles two members changing the same passage (a clash) rather than overwriting either.
- 2026-10-14: the product thinker ruled on hosting, verbatim: "no hosting server, [...] we would want to collaborate with a partner offering that, not us. never.", clarified the same day to say it is offered only through a partner Read as: abcd never runs a server that holds or serves project content; anything that needs one is offered only through a partner who runs it. This conflicts with the same day's "Switch plus GitHub App" ruling insofar as building on GitHub's release webhooks needs a server to receive them, and is put back to the product thinker once the research note on partner products (2026-10-14) is in.
- 2026-10-14: asked how people sign in on abcdesign.app and for abcdev.app's live view, told that a "Sign in with GitHub" button needs a secret held by a small service (GitHub paused removing that need in August 2026) and that an abcd service holding it is ruled out, the product thinker chose "Paste a GitHub key": each person pastes a personal access token they create on GitHub; no server or partner is involved, and the person handles the key themselves.
- 2026-10-14: the product thinker reset the shape, verbatim: "this is all too complicated. Let's rethink. if a user opts-in, abcd automatically builds a github.io site with each release. We should test that with the abcd repo at one point so that it no longer has to deploy to cloudflare. The cloudflare abcdev.app will then become a polished version specifically for the abcd community (if there's a need for that); what's important is abcdesign.app, which must work for any abcd-managed repo where a team member can host a session to, initially, edit the brief only, e.g. abcdesign.app/acme/widget/brief/ (later intents/ etc., but not now). That's the most urgent thing for PTs: To be able to edit the brief; the current suggested solution (worktree) isn't ideal; we need the team to live build and/or coordinate here. If they sit in Stoa, or MS-Teams for example and work on that collaboratively." Read for this intent: the team must be able to build the brief together live, or at least coordinate live, while sitting in a meeting tool; the shared branch chosen earlier the same day is not enough on its own (read as the "worktree" named in the ruling), which reverses "Shared branch first". How live editing runs with no abcd server and no key is put back to the product thinker.
- 2026-10-14: asked how a team edits live with no abcd server and no key (options: the host's browser connecting the others directly; the host's laptop over Tailscale; the team's own relay deployed once to their own free hosting account from a blueprint abcd ships, the pattern the rascals demo repository already uses on Render; a partner service), the product thinker ruled, verbatim: "ok, we offer tailscale and the rascals pattern as build-in defaults; partner services come later". So abcd ships two built-in ways to host a session: the host's own machine reached over Tailscale, and the team's own relay deployed from abcd's blueprint to the team's own account; both are run by the team, never by abcd. Partner services are a later intent. This supersedes, for this intent, the same day's "Shared branch first".
- 2026-10-14: asked how people sign in to a team session now that the team's relay or host machine could hold a GitHub sign-in secret, the product thinker answered, verbatim: "why not the rascals pattern, which shows a QR code or number ot join". Read as: people join a session the way a phone joins a rascals game, by scanning a QR code or typing a short code the host's screen shows; they do not sign in to GitHub to join. Only the host, who submits the result, needs GitHub access (on their own machine, or the key held on the team's relay). This supersedes "Paste a GitHub key" for everyone but the host. What it does to the team-only rule (admission becomes holding the host's code, not a GitHub access check) and to naming every member on the pull request is put back to the product thinker.
- 2026-10-14: asked how the team-session plan and the group-editing plan are cut, the product thinker chose "Fold plan 2 into 1": the group-editing draft (itd-2610052000424067) folds into the team-session plan (itd-2610091016262335), so the first release of abcdesign.app is one plan, a team editing its brief together live; the group-editing draft is retired as replaced.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
