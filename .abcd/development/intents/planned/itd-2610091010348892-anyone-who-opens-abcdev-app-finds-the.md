---
id: itd-2610091010348892
slug: anyone-who-opens-abcdev-app-finds-the
spec_id: spc-2610141709461788
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-135]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2610052000411162]
related_adrs: [adr-47, adr-2610091010387033]
---

# An opted-in project's record is built at each release onto its own GitHub Pages

## Press Release

> When a project switches it on at setup, abcd's release workflow builds the project's about page and development record at each release and publishes them on the project's own GitHub Pages, for example acme.github.io/widgets/. Every page opens with a line saying it is the record of an abcd-managed repository, linking to the repository, and every word on it comes from the repository. The project's own workflow builds it and GitHub hosts it; abcd runs nothing. abcd's own record moves to the same route, built exactly the way any other project's is.

## Why This Matters

Every abcd-managed project should have the about page and development record abcd already builds for itself, kept current, without anyone running a server: GitHub builds and hosts it. The product thinker, 2026-10-14, verbatim: "abcd must provide this (and keep current!) for any abcd-managed repo", and "if a user opts-in, abcd automatically builds a github.io site with each release".

## Mechanism

We expect owners to switch it on because it costs one answer at setup and nothing to run afterwards: the project's own release workflow builds the pages and GitHub hosts them. The claim fails if owners who say yes find the pages missing or stale after a release.

## Scope Conditions

