---
id: adr-2609292116133348
slug: a-dependency-bump-inside-the-bound-is-re-authored-as-the
status: accepted
date: 2026-09-29
supersedes: null
superseded_by: null
related_intents: [itd-2609221842494980]
related_rfcs: []
related_adrs: []
---

# ADR-2609292116133348: A dependency bump inside the bound is re-authored as the owner and only pushed by an App

## Context

The attribution gate (`scripts/check-attribution.sh`) refuses a machine as the
author or committer of any commit, so a pull request dependabot opens is not
mergeable as authored, and every bump was landed by a person re-authoring it by
hand. The rule is deliberate: the contributor graph is built from those two
fields, and a squash merge re-appends a mis-identified branch author as a
co-author. [iss-2609221820487644](../../../work/issues/open/iss-2609221820487644-the-attribution-gate-should-let-a-dependabot-dependency-bump-through-without-a-human-re-authoring-it.md)
asked for bumps to land without that hand step, and said that because it
reverses a stated rule it needs this record and a brief invariant.

The product thinker ruled on 2026-09-22 for route 2 of the capture's three
shapes (a workflow re-authors the bump before the gate runs; the gate does not
change) and planned it as
[itd-2609221842494980](../../intents/planned/itd-2609221842494980-a-dependency-bump-lands-without-a-person-re-authoring-it-a.md),
whose `## Decisions` fix the bound as the diff's shape, the message and the
opt-in. A ruling on 2026-09-29 settled the credential: a GitHub App the person
creates and installs pushes the result, and the landed commit's author is the
person. The constraint that shapes the rest is the platform's: a
`pull_request` run that dependabot starts reads only Dependabot secrets, and a
push made with the run's own `GITHUB_TOKEN` starts no checks.

## Decision

We will re-author an in-bound dependency bump as the repository owner, and let
a GitHub App do nothing but push it.

1. **The owner authors and commits; the App only pushes.** The re-authored
   commit has the same tree and parent as the bot's, the owner named in
   `.abcd/config/dependency-reauthor.conf` as author AND committer, a message
   naming the bot, its commit and the workflow, and `Assisted-by: None`. The
   App's identity appears in neither field, because the gate refuses a `[bot]`
   identity in both roles, and the gate stays unchanged.
2. **The bound.** A pull request is re-authored only when all of these hold:
   its author is a bot the declaration names and the run's actor is that same
   bot (a person who pushes a bot-authored commit onto the bot's branch starts
   a run whose actor is that person); its branch lives in this repository and
   starts with the prefix the row declares, and its name passes an allowlist
   and a length cap before any message repeats it; it carries exactly one
   commit, authored by that bot; and that commit only MODIFIES files the row
   names, at the directory `.github/dependabot.yml` declares for the ecosystem.
   A file added, deleted or renamed, or the same name in any other directory,
   is outside the bound, because a new manifest carves a module out of the
   tree the checks run over. Anything outside the bound is left alone, and the
   run names the clause it failed. The script and the declaration are checked
   out from the pull request's base, so the branch under judgement cannot
   change them.
3. **The accepted residual.** The bound judges which files a bump changes,
   never what it writes in them: a manifest's content inside the bound (a
   `go.mod` `replace`, `toolchain` or `tool` directive, a `go.sum` line) is
   re-authored with no person reading it. This follows from ruling 2 (the
   bound is the diff's shape) and is accepted here rather than left implicit.
4. **The secrets are Dependabot secrets.** The App's id and private key are
   stored as the Dependabot secrets `DEPENDENCY_REAUTHOR_APP_ID` and
   `DEPENDENCY_REAUTHOR_APP_KEY`, the only store a dependabot-started
   `pull_request` run reads. The run mints a short-lived installation token
   scoped to this repository's contents, pushes under a lease pinned to the
   bot's commit, and revokes the token on exit.
5. **Refusal, never a fallback.** While the owner is unset or either secret is
   absent, an in-bound bump is refused by name and stays a person's to land.
   The workflow never pushes with `GITHUB_TOKEN` and never keeps the bot as
   author.

## Alternatives Considered

- **The gate exempts a manifest-only commit on the bot's branch.** Rejected by
  the 2026-09-22 ruling: it puts a machine into the contributor graph and
  widens the one gate every change passes.
- **Dependabot off, bumps by hand on a schedule.** Rejected by the same
  ruling: it keeps the hand step the capture asked to remove.
- **Push with a token the person creates.** Considered first (the 2026-09-29
  rulings) and refined to the App: an installation token is minted per run,
  scoped to one repository's contents and revoked on exit, where a personal
  token is long-lived and carries its owner's whole reach.
- **Re-author inside the bound, the App only pushing (chosen).** The gate
  and the rule behind it stay as they are, and the one thing a person gives up
  is stated: authorship asserted by a workflow for commits inside this bound
  and no others.

## Consequences

- A bump inside the bound lands with no person re-authoring it, once the
  person has set the owner, created and installed the App, and stored its two
  secrets; until then the workflow refuses and a person lands the bump as
  before.
- A change to the bound, the residual or the credential's placement is a
  change to this record and to the brief invariant that cites it, never merely
  a code path.
- An Actions bump stays outside the bound (a workflow file is never a
  manifest) and is still landed by a person.
- The token mint is about forty lines of shell (JWT signing with `openssl`,
  the exchange with `curl`) that abcd maintains and re-audits, in exchange for
  no new pinned action.
