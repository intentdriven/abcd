---
id: adr-44
slug: remote-mutation-and-caller-identity
status: accepted
date: 2026-08-19
supersedes: null
superseded_by: null
related_intents: [itd-92]
related_rfcs: []
related_adrs: [adr-43]
---

# ADR-44: abcd never mutates a remote uninvited, and identity derives from caller-local facts

## Context

Two rules surfaced during the collaborator-fence field work
([`2026-08-19-collaborator-fence-field-research.md`](../../research/notes/2026-08-19-collaborator-fence-field-research.md))
and were living, homeless, in a draft intent's prose. Both are trust rules that
outlive any one feature — the itd-84 routing (capability → intent, trust rule →
ADR + brief invariant) puts them here, following the auto-merge and itd-111
precedents. One of them already binds shipped code.

## Decision

1. **abcd never mutates a remote uninvited.** Reading remote state (a probe, a
   doctor, a drift diff) is the default and needs no ceremony. Writing remote
   state — settings, rulesets, environments, webhooks, roles — happens only
   through an explicit, dedicated verb the user invokes and confirms, never as
   a side effect of install, doctor, or any read-shaped command. A verb that
   applies remote configuration owns its own ordering (a required check's
   workflow lands on the default branch before any ruleset requires its
   context) and refreshes the committed mirror in the same change.
2. **Identity derives from caller-local facts, never from repository
   ownership.** Any probe that needs to know who the caller is — to redact
   their identity, to suppress their public handle, to scope a verdict — reads
   caller-local sources (git `user.name`/`user.email`, the login embedded in a
   `users.noreply.github.com` address where present, `$HOME`). It never infers
   the caller from who owns the remote: ownership and identity diverged the day
   this repository transferred to an organisation, and the launch scanner's
   ownership-keyed suppression produced 13 false hard-fails within hours
   (iss-283; fixed in `internal/adapter/scanner/identity.go`, pinned by
   `TestIdentityNameEqualsNoreplyLogin`).

## Alternatives Considered

- **Keep both rules as intent prose.** Rejected: an intent ships and closes;
  these rules bind every future remote-touching verb and one already-shipped
  subsystem. Prose in a draft is not a home a linter or a reviewer can hold
  other code to.
- **Auto-apply safe-looking remote settings at install.** Rejected outright:
  changing a stranger's repository settings uninvited is the wrong altitude
  whatever the setting, and "safe-looking" is exactly the judgement the
  confirm step exists to give the human.

## Consequences

- itd-92's doctor is read-only by construction; the apply capability is a
  separate intent bound by rule 1.
- The corresponding brief invariant is invariant 10 in
  [`02-constraints/03-invariants.md`](../../brief/02-constraints/03-invariants.md),
  landed with this record's acceptance.
- Rule 2 is already enforced in the launch scanner; any new probe (the doctor)
  inherits it as a requirement, not a suggestion.
