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
7. **The page set's edges (2026-09-26).** The status page is the record health page; the timeline is the genealogy the dashboard carries; the landing page and the record pages cannot be switched off beneath the explorer. The composition setup derives quotes the recorded identity block and composes the landing page from `docs/README.md`, and the static inputs are seeded as byte copies of abcd's own.
8. **The account is the token's (2026-09-26).** The provider account is the one the credential reaches; a token reaching none or several is refused rather than guessed at, so no account identifier is configured anywhere.

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

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the renderer is built and proved on abcd's own site; the gap is only the verb from the checkout to a live address; we expect a managed repository's site to gain readers who never open the tree; shown wrong if none does within a release
