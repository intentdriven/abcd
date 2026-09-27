---
id: itd-2609061543533170
slug: abcd-sets-up-a-managed-repository-s-release-rendered-site-en
spec_id: spc-2609212141407459
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609212103568351]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-100, itd-131]
related_adrs: [adr-2609212115255771]
---

# One verb takes a managed repository's site from the checkout to a live address

## Press Release

> **One verb sets up a managed repository's site end to end, and the same pages abcd's own site has render from that repository's record.**
>
> "The explorer, the graph, the timeline: I had them for abcd and wanted them for every repository abcd manages," said a product thinker looking at the record browser. "Now one verb writes the workflow and the environments, and when I have given abcd a hosting credential it creates and routes the host too. Any managed repo's site looks like abcd's, with its own record in it."

## Why This Matters

The renderer already exists: `abcd site build` composes the landing page from the identity block and the record export into the explorer, record pages, the relationship graph, the timeline and the glossary, and it is how abcd's own site is made. What no record covered was the distance from the verb to a live address for a repository abcd manages. Ruled 2026-09-21: everything, when a credential is present; the provider behind an adapter seam; the pages the same set for every repository, switched off per repository, never on to something extra.

## Mechanism

We expect a managed repository whose site is live to be read by people who never open the tree, because the record is written to be read and a checkout is where nobody reads it; shown wrong if no managed repository's site gains a reader within a release of it shipping.

## Scope Conditions

- Holds for a repository abcd manages whose forge runs the release workflow abcd scaffolds; a forge without workflows is out of reach. <!-- cond: cond-2609212141406389 -->
- Holds where the hosting provider has an API the adapter can create and route through; a provider without one is the person's step. <!-- cond: cond-2609212141407649 -->

## What's In Scope

