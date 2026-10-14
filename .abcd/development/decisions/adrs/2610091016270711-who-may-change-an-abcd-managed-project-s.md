---
id: adr-2610091016270711
slug: who-may-change-an-abcd-managed-project-s
status: accepted
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

On 2026-10-14 the product thinker narrowed these requirements for the first release, verbatim: "for now, public projects only in abcd, but private ones must be an option later". The private-repository requirements above stand for the later intent that adds private repositories; the decision here must leave room for them.

## Decision

Decided by the product thinker's rulings of 2026-10-14, recorded verbatim in itd-2610091016262335's Decisions:

1. **abcd runs no server and holds no key.** A session runs on the team's own machine, reached over Tailscale, or on the team's own relay, deployed once from a blueprint abcd ships to the team's own account (the pattern the rascals demo repository uses on Render). Partner-run hosting is a later intent.
2. **Joining is by the host's QR code or short code.** There are two kinds of participant:
   - A **team member** signs in with GitHub through the team's own small GitHub sign-in app, whose secret stays on the team's relay or host machine (pasting a personal access token is the fallback). They are admitted as a team member when GitHub reports that they can change the repository (write access or above). Their changes are credited to them.
   - A **guest** joins by code alone. A guest's change is a suggestion until a team member approves it, and the approval is recorded with an `Approved-by:` line beside the guest's display name.
3. **A change lands only through GitHub, as a proposal the team decides on.** The team chooses, per session, between each edit as its own pre-filled GitHub issue (credited to its author or approver) and the whole session as one pull request (authored by the submitter, with a co-author line per credited team member). abcd never writes into the repository itself. The team's own agent session picks these up at its start, the way the inbox surfaces reports; a GitHub Action on the project's own key is a later addition.
4. **Scope of the first release:** the brief only, public and private repositories alike, with no product thinker / technical facilitator toggle. abcdev.app is read-only and offers no editing.

## Alternatives Considered

- **A relay abcd runs.** Rejected: abcd never runs a server holding or serving project content ("not us. never.").
- **A partner's live-collaboration service** (Liveblocks, Tiptap's cloud). Deferred to a later intent: the partner holds a readable copy and someone pays monthly.
- **Browsers connecting directly, the host's browser as the hub.** Not chosen: unreliable on company networks without a relay.
- **A shared GitHub branch refreshed every few seconds.** Chosen first, then reversed the same day: not live enough for a team working in a call.
- **Pasting a personal access token for everyone.** Kept only as the fallback once the team's own relay or host can hold its sign-in secret.
- **A GitHub App owned by abcd, or GitHub's roles deciding who is product thinker or facilitator.** Rejected: abcd holds no key, custom repository roles are enterprise-only, and personal repositories have no teams (research note 2026-10-14-github-roles-admission-and-issue-links.md).

## Consequences

- Each team does a one-off setup: deploying the relay or joining the host's Tailscale network, and registering its own GitHub sign-in app. Keeping that relay and its secret safe is the team's responsibility, not abcd's.
- Anyone the host gives the code to sees the brief, a private one included; admission of guests is the host's call.
- The session needs per-person change tracking and a suggestion mode for guests.
- abcd runs no network surface of its own anywhere. The relay is a verb in the abcd binary that the team starts on its own machine or deploys to its own account, admitted as the second listener by the 2026-10-14 amendment of adr-2610032150581128. The hosted page at abcdesign.app is static.
