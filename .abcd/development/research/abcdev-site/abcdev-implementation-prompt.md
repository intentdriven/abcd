# abcdev.app — the implementation prompt, and the two architecture questions behind it

*21 August 2026. Companion to the plan (`abcdev-site-plan.md`, v4) and the prototype (v6). Part 1 is the prompt to paste into a facilitator session that has the abcd plugin; Part 2 is the adversarial review of the two questions you asked — one repo or two, deploy per release or per merge — with the recommendation the prompt already assumes.*

---

## Part 1 — the prompt

Paste everything inside the fence into a session of your agent harness with the abcd plugin loaded, in a checkout of `Partnermedia/abcd` on a fresh branch. It records the work first (intents, decisions, issues, research) and only then builds, so the repo carries the *why* before any code lands. Where it names a verb, that verb exists in the current CLI reference; ADRs and research notes are written by hand because no verb mints them.

````text
You are the facilitator for this repository (abcd). Before anything else, read
AGENTS.md and .abcd/development/README.md, run `abcd ahoy`, and keep the INTENTS,
ISSUES, DOCUMENTATION and COMMITTING rule domains in mind throughout (*INTENTS etc.
activates them explicitly).

CONTEXT
We are giving abcd a website at abcdev.app that is GENERATED FROM THIS REPOSITORY
and nothing else: `/` becomes a landing page for product thinkers, the MkDocs
rendering of docs/ moves from `/` to `/docs/`, and a record explorer (`/record/…`,
`/contributors/`, `/references/`) renders the development record. Everything has
been investigated and prototyped outside the repo; your first job is to bring
that work INTO the record the abcd way, your second job is to build it.

The design artefacts are in `.abcd/.work.local/scratch/site-plan/` (local, not
committed):
  - abcdev-site-plan.md      the investigation and plan (SOTA survey, decisions,
                             URL map, build pipeline, record.json schema, pages,
                             design system, rollout, open decisions, risks)
  - abcd-web.html            the clickable prototype (open it in a browser; the
                             "notes for the team" toggle explains every section
                             and names the source of every block)
  - abcd-readme-migration.zip  a bundle containing: docs/explanation/{rationale,
                             roles,artefacts,process}.md and docs/how-to/install.md
                             (README's product text moved verbatim, plus a
                             per-OS CLI section), a slim README.md, MIGRATION.md
                             (the line-by-line map), .abcd/site.json (the
                             composition manifest), site-src/ui.json (the complete
                             list of words the generator may add), docs/assets/img/
                             (role portraits, artefact icons, process-loop.svg),
                             and compose.py + build_data.py — the Python reference
                             implementation of the composition and layout rules
                             that the Go generator must reproduce.
Read the plan in full and open the prototype before you write a single record.

THE ONE RULE THAT GOVERNS EVERYTHING
No text is written for the website. Every sentence the site renders is a span of
a file in this repository, selected by path and heading through .abcd/site.json;
the only words the generator may add are the interface strings in
site-src/ui.json (button labels, tab captions, tile captions), and numbers,
dates, file names and asset names. Every picture is a committed asset under
docs/assets/img/, referenced from a docs page like any other image; the build
optimises rasters and inlines SVGs, it never draws. A build check must fail when
a rendered text node has no source, when rendered text violates docs-lint, when
an `abcd …` snippet does not match the generated CLI reference, or when a
cross-reference is unresolved and not in the committed baseline
(.abcd/site-baseline.json — a ratchet: whatever the build finds today is the
baseline — the investigation counted six, an in-repo re-count found eight, adr-22
pointing at adr-14, adr-15 and adr-17 — and fixing one shrinks it). If you ever feel the need to write a sentence for the
site, the sentence belongs in docs/ or in the record, and the site renders it
from there.

