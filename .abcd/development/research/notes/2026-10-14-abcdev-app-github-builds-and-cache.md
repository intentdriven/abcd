# abcdev.app: GitHub rate limits, build triggers, per-user tokens and cache isolation

Written for iss-2610091010558682, for itd-2610091010348892 and adr-2610091010387033, from the planning interview of 2026-10-14. Produced by a research agent and saved unedited; the record holds it as the agent returned it.

*Research note, 2026-10-14. Recency is assessed against 2026-10-14. Most GitHub docs pages carry no date. The dated sources are: GitHub Terms of Service, effective 2026-04-27; Vercel GitHub docs, last updated 2026-09-17; GitHub changelog, 2025-05-08; one community thread, 2022–2025, which is older and should be treated with care.*

**Evidence tiers.** **[EVIDENCE]** means the platform's own normative documentation or a standard (an RFC). **[CONSENSUS]** means several practitioners or services do the same thing independently. **[CONTESTED]** means reasonable readings disagree. **[ANECDOTE]** means a single unverified report. **[INFERENCE]** means my design reasoning; it is not a sourced fact. Every page was read through a summarising fetch tool, so the quotations below are as the tool returned them. Check the wording against the live page before quoting it in anything public.

## Bottom line

Make **installing an abcdev GitHub App** half of the opt-in, with the config file as the other half. That one choice covers most of the rulings.

- **Anonymous pages cost no GitHub requests at view time.** They are built from the release webhook using that owner's installation token, which has its own budget per installation. So anonymous traffic can at worst drain the budget of the one installation behind the page being built, never a single site-wide pool. Building happens only on releases or through a rate-limited queue.
- **Signed-in visitors use their own token.** They sign in through the same App and get a GitHub App user access token. Their live builds, and the access check before a cached private page, spend that person's allowance.
- **An OAuth App is the wrong tool here.** It cannot ask for read-only access to private repositories.
- **Private pages never enter the shared cache.**

---

## 1. What each auth mode allows per hour, and who it is charged to

| Auth mode | Primary limit | Charged to | Source / tier |
|---|---|---|---|
| Unauthenticated | 60 requests/hour, public data only | Originating IP address | [REST rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api) [EVIDENCE] |
| Personal access token, or any user-authorised token (OAuth app or GitHub App acting for the user) | 5,000 requests/hour | **The user.** One budget shared across all their apps and tools | same [EVIDENCE] |
| GitHub App **user access token** | "dictated by the primary rate limits for the authenticated user" | The user | same [EVIDENCE] |
| GitHub App **installation token** | 5,000/hour base. Over 20 repos or 20 org users, +50/hour per repo and per user, capped at 12,500. Enterprise Cloud: 15,000 | **The installation** | same; GraphQL equivalent in points at [GraphQL limits](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api) [EVIDENCE] |
| OAuth app client ID and secret, public data only | 5,000/hour per app (15,000 if owned by an Enterprise Cloud org) | **The app, one site-wide pool** | REST rate limits [EVIDENCE] |
| `GITHUB_TOKEN` in Actions | 1,000/hour per repository (15,000 for Enterprise Cloud) | The repository | REST rate limits [EVIDENCE] |
| GraphQL | 5,000 points/hour per user or per installation. Users acting through Enterprise Cloud-owned apps get 10,000 | Same split as REST | [GraphQL limits](https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api) [EVIDENCE] |

**Points that matter for the design:**

