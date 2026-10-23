---
id: itd-2610040740135705
slug: the-technical-facilitator-and-other-team
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610032150577708]
related_adrs: [adr-2610032150581128]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# The dashboard for more people: the facilitator's view and the team's, configurable

## Press Release

> The technical facilitator and other team members open abcd's dashboard with a view made for their role, and the project chooses who sees what.

_Filed by the facilitator on 2026-10-04 from the product thinker's words at the dashboard interview (itd-2610032150577708 decision 8): "for the techincal facilitator, for other team embers (configurable)"._

## Why This Matters

_Facilitator-written._

The first version is made for the product thinker. The technical facilitator needs the developers' side of the record (the product thinker's description at decision 7 there: "everything the developers need in addition to the public facing website content"), and other team members need a view for their role. Who sees what is the project's choice.

Typed links: builds on itd-2610032150577708.

## Decisions

1. 2026-10-04, the product thinker (decisions 7 and 8 there): the product thinker first; the facilitator and other team members later, configurable.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [dashboard-sites SOTA note](../../research/notes/2026-10-23-dashboard-sites-sota.md).

- **GitHub cannot tell the product thinker from the facilitator:** custom repository roles are Enterprise Cloud only, teams exist only in organisations, a personal repository has only an owner and write collaborators; for GitHub-side admission a configuration file keyed on GitHub user ids was the one mechanism that works everywhere (the GitHub roles note of 2026-10-14). That finding rules out GitHub as the role source; it says nothing about the tailnet identifiers this dashboard keys on, which are a different space.
- **The tailnet can carry per-person roles, attested by the daemon:** a grant's `app` capability (`domainName/capabilityName` with JSON parameters) reaches the application through the lookup's `CapMap`; golink grants `"tailscale.com/cap/golink": [{"admin": true}]` and reads it "via the Tailscale daemon whois API" (Tailscale grants; golink). No header can forge it; but it lives in the tailnet's policy file, which the tailnet's admin edits, not the project.
- **Each device is already let in separately:** the lookup names a node and a profile per connection and the devices-seen file lists them; a per-person view needs a mapping from profile to role, not a new admission step.
- **A view per role is the cheap cut; hiding records is the hard one:** every page comes from one renderer by id, and the board intent's facilitator view is the same read arranged differently.

**Path 2 — native, shaped to golink's capability model, with the role source as the seam: the project's configuration first (keyed on the stable user id the tailnet lookup reports, not the display login), a tailnet capability as an optional second source later.** A role list in the repository is shared working data read by a per-machine listener, and a role in the tailnet's policy file is the tailnet admin's to grant: two authorities for one fact, which the question below puts to the fit-challenge. The first release chooses a view per role and hides nothing. The independent fit-challenge runs at plan time.

## Open Questions

- Which roles exist beyond the two, and where the project declares who sees what.
- Whether each person is let in separately, and how a view is chosen per person.
- (SOTA 2026-10-23) Whether a role declared in the repository and a role granted in the tailnet's policy file may both exist, and which wins when they disagree; two authorities for one fact is the thing to challenge.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
