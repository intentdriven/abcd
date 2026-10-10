---
id: adr-43
slug: inbound-equals-outbound-and-the-org-role
status: accepted
date: 2026-08-19
supersedes: null
superseded_by: null
related_intents: [itd-92]
related_rfcs: [rfc-2]
related_adrs: [adr-41]
---

# ADR-43: Contributions are inbound = outbound MIT, and the trust boundary is the organisation's role ladder

## Context

The repository is public, holds outside collaborators, and moved from a
personal account to an organisation on 2026-08-19. Until that move there was no
role between "no access" and "can publish": on a personal repository a
collaborator holds write — push, merge, and (because the release pipeline fires
on a push to the default branch, and the plugin marketplace serves the repo
root) effectively publish. Every artefact governing contribution still
described a private solo project, including a deferred decision — "a human-only
`Signed-off-by:` (DCO) is deferred until the repo is public or takes its first
outside contribution" — whose trigger had already fired.

This record holds only what is settled and hard to reverse. The forward-looking
ladder above the entry rung is [rfc-2](../../roadmap/rfcs/rfc-2-the-contributor-trust-ladder.md);
the operative contributor-facing text is `CONTRIBUTING.md`, which this record
cites rather than restates (the adr-41 property).

## Decision

1. **Inbound = outbound.** A contribution is accepted under the repository's
   MIT licence, exactly as it goes out. No CLA, and no DCO: for a small MIT
   project a plain inbound = outbound statement carries the same assurance as a
   `Signed-off-by:` ritual without the ceremony, and the kernel's own position —
   only a human may ever sign one — is preserved trivially by having none. The
   deferral is closed by this decision, not by silence.
2. **The trust boundary is the organisation's role ladder.** Access decisions
   are expressed as organisation/repository roles, not ad-hoc grants: Triage is
   the entry rung for new collaborators (issues and pull requests, no push),
   and higher rungs are individual maintainer decisions about people, recorded
   by the role itself. The publish surface behind the ladder is fenced
   server-side — required checks and merge queue without bypass, code-owner
   review over the paths that ship behaviour (`.github/CODEOWNERS`), a
   required-reviewer environment between merge and release — and the applied
   rulesets are mirrored in-tree under `.abcd/work/rulesets/` (the live-vs-tree
   drift check is itd-92's remit).

## Alternatives Considered

- **DCO via `Signed-off-by:`.** The kernel's mechanism, and the original
  deferred plan. Rejected: it adds a per-commit ritual whose whole content — "I
  may submit this under the licence" — the inbound = outbound statement already
  carries at the project scale this repository has. Revisit if the project ever
  needs provenance stronger than the licence statement (e.g. relicensing).
- **A CLA.** Assigns copyright and needs legal infrastructure; out of scale for
  a two-maintainer MIT project, and hostile to casual contribution.
- **Trust by ad-hoc collaborator grants (the personal-repo shape).** Rejected
  by the move itself: it has no rung between stranger and publisher, which is
  what made the migration urgent once invitations went out.

## Consequences

- `CONTRIBUTING.md` states the licence position and the intake rules; this
  record is why they are settled.
- New collaborators enter at Triage. A rung above Triage is a maintainer
  decision about a person, never an automatic graduation — the ladder's shape
  and thresholds stay in rfc-2 until experience settles them.
- The DCO deferral text retires from every surface that carried it; work no
  human signs off is still disclosed under the attribution convention
  (`Assisted-by:`), which is unchanged by this record.
