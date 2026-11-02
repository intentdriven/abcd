---
id: itd-2611011325428560
slug: in-a-private-repository-with-one
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610150602547271]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A solo private repo can self-attest its gates, labelled so

## Press Release

> In a private repository with one maintainer, the person can switch on self-attestation: abcd's local gates post their result under a name that says it was not independently verified, and the hosted run is skipped. Only the person can switch it on, by an act no agent can perform.

## Why This Matters

> A solo maintainer of a private repository pays hosted CI to repeat checks their own machine has just run. Re-running them is honest but costly; skipping them silently is cheap but dishonest. The honest middle is to post the local result under a name that says what it is. GitHub verifies nothing produced on the person's machine: any token with write access can set any commit status, artifact attestations need GitHub Actions (and Enterprise Cloud for a private repository), and SLSA calls a build run on the developer's own machine Build L0, its provenance "trivial to bypass or forge". So the label must say "not independently verified", and the switch must be one no agent can flip. Today an agent can: the interactive guard does not protect the person's machine configuration from an agent's writes, and the agents' forge login holds admin rights (iss-2610170725144804).

## Mechanism

> We expect the option to stay honest because both halves rest on things an agent cannot do: the label is fixed by abcd and never reuses the name of an independent check, and the switch is set by an act that needs the person present (a touch of a security key or a fingerprint on a key the agents' processes cannot use without it, per OpenSSH's FIDO keys and Apple's user-presence access control). It would be shown wrong by any route through which an agent turns the option on through abcd's own verbs, or posts the labelled status through abcd while it is off; an agent that edits abcd's code or the grant record directly is outside what the switch can stop, which the record states.

## Scope Conditions

- Private repositories with one maintainer and no outside contributors; a repository that accepts outside contributions never self-attests their changes.
- The technical facilitator is declared a person (itd-2610150602547271); a repository whose facilitator is abcd cannot switch it on.
- The person has a security key or a Mac with a fingerprint reader; without one the option is not offered, rather than offered with a weaker switch.
- The merge queue's run, where one exists, is never replaced: self-attestation stands in for the pull-request run only.

## Acceptance Criteria

- Given a private repository with one maintainer and a person technical facilitator, when the person turns self-attestation on, then abcd requires a touch of their security key or fingerprint, records the grant on the person's machine, and refuses the same change from any agent session.
- Given self-attestation on, when the local gates pass for a change by the person or their agents, then abcd posts a status named for self-attestation (for example `self-attested/local-gates`), described as run on the person's machine and not independently verified, and the hosted pull-request run is skipped.
- Given self-attestation off, when abcd's own landing runs, then abcd posts no self-attested status; and the record states plainly that an agent holding the person's forge token can still post one by a direct API call, which the label and the landing report make visible but cannot prevent.
- Given a pull request from anyone but the person and their agents, when it is opened, then the full hosted check runs and no self-attested status counts for it.
- Given the landing report of itd-2611011325403127, when a change landed on self-attestation, then the report says so, so the product thinker can see which landings no independent check examined.

## Open Questions

1. Is the self-attested status ever a required check? Posted from the person's token it is the agent's word (any agent holding the token can post it, and a ruleset cannot pin a status to a user token), so as specified it is an honest label beside the hosted run, not a replacement for it; replacing the hosted run needs an identity the agents cannot use, such as an App whose signing needs the person's touch per post (itd-2610300955031405) or the checker of itd-2611011325430933.
2. Is the human touch asked once (to switch on, for a period) or per push? Per push is stronger and stops autonomous landing.
3. Does the switch expire (for example after 30 days) so it is re-confirmed?
4. Should abcd also give the agents a forge token without admin rights, so the ruleset that requires the status cannot be edited by them? That is a wider change than this option.

## Decomposition

Hand-run four-piece table, 2026-11-01, one of five drafts filed together from the investigation of hosted CI and local compute costs (a private ideate verdict, reframed). The product thinker ruled on 2026-11-01 that the work is filed as separate intents in this order: measure (itd-2611011325403127), fewer tests (itd-2611011325421304), self-attestation (itd-2611011325428560), an independent checker (itd-2611011325430933), and the check profiles new repositories are offered (itd-2611011325449761), the fifth at the product thinker's own request the same day.

| Part | Kind | Home |
|---|---|---|
| the labelled self-attested status, the switch, the conditions, the report flag | capability | this intent |
| a self-attested result is labelled as such, never counted as independent verification, and only the person, by an act no agent can perform, turns it on | trust rule | adr-2611011327164895 (proposed), and a brief invariant when it is accepted |
| posting the status, recording the grant, the touch | plumbing | the landing chapter |
| (none) | stance | none |

Verdict: SPLIT, intent and ADR. Typed links: `builds_on` itd-2610150602547271 (a repository declares whether its technical facilitator is a person); consistent with adr-2610120937435856 (no cloud receipt trusted) and adr-2610240850193281 (the person alone grants autonomy); `refines` the commit-status criterion of itd-2610261025516272 (advisory).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