- **Secondary limits.** No more than 100 concurrent requests (shared by REST and GraphQL), 900 REST points/min, 2,000 GraphQL points/min, and 2,000 OAuth token requests per hour. These "may change without notice" ([REST rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)) [EVIDENCE].
- **Sharing between apps.** "requests made by a higher-limit app reduce the remaining budget available for lower-limit authentication methods" (same page) [EVIDENCE]. In practice, a signed-in user's live build competes with every other tool that user runs.
- **Conditional requests are free when authenticated.** A request "does not count against your primary rate limit if a `304` response is returned" and it "was made while correctly authorized with an `Authorization` header" ([REST best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api)) [EVIDENCE]. This makes authenticated polling with an ETag close to free. Unauthenticated 304s are not covered.
- **Unauthenticated access was tightened in 2025.** Since May 2025, unauthenticated limits also cover "cloning repositories over HTTPS, anonymously interacting with our REST APIs" and "downloading files from raw.githubusercontent.com". The stated reason is scraping ([GitHub changelog 2025-05-08](https://github.blog/changelog/2025-05-08-updated-rate-limits-for-unauthenticated-requests/)) [EVIDENCE]. The entry I read gives no numbers for clone or raw downloads.
- **API terms.** "Abuse or excessively frequent requests to GitHub via the API may result in the temporary or permanent suspension", and "You may not share API tokens to exceed GitHub's rate limitations" ([GitHub ToS §H](https://docs.github.com/en/site-policy/github-terms/github-terms-of-service)) [EVIDENCE].
- **New token format.** Since 2026-04-27, installation tokens are being rolled out in a stateless format, `ghs_APPID_JWT`. Code that assumes 40 characters may break ([installation token docs](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)) [EVIDENCE].

## 2. Build triggers

| Trigger | How it behaves | Fit for "public at release, live for signed-in visitors" |
|---|---|---|
| **App webhook `release`** | Available to GitHub Apps with Contents read. Actions: created, deleted, edited, prereleased, published, released, unpublished ([webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads)) [EVIDENCE]. A "latest" release is "the most recent non-prerelease, non-draft release, sorted by the created_at attribute" ([releases API](https://docs.github.com/en/rest/releases/releases)) [EVIDENCE]. | **Best fit for the public snapshot.** It fires exactly at release, centrally for every installation ("a single webhook", per [Apps vs OAuth](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps)). |
| **Owner's workflow pushes to the site** | The workflow runs on `release`. `published` covers both stable releases and pre-releases; `prereleased` does not fire for a pre-release published from a draft ([workflow triggers](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)). The site can authenticate the push with Actions OIDC: custom `aud`, `id-token: write`, claims including `repository_id` and `repository_visibility`, and the advice to require a matching `repository_id` ([OIDC reference](https://docs.github.com/en/actions/reference/security/oidc)) [EVIDENCE]. | **Good for public pages only.** It costs the site zero GitHub requests. It cannot give a live view: nothing runs between releases. Releases created with `GITHUB_TOKEN` will not start this workflow (workflow-triggers page; the statement sits in the `pull_request` section). |
| **Scheduled polling** | GitHub says: "subscribe to webhook events instead of polling" ([REST best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api)) [EVIDENCE]. Scheduled workflows in public repos are disabled after 60 days without activity ([workflow triggers](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)) [EVIDENCE]. | **Use only as a backstop.** Webhooks are lost when the site is down: "GitHub does not automatically redeliver failed deliveries" ([failed deliveries](https://docs.github.com/en/webhooks/using-webhooks/handling-failed-webhook-deliveries)) [EVIDENCE]. A daily conditional `GET /releases/latest` on the installation token returns 304 for free and catches the misses. |
| **On request** | pkg.go.dev works this way: a "Request" button and proxy fetches, plus monitoring the module index every few minutes ([pkg.go.dev/about](https://pkg.go.dev/about)) [EVIDENCE]. | **Right for the signed-in live view only,** charged to the visitor's token. For anonymous visitors it is the abuse path: queue it and rate-limit it, never build inline. |

**Webhook hygiene** [EVIDENCE]:
- Verify `X-Hub-Signature-256` with a constant-time comparison ([validating deliveries](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries)).
- Answer with a 2xx within 10 seconds and process from a queue ([webhook best practices](https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks)).
- Deduplicate on `X-GitHub-Delivery`. A redelivery carries the same ID (same page).

## 3. Signed-in visitors pay for their own view, and their access is checked first

**Token type: a GitHub App user access token, not OAuth** [EVIDENCE]:
- **Narrow reach.** A user access token "only has permissions that both the user and the app have". The app "can only access resources in an account where it is installed" ([user token generation](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app); [acting on behalf of a user](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user)).
- **Charged to the user.** Requests "will be attributed to that user". Rate limits follow the user's limits ([REST rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)).
- **Lifetimes.** The token expires after 8 hours and the refresh token after 6 months. Use the web flow with `state` and PKCE (user token generation page).
- **OAuth is broader than needed.** The `repo` scope "Grants full access to public and private repositories", read and write, and no read-only private scope is listed ([OAuth scopes](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps)). "you can't scope source code access to read-only" ([authorizing OAuth apps](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/authorizing-oauth-apps)). GitHub says "GitHub Apps are preferred to OAuth apps" ([Apps vs OAuth](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps)).
- **Consequence:** a contributor gets a live view of a private repo only if its owner installed the App on it. That matches the opt-in ruling. A practitioner thread confirms the trap: a user token saw no private repos until the App was installed on the account ([community discussion #36463](https://github.com/orgs/community/discussions/36463), 2022–2025) [ANECDOTE, consistent with the docs].

**Access check before serving a cached private page.** Two options:

| Option | Cost and who pays | Notes |
|---|---|---|
| **Visitor's token: `GET /repos/{owner}/{repo}`** | 1 request on **the visitor's** budget. 0 with an ETag and a 304 | Works with user access tokens; needs Metadata read ([GitHub App permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps)) [EVIDENCE]. GitHub returns 404 rather than 403 "to avoid confirming the existence of private repositories" ([troubleshooting](https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api)) [EVIDENCE], so treat any non-200 as a refusal. The response includes a `permissions` object, but the page does not say outright that it reflects the caller ([repos API](https://docs.github.com/en/rest/repos/repos#get-a-repository)). Do not rely on its fields without testing. |
| **Collaborator permission endpoint: `GET /repos/{o}/{r}/collaborators/{username}/permission`** | 1 request on **the installation's** budget (the site side) on every check | It returns `permission` (admin/write/read/none) and `role_name` ([collaborators API](https://docs.github.com/en/rest/collaborators/collaborators)). It is callable with an installation token under Metadata read (permissions page) [EVIDENCE]. The neighbouring "check if collaborator" endpoint requires push access for classic tokens. |

**Recommendation [INFERENCE]:** use the visitor's own token. Cache the yes/no decision per `(github_user_id, repository_id)` with a short TTL, accepting a short lag after access is revoked. Drop all of a user's tokens on `github_app_authorization` revocation ([webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads)).

## 4. Cache keys that keep private pages private

- **The HTTP standard.** "A shared cache MUST NOT use a cached response to a request with an Authorization header field" unless the response explicitly allows shared storage. `private` means "a shared cache MUST NOT store the response". `Vary` only controls whether a stored response may be reused ([RFC 9111](https://www.rfc-editor.org/rfc/rfc9111.html) §3.5, §5.2.2.7, §4.1) [EVIDENCE].
- **The shared cache holds public pages only** [INFERENCE]. Key them on `(repository_id, release tag's commit sha, renderer version)`, never on owner/name, which changes on rename or transfer. Store a page only if the repo was `private: false` both when it was built and when it is served.
- **Private pages bypass the shared cache** [INFERENCE]. Keep them at the origin, keyed on `(repository_id, sha)`, marked `Cache-Control: private` (or `no-store`), and check access on every request as in §3.
  - Serve them from a separate hostname, never from the same paths as public pages. GitHub uses this pattern itself: private Pages sites get "a unique subdomain" ([Pages visibility](https://docs.github.com/en/pages/getting-started-with-github-pages/changing-the-visibility-of-your-github-pages-site)) [EVIDENCE].
  - Do not count on `Vary: Cookie` at a CDN for isolation.
- **Purge on these events** ([webhook events](https://docs.github.com/en/webhooks/webhook-events-and-payloads)) [EVIDENCE for the events; INFERENCE for the policy]:
  - `repository` privatized, deleted or transferred.
  - `installation_repositories` removed, and `installation` deleted. The repo has left the opt-in.
  - A repo made private must lose its public release snapshot at once, or the last-release page keeps leaking it.
- **Keep the two kinds of build apart** [CONTESTED reading of the ToS]. Never let a build made with a signed-in user's token fill the anonymous public cache. One reading of §H, "You may not share API tokens to exceed GitHub's rate limitations", is that serving everyone from one person's quota is exactly that. The live and release builds are also different snapshots, so they need different keys anyway.

## 5. How comparable services do it

| Service | Opt-in | Rebuild trigger | Private repos | Takedown / abuse |
|---|---|---|---|---|
| **Read the Docs** | A GitHub App with repo selection, replacing the older OAuth integration ([RTD git integration](https://docs.readthedocs.com/platform/stable/reference/git-integration.html)) | Webhooks: build the latest commit, sync versions from tags, PR builds (same page) | Paid Business plan only. Clones with a token "valid for one hour and scoped to the repository" (same). Private versions "will 404 when visited by people without viewing permissions" ([privacy levels](https://docs.readthedocs.com/platform/stable/commercial/privacy-level.html)). The SSO option grants access to anyone with permissions on the Git repo ([SSO](https://docs.readthedocs.com/platform/stable/commercial/single-sign-on.html)). **Revoking repo access stops builds, but the docs stay up** (git-integration page). | A DMCA policy is listed ([RTD policies](https://docs.readthedocs.com/platform/stable/reference/policies.html)); I did not open its text |
| **pkg.go.dev** | None needed: any module fetched through the proxy, or requested with the "Request" button | Index polled "every few minutes", plus on request | No private modules mentioned | The owner publishes a `retract`, or files a request to "hide your package documentation". Only limited information is shown when no suitable license is detected ([pkg.go.dev/about](https://pkg.go.dev/about)) |
| **Vercel** | GitHub App. A personal-account repo can only be connected by its **Owner**. In an org, an Owner or a member with access; outside collaborators cannot connect | "deploy every push by default". Pull requests from forks need authorisation. A GitHub Actions `--prebuilt` path deploys without giving Vercel the source ([Vercel for GitHub](https://vercel.com/docs/git/vercel-for-github), updated 2026-09-17) | Supported through the App | Not covered on that page |
| **Netlify** | GitHub App that installs on all repos or selected ones. Tokens "expire after one hour" ([Netlify repo permissions](https://docs.netlify.com/git/repo-permissions-linking/)) | Continuous deployment on push | Deploy keys for private submodules | Not covered |
| **GitHub Pages (private)** | n/a | n/a | Enterprise Cloud only. Viewable only by "people with read access to the repository" ([Pages visibility](https://docs.github.com/en/pages/getting-started-with-github-pages/changing-the-visibility-of-your-github-pages-site)) | n/a |

**Patterns across these services** [CONSENSUS]:
- Every hosted builder has moved to a GitHub App with repo selection and one-hour, repo-scoped tokens (Read the Docs, Netlify, Vercel).
- Private content is a paid or enterprise tier everywhere.
- Unauthorised visitors get a 404, never a 403.
- Read the Docs leaves stale content up after access is revoked. abcdev should not copy that for private repos.

**Legal basis for showing repo content off GitHub** [EVIDENCE]:
- The licence users grant each other in GitHub's ToS covers use only "through the Service as permitted by GitHub's functionality" (§D.5). So it does not cover display on abcdev.app.
- §D.8 adds that the Terms "do not restrict lawful access to or use of the contents of public repositories by third parties" ([GitHub ToS](https://docs.github.com/en/site-policy/github-terms/github-terms-of-service), effective 2026-04-27).
- So the right to display rests on the **repository's own licence plus the owner's opt-in**, not on GitHub's terms.
- Private content is "confidential" to its owner (§E).
- GitHub's Acceptable Use Policy requires using personal information only "for the purpose for which that User has authorized it", securing it, and acting on removal requests ([AUP](https://docs.github.com/en/site-policy/acceptable-use-policies/github-acceptable-use-policies)). Commit author names and emails in a development record fall under this.

---

## Ranked recommendations (best value for effort first)

1. **Make "config file plus App installed on the repo" the opt-in.** Ask for only Contents:read and Metadata:read. Subscribe to `release`, `repository`, and the default `installation`, `installation_repositories` and `github_app_authorization` events. Each owner's builds then spend their own installation's budget, and the same install unlocks user tokens for private repos. *Source: Apps vs OAuth; REST rate limits; webhook events. [EVIDENCE]*
2. **Never call GitHub while serving an anonymous request.** Anonymous visitors read release snapshots only. Build on `release` `published` with an installation token down-scoped to that one repo via `repositories` ([installation tokens](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)). An anonymous cache miss puts a job in a queue with a per-repo cooldown and a global limit, never an inline build. Schedule builds by `X-RateLimit-Remaining`. *[EVIDENCE for the mechanics; INFERENCE for the policy]*
3. **Use GitHub App user access tokens for signed-in visitors.** Web flow with PKCE and `state`. Live builds and access checks are charged to the visitor. Store refresh tokens as secrets that stay valid for 6 months. *Source: user token generation page. [EVIDENCE]*
4. **Check access with the visitor's token,** `GET /repos/{owner}/{repo}` with an ETag, and cache the decision briefly per `(user_id, repository_id)`. *Source: troubleshooting (404 hides existence); best practices (free 304). [EVIDENCE + INFERENCE]*
5. **Keep public and private caches separate:** public pages in the shared CDN keyed on `repository_id`; private pages origin-only, marked `private`, on a separate host, purged on visibility change, removal or uninstall. *Source: RFC 9111; Pages unique subdomain. [EVIDENCE + INFERENCE]*
6. **Fetch each build in as few requests as possible:** one archive download per build instead of a contents call per file, GraphQL for metadata, conditional requests everywhere. This protects the visitor's shared 5,000/hour. *[INFERENCE; archive-download billing unverified, see risks]*
7. **Run a daily backstop** that redelivers missed webhooks through the App webhook deliveries API and makes a conditional `GET /releases/latest`. *Source: failed deliveries; best practices. [EVIDENCE]*
8. **Publish a removal path before launch:**
   - The owner removes the config or uninstalls the App, and the site purges at once.
   - Third parties get a request form (the pkg.go.dev model) and a DMCA channel.
   - Personal-information removal requests are honoured, per the AUP.
   - *[CONSENSUS across RTD and pkg.go.dev; EVIDENCE for the AUP duty]*
9. **Optional: let owners push builds from their own workflow** using OIDC (custom `aud`, require `repository_id` and `event_name`), for owners who will not install an App. **Public only**, no live view. *Source: OIDC reference. [EVIDENCE]*

## Not worth adopting

- **OAuth App sign-in.** Getting private repos needs `repo`, which is full read and write on every private repo the user can reach, with no read-only option. Rate limits still fall on the user, so it gains nothing over a GitHub App. *(OAuth scopes; authorizing OAuth apps)*
- **Unauthenticated fetches by the server for anonymous builds.** 60/hour per IP, and since 2025 clone and raw downloads are throttled too. *(REST rate limits; 2025 changelog)*
- **OAuth client ID and secret for public data.** One 5,000/hour pool for the whole site: exactly the allowance anonymous traffic would drain.
- **Polling as the main trigger.** GitHub advises webhooks instead, and owner-side scheduled workflows go dormant after 60 days.
- **The collaborator permission endpoint as the gate on every request.** It charges the site's installation for every signed-in page view. Keep it only as a fallback.
- **Filling the public cache from signed-in users' builds.** It risks the §H rule on sharing tokens, and it mixes the release and live snapshots.
- **Private GitHub Pages as the base.** Enterprise Cloud only.

## Open risks

1. **Unpublished limits.** Secondary limits "may change without notice". The 2025 changelog gives no numbers for clone or raw downloads. I found no document saying whether the archive download's redirect to codeload counts against the REST budget.
2. **The visitor's budget is shared** with every other tool they use ("higher-limit app reduce the remaining budget"). A large live build can starve their other tools, so each build needs a request cap.
3. **Access lags after revocation.** The access-decision TTL is how long a removed collaborator can still see cached private pages. Choose it deliberately.
4. **No automatic redelivery.** A missed `release` or `privatized` webhook leaves the site stale, or worse, still showing a now-private repo publicly, until the backstop runs. Make visibility purges fail closed: re-check `private` at serve time for public pages whose last check is old.
5. **Org-owned private repos need the org to install the App.** Contributors cannot turn on the live view themselves. This is a product constraint, not a bug.
6. **Legal footing is the repo's licence plus opt-in, not GitHub's ToS (§D.5).** Records of unlicensed repos and personal information inside records need a policy. pkg.go.dev limits what it shows without a suitable licence.
7. **Unread details.** I could not confirm whether `GET /repos`' `permissions` object describes the caller. I did not open the text of Read the Docs' DMCA policy.
8. **Open product question.** What does an anonymous visitor see for an opted-in public repo with no releases yet? The rulings leave this undefined.
9. **Token handling.** The 2026 installation-token format change means code must not assume a token length. Six-month refresh tokens make the token store a high-value target.

## Sources

- https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
- https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api
- https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/rate-limits-for-github-apps
- https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api
- https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api
- https://docs.github.com/en/rest/collaborators/collaborators
- https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps
- https://docs.github.com/en/rest/repos/repos#get-a-repository
- https://docs.github.com/en/rest/releases/releases
- https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps
- https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps
- https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/authorizing-oauth-apps
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-with-a-github-app-on-behalf-of-a-user
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app
- https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app
- https://docs.github.com/en/webhooks/webhook-events-and-payloads
- https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
- https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks
- https://docs.github.com/en/webhooks/using-webhooks/handling-failed-webhook-deliveries
- https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows
- https://docs.github.com/en/actions/security-for-github-actions/security-hardening-your-deployments/about-security-hardening-with-openid-connect
- https://docs.github.com/en/actions/reference/security/oidc
- https://docs.github.com/en/pages/getting-started-with-github-pages/changing-the-visibility-of-your-github-pages-site
- https://docs.github.com/en/site-policy/github-terms/github-terms-of-service
- https://docs.github.com/en/site-policy/acceptable-use-policies/github-acceptable-use-policies
- https://github.blog/changelog/2025-05-08-updated-rate-limits-for-unauthenticated-requests/
- https://github.com/orgs/community/discussions/36463
- https://www.rfc-editor.org/rfc/rfc9111.html
- https://docs.readthedocs.com/platform/stable/reference/git-integration.html
- https://docs.readthedocs.com/platform/stable/commercial/privacy-level.html
- https://docs.readthedocs.com/platform/stable/commercial/single-sign-on.html
- https://docs.readthedocs.com/platform/stable/reference/policies.html
- https://pkg.go.dev/about
- https://vercel.com/docs/git/vercel-for-github
- https://docs.netlify.com/git/repo-permissions-linking/
