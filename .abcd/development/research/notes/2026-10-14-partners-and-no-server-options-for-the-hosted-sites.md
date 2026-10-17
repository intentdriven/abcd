# Research note: products that could build on abcd's repository files for abcdev.app and abcdesign.app

Written for itd-2610091010348892, itd-2610091016262335 and itd-2610052000424067, from the planning interview of 2026-10-14, after the product thinker ruled that abcd never runs a server holding or serving project content. Produced by a research agent and saved unedited.

**Date:** 2026-10-14. Recency is judged against this date.

**Constraints (product thinker rulings, 2026-10-14):**
- abcd never runs a server that holds or serves project content.
- Where hosting is needed, a partner runs it. Nothing here is framed as a paid abcd feature.
- Group editing works through a shared GitHub branch. The page polls often, handles clashes, and Submit turns the branch into one pull request. There is no live relay.

**Evidence tiers:**
- **[DOC]**: the vendor's own documentation or pricing page. It shows what the product claims about itself. I did not test it.
- **[EVIDENCE]**: a measurement or dataset.
- **[CONSENSUS]**: several independent practitioners agree.
- **[CONTESTED]**: credible sources disagree.
- **[ANECDOTE]**: one person's report, or a vendor claim with no method behind it.

Almost everything below is [DOC]. I tested nothing hands-on, and I found no study comparing these products. Where the evidence is thin, I say so.

## Summary

1. **Release-time pages need no abcd server.** The project owner's own GitHub Actions workflow runs when a release is published, renders the site with abcd's existing renderer, and publishes it to the owner's GitHub Pages. GitHub hosts it, and it is free for public repositories. That covers the "as of the last release" half of abcdev.app.
2. **No product renders abcd's record model.** That means intents moving through drafts, planned and shipped folders, plus ADRs, the issue ledger and the about page. abcd keeps building the renderer either way.
3. **A browser-only "Sign in with GitHub" button is impossible today.** GitHub requires a client secret for the token exchange and blocks browser calls to it. Its plan to support single-page apps was paused in August 2026. Without a server, users must paste their own access token, or a partner must host sign-in. This is the one hard wall for abcdesign.app and for the signed-in "live" view of abcdev.app.

## 1. What works with no server at all

### 1a. Owner-run builds published to GitHub Pages (abcdev.app, release view)

