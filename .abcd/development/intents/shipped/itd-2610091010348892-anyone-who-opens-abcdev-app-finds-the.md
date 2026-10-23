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
related_intents: [itd-2610052000411162, itd-2610142145236613]
related_adrs: [adr-47, adr-2610091010387033]
impact: additive
---

# An opted-in project's record is built at each release onto its own GitHub Pages

## Press Release

> When a project switches it on at setup, abcd's release workflow builds the project's about page and development record at each release and publishes them on the project's own GitHub Pages, for example acme.github.io/widgets/. Every page opens with a line saying it is the record of an abcd-managed repository, linking to the repository, and every word on it comes from the repository. The project's own workflow builds it and GitHub hosts it; abcd runs nothing.

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

- **Given** setup in a public abcd-managed repository, **when** the owner answers yes to a project site, **then** the switch is set in the project's abcd settings and abcd's release workflow is added, and nothing else in the repository changes: no file is written for the website.
- **Given** a project with the switch on, **when** it publishes a release, **then** its about page and development record appear at its own GitHub Pages address, as of that release.
- **Given** any page of the site, **when** a visitor reads it, **then** every word on it comes from what abcd writes to the repository anyway (its identity block, brief, development record and changelog), and hand-written documentation is not rendered: it is read on GitHub.
- **Given** any page of the site, **when** a visitor opens it, **then** it opens with a line saying it is the record of an abcd-managed repository, linking to the repository.
- (technical) **Given** text on a page that cannot be traced to a file abcd writes, **when** the workflow builds, **then** the build fails, as the site's single-source rule already requires; the site's interface words, styles and scripts come from the abcd binary, never from files in the repository.
- (technical) **Given** the release workflow, **when** it runs, **then** it uses no secret beyond the short-lived token GitHub gives each run, and abcd runs nothing.
- (technical) **Given** the owner switches the site off, **when** the next release is published, **then** nothing is published, and abcd's setup can remove the workflow.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md). The 2026-10-14 notes on GitHub builds and on no-server options already settle the route; this pass checks the mechanics of a release-triggered Pages build.