- **`abcd site setup`**: writes the site composition from the identity block and the record, the render-on-release-then-deploy workflow and the environments it needs, and prints the exact remaining step for the person.
- **The credentialled path**: with a hosting credential configured, the same verb creates and routes the host through the provider adapter and reports the live address; without one it stops at the step above and says so.
- **The provider seam**: one adapter ships (the provider abcd's own site uses); a second is a later intent, not a change to the verb.
- **The pages**: landing, explorer, record pages, graph, timeline, glossary and status render for every managed repository from its own text; the site configuration switches pages off.
- **Re-runnable and credential-clean**: a second run changes nothing current and says so; the hosting credential is resolved by name through the credential store (itd-2609221017023290) and never written into the repository.
- **Security review** before the lane ships.

## What's Out of Scope

- A second provider.
- Custom pages beyond the set.
- Any change to the renderer's page shapes.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Everything when a credential is present; the person's step otherwise (ruled 2026-09-21).
2. The provider is an adapter behind a seam, one shipped.
3. The same pages for every repository, opt-out per page.

Taken in the implementing lane (autonomous run A, 2026-09-26), within the rulings above:

4. **The credential's interim source (2026-09-26).** The credential store this intent resolves through is itd-2609221017023290, planned and not built. Until it lands, the hosting credential is read by name through one narrow interface (`internal/core/credential`, `Resolve(name)`) from one machine-scoped file, `~/.abcd/credentials.json`, refused unless it is a regular file this uid owns at mode 0600 or tighter. itd-2609221017023290 is the successor: it replaces the source behind the interface, and no reader changes.
5. **Secrets are the person's step (2026-09-26).** The forge encrypts an environment secret before it accepts it, and doing that here would add a dependency and pass the value through abcd. The verb reads which secret names the deploy environment holds and prints the exact `gh secret set` command for each missing one.
6. **Render on release, whoever made it (2026-09-26).** The workflow runs on `release: published`, on the `release` workflow completing on the default branch (a release created with the workflow's own token fires no release event), and on dispatch. It renders with abcd's latest release, checksum- and attestation-verified, so the file does not change when abcd does.

   *Addendum (2026-09-26, the lane's security review, iss-2609260120384106).* As first written the workflow never fired by itself on a repository `launch scaffold` laid out: `auto-release` runs `release` by `workflow_call`, inside its own run, so no run named `release` completes; a hand-pushed tag's `release` run has the tag as its head branch, which the trigger's branch filter dropped; and the token-made release fires no `release: published`. The `workflow_run` entry therefore names both `release` and `auto-release` and filters no branch, and the render job's gate admits a head branch that is the default branch or starts with `v`, beside the fork and pull-request conditions it already held. `auto-release` completes on every push to the default branch, so a push that released nothing redeploys the latest published release: the bytes the site already serves, unless a dispatch rolled it back to an older tag, which that push replaces with the latest release again.
7. **The page set's edges (2026-09-26).** The status page is the record health page; the timeline is the genealogy the dashboard carries; the landing page and the record pages cannot be switched off beneath the explorer. The composition setup derives quotes the recorded identity block and composes the landing page from `docs/README.md`, and the static inputs are seeded as byte copies of abcd's own.
8. **The account is the token's (2026-09-26).** The provider account is the one the credential reaches; a token reaching none or several is refused rather than guessed at, so no account identifier is configured anywhere.
9. **The supply-chain posture and the exit-1 output, named (2026-09-26, the lane's security review).** Accepted as they stand, and named here so no reader has to infer them. The render job takes abcd's latest release, bound to `intentdriven/abcd`'s release workflow by an attestation check; the checksums ship in the same release, so they prove transfer integrity only. A compromised abcd release therefore runs in a job holding a read-only token and can deface the site, and cannot reach the deploy secrets, because the deploy job runs the provider's tool and not abcd. That tool, wrangler, is installed from npm at a named version with no integrity pin, as in abcd's own site workflow. The environments admit tags `v*` and setup checks no tag ruleset, so the right to push a tag is enough to run modified workflow content in `site`. A `--json` run that exits 1 prints two JSON documents, the result and then the CLI's error envelope; that is the repository-wide convention, and a strict consumer of this verb's `--json` reads both.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a managed repository, **when** `abcd site setup` runs without a hosting credential, **then** the composition, the render-then-deploy workflow and the environments are written, and the exact remaining step is printed.
- **Given** a hosting credential configured, **when** the verb runs, **then** the host is created and routed through the adapter and the live address is reported; nothing is written into the repository but the files above.
- **Given** the provider seam, **when** the adapter list is read, **then** one provider is present and the seam's interface is the one a second would implement.
- **Given** any managed repository, **when** its site renders, **then** the page set is abcd's own (landing, explorer, record pages, graph, timeline, glossary, status) from that repository's text, with pages switched off per its configuration.
- **Given** a second run, **when** nothing has changed, **then** it writes nothing and says so.
- **Given** the lane, **when** it ships, **then** a security review of the workflow writes and the provider calls is on its record.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-070279698e5f -->
Fidelity review — receipt rcp-070279698e5f (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:82d9f21abe685773ec84da6e26aae97b2d9f21bf899ac42b23c5c1692ad6289e
Input attestations: diff:a4d0980a^1..a4d0980a (PR #722 feat/site-setup, read at main 811fba17)@sha256:a0d059f6b94e86f8fe112bf5bb06de3b73a44d7472fc0bade8ada6effb6255fd;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: with no credential the harness test sees every setup file written on disk, both forge environments created with branch+tag policies, the provider never called, and the remaining steps naming the credential store path, the gh secret set commands and git add; the credential step is composed at setup.go:290-296
  evidence: internal/core/site/setup_test.go:220 — "func TestSetupWithoutACredentialWritesTheRepositoryHalfAndSaysWhatRemains"
  evidence: internal/core/site/setup.go:290 — "if hc.Status == HostNoCredential {"
  evidence: internal/core/site/setup.go:752 — "cmd := "gh secret set " + name + " --env " + EnvDeploy"
- ac-2 — MET: with a credential the fake host ends holding the worker and the routed domain, the address is reported, and git status -uall equals exactly the setup file list with the token in none of them; the host stage runs Inspect then Create/Route through the Provider interface after confirmation
  evidence: internal/core/site/setup_test.go:267 — "func TestSetupWithACredentialCreatesRoutesAndReportsTheHost"
  evidence: internal/core/site/setup_test.go:308 — "the repository changed in\n %v\nwant exactly"
  evidence: internal/core/site/setup.go:664 — "func setupHost(ctx context.Context, adapter hosting.Adapter"
  evidence: internal/adapter/hosting/cloudflare/cloudflare.go:365 — "func (c *client) Create(ctx context.Context, s hosting.Site) error"
- ac-3 — MET: the provider list is one adapter, cloudflare, and a test pins Providers() to exactly [cloudflare]; the seam is the hosting.Adapter interface (repository half as data, Connect for the host half) which is what a second provider implements
  evidence: internal/core/site/providers.go:13 — "var adapters = []hosting.Adapter{cloudflare.Adapter{}}"
  evidence: internal/adapter/hosting/hosting.go:58 — "type Adapter interface {"
  evidence: internal/core/site/setup_test.go:336 — "func TestTheProviderListHasOneAdapterBehindTheSeam"
- ac-4 — MET_WITH_CONCERNS: the composition setup writes builds every page of the set from the fixture repository's own text and the switches remove pages and their links; the concern is that the set's edges are narrower than the bullet reads: the timeline is a panel of the dashboard rather than a page and status is the record health page (Decision 7), and the rendered header links a docs page a managed repository does not build (open iss-2609260928152365)
  evidence: internal/core/site/setup_test.go:349 — "func TestSetupGivesAManagedRepositoryTheWholePageSet"
  evidence: internal/core/site/pages.go:37 — "var PageNames = []string{"landing", "explorer", "record_pages", "graph", "timeline", "glossary", "status"}"
  evidence: internal/core/site/pages_test.go:97 — "func TestASwitchedOffPageIsGoneAndNothingLinksToIt"
  evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:68 — "the timeline is the genealogy the dashboard carries"
  evidence: .abcd/work/issues/open/iss-2609260928152365-site-header-links-docs-without-a-docs-build.md:1 — "site-header-links-docs-without-a-docs-build"
- ac-5 — MET: a second run after a credentialled first run reports no_change, every file current or kept, no forge write and no host write; the CLI prints the status on its first line
  evidence: internal/core/site/setup_test.go:375 — "func TestASecondRunWritesNothingAndSaysSo"
  evidence: internal/core/site/setup.go:331 — "res.Status = StatusNoChange"
  evidence: internal/surface/cli/site.go:296 — "fmt.Fprintf(w, "abcd site setup — %s\n", termsafe.Sanitize(res.Status))"
- ac-6 — MET_WITH_CONCERNS: the lane's security review is on the record as Decision 6's addendum and Decision 9 (the workflow's supply-chain posture and the provider calls' blast radius, named) and as four category-security/bug findings captured from the review of lane sitesetup, three resolved in the same PR; the concern is that no dated review directory under .abcd/work/reviews/ holds the review itself, so its scope is recoverable only from its findings and the decisions that summarise it
  evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:70 — "9. **The supply-chain posture and the exit-1 output, named (2026-09-26, the lane's security review)."
  evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:67 — "*Addendum (2026-09-26, the lane's security review, iss-2609260120384106).*"
  evidence: .abcd/work/issues/resolved/iss-2609260120383554-abcd-site-setup-rewrites-an-existing-deployment-environment.md:8 — "found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane sitesetup)""
  evidence: .abcd/work/issues/open/iss-2609260120380520-the-machine-credential-file-internal-core-credential-abcd.md:6 — "category: "security""

Gap audit:
- honoured:
  - one verb writes the composition, the render-then-deploy workflow and the environments, and prints the remaining step
    evidence: internal/core/site/setup_test.go:220 — "TestSetupWithoutACredentialWritesTheRepositoryHalfAndSaysWhatRemains"
  - with a credential the host is created and routed and the live address reported; the credential never enters the repository or the report
    evidence: internal/core/site/setup_test.go:267 — "TestSetupWithACredentialCreatesRoutesAndReportsTheHost"
    evidence: internal/adapter/hosting/cloudflare/cloudflare_test.go:137 — "func TestTheCredentialNeverReachesAnError"
  - one provider behind a seam a second would implement
    evidence: internal/adapter/hosting/hosting.go:58 — "type Adapter interface {"
  - the site renders on release whoever made it, fork runs never render
    evidence: internal/core/site/setupsrc/site.yml.tmpl:46 — "workflows: [release, auto-release]"
    evidence: internal/core/site/setup_test.go:766 — "func TestTheWorkflowFiresOnEveryReleasePath"
    evidence: internal/core/site/setup_test.go:743 — "func TestAWorkflowRunFromAForkNeverRenders"
  - re-runnable: a second run writes nothing and says so
    evidence: internal/core/site/setup_test.go:375 — "TestASecondRunWritesNothingAndSaysSo"
  - wired on the CLI and the plugin surface
    evidence: internal/surface/cli/site_setup_test.go:20 — "func TestSiteSetupIsWiredAndReachesNoNetwork"
    evidence: commands/site.md:89 — "## `setup` — take a managed repository's site to a live address"
- diverged:
  - the hosting credential is resolved through the credential store itd-2609221017023290; delivered through an interim owner-only ~/.abcd/credentials.json behind credential.Resolve, signed off as Decision 4 with the store intent named as successor
    evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:63 — "4. **The credential's interim source (2026-09-26)."
    evidence: internal/core/credential/credential.go:1 — "package credential"
  - everything when a credential is present (ruling 1); delivered with the deploy secrets left as the person's step, the verb printing one gh secret set command per missing secret, signed off as Decision 5
    evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:64 — "5. **Secrets are the person's step (2026-09-26)."
    evidence: internal/core/site/setup.go:728 — "func secretSteps("
  - the page set names timeline and status as pages; delivered as the dashboard's genealogy panel and the record health page, signed off as Decision 7
    evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:68 — "7. **The page set's edges (2026-09-26)."
    evidence: internal/core/site/setup_test.go:366 — "the managed repository's dashboard lacks the timeline"
- missing:
  - a review artefact for the lane's security review under .abcd/work/reviews/ (the charter's home for a review conducted outside abcd's verbs); only its findings and the decisions summarising it are on the record
    evidence: .abcd/work/reviews/README.md:1 — "# Reviews"
    evidence: .abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s-release-rendered-site-en.md:70 — "named here so no reader has to infer them"

Scope-condition dispositions:
- cond-2609212141406389 — survived: the written workflow fires on release: published, on the release and auto-release workflows completing, and on dispatch, and a test asserts each path; a forge without workflows has nothing to run it
  evidence: internal/core/site/setupsrc/site.yml.tmpl:46 — "workflows: [release, auto-release]"
  evidence: internal/core/site/setup_test.go:766 — "func TestTheWorkflowFiresOnEveryReleasePath"
- cond-2609212141407649 — survived: the one shipped provider creates and routes through its API and the no-credential path leaves the host as the person's step in the provider's console, as the condition assumed
  evidence: internal/adapter/hosting/cloudflare/cloudflare.go:379 — "func (c *client) Route(ctx context.Context, s hosting.Site) error"
  evidence: internal/core/site/setup.go:296 — "in the provider's own console"

## Grounds

- pursued: the renderer is built and proved on abcd's own site; the gap is only the verb from the checkout to a live address; we expect a managed repository's site to gain readers who never open the tree; shown wrong if none does within a release