- A public abcd-managed repository on GitHub with GitHub Pages available to it; private repositories are a later intent (Pages on a private repository is access-controlled only on GitHub's enterprise plan). <!-- cond: cond-2610141709464490 -->
- Releases are made as GitHub releases. <!-- cond: cond-2610141709468073 -->
- The record fits GitHub Pages' limits (a site of at most 1 GB). <!-- cond: cond-2610141709463916 -->

## Acceptance Criteria

_Drafted 2026-10-14 from the product thinker's rulings; accepted as drafted the same day (Decisions)._

- **Given** setup in a public abcd-managed repository, **when** the owner answers yes to a project site, **then** the switch is set in the project's abcd settings and abcd's release workflow is added, and nothing else in the repository changes.
- **Given** a project with the switch on, **when** it publishes a release, **then** its about page and development record appear at its own GitHub Pages address, as of that release.
- **Given** any page of the site, **when** a visitor opens it, **then** it opens with a line saying it is the record of an abcd-managed repository, linking to the repository.
- **Given** abcd's own repository, **when** it moves to this route and publishes a release, **then** its record is built the same way as any other project's, and its Cloudflare deploy can be retired.
- (technical) **Given** text on a page that cannot be traced to a file in the repository, **when** the workflow builds, **then** the build fails, as the site's single-source rule already requires.
- (technical) **Given** the release workflow, **when** it runs, **then** it uses no secret beyond the short-lived token GitHub gives each run, and abcd runs nothing.
- (technical) **Given** the owner switches the site off, **when** the next release is published, **then** nothing is published, and abcd's setup can remove the workflow.

## Open Questions


## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-09: the product thinker ruled that abcdev.app and abcdesign.app are both kept: abcdev.app shows a project's landing page and record, abcdesign.app is where anyone opens and edits its brief (itd-2610052000411162). Answer verbatim: "Keep both sites".
- 2026-10-09: the product thinker ruled that the reach and load rules set for abcdesign.app apply to abcdev.app too. Answer verbatim: "same applies to abcdev.app, too (not just abcdesign.app)". So public and private repositories are both supported, a private one only through the person's own GitHub sign-in; read-only builds without a sign-in are rate-limited so bots cannot exhaust the site's GitHub allowance; a signed-in build spends the person's own allowance; and a shared cache serves public pages to everyone but a private page only to people the repository admits (adr-2610091010387033).
- 2026-10-14: the product thinker ruled that every project page, abcd's included, stays on abcdev.app and shows only what can be built from the project's repository. Answer verbatim: "remove intentdriven.sh/abcd; all goes through abcdev.app, although only stuff that can be build via the public repo. It doesn't shown anything for a private repo, nor will it add content that's not in the public repo". Clarified the same day, verbatim: "of course it can be used if a person has access to the repo; public only for people without sign in". So the 2026-10-09 reach ruling stands: a visitor not signed in sees public repositories only, and a person signed in with GitHub also sees the private repositories they have access to. This withdraws a proposal made earlier the same day to put a project's "what it is about" pages on a separate site (intentdriven.sh/abcd/).
- 2026-10-14: the product thinker ruled that a project's page at abcdev.app/<owner>/<repo>/ shows its about page and its development record, exactly as abcd's own front page does today, and that abcd provides that page, kept current, for every abcd-managed repository. Answer verbatim: "About page plus record, exactly as now -- abcd must provide this (and keep current!) for any abcd-managed repo". This supersedes the same day's earlier remark that abcd's page there "would only show the dev dashboard etc., not what abcd is about".
- 2026-10-14: the product thinker ruled that a public project appears on abcdev.app only once its owner switches it on in the repository's abcd configuration, and that abcd's setup asks the owner whether to. Answer verbatim: "repo owner should switch it on in config; ask at setup". This settles the opt-in half of the first open question (which repositories are rendered); switching the setting off is the owner's way to withdraw.
- 2026-10-14: the product thinker ruled on authorship: every project page opens with a default line saying it is the record of an abcd-managed repository, linking to that repository on GitHub, and its report button links to GitHub's own reporting, not to abcd, because GitHub hosts the content and abcd does not. Answer verbatim: "A default line at the top saying this is the record for an abcd-managed repo, with link to the github repo. A report button links to Github, not abcd; we're not hosting that content, github is". This settles the open question on what a page says about who wrote it.
- 2026-10-14: asked what abcdev.app/ itself shows once every project lives at abcdev.app/<owner>/<repo>/, the product thinker chose "A list of projects": the front page lists every switched-on project, abcd among them, and abcd's own about page and record move to abcd's project address, one click deeper.
- 2026-10-14: asked how soon after a change a project's page must show it, given that each rebuild spends part of GitHub's allowance, the product thinker chose "At each release" over minutes after each change and once a day.
- 2026-10-14: asked whether the release pace also binds the signed-in team working from the record, the product thinker ruled, verbatim: "public: Release only; contributors/maintainers: immediately/live, it's an abcd feature". So a visitor sees the page as of the project's last release, and a signed-in person with access to the repository (its contributors) sees the record as it is now, rebuilt on that person's own GitHub allowance (the 2026-10-09 reach ruling).
- 2026-10-14: asked what happens to the routes for people outside the team now that abcdesign.app is team-only, the product thinker ruled, verbatim: "abcdev.app is the public facing site, abdesign.app the one for the team (an optional link connects both if the visitor to abcdev has access to abcdesign); but abcdev should have a simple option for anyone to submit an issue, even if they cannot access abcdesign: if they make a change, they should have the option to submit an issue for that specific change, pre filled template on gh they can submit." Read as: abcdev.app is public and abcdesign.app is the team's; abcdev.app shows a link to abcdesign.app only to a visitor who has access to it; on abcdev.app anyone may make a change and submit it as a pre-filled GitHub issue for that one change. The earlier same-day routes for outsiders on abcdesign.app (editing there, the downloaded file, choosing a route by the size of the change) are not carried forward; itd-2610052000411162's press release, which puts the outsider route on abcdesign.app, is rewritten at planning to put it on abcdev.app.
- 2026-10-14: asked whether the route for proposing a change to a public brief stays its own plan or folds into the abcdev.app project pages, the product thinker rejected both, verbatim: "no, abcdev is read only, abcdesign.app is editing for authorised people; they can submit issues as described, pull requests for large work etc." Read as: abcdev.app offers no way to propose a change, for anyone; editing happens only on abcdesign.app, by the people admitted there, who submit a small change as a pre-filled GitHub issue and larger work as a pull request. This reverses the same day's earlier ruling that abcdev.app offers anyone a pre-filled issue for a change, and narrows the ruling that every change on abcdesign.app becomes a pull request.
- 2026-10-14: asked what opting in takes besides the configuration switch, given the research note of the same day (2026-10-14-abcdev-app-github-builds-and-cache.md: hosted builders use a GitHub App installed per repository, building on the release webhook with that installation's own rate budget, and a signed-in person gets read-only private access only where the App is installed), the product thinker (answering the facilitator's question) chose "Switch plus GitHub App": a project opts in by the switch in its abcd configuration AND by installing abcd's GitHub App on the repository; setup asks for both, and an organisation's repository needs an organisation owner to install it.
- 2026-10-14: after the research findings, the product thinker ruled, verbatim: "for now, public projects only in abcd, but private ones must be an option later". Read as: abcdev.app and abcdesign.app serve public repositories only in their first release; a private repository is shown or edited nowhere until a later intent adds it, and the design must leave room for that (the 2026-10-09 rules for a private repository, reached only through the person's own GitHub sign-in and never cached beyond those with access, become that later intent's requirements).
- 2026-10-14: the product thinker ruled on hosting, verbatim: "no hosting server, [...] we would want to collaborate with a partner offering that, not us. never.", clarified the same day to say it is offered only through a partner Read as: abcd never runs a server that holds or serves project content; anything that needs one is offered only through a partner who runs it. This conflicts with the same day's "Switch plus GitHub App" ruling insofar as building on GitHub's release webhooks needs a server to receive them, and is put back to the product thinker once the research note on partner products (2026-10-14) is in.
- 2026-10-14: asked which address a project's pages use now that abcd runs no server holding or serving project content (research note 2026-10-14-partners-and-no-server-options-for-the-hosted-sites.md: a project's own GitHub workflow can build its pages at each release and publish them on its own free GitHub Pages), the product thinker chose "Project's own Pages address": a project's pages live at its own GitHub Pages address (for example acme.github.io/widgets/), built at each release by the project's own workflow, which abcd ships; abcdev.app lists the switched-on projects and links to them. This replaces the earlier abcdev.app/<owner>/<repo>/ address, and abcd's own pages move to abcd's own Pages address on the same terms.
- 2026-10-14: asked how a project opts in, given that keeping the GitHub App means a small job on abcd's side holding a secret key, the product thinker answered, verbatim: "no key holding. Alternatives?" So abcd holds no key of any kind, and the GitHub App is dropped from the opt-in (reversing the same day's "Switch plus GitHub App"); the alternatives were put back to the product thinker.
- 2026-10-14: asked how people sign in on abcdesign.app and for abcdev.app's live view, told that a "Sign in with GitHub" button needs a secret held by a small service (GitHub paused removing that need in August 2026) and that an abcd service holding it is ruled out, the product thinker chose "Paste a GitHub key": each person pastes a personal access token they create on GitHub; no server or partner is involved, and the person handles the key themselves.
- 2026-10-14: asked how abcdev.app's list learns which projects switched on, the product thinker set the question aside and reset the shape, verbatim: "this is all too complicated. Let's rethink. if a user opts-in, abcd automatically builds a github.io site with each release. We should test that with the abcd repo at one point so that it no longer has to deploy to cloudflare. The cloudflare abcdev.app will then become a polished version specifically for the abcd community (if there's a need for that); what's important is abcdesign.app, which must work for any abcd-managed repo where a team member can host a session to, initially, edit the brief only, e.g. abcdesign.app/acme/widget/brief/ (later intents/ etc., but not now). That's the most urgent thing for PTs: To be able to edit the brief; the current suggested solution (worktree) isn't ideal; we need the team to live build and/or coordinate here. If they sit in Stoa, or MS-Teams for example and work on that collaboratively." Read for this intent: a project that opts in gets a github.io site built by abcd's workflow at each release, on its own GitHub Pages; abcd's own repository is to be moved onto that route at some point, retiring its Cloudflare deploy; abcdev.app becomes, if there is a need, a polished site for the abcd community only, and no longer lists or renders other projects. The earlier rulings on listing, the opt-in signal, the front-page list and the live view for signed-in contributors are superseded; this intent's press release is rewritten at planning.
- 2026-10-14: walked through the rewritten press release above, the product thinker answered "Stands as written".
- 2026-10-14: offered the criteria walk on the drafted Why This Matters, Mechanism, Scope Conditions and Acceptance Criteria, the product thinker answered, verbatim: "auto-approve all". Every drafted criterion, the technical ones included, is accepted as written.

## Grounds

- pursued: owners switch the project site on because it costs one answer at setup and nothing to run afterwards; shown wrong if owners who say yes find the pages missing or stale after a release