STEP 1 — RECORD THE WORK (no code yet)
1a. Idea admission. Run /abcd:ideate for the idea "abcd generates its own website
    from the repository" using the plan's §3 (state of the art) and §4 (decisions)
    as the primary-source research leg, the existing record as the grill leg, and
    Part 2 of the document this prompt came from as the adversarial leg; write the
    verdict with `abcd ideate record abcdev-site --verdict-json …`. Ideate is
    optional in abcd; we want it here because the blast radius (a public surface,
    a new CI job with deploy credentials) is large — see the principle
    adversarial-review-scales-with-blast-radius.
1b. Research note. Move abcdev-site-plan.md into .abcd/development/research/ as a
    dated note (keep its text; add frontmatter in the style of the neighbouring
    notes). Commit the prototype beside it only if the research directory already
    holds binary-ish artefacts; otherwise leave it in scratch and link the
    artifact URL from the note. The migration bundle's files are NOT research —
    they go to their real homes in step 2.
1c. Decisions. Write two ADRs in .abcd/development/decisions/adrs/ (MADR, next
    sequential ids, frontmatter like adr-45), each with the considered options
    and the consequences spelled out:
      - "abcdev.app is rendered from this repository alone": the site is a
        surface of the record (sibling of positioning.json's surfaces), generated
        by an `abcd site` verb from repo text and committed assets under the
        single-source rule above; SSG-agnostic — MkDocs Material stays for /docs/
        for now, the SSG is replaceable by a later ADR; a single repository, no
        abcd-web repo (Part 2, question 1 has the reasoning and the rejected
        alternative; record both); and an amendment to adr-30's boundary — the
        record is never bundled into the launch payload, but the site renders it
        read-only at /record/.
      - "The website deploys on release, not on merge": production at abcdev.app
        is built from the release tag by a workflow triggered by `release:
        published`, with the released binary doing the rendering; main deploys to
        a clearly labelled preview; docs-only fixes ship as patch releases under
        the changelog-driven release flow (adr-37); a workflow_dispatch rebuilds
        from the latest tag for emergencies, never from main (Part 2, question 2).
    Link both ADRs to each other and to adr-37 in frontmatter.
1d. Intents. File the website as intents, one press release each, portable on
    their own, with builds_on chaining them to the umbrella:
      - umbrella: `abcd intent "Alice opens abcdev.app and understands who abcd
        is for, what she and her facilitator own, and how to install it — from
        one page rendered from the repository"` — the landing page: hero from
        rationale.md and the Identity block, chapters a Roles, b Artefacts,
        c Process, d Install rendered from the four docs pages, the newest
        shipped intent with a MET audit quoted as the only testimonial.
      - `abcd intent "…"` for the record explorer: dashboard, the relationship
        chart (one chart, two arrangements: the date coil with month zones and
        the links-only force layout; ego focus with arrowheads; the card with
        GitHub-palette pills and the date continuum), the genealogy timeline,
        the contributors page (humans as authors of record, Assisted-by
        disclosure as disclosure), the references page from references.csl.json.
        Decompose further if a press release cannot carry it — see the
        principle decompose-before-filing — but keep each intent user-facing.
      - `abcd intent "…"` for `abcdev.app/install.sh` as the one new
        distribution endpoint (the existing one-liner, served from our domain,
        checksum-verified, no redirects, no Homebrew).
      - the README → docs move is plumbing that enables the above: it goes
        straight into the brief (see artefacts.md: plumbing skips the press
        release), not into an intent. `abcd site build` and its checks are
        plumbing too.
    Write the press releases in the voice of the existing shipped intents
    (itd-100 is the model: a named persona from personas.json feeling the
    difference, a quote, then Given/When/Then acceptance criteria). Acceptance
    criteria must be checkable by the audit: cite the check, the URL, the file.
    Then `abcd intent plan <itd-N>` each one you are ready to build, and
    `abcd intent ready` before you start it.
1e. Issues. Capture what the investigation found, one `abcd capture` each with
    an honest severity and --found-at where it applies:
      - eight unresolved references (adr-22→adr-14/15/17, adr-25→adr-8,
        adr-27→adr-16, adr-28→adr-18, adr-35→adr-4 are `supersedes` targets retired
        under "retire the name"; itd-3→spc-1) — nothing checks supersedes targets
        today
      - the spec link is recorded from both ends (intent spec_id ↔ spec
        implements, 32 pairs) and 20 related_* pairs are listed in both files
      - issues carry no typed cross-references at all (only body mentions)
      - one early commit carries "Claude" in the git author field, pre-policy;
        two are dependabot — the contributors page needs a rule for bot/tool
        authors
      - Cloudflare non-production branch builds run only the version command
      - MkDocs 1.x unmaintained since Aug 2024; Material in maintenance mode since
        6 Nov 2025 (12 months of security fixes "at least"); MkDocs 2.0 removes
        plugins — an SSG decision is due before ~Nov 2026
      - the Claude project description and the July landing-page asset point at
        <personal account>/abcd and at docs paths that no longer exist
      - the brief's own press release tells the lifeboat story, not the
        product-thinker story README tells — the brief records this as open
    Do not resolve any of them in this step.
1f. Brief. Update .abcd/development/brief/ so that everything it says about the
    website reads true right now: the site exists as a design and a plan, the
    generator does not yet — mark the passages not yet real the way the brief
    marks ambition, and let the shipping change remove the marks.
1g. Commit the record (Assisted-by trailer as CONTRIBUTING.md requires) before
    any code. Run `abcd docs lint` and `abcd changelog` and make sure both are
    clean.

STEP 2 — BUILD, IN THE PLAN'S PHASES
Follow the plan's §6 in order; each phase is a spec closed with `abcd spec close`
and its intent audited with `abcd intent audit` before the next begins.
  Phase 1 — the move: unpack the migration bundle to its real paths (docs pages,
    docs/assets/img/, .abcd/site.json, site-src/ui.json, the slim README.md),
    register README's universal one-liner and the two per-OS forms in
    install.md as surfaces that must agree (a test: same URLs, same checksum
    step, only `uname -s` resolved), move MkDocs to /docs/ with _redirects for
    the old root URLs, keep positioning.json's readme-strapline check passing
    and add the site-hero surface. docs-lint must gate the moved text exactly as
    it gated README.
  Phase 2 — `abcd site build` (Go, in the binary): walk .abcd/site.json, compose
    the landing page and the record pages from repo text, emit record.json
    (nodes, the deduplicated typed links, lifecycle from directory, dates from
    git, the coil and the by-links layouts precomputed, releases from
    CHANGELOG.md, contributors from git with .mailmap, references from the
    CSL-JSON), inline SVG assets, optimise rasters, and write site/. Reproduce
    compose.py and build_data.py faithfully — they are the spec for the
    composition rules, including the provenance attribute on every block.
    `abcd site check` runs the four checks above and the mobile audit (no
    horizontal overflow at 360/390/768/1360; screenshot every route).
  Phase 3 — the workflow: .github/workflows/site.yml on `release: published`
    (+ workflow_dispatch with a tag input): check out the tag, download and
    checksum-verify the released binary, `abcd site build`, mkdocs build into
    site/docs/, `abcd site check`, attest site.tar.gz as a release asset, deploy
    with wrangler to the production Worker from a protected GitHub Environment;
    the same workflow on push to main deploys to the preview target with the
    unreleased label rendered from the build metadata. Turn off Cloudflare's
    automatic production builds; record the change in wrangler.jsonc's comments.
  Phase 4 — install.sh: the template under site-src/, rendered with the latest
    tag at build time, served as /install.sh with the right content type, and the
    install.md lead sentence updated in the same change so the site never
    previews a command the repo does not ship.
At every phase: nothing on the site that is not in the repo; every picture a
committed asset; every page perfect at 390 px or the element removed from the
mobile view rather than degraded; light and dark; `prefers-reduced-motion`
respected.

WHAT NOT TO DO
Do not create a second repository. Do not deploy production on merge. Do not
write copy for the site — not a tagline, not a button label outside ui.json,
not an alt text that is not in the docs page. Do not add a Node toolchain to
the build (Playwright for the mobile audit may live in an optional CI job). Do
not let the website read the GitHub API at runtime (60 requests an hour
unauthenticated); everything is injected at build time. Do not change what the
record says to make the site look better — if the record is wrong, capture an
issue.

DEFINITION OF DONE
`abcd intent audit` returns MET for every website intent; `abcd site check`
passes on the tag; the release workflow has deployed abcdev.app from a tag and
the footer names that tag and commit; the README is a contributor page and
docs-lint is clean; the two ADRs are accepted; the baseline holds exactly the
unresolved references that existed before this work, or fewer.
````

---

## Part 2 — the two architecture questions, reviewed adversarially

You asked for this to be thought through rather than asserted, so each question is argued both ways against what comparable projects actually do, with the failure modes of the recommended option named and mitigated rather than waved away.

### Question 1 — a second repository (`abcd-web`) or one?

**The case for a second repo, made as strongly as it can be.** A website has its own toolchain (templates, CSS, fonts, an image pipeline, screenshot tests), its own cadence, its own contributors and its own secrets. Putting it beside a Go CLI means every CLI contributor inherits website CI, every website change shows up in the product's history, and the Cloudflare deploy token lives in the repository that also signs releases — a wider blast radius than necessary. A separate repo also gives the cleanest reproducibility story: `abcd-web@<sha>` checks out `Partnermedia/abcd@<tag>` and builds, so the site for a release is a pure function of two pinned references. And there is precedent: Kubernetes, Go, Rust and GitLab all keep their websites in separate repositories; Atuin, a single-binary CLI like abcd, moved its docs to a separate `atuinsh/docs` repository ([atuinsh/docs](https://github.com/atuinsh/docs)); Fabrizio Ferri Benedetti's survey of docs-as-code topologies treats "docs repo separate from code" as a legitimate, common shape ([passo.uno](https://passo.uno/docs-as-code-topologies/)).

**Why it loses here.** Three things about abcd specifically.

First, the single-source rule is a property of *one tree at one commit*. The manifest that says where each block comes from (`.abcd/site.json`), the allowlist of words the generator may add (`ui.json`), docs-lint, the record and the docs must be reviewable in the same pull request that changes a docs page, and checkable by one CI job on one commit ("every rendered text node has a source *in this commit*"). Split them and the rule acquires a second repository of text — templates, ui strings, redirects, alt texts — plus a version-compatibility problem between the record's schema and the renderer that reads it. Every separate-docs-repo project in the precedent list has exactly that coordination tax and pays it because its docs are large, localised or multi-product; abcd's are neither.

Second, the generator is product code. `abcd site build` is a verb of the binary, and the site is the binary rendering its own record — dogfooding, and the strongest possible demonstration of "the record is machine-readable". A second repo would either re-implement record parsing or depend on the binary anyway, at which point the templates it needs are the only thing living elsewhere.

Third, the peer group that looks like abcd keeps its site in-repo: GoReleaser (`www/`, moved from Material to Hugo inside the same repo in March 2026), golangci-lint (`docs/`), Task (`website/`), mise (`docs/`), chezmoi (`assets/chezmoi.io`), uv and ruff (`docs/`). Atuin is the exception that proves the rule: it split when docs outgrew the CLI — and still carries `docs/docs` in the main repo ([atuin/docs/docs](https://github.com/atuinsh/atuin/tree/main/docs/docs)).

**The legitimate concerns, answered inside one repo.** Toolchain bloat: the site build is Go plus the MkDocs the repo already has; the mobile screenshot audit (Playwright) is an optional CI job, not a build dependency. Blast radius: the Cloudflare token is scoped to deploy one Worker and lives in a protected GitHub Environment with the release workflow's existing trust chain (verified commit, attested artefacts); it never touches release signing. CI time: seconds. History noise: `site-src/` is one directory, and CODEOWNERS can route it.

**Verdict: one repository.** The plan's hybrid is the right shape: `site-src/` + `abcd site build` in the binary, MkDocs kept for `/docs/` behind an SSG-agnostic boundary. Record the rejected alternative in the ADR so the question is not reopened without new evidence — the evidence that would reopen it is localisation, a second product on the same site, or documentation contributors who are not CLI contributors.

### Question 2 — deploy on every release, or on every merged PR?

**Your instinct (release, not merge) is right, and here is the adversarial pass on it.**

*The case for deploying on merge.* It is the default of every hosted platform (Cloudflare, Vercel, Netlify deploy the production branch on push), it keeps the record explorer current to the hour, and it means a typo fix reaches the site in minutes. The record is the most alive part of this repository — 1,188 commits in 45 days against 10 releases — so a release-bound site shows a record that is up to four or five days stale.

*Why release wins anyway.* The site's job is to describe a product someone can install. If production follows main, the site can document a verb that no downloadable binary has, and the install command, the version badge, the changelog and the docs can disagree for days at a time. Versioned docs would fix that and are far too heavy for a project this size. Deploying from the tag makes the whole site a single statement — "this is abcd at v0.6.1" — which is the same promise the brief makes about itself ("everything it says reads true right now"), the same promise immutable releases make about assets, and it lets the site be a release artefact: built by the released binary, attested alongside the binaries with `actions/attest-build-provenance` ([GitHub docs on artifact attestations](https://docs.github.com/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds)), so anyone can verify that abcdev.app is the rendering of a specific tagged commit. No comparable project does that yet; it is cheap for abcd because the release workflow already verifies and attests.

*The failure modes, and what the prompt does about each.*

- **A docs-only fix has to wait for a release.** Under changelog-driven releasing (adr-37) a docs fix is a CHANGELOG line and a patch version; that is a feature, not a cost — the fix gets a version and a date. For a genuine emergency (a wrong command, a security notice), the workflow accepts `workflow_dispatch` with a tag input and rebuilds from the *latest tag*, never from main, so the invariant survives the emergency.
- **The record explorer is stale between releases.** The same workflow deploys every push to main to a preview target (a separate hostname or the Worker's preview URL) with an "unreleased — main@sha" label rendered from build metadata. The team gets the live dashboard; the public gets the released one. This is the standard production-from-tags, preview-from-branches split, and it also replaces the Cloudflare branch builds that currently run only the version command.
- **A half-finished release.** Trigger on `release: published`, in a separate workflow from `release.yml`, so a site failure can neither block nor taint a release; make the job idempotent (re-running deploys the same tag). Only full releases deploy production; pre-releases deploy to preview.
- **Rendering drift between the binary and the site.** The workflow downloads the *released* binary (checksum-verified, like the install one-liner does) and renders with it, rather than building from source — the site is produced by the same bytes users run.
- **Cloudflare's Git integration keeps deploying main.** Turn automatic production builds off and deploy with `wrangler` from Actions ([cloudflare/wrangler-action](https://github.com/cloudflare/wrangler-action)); the dashboard's build command becomes a comment in `wrangler.jsonc`, which is where the repo already documents it.

**Verdict: production on `release: published` from the tag, preview on push to main, emergencies by dispatch from the latest tag.** One consequence worth stating in the ADR: the cadence of the website becomes the cadence of releases, which makes small, frequent releases slightly more valuable than they already are — and abcd's ten in 45 days suggests that is the cadence anyway.

### What to record

Two ADRs, as the prompt says, each with its rejected option; one issue for the Cloudflare build quirk; and a sentence in the brief's internals section that the website is a surface of the record. Nothing else about this decision needs a home — the workflow file and the ADRs are the record.

Sources: [cloudflare/wrangler-action](https://github.com/cloudflare/wrangler-action) · [Using artifact attestations — GitHub Docs](https://docs.github.com/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds) · [actions/attest-build-provenance](https://github.com/actions/attest-build-provenance) · [Docs-as-code topologies — passo.uno](https://passo.uno/docs-as-code-topologies/) · [atuinsh/docs](https://github.com/atuinsh/docs) · [atuin/docs/docs](https://github.com/atuinsh/atuin/tree/main/docs/docs)
