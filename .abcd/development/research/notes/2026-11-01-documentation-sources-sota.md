# Which documentation is authored and which is generated, and a second authored surface for learning material

Two state-of-the-art reviews run on 2026-11-01 for itd-2611011151475128,
itd-2611011151512956 and itd-2611011239589823. Recency was assessed against
that date. Evidence tiers: CONSENSUS (several independent practitioners or a
framework's own definition), EVIDENCE (a case study or measurement), ANECDOTE
(one practitioner or a vendor's own claim).

## Review 1: authored or generated, by documentation type

| Type | Source | Reason | Strongest source |
|---|---|---|---|
| Explanation | Authored in the design record, a chosen part rendered | Explanation covers "why things are so", weighs alternatives and opinions; machine facts hold none of it. Diátaxis also warns explanation "tends to absorb other things", so a selection renders, not the whole record. | [Diátaxis: explanation](https://diataxis.fr/explanation/) |
| Reference | Generated from the command tree and schemas, descriptions written at the source | It "should mirror the structure of the product"; Diátaxis warns that generated reference is not all the documentation required. | [Diátaxis: reference](https://diataxis.fr/reference/) |
| How-to | Authored, its steps run as tests | Written "from the perspective of the user, not of the machinery"; a command tree holds no goals. | [Diátaxis: how-to guides](https://diataxis.fr/how-to-guides/) |
| Tutorial | Authored, its steps run as tests | "A pedagogical problem" needing a narrative and testing. | [Diátaxis: tutorials](https://diataxis.fr/tutorials/) |
| README | Composed: stable text from the record, generated regions for what changes | Stable knowledge is written, volatile knowledge is moved out. | [Martraire](https://speakerdeck.com/cyriux/leaving-documentation-or-living-documentation) |

Diátaxis itself never says which type is authored or generated; each placement
above is inferred from its definition of the type.

Findings, ranked:

1. Mark every generated file or region (CONSENSUS): Go's `Code generated … DO
   NOT EDIT.` line ([cmd/go](https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source));
   Tessl marks its output as generated from a spec ([Böckeler, 2025](https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html)).
2. Catch hand edits by regenerating and failing on any difference (CONSENSUS):
   living documentation kept in sync with its sources, failing the build when
   it is not ([Bolognesi on Martraire, 2022](https://engineering.facile.it/blog/eng/on-living-documentation)).
3. Generate CLI reference from the command tree (CONSENSUS):
   [cobra docgen](https://github.com/spf13/cobra/blob/main/site/content/docgen/_index.md).
4. Run how-to and tutorial steps as tests rather than generate them
   (CONSENSUS): [Go examples](https://go.dev/blog/examples),
   [rustdoc](https://doc.rust-lang.org/rustdoc/write-documentation/documentation-tests.html),
   [mdBook test](https://rust-lang.github.io/mdBook/cli/test.html),
   [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript),
   [Doc Detective](https://docs.doc-detective.com/) (the vendor's own claim).
5. Point a rendered page's edit link at its source (CONSENSUS):
   [Docusaurus `editUrl`](https://docusaurus.io/docs/api/plugins/@docusaurus/plugin-content-docs).
6. Reused content is read-only where it appears; edit the source (CONSENSUS):
   [PTC Arbortext](https://support.ptc.com/help/arbortext/r8.3.3.0/en/editor/tutorial/help17050.html).
7. The one mechanical write-back found depends on span-level link comments
   (ANECDOTE): [`org-babel-detangle`](https://doc.emacsen.de/30.2/fun/org-babel-detangle.html).
8. Where AI tools write documentation back, they propose a change for review
   and never commit it (ANECDOTE, vendor):
   [Mintlify Autopilot](https://www.mintlify.com/docs/autopilot). No tool or
   study was found that detects a hand edit to a rendered page and drafts the
   matching edit to its source; catching the edit is established, drafting the
   source edit is not.
9. Hand-written prose references go stale (EVIDENCE): most of more than 3,000
   GitHub projects carried an outdated code-element reference at some point
   ([Tan, Wagner and Treude](https://arxiv.org/abs/2212.01479)).

Not worth adopting: generating documentation from spec-driven tools (spec-first,
not spec-anchored over time, per Böckeler); embedding-based drift detectors
(vendor claims only); rendering the whole design record as user documentation;
forcing how-to guides into the command tree's shape.

## Review 2: a second authored surface for learning material

Verdict: supported, with changes.

1. Separate the owners, not the destination (EVIDENCE): MDN merged its
   separate curriculum into Learn web development because readers found two
   learning resources confusing ([MDN blog](https://developer.mozilla.org/en-US/blog/curriculum-learn-web-development),
   [discussion #750](https://github.com/orgs/mdn/discussions/750)); Rust keeps
   separate teams for the Book, Rustlings and the Reference in one
   organisation and toolchain ([Rust teams](https://rust-lang.org/governance/teams/lang/)).
2. How-to guides belong with the product owners, tutorials may belong to an
   education team (CONSENSUS, framework): a how-to serves a competent user at
   work, a tutorial a learner at study
   ([Diátaxis: tutorials and how-to](https://diataxis.fr/tutorials-how-to/)).
3. Make tutorials and exercises runnable and run them in CI (CONSENSUS):
   [Rust Reference dev guide](https://rust-lang.github.io/reference/dev-guide/print.html),
   [Exercism configlet](https://exercism.org/docs/building/configlet),
   [nbval](https://github.com/computationalmodelling/nbval).
4. Fill release facts into authored pages instead of typing them (EVIDENCE,
   one case): a contributing tutorial named a release years out of date
   ([Django ticket #37217](https://code.djangoproject.com/ticket/37217)).
5. Route product changes to the content owners with a three-state
   docs-impact mark and a release check (CONSENSUS):
   [Mattermost labels](https://docs.mattermost.com/developers/contribute/more-info/getting-started/labels),
   [Kubernetes release Docs handbook](https://github.com/kubernetes/sig-release/blob/master/release-team/role-handbooks/docs/README.md),
   [CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners).
6. Fix generated pages at their source (CONSENSUS):
   [Kubernetes: contribute upstream](https://kubernetes.io/docs/contribute/generate-ref-docs/contribute-upstream/);
   keep code and content apart, validated in CI ([Stripe Markdoc](https://stripe.dev/blog/markdoc)).
7. Start editing for non-engineers with the forge's editor and previews; add a
   content system only once such a team exists (ANECDOTE):
   [Tom Johnson, 2024](https://idratherbewriting.com/blog/thoughts-on-docs-as-code-promise).
8. Regenerate terminal recordings from scripts; stamp hand-recorded video with
   a version and a review date (ANECDOTE, tooling):
   [vhs-action](https://github.com/charmbracelet/vhs-action).

Counter-evidence: learning content hosted on an outside platform stops working
when the platform closes ([Kubernetes tutorials on Katacoda](https://v1-36.docs.kubernetes.io/blog/2023/02/14/kubernetes-katacoda-tutorials-stop-from-2023-03-31/));
a separate authoring toolchain can be archived ([Codelabs tools](https://github.com/googlecodelabs/tools));
stale content is the default without automated checks
([Kubernetes 2019 docs review](https://v1-36.docs.kubernetes.io/blog/2020/01/21/reviewing-2019-in-docs/)).

Not worth adopting: a separate learning site or repository; true two-way sync
between rendered pages and their sources (no precedent; a one-way render with
a match check gives the same safety); a git-backed content system from day
one; freshness dates as the only control; version placeholders in tutorials
(substitute real values when the page is built instead).