- **The official actions and limits:** the deploy job needs `pages: write` and `id-token: write` and "should target the `github-pages` environment"; the artefact is "a single gzip archive containing a single tar file", with no symbolic or hard links, "under 10GB (we recommend under 1 GB!)" and an "officially supported maximum size limit" of 1 GB; sites may be no larger than 1 GB, deploys time out at 10 minutes, and the ten-builds-an-hour limit does not apply to a custom workflow. The current majors are `configure-pages@v6`, `upload-pages-artifact@v5` and `deploy-pages@v5` (each repository's tags, checked 2026-10-23); GitHub's starter workflow still pins older ones (`configure-pages@v5`, `upload-pages-artifact@v3`), with `concurrency: group: "pages"`. `upload-pages-artifact` excludes dotfiles unless `include-hidden-files: true`, which matters if the rendered site mirrors any dot-path.
- **A release-triggered run is on the tag, and the recommended environment rule refuses it:** GitHub recommends "a deployment protection rule so that only the default branch can deploy to this environment"; the rule "is matched against the `GITHUB_REF` of the workflow run"; branch and tag patterns are set separately. One project's release-triggered docs deploy failed with "Tag 'v1.0.1' is not allowed to deploy to github-pages" and was reworked to build from the default branch. The fix is a tag rule (`v*`) on the environment, set by the owner, or a job that checks out the release's commit while running on the default branch's ref.
- **The site lives under a path and the action says which:** `configure-pages` outputs `base_url`, `origin`, `host` and `base_path` ("/my-repo" or empty); its `enablement` input needs more than `GITHUB_TOKEN` (a PAT with `repo`, or an App with `administration:write` and `pages:write`). Enabling Pages with the Actions source is a one-time step the owner does by hand or setup does with the owner's own forge login, never with a key abcd holds.
- **A release created with `GITHUB_TOKEN` does not start a `release`-triggered workflow** (GitHub's general rule for `GITHUB_TOKEN` events, via the GitHub builds note); the Pages rule that such commits do not trigger a build is the branch-publishing source's, not the Actions source this route uses. Where abcd scaffolds the release workflow, the site job belongs inside it.
- **Private repositories:** an access-controlled Pages site needs Enterprise Cloud; on other plans the site is public (Pages visibility). The first scope condition stands; the later intent must say "your record would be public" before offering the switch.

**Path 1 for hosting — GitHub Pages through the project's own workflow and the three official actions, as ruled — with the renderer native and base-path-aware.** Setup prints the one settings step (enable Pages with the Actions source; add the tag rule if the job runs on the tag) as verified, linked instructions to a person. This pass came after the build was planned and had no fit-challenge before the intent shipped, so it stands as notes for a later revision.

Questions this pass raised, kept as notes for a later revision of the shipped work:

- The first criterion says "nothing else in the repository changes"; enabling Pages and the environment's tag rule are settings, not files. Whether setup does those steps through the owner's own login (an outbound call under adr-38's "when asked") or prints them as instructions, and whether the criterion should say so.
- Whether the site job is copied into each project's workflow or called as a reusable workflow from abcd's repository; the ruling's wording favours the copy, which setup then keeps current.

## Open Questions


## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-8ae5e776b270 -->
Fidelity review — receipt rcp-8ae5e776b270 (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:acc7493119fb9278e755bd63e127968180bdfbf470a36966e4e9900cc18ab7c7
Input attestations: diff:6fd30326d137092947945773dd9bf4c2b95952f2..8281224047292549fc71e9ab9f99fb1e30882041 (feat/hosted-sites)@sha256:98969c7180a6262d673c33126e19770148d85a0290b923ab28bc55a11071865e;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Setup asks the switch question before writing; a yes writes .abcd/site.json in project mode with hosting.publish true plus .github/workflows/site.yml and seeds no site sources; the test asserts the changed paths are exactly those two and that no composition key is present (passes at HEAD).
  evidence: internal/core/site/setup.go:362 — "if req.Asker == nil || !req.Asker.Confirm(switchQuestion(adapter, prof, takeover, manifestBytes == nil, project)) {"
  evidence: internal/core/site/setup.go:728 — "// A project site is seeded with nothing: its interface words, styles"
  evidence: internal/core/site/setup_pages_test.go:70 — "var pagesFiles = []string{ManifestRelPath, SiteWorkflowRelPath}"
  evidence: internal/core/site/setup_pages_test.go:94 — "if got := changedPaths(t, h.harness); !reflect.DeepEqual(got, want) {"
- ac-2 — MET_WITH_CONCERNS: The workflow fires on release: published, checks out the released tag, builds with --project below /< repo>/ and deploys through upload-pages-artifact/deploy-pages, and setup creates a workflow-built Pages site. Concern: the workflow downloads abcd's latest release, and no abcd tag contains the --project commit (54f5060ad is only on origin/feat/hosted-sites; latest tag v0.15.1), so a switched-on project's release build fails until abcd cuts a release carrying it, which setup itself admits in a note. No live end-to-end deploy to a real github.io address is evidenced; the proof is a golden file and an in-process fake.
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:54 — "release:"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:129 — "ref: refs/tags/${{ steps.release.outputs.tag }}"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:218 — ""${RUNNER_TEMP}/abcd" site build --project --out site --version "${TAG#v}" --commit "$(git rev-parse --verify HEAD)" --base-path /widgets/"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:285 — "uses: actions/deploy-pages@368f82528645a54fb793d4d04e342629a3f51346 # v5.0.1"
  evidence: internal/adapter/hosting/githubpages/githubpages.go:292 — "map[string]string{"build_type": "workflow"}"
  evidence: internal/core/site/setup.go:80 — "const projectReleaseNote = "the workflow builds with abcd's latest release, so its build fails until abcd publishes a release " +"
- ac-3 — MET_WITH_CONCERNS: The about page is built in code from the identity block (only when abcd wrote its file), the brief's opening, met shipped intents' press releases and CHANGELOG.md; the references page is dropped, and the test asserts no README/docs words on any page and every data-src span is abcd-written. Concerns: 'what abcd writes' is enforced as a path prefix (.abcd/** but the local tier, plus CHANGELOG.md), so a person's own file placed under .abcd/ passes as abcd's; and with no identity block the page is titled by the repository name from the origin remote, which is not a file abcd writes.
  evidence: internal/core/site/project.go:129 — "func abcdWrittenSource(rel string) bool {"
  evidence: internal/core/site/project.go:133 — "return strings.HasPrefix(rel, ".abcd/") && !strings.HasPrefix(rel, ".abcd/.work.local/")"
  evidence: internal/core/site/project.go:208 — "if !ok || !fsutil.ValidRelPath(cfg.Block.File) || !abcdWrittenSource(cfg.Block.File) {"
  evidence: internal/core/site/project_test.go:104 — "for _, words := range []string{readmeOnlyWords, docsOnlyWords, "Who this is for"} {"
  evidence: internal/core/site/project_test.go:129 — "if _, ok := files["references/index.html"]; ok {"
- ac-4 — MET: Every page's head is followed by the managed line, whose words are the binary's managed_record string, wrapped in a link to the repository address taken from the origin remote; a project test asserts it on every built .html page (passes at HEAD).
  evidence: internal/core/site/setupsrc/ui.json:107 — ""managed_record": "This is the record of an abcd-managed repository","
  evidence: internal/core/site/compose.go:303 — "b.WriteString(c.managedLine())"
  evidence: internal/core/site/compose.go:316 — "if c.repo.Repository != "" {"
  evidence: internal/core/site/project_test.go:261 — "func TestEveryProjectPageOpensWithTheManagedRepositoryLine(t *testing.T) {"
- ac-5 — MET: With --project the gate also fails any data-src span on any page whose file abcd does not write, on top of the existing word-provenance walk; the workflow runs lint site --project after the build; interface words come from the binary's embedded ui.json and the stylesheet/scripts/headers are copied from setupsrc, which the test compares byte for byte.
  evidence: internal/core/site/project.go:524 — "func (c *checker) checkProjectSources() {"
  evidence: internal/core/site/check.go:372 — "c.checkProjectSources()"
  evidence: internal/core/site/project.go:150 — "func projectUI() (UI, error) {"
  evidence: internal/core/site/project_test.go:173 — "func TestTheProjectGateRefusesASpanFromDocsOrTheReadme(t *testing.T) {"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:219 — ""${RUNNER_TEMP}/abcd" lint site --project --out site --base-path /widgets/"
- ac-6 — MET: The Pages workflow's only credential is github.token; the deploy job has exactly contents read, pages write and id-token write; the adapter declares no secrets and no credential name; a test refuses any secrets. reference and any shell interpolation. The build runs on the project's own runner, so no abcd service runs anything.
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:105 — "GH_TOKEN: ${{ github.token }}"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:242 — "permissions:"
  evidence: internal/adapter/hosting/githubpages/githubpages.go:73 — "func (Adapter) Secrets() []string { return nil }"
  evidence: internal/core/site/setup_pages_test.go:311 — "func TestThePagesWorkflowHoldsNoSecretAndInterpolatesNothingIntoAShell(t *testing.T) {"
- ac-7 — MET: The render job's switch step reads hosting.publish at both the released tag and the head of the default branch (missing or malformed reads as off) and gates every later step and the deploy job; the test runs the step with bash over real git checkouts. Setup with the switch off touches nothing remote and offers to remove the workflow, keeping a hand-edited one.
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:154 — "if ! git show "origin/main:.abcd/site.json" 2>/dev/null | jq -e '.hosting.publish == true' >/dev/null 2>&1; then"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:236 — "if: needs.render.outputs.publish == 'true' && toJSON(github.event.repository.private) == 'false'"
  evidence: internal/core/site/setup.go:1198 — "if req.Asker == nil || !req.Asker.Confirm("Remove "+SiteWorkflowRelPath+", the release workflow abcd added? The project site is switched off in "+"
  evidence: internal/core/site/setup_pages_test.go:238 — "func TestASwitchedOffSiteRemovesTheWorkflowAndPublishesNothing(t *testing.T) {"
  evidence: internal/core/site/setup_pages_switch_test.go:58 — "func TestTheSwitchIsReadAtTheTagAndTheDefaultBranchAndOnlyWhilePublic(t *testing.T) {"

Gap audit:
- honoured:
  - switching it on at setup costs one answer and writes only the switch and the workflow
    evidence: internal/core/site/setup.go:362 — "switchQuestion(adapter, prof, takeover, manifestBytes == nil, project)"
    evidence: internal/core/site/setup_pages_test.go:70 — "var pagesFiles = []string{ManifestRelPath, SiteWorkflowRelPath}"
  - the project's own workflow builds at each release and publishes to its own GitHub Pages address (owner.github.io/< repo>/)
    evidence: internal/adapter/hosting/githubpages/githubpages.go:116 — "return "https://" + strings.ToLower(owner) + ".github.io" + basePath(s)"
    evidence: internal/core/site/testdata/site-workflow-github-pages.yml:279 — "uses: actions/upload-pages-artifact@fc324d3547104276b827a68afc52ff2a11cc49c9 # v5.0.0"
  - every page opens with a line saying it is the record of an abcd-managed repository, linking to it
    evidence: internal/core/site/compose.go:314 — "func (c *composer) managedLine() string {"
  - GitHub hosts it and abcd runs nothing: no secret beyond the run's token
    evidence: internal/adapter/hosting/githubpages/githubpages.go:70 — "func (Adapter) CredentialName() string { return "" }"
    evidence: internal/core/site/testdata/site-workflow-github-pages.yml:48 — "github-pages the deploy job. Same policy. No secrets: the run's own token."
- diverged:
  - every word on it comes from the repository: enforced as a path prefix (.abcd/** and CHANGELOG.md), and the title can come from the origin remote's repository name rather than a file
    evidence: internal/core/site/project.go:133 — "return strings.HasPrefix(rel, ".abcd/") && !strings.HasPrefix(rel, ".abcd/.work.local/")"
    evidence: internal/core/site/project.go:508 — "title := c.repo.Name"
  - at each release: only a published, non-draft, non-pre-release GitHub release tagged vX.Y.Z is rendered
    evidence: internal/core/site/testdata/site-workflow-github-pages.yml:114 — "if ! printf '%s' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-+] [0-9A-Za-z.-]+)?$'; then"
    evidence: internal/core/site/testdata/site-workflow-github-pages.yml:121 — "echo "::error::$tag is a pre-release or a draft; the site renders published releases only""
- missing:
  - a working build at a switched-on project's next release today: the workflow uses abcd's latest release, and no released abcd carries site build --project yet
    evidence: internal/core/site/setup.go:80 — "the workflow builds with abcd's latest release, so its build fails until abcd publishes a release"
    evidence: internal/core/site/testdata/site-workflow-github-pages.yml:169 — "tags="$(gh release list --repo intentdriven/abcd --exclude-drafts --exclude-pre-releases \"

Scope-condition dispositions:
- cond-2610141709464490 — survived: The delivery encodes the public-only assumption at every layer: setup refuses a private repository or an unreadable visibility before asking or writing, the adapter's Inspect refuses one, and the workflow's switch step and deploy job publish nothing for a repository made private later.
  evidence: internal/core/site/setup.go:307 — "return SetupResult{}, fmt.Errorf("site setup: %s is private, and %s publishes public repositories only for now", forge.Repo(), adapter.Name())"
  evidence: internal/adapter/hosting/githubpages/githubpages.go:236 — "the project site is published for public repositories only"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:146 — "if [ "${REPO_PRIVATE:-}" != false ]; then"
- cond-2610141709468073 — narrowed: The workflow does rely on GitHub releases (release: published, gh release view), but it renders only a published, non-draft, non-pre-release release whose tag matches vX.Y.Z, and the environments admit only tags v*; a GitHub release under any other tag scheme publishes nothing.
  narrowing: holds only for published, non-draft, non-pre-release GitHub releases tagged vX.Y.Z (environment policy: tags v*), not for every GitHub release
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:114 — "if ! printf '%s' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-+] [0-9A-Za-z.-]+)?$'; then"
  evidence: internal/core/site/testdata/site-workflow-github-pages.yml:118 — "state="$(gh release view "$tag" --repo "$GITHUB_REPOSITORY" --json isPrerelease,isDraft \"
- cond-2610141709463916 — untested: Nothing in the delivery measures or bounds the built site's size against GitHub Pages' 1 GB limit, and no test builds a site near it; the assumption was neither exercised nor contradicted.
<!-- abcd-review-end receipt=rcp-8ae5e776b270 -->

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
- 2026-10-14: the move of abcd's own repository to this route is split out into its own intent, itd-2610142145236613, planned now and built later. The product thinker asked, verbatim: "keep working so that we have this intent planned and ready to implement. We'll have to delay building it but I want it to be ready". The press release's last sentence ("abcd's own record moves to the same route, built exactly the way any other project's is.") and the fourth criterion ("Given abcd's own repository, when it moves to this route and publishes a release, then its record is built the same way as any other project's, and its Cloudflare deploy can be retired.") move to that intent; this intent ships without them.
- 2026-10-15: the product thinker ruled that a project's site renders only what abcd already writes to the repository, as its single source, and nothing written for the website. Pictures are the exception: abcd is later to ask the person for images, or offer to generate some, depending on the context (captured as iss-2610150529485165). Read as: setup writes no composition and no site sources into the repository; the about page is composed from abcd's own writing; the interface words, styles and scripts ship in the binary. The first and fourth criteria are tightened and a criterion is added to say so.
- 2026-10-15: asked where hand-written docs live, the product thinker ruled that an abcd-managed repository's docs are read on GitHub by default, and that abcd's own docs are rendered in high quality on abcdev.app until a community site arrives. For this intent: a project's site renders no docs; its docs are read on GitHub.

## Grounds

- pursued: owners switch the project site on because it costs one answer at setup and nothing to run afterwards; shown wrong if owners who say yes find the pages missing or stale after a release