- **Release trigger.** A workflow can run on `release: published`. Draft releases trigger nothing except `published` ([Actions events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)) [DOC]. No webhook receiver is needed, because GitHub runs the build inside the owner's repository. This matches abcd's existing decision that the website deploys on release, not on merge (`.abcd/development/decisions/adrs/0048-website-deploys-on-release-not-on-merge.md`).
- **Any generator can publish.** A custom workflow uses `configure-pages`, `upload-pages-artifact` and `deploy-pages`. The deploy job needs `pages: write` and `id-token: write` ([custom workflows](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)) [DOC].
- **Limits.** A published site can be at most 1 GB. There is a soft limit of 100 GB of bandwidth a month, and deployments time out after 10 minutes. The 10-builds-per-hour soft limit does not apply to custom workflows ([Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits)) [DOC].
- **Prohibited use.** Pages may not be used mainly for commercial transactions or commercial SaaS, or for handling passwords ([Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits)) [DOC]. A project's own record site is fine. A central abcd product site should stay informational.
- **Private repositories.**
  - A Pages site restricted to people with read access requires GitHub Enterprise Cloud ([visibility](https://docs.github.com/en/pages/getting-started-with-github-pages/changing-the-visibility-of-your-github-pages-site)) [DOC].
  - Pro and Team can publish Pages from private repositories, but the site is not access-controlled ([plans](https://docs.github.com/en/get-started/learning-about-github/githubs-plans)) [DOC].
  - So a private repository's record either goes public on Pages, or needs Enterprise Cloud, or is rendered in the browser with the user's own token (1b).
- **Cost.** abcd ships a reusable workflow or Action. The owner pays nothing on a public repository.

### 1b. A static client-side app that talks to GitHub directly (abcdesign.app, and abcdev.app's live view)

abcd would serve only the app's code from static hosting. The content travels from GitHub straight to the user's browser.

**Reading:**
- The REST API "supports CORS for AJAX requests from any origin" ([CORS](https://docs.github.com/en/rest/using-the-rest-api/using-cors-and-jsonp-to-make-cross-origin-requests)) [DOC].
- Without a token, the limit is 60 requests an hour per IP address. With a user token it is 5,000 an hour, shared with everything else acting for that user ([rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)) [DOC].
- A record of a few hundred files therefore cannot be read through the API by anonymous visitors. Two workarounds:
  - Anonymous reads from `raw.githubusercontent.com` at the release tag. One developer reports it sends `Access-Control-Allow-Origin: *`, sits outside the 60-an-hour limit, and does not serve private repositories ([diffdump #11, Jul 2026](https://github.com/hsyntax/diffdump/issues/11)) [ANECDOTE]. GitHub does not document the raw host's limits.
  - A single JSON bundle that the owner's release workflow writes to the owner's Pages site. A GitHub engineer said in 2021 that Pages sends `access-control-allow-origin: *` for public sites only ([community #22399](https://github.com/orgs/community/discussions/22399)) [ANECDOTE: an informal forum reply].
- Either way, test it with a real fetch before relying on it.

**Writing to a shared branch:**
- Updating a file through the Contents API requires the blob SHA of the version being replaced ([contents](https://docs.github.com/en/rest/repos/contents)) [DOC]. That gives optimistic concurrency: a save made against a stale version is rejected, and the app can show the clash.
- Create, update and delete calls "must be used serially", meaning one at a time (same page) [DOC].
- For multi-file saves, the Git Data API's ref update with `force: false` makes "sure the update is a fast-forward", so it never overwrites a teammate's work ([refs](https://docs.github.com/en/rest/git/refs)) [DOC].
- A commit created through the API is authored by "the authenticated user" by default ([git commits](https://docs.github.com/en/rest/git/commits)) [DOC]. With each member's own token, each save is credited to that person.

**Frequent refresh without a relay:**
- Send `If-None-Match` with the ETag from the last response. "A conditional request does not count against your primary rate limit if a 304 response is returned."
- GitHub still prefers webhooks to polling, and recommends polling only "as often as you need to" ([best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api)) [DOC].

**One pull request naming everyone:**
- `Co-authored-by` trailers credit co-authors when the address is tied to their GitHub account. GitHub's no-reply address works for people who keep their email private ([multiple authors](https://docs.github.com/en/pull-requests/committing-changes-to-your-project/creating-and-editing-commits/creating-a-commit-with-multiple-authors)) [DOC].
- GitHub's docs do not say whether a squash merge keeps the authors of individual commits ([squash config](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/configuring-commit-squashing-for-pull-requests)). So the Submit step should write explicit trailers into the pull request rather than rely on the merge.

**Sign-in is the wall:**
- **Web flow.** The token exchange (`POST https://github.com/login/oauth/access_token`) lists `client_secret` as Required, and "CORS pre-flight requests (OPTIONS) are not supported" ([authorizing OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)) [DOC]. GitHub App user tokens also require the secret in the web flow ([GitHub App user tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)) [DOC].
- **PKCE did not remove the secret.** PKCE (a sign-in hardening method) arrived on 14 July 2025, but "GitHub does not distinguish between public and confidential clients" ([changelog](https://github.blog/changelog/2025-07-14-pkce-support-for-oauth-and-github-app-authentication/)) [DOC]. A GitHub staff member wrote on 15 July 2025: "all of them require access to the client secret", and "We still have to add support for CORS on the token endpoint" ([community #15752](https://github.com/orgs/community/discussions/15752)) [ANECDOTE: staff forum post].
- **The fix is paused.** Roadmap item #1153, "Single page app support for GitHub Apps", would have allowed PKCE with no secret and CORS on `/access_token`. It targeted Q4 2025, moved to Exploring in June 2026, and was set to **Paused on 13 August 2026** ([roadmap #1153](https://github.com/github/roadmap/issues/1153)) [DOC].
- **Device flow** needs no client secret, but it must be enabled in the app's settings ([authorizing OAuth apps](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)) [DOC]. One user reported CORS errors on `github.com/login/device/code` in January 2025 (in [#15752](https://github.com/orgs/community/discussions/15752)) [ANECDOTE]. Treat it as blocked until tested.
- **What works today without a server:** the user pastes a fine-grained personal access token. Sveltia CMS, an MIT-licensed successor to Decap, offers this and recommends it for solo users. For several users it recommends its self-installed Cloudflare Worker authenticator instead, because tokens are a burden. Its maintainers decline to run that authenticator as a hosted service because of "legal, privacy, and security liabilities" ([sveltia-cms-auth](https://github.com/sveltia/sveltia-cms-auth), [sveltia-cms](https://github.com/sveltia/sveltia-cms)) [DOC].

### 1c. Native GitHub surfaces that cost nothing and need no app

- **Small changes as pre-filled issues.** A link such as `.../issues/new?title=&body=&labels=&template=` opens a pre-filled issue.
  - The `labels` and `assignees` parameters work only if the person has permission to set them, and otherwise the link returns 404. An over-long URL returns 414.
  - The parameters can also fill fields in issue form templates ([creating an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-an-issue)) [DOC].
  - Issue forms are YAML files under `.github/ISSUE_TEMPLATE`, and are still in public preview ([issue forms](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)) [DOC].
- **Larger changes from people without write access.** The GitHub web editor will "automatically fork the repository and open a pull request". It cannot edit a protected branch directly ([editing files](https://docs.github.com/en/repositories/working-with-files/managing-files/editing-files)) [DOC].
- **github.dev.** Free for any signed-in user. Commits go to the current branch. Only web extensions run, and there is no terminal ([github.dev](https://docs.github.com/en/codespaces/the-githubdev-web-based-editor)) [DOC].
- **GitHub Projects.** Table, board and roadmap views, up to 50 fields, and draft issues ([Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects/learning-about-projects/about-projects)) [DOC]. It tracks issues and pull requests, not repository files.

### 1d. What is impossible without any server (plainly)

1. **A "Sign in with GitHub" button that completes in the browser.** The web flow requires a secret and has no CORS. Device flow is reportedly CORS-blocked. Single-page app support is paused (sources in 1b). The only way round it is pasting a token.
2. **Detecting GitHub App installs as the opt-in signal.** A GitHub App can run with webhooks turned off; deselecting "Active" is allowed ([registering an app](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app)) [DOC]. But listing an app's installations requires signing with the app's private key, which cannot ship in a browser. The options:
   - A scheduled job in abcd's own Actions that holds the key as a secret. That is compute, not a content server, so the product thinker must decide whether it is allowed.
   - Replace the App with a signal anyone can see, such as a repository topic plus the config switch.
3. **Instant updates to a central abcdev.app index when a release is cut.** No server means no webhook receiver. A scheduled job runs at most every 5 minutes, "can be delayed during periods of high loads", and is turned off in a public repository after 60 days without activity ([Actions events](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)) [DOC]. Per-repository pages are unaffected, because each owner's own release workflow builds them.
4. **Real-time co-editing and presence.** Without a relay you can only poll. The ruling already accepts this.
5. **Rendering a private repository centrally without a sign-in.** It needs Enterprise Cloud Pages (1a) or a signed-in browser (1b).
6. **Inviting a colleague who lacks write access to edit the shared branch.** GitHub's own collaborator invitation is the only grant, and the repository owner must send it. Without it, the colleague's route is the fork-and-pull-request path in 1c.

## 2. Product catalogue: partners and components

"Fit" means fit with abcd's layout: Markdown with YAML frontmatter under `.abcd/development/…`. **No product understands abcd's intent lifecycle, ADRs or issue ledger as such.** Every option needs at least templates for them, and I tested none against a `.abcd/` dot-directory. That is a real risk, because tools often skip dot-directories.

### 2.1 Read-only rendering

| Product | Licence / cost | Fit with abcd files | Who hosts | Notes |
|---|---|---|---|---|
| **Astro + Starlight** | MIT ([repo](https://github.com/withastro/starlight)) | Configuration plus templates. Astro's `glob()` loader reads Markdown "from anywhere on the filesystem" with a `base` path and checks frontmatter against a schema ([content collections](https://docs.astro.build/en/guides/content-collections/)). Starlight requires a `title` on every page ([authoring](https://starlight.astro.build/guides/authoring-content/)) | Owner's Pages | The best fit of the generators for typed frontmatter [DOC]. abcd already has its own renderer, so this is a fallback, not a requirement. |
| **Docusaurus** | MIT ([repo](https://github.com/facebook/docusaurus)) | Unknown fields need a `parseFrontMatter` hook ([docs](https://docusaurus.io/docs/markdown-features)) | Owner's Pages | Adapter code needed [DOC]. |
| **Hugo** | Apache-2.0 ([repo](https://github.com/gohugoio/hugo)) | Mounting a non-`content/` folder is possible, but the mounts docs page returned 404, so this is **unverified this run** | Owner's Pages | |
| **Material for MkDocs** | In maintenance mode since 5 Nov 2025. Fixes promised for "at least the next 12 months". Successor Zensical is MIT and not yet at feature parity ([blog](https://squidfunk.github.io/mkdocs-material/blog/2025/11/05/zensical/)) | Configuration plus plugins | Owner's Pages | **Do not adopt for new work** [DOC]. |
| **Read the Docs** | Community: free for open source, ad-supported, public repositories only. Business: $50 / $150 / $250 a month; Enterprise from $10k a year ([pricing](https://about.readthedocs.com/pricing/)) | "Any tool that generates HTML" through `build.commands` ([build customization](https://docs.readthedocs.com/platform/stable/build-customization.html)), so abcd's renderer could run there | **Partner** | A ready partner for public read-only pages if an owner does not want Actions [DOC]. Paid Business tiers, not Enterprise, are needed for private repositories and team sign-in. |
| **GitBook** | Free plan: 1 user. Essential $65/site/mo; Ultimate $249/site/mo, which adds "Authenticated access"; plus $12/user/mo ([pricing](https://www.gitbook.com/pricing)). An open-source programme exists; "Contributors can update docs through GitHub or GitLab sync without a paid GitBook seat" ([open source](https://www.gitbook.com/solutions/open-source)) | Git Sync with `.gitbook.yaml` `root`. With no `SUMMARY.md`, it infers navigation from folders ([content config](https://gitbook.com/docs/getting-started/git-sync/content-configuration)). Files not listed in `SUMMARY.md` do not appear, and it may create new files rather than reuse existing ones ([troubleshooting](https://gitbook.com/docs/getting-started/git-sync/troubleshooting)) | **Partner** | Whether GitBook keeps unknown frontmatter fields is **undocumented**, a risk for abcd's records [DOC]. |
| **Mintlify** | Starter $0 (5 editors); Pro $450/mo annual ([pricing](https://www.mintlify.com/pricing)) | Its own docs format; would need an adapter | **Partner** | Too expensive for abcd's users relative to fit. |
| **Backstage (TechDocs + ADR plugin)** | Apache-2.0, CNCF Incubation ([repo](https://github.com/backstage/backstage)) | The ADR plugin parses MADR (a common ADR template) 2.x/3.x only, and recognises frontmatter only in the MADR 3.x form. Decorators change how content is displayed, not how it is parsed ([ADR plugin](https://github.com/backstage/community-plugins/blob/master/workspaces/adr/plugins/adr/README.md)) | A Backstage instance someone runs | **Anti-recommendation at this scale.** Porsche estimates "~15% of one engineer's time" to keep an instance running, and moved Backstage from Trial to Assess in July 2025 ([Porsche radar](https://opensource.porsche.com/porschedigital-technology-radar/platforms-and-operations/backstage/)) [ANECDOTE: one organisation]. |
| **Log4brains** | Apache-2.0. v1.1.0 on 17 Dec 2024, "After a very long pause" ([releases](https://github.com/thomvaill/log4brains/releases), [repo](https://github.com/thomvaill/log4brains)) | Configurable `adrFolder`; MADR is the default template | Owner's Pages | ADRs only, and maintenance is intermittent. Not worth adopting. |

### 2.2 Browser editing that lands as commits or pull requests

| Product | Licence / cost | Pull requests? | Server or partner needed | Notes |
|---|---|---|---|---|
| **Decap CMS** | MIT. Active: 3.17.0 makes "Decap Turbo … generally available" ([releases](https://github.com/decaporg/decap-cms/releases); dated "14 Oct" with no year, which implies today) | Editorial workflow: a branch `cms/<collection>/<slug>` plus a pull request; publish merges it ([editorial workflow](https://decapcms.org/docs/editorial-workflows/)). **Open Authoring**: people without write access edit in their own fork, and moving a card to "Ready to Review" opens a pull request ([open authoring](https://decapcms.org/docs/open-authoring/)) | The GitHub backend needs an authentication server (Netlify or Turbo), and "all users must have push access" except under Open Authoring ([GitHub backend](https://decapcms.org/docs/github-backend/)) | The closest existing match to abcdesign's "larger work as a pull request" [DOC]. |
| **Decap Turbo** | Free (1 site, 1 seat); Pro €19/mo (5 seats); extra €10/site and €6/seat ([Turbo](https://decapcms.org/turbo/)) | Undocumented for Turbo | **Partner.** "Turbo mirrors the repo into its own database" and makes Git calls "using the organization's Git connection" ([how it works](https://decapcms.org/docs/turbo-how-it-works/)) | Project content does pass through the partner. Commit authorship is undocumented [DOC]. Just reached general availability, so evidence is thin. |
| **Sveltia CMS** | MIT, public beta ([repo](https://github.com/sveltia/sveltia-cms)) | Editorial workflow status **unverified** | Token sign-in needs none; button sign-in needs a self-installed Worker | **[CONTESTED]**: it calls Decap "neglected for years", but Decap shipped releases monthly through 2026 ([releases](https://github.com/decaporg/decap-cms/releases)). |
| **TinaCMS / TinaCloud** | Apache-2.0 ([repo](https://github.com/tinacms/tinacms)). Free: 2 users. Team $24/project/mo. Team Plus $41/project/mo is the lowest tier listing the editorial workflow ([pricing](https://tina.io/pricing)) | Saving to a protected branch creates a branch and opens a **draft pull request** ([editorial workflow](https://tina.io/docs/tinacloud/editorial-workflow)) | **Partner** (TinaCloud). Self-hosting needs your own auth, database and Git modules, and "some features are not available" ([self-hosted](https://tina.io/docs/self-hosted/overview)) | The partner path that matches the shared-branch-then-pull-request model best. Self-hosting fails the no-server rule. |
| **Keystatic** | MIT ([repo](https://github.com/Thinkmill/keystatic)) | Branch picker; pull requests not documented | Needs Node.js for its API routes, and collaborators need write access ([GitHub mode](https://keystatic.com/docs/github-mode)) | Fails the no-server rule. |
| **Pages CMS** | MIT; the hosted app is free ([site](https://pagescms.org/)); configured by `.pages.yml` ([docs](https://pagescms.org/docs/)) | Undocumented | **Partner** (free hosted app). Its GitHub App asks for **Administration read and write**, Actions read and write, and Contents read and write ([GitHub App](https://pagescms.org/docs/guides/installing/github-app/)) | Collaborators invited by email commit under the app's identity unless `commit.identity: user` is set ([collaborators](https://pagescms.org/docs/configuration/collaborators/)). The broad permissions and machine-authored commits are a poor match for abcd's human-author rule. |
| **CloudCannon** | Standard $49–55/mo (3 users); Team $300–350/mo; no free plan ([pricing](https://cloudcannon.com/pricing/)) | Branching workflows on Team | **Partner** | Too expensive for small open-source teams. |
| **Front Matter CMS** | MIT; maintained but slow, per a 2026 note ([repo](https://github.com/estruyf/vscode-front-matter)) | No | None | A desktop VS Code tool; web support unverified ([docs](https://frontmatter.codes/docs)). |
| **GitBook change requests** | See 2.1 | No. Merging a change request "commit[s] its changes to GitHub" on the synced branch ([enabling GitHub sync](https://gitbook.com/docs/getting-started/git-sync/enabling-github-sync), [change requests](https://gitbook.com/docs/collaboration/change-requests)) | **Partner** | Review happens inside GitBook, not as a GitHub pull request. To get one, sync GitBook to a side branch and open the pull request from there. |
| **HackMD** | Free: 3 teammates, 20 GitHub pushes a month. Prime $5/seat/mo ([pricing](https://hackmd.io/pricing)) | No. A push makes a commit to a branch or file you choose; HackMD recommends an admin account ([sync](https://hackmd.io/@docs/sync-a-note-with-github)) | **Partner** | One note per file. Suits one brief, not the whole record. |
| **Obsidian Git / Relay** | Obsidian Git is MIT; mobile is "very unstable" ([repo](https://github.com/Vinzent03/obsidian-git)). Relay's Git sync needs Premium at $18/user/mo ([Relay](https://relay.md/)) | No | Local, or Relay | A desktop path for writers, not a browser product. |

### 2.3 Collaboration engines (real-time co-editing services)

- **Liveblocks.** Pro $25–30/mo, Team from $500–600/mo, metered per user-minute and per comment ([pricing](https://liveblocks.io/pricing)).
- **Tiptap.** The editor is MIT. Tiptap's collaboration cloud costs $49–1,199/mo ([pricing](https://tiptap.dev/pricing)), and its MIT Hocuspocus server can be self-hosted ([repo](https://github.com/ueberdosis/hocuspocus)).
- **Verdict.** Both are live relays, which the ruling excludes. They are also building blocks, not a product: abcd would still build the app and its GitHub write-back. **Rejected.**

### 2.4 Product-management tools

- **Linear.** Comments, status, labels and assignee sync both ways with GitHub Issues, aimed at open-source maintainers ([integration](https://linear.app/integrations/github)). Paid plans are $10–16 per user per month ([pricing](https://linear.app/pricing)); the page does not say which plan includes the sync. It syncs issues, not files.
- **Productboard.** "The connection is push-only" from Productboard to GitHub issues, with open or closed status flowing back ([integration](https://www.productboard.com/integrations/github/)). Credit-based plans with a 5-maker minimum ([pricing](https://www.productboard.com/pricing/)).
- **Jira.** Links branches, commits and pull requests through issue keys ([Atlassian](https://support.atlassian.com/jira-cloud-administration/docs/integrate-with-github/)). It does not sync files.
- **Notion.** One-way sync of pull requests and issues ([guide](https://www.notion.com/help/guides/bring-your-codebase-into-context-with-notions-github-integration)).
- **Confluence.** One-way publishers from Markdown to Confluence: `mark` (Apache-2.0, with optional frontmatter) ([repo](https://github.com/kovetskiy/mark)), and the commercial *GitHub Markdown Sync for Confluence* ([marketplace](https://marketplace.atlassian.com/apps/2014743641/github-markdown-sync-for-confluence)).
- **Verdict.** None of these reads intent files. Each could only receive a copy if abcd mirrored intents to issues. That is optional, and a scope increase.

### 2.5 Anything already doing "intents with acceptance criteria rendered from a repo"?

**No product renders and hosts a product record like abcd's.**

| Tool | What it has | What it lacks |
|---|---|---|
| **GitHub Spec Kit** (MIT) | Its spec template has user stories with "**Given** … **When** … **Then**" scenarios and success criteria ([template](https://github.com/github/spec-kit/blob/main/templates/spec-template.md), [repo](https://github.com/github/spec-kit)) | No frontmatter and no renderer [DOC]. The closest content model. |
| **Kiro specs** | `requirements.md` with user stories and acceptance criteria ([docs](https://kiro.dev/docs/specs/)) | No rendering or sharing documented. |
| **Backlog.md** (MIT) | Tasks as `.md` files in the repo with acceptance-criteria checklists, plus a browser board that edits them | Runs only on the local machine (`127.0.0.1`), "no server, no account" ([repo](https://github.com/MrLesk/Backlog.md)). The closest match to abcdesign's editing interaction, and it is local-only. |
| **SmartBear Cucumber for Jira "Living Documentation"** | Shows Gherkin acceptance specs (Given/When/Then feature files) from GitHub, GitLab or Bitbucket inside Jira ([product page](https://smartbear.com/product/cucumber-for-jira/living-documentation/)) | Vendor claims only; Gherkin `.feature` files, not Markdown [ANECDOTE/MARKETING]. |

## 3. Ranked shortlist (value for effort, under the rulings)

1. **Owner-run release workflow to the owner's GitHub Pages, using abcd's own renderer.** Replaces abcdev.app's per-repository "as of the last release" rendering. abcd's cost: one reusable workflow. Owner's cost: free on a public repository. GitHub hosts. [DOC, see 1a]
2. **Native GitHub surfaces for abcdesign's change paths.**
   - Pre-filled issue links and issue forms carry small changes.
   - The web editor's automatic fork-and-pull-request, or github.dev, carries large ones.
   - Replaces "small changes as issues" outright, and is a usable stopgap for "larger work as a pull request". Cost: zero. [DOC, see 1c]
3. **Static client-side abcdesign.app, with token sign-in.**
   - Covers the shared branch (blob-SHA clash detection, fast-forward-only ref updates), ETag polling, Submit as one pull request with `Co-authored-by` trailers, and the view toggle.
   - Replaces abcdesign's core with no server. Cost: abcd builds the app, and users suffer the token-paste sign-in (Sveltia ships this model).
   - It also makes "private later" easier, because content only ever reaches the browser with the user's own token. [DOC, see 1b]
4. **A sign-in partner, if token paste proves unacceptable.** **TinaCloud** fits best: draft pull requests from protected branches, from $41/project/mo, Apache-2.0 core. **Decap with Open Authoring** is the only option that turns a non-writer's edits into a fork pull request; button sign-in for it comes from Netlify or Turbo. Each partner adds a vendor holding tokens or content, and Turbo mirrors the repository. [DOC]
5. **Read the Docs Community** as a ready partner host for public read-only pages. It replaces step 1 for owners who do not want Actions, at the cost of ads. [DOC]
6. **A central abcdev.app index of links only**, built by a scheduled job in abcd's own repository from a repository topic. It holds names and links, not content. Freshness is bounded by the 5-minute schedule and its delays (1d.3). [DOC]

## 4. What no product covers (abcd still builds it)

- Rendering abcd's record model: intent lifecycle folders, press-release-first intents with Given/When/Then, ADRs, issue ledger, about page. abcd's existing renderer and its `site-render` gate already do this.
- The product-thinker / technical-facilitator view toggle.
- Group editing on a shared branch: clash handling, the refresh loop, and Submit producing one pull request that names every member.
- Opt-in discovery across repositories, and the "release vs live" switch.
- Button sign-in with no server. Today only a partner or GitHub can provide it (1d.1).

## 5. Not worth adopting

- **Backstage.** Running cost sits at platform-team scale, and its ADR plugin understands MADR only (2.1).
- **Material for MkDocs.** In maintenance mode (2.1).
- **Log4brains.** Intermittent maintenance, ADRs only (2.1).
- **Mintlify and CloudCannon.** Price far exceeds fit for small open-source teams.
- **Liveblocks and Tiptap Cloud.** Relays the ruling excludes, and building blocks rather than products.
- **Keystatic and self-hosted Tina.** Need a server.
- **Pages CMS.** Its GitHub App asks for repository Administration rights and commits under the app's identity by default, which conflicts with abcd's rule that every commit has a human author (see `AGENTS.md`, attribution section).
- **Notion, Jira and Productboard.** None reads repository files (2.4).

## 6. Open risks

- **GitHub sign-in roadmap.** #1153 is paused. If it ships, a browser-only button becomes possible. Until then, a partner holds the client secret or users paste tokens.
- **Undocumented anonymous-read limits.** raw.githubusercontent.com limits and Pages CORS rest on forum posts. Test with real fetches before designing abcdev.app's anonymous path around them.
- **The `.abcd/` dot-directory with generators and partners.** Untested. GitBook's handling of unknown frontmatter is undocumented, and it can create duplicate files.
- **Squash merges and co-authors.** GitHub does not document whether a squash keeps individual commit authors. Write explicit trailers.
- **Scheduled jobs.** They are turned off after 60 days without activity in public repositories, and run late under load.
- **Decap Turbo.** Reached general availability today. Evidence is thin, and commit authorship is undocumented.
- **Decisions for the product thinker:**
  - Does a stateless sign-in function, or a scheduled abcd Actions job holding a GitHub App key, count as "a server"? Neither holds content.
  - Does an abcd-served client app that fetches content straight from GitHub count as "serving project content"?
- **Overall evidence.** Almost all of it is [DOC]. I found no independent evaluation of any Git-based CMS or of GitBook's Git Sync fidelity.

Earlier repository research on the site is in `.abcd/development/research/abcdev-site/`. I did not reread it for this note.
