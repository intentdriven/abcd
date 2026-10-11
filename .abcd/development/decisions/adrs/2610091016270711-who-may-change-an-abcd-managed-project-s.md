---
id: adr-2610091016270711
slug: who-may-change-an-abcd-managed-project-s
status: proposed
date: 2026-10-09
supersedes: null
superseded_by: null
related_intents: [itd-2610091016262335, itd-2610052000411162, itd-2610040740122709, itd-2610040740135705]
related_rfcs: []
related_adrs: [adr-2610032150581128]
---

# ADR-2610091016270711: Who may change an abcd-managed project's design on abcdesign.app, and how a change lands

## Context

The product thinker proposed on 2026-10-09 (itd-2610091016262335) that the product thinker and the technical facilitator of an abcd-managed project make its design decisions on abcdesign.app, the product thinker on the brief and the intents, the facilitator on the technical decisions, while everyone else sees the same pages read-only, drawn from the repository. On the same day they ruled that this stands beside the earlier draft in which anyone opens a public brief and submits a proposal (itd-2610052000411162), and that the local dashboard's planned routes for acting (itd-2610040740122709, itd-2610040740135705) are kept as well, so a team may decide locally or on the hosted site.

A hosted page that changes a project's record is a trust boundary abcd has not drawn: abcd's only network surface today is the local dashboard, reachable only through the person's own Tailscale network (adr-2610032150581128). Open before this record can be decided:

- How a person proves they may change a project: the forge's own sign-in and the repository's permissions, or something abcd issues.
- Which change each role may make (the product thinker the brief and the intents, the facilitator the technical decisions), and who checks it.
- How a change lands in the repository: a pull request the other role reviews, a direct commit, or a proposal the local session applies; and what is written under the person's name.
- What a reader without access sees, and what is never shown (a private repository, an unpublished idea).

Requirements the product thinker set on 2026-10-09, which any decision here must meet:

- Public and private repositories are both supported, on abcdesign.app and on the local route; a private repository is reached only through the person's own GitHub sign-in.
- A read-only build for someone not signed in is rate-limited, so anonymous traffic, bots included, can never exhaust the GitHub allowance the site serves from.
- A build for a signed-in person spends that person's own GitHub allowance, never the site's.
- A shared cache is wanted where it is possible: a page built from a public repository can be served to everyone, but one built from a private repository with one person's sign-in may be served only to people that repository also admits, so a private cache entry is keyed to access, never shared beyond it.


## Decision

_Not yet decided; settled at the planning interview of itd-2610091016262335._

## Alternatives Considered

_To be laid out at planning: forge sign-in with repository permissions and changes as pull requests; forge sign-in with direct commits; abcd-issued pairing as on the local dashboard; read-only hosting with every change made in the local session._

## Consequences

_To be written with the decision._
