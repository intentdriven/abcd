---
id: adr-2610091010387033
slug: abcdev-app-may-render-public-abcd
status: proposed
date: 2026-10-09
supersedes: null
superseded_by: null
related_intents: [itd-2610091010348892, itd-135, itd-2610052000411162]
related_rfcs: []
related_adrs: [adr-47, adr-48]
---

# ADR-2610091010387033: abcdev.app may render public abcd-managed repositories other than abcd

## Context

The product thinker proposed on 2026-10-09 (itd-2610091010348892) that abcdev.app renders the landing page and development record of any public abcd-managed project on GitHub, at `abcdev.app/<owner>/<repo>/`, with abcd's own pages moving to `abcdev.app/intentdriven/abcd/` as the showcase.

That reverses [adr-47](0047-abcdev-app-rendered-from-this-repository.md) decision 1, "The website is a surface of this repository and of nothing else", and touches its constraint that the build depends on no runtime API calls because the unauthenticated GitHub API allows 60 requests an hour. It also sits beside the product thinker's 2026-10-05 choice of a separate site, abcdesign.app, where anyone opens and edits a public project's brief (itd-2610052000411162); on 2026-10-09 they ruled that both sites are kept: abcdev.app shows a project's record, abcdesign.app edits its brief.

Open before this record can be decided:

- Which repositories are rendered: every public repository carrying an abcd-managed marker, or only those whose owners opt in, and how an owner withdraws one.
- What may be rendered from a repository abcd's maintainers do not control, under abcd's domain, and what the page says about who wrote it (the site's single-source rule renders spans of a repository's own files; a third party's files are not abcd's).
- How builds run: on a schedule, on the owner's request, or from the owner's own release, within the GitHub API's limits, and what abuse or takedown path exists. On 2026-10-09 the product thinker required, for abcdesign.app and the local route, that a read-only build for someone not signed in is rate-limited so anonymous traffic, bots included, cannot exhaust the allowance the site serves from, and that a signed-in build spends the person's own allowance (adr-2610091016270711); on the same day they ruled that the same applies to abcdev.app: "same applies to abcdev.app, too (not just abcdesign.app)".

Requirements the product thinker set on 2026-10-09, which any decision here must meet:

- Public and private repositories are both supported; a private repository is reached only through the person's own GitHub sign-in.
- A read-only build for someone not signed in is rate-limited, so anonymous traffic, bots included, can never exhaust the GitHub allowance the site serves from.
- A build for a signed-in person spends that person's own GitHub allowance, never the site's.
- A shared cache serves a page built from a public repository to everyone; a page built from a private repository is served only to people that repository admits, never beyond.

## Decision

_Not yet decided; settled at the planning interview of itd-2610091010348892, after the research note on building from GitHub within its rate limits._

## Alternatives Considered

_To be laid out at planning: render on opt-in only; render every public abcd-managed repository; keep abcdev.app to abcd alone (adr-47 as it stands) and serve other projects elsewhere._

## Consequences

_To be written with the decision._
