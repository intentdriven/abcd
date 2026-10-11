---
id: itd-92
slug: abcd-verifies-branch-protection-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# abcd tells every repo the truth about its fence

## Press Release

> **The doctor reports what a managed repository's environment actually holds
> — and what it cannot — one honest verdict per fence piece, for every user
> from a bare local checkout to an organisation.**
>
> Alice, a hobbyist keeping their project in a plain local git repository — no
> forge, no CI — runs the same `abcd ahoy` doctor as Bob, a solo developer on
> a personal GitHub account, and Carol, the maintainer of an org-hosted
> project with outside collaborators. None of them is told to become someone
> else first. The doctor probes read-only, and renders one verdict per fence
> piece from a small vocabulary drawn strictly from what a read-only probe
> can decide *for this caller*: **applied** (with its stated limit — for a
> forge-side required check this asserts registration, never a living
> executor), **applied but drifted** from the committed mirror, **absent**
> (not configured; where the plan cannot be read, enablement-undetermined is
> said rather than guessed), **unverifiable for this caller** (no `gh`, no
> auth, admin-only read — never rendered green), **rule without executor**
> (an in-tree rule nothing enforces), and **unsupported** (no forge, or a
> forge abcd has no adapter for — an environment fact, never an error).
>
> "It doesn't nag me to buy an organisation," said Alice. "It tells me what
> my setup protects, what it can't, and exactly what I'm trusting to luck."

## Why This Matters

Captured 2026-07-17, the day abcd-cli's own `main` was protected by hand.
Extended 2026-08-19 from a full day of building the collaborator fence on
this repository manually, recorded in
[`2026-08-19-collaborator-fence-field-research.md`](../../research/notes/2026-08-19-collaborator-fence-field-research.md),
and reworked the same day after two independent adversarial reviews. The
evidence reframes the original scope: the fence is a **capability ladder**
whose rungs exist or not depending on forge, plan, ownership — and caller,
since verdicts are caller-relative (an admin-only read is unverifiable to a
non-admin collaborator on the same repo). The tool's job for the amateur
coder is to say truthfully which rung they stand on, in the idiom the
banlist's `reach:` lines already ship.

Per [`enforcement-claims-are-facts`](../../principles/enforcement-claims-are-facts.md),
a doctor verdict is a report, never enforcement — and the vocabulary's own
limits are part of the report (registration is not a living executor; absent
is not proof of unavailability). The trust rules the doctor and any later
apply verb answer to are [adr-44](../../decisions/adrs/0044-remote-mutation-and-caller-identity.md):
never mutate a remote uninvited; identity from caller-local facts only.

## What's In Scope

- The read-only doctor pass over an enumerated fence-piece registry (the
  spec's first deliverable is that closed list — no "every piece" claim
  without it), rendering the verdict vocabulary above per piece, per caller.
- The live-vs-mirror drift diff against `.abcd/work/rulesets/` (closes
  iss-277), normalising server-assigned ids and timestamps so a re-apply is
  not a false drift.
- The fence-piece registry extends beyond branch rulesets to the full
  API-visible repo configuration. Repo settings (merge options,
  `delete_branch_on_merge`, auto-merge), Actions permissions with the default
  workflow-token scope, and the security-and-analysis toggles (secret scanning,
  push protection, Dependabot) each get a verdict from the same vocabulary. A
  setting the public API cannot read or set is reported **unsupported (API)**
  with the manual step named, never a false green. The fork-PR
  outside-collaborator approval policy is the known case, its Actions `/access`
  endpoint being private/internal-only. The live-vs-mirror drift diff extends to
  a `repo-settings.json` mirror sibling (iss-2608270512210664). Applying any of
  this stays the separate adr-44-bound apply intent below.
- The pull-request-review verdict is maintainer-count-aware. A solo repo where
  the author cannot self-approve is correctly served by
  `required_approving_review_count: 0`, whereas a repo with multiple maintainers
  expects a non-author approval (`count >= 1`, with GitHub blocking
  self-approval). The doctor reports this piece against that context rather than
  a fixed threshold; setting the value is the apply intent's job. (GitHub's
  `require_extra_approval_for_unattributed_changes` already dovetails with the
  four-role attribution model.)
- The tier summary: which rung this repository stands on, and what that rung
  does not hold, in plain words.
- Loud degradation throughout ([`loud-staging`](../../principles/loud-staging.md)):
  a probe that cannot answer says so; the summary never renders green above
  an unverifiable piece.

## What's Out of Scope

- **Applying** remote configuration — a separate intent, bound by adr-44
  rule 1, owning the bootstrap ordering the field note records twice.
- **The launch-gate refusal policy** (what a cut refuses on, per tier) — a
  decision record of its own once the doctor has field experience; until it
  exists the launch preflight makes no claim on these verdicts.
- Non-GitHub forge adapters — such forges report **unsupported**, honestly,
  rather than a fake unverifiable.
- Cost/plan advice (what the user's tier makes free) — planning-interview
  material, fed by field-note lesson 8.

## Acceptance Criteria

- Given Alice's repository has no git remote, When the doctor runs, Then
  every forge-side piece in the registry reports **unsupported** naming the
  absent remote, the local pieces (hooks, banlist, guard, preflight) report
  their true state, and the run exits as a supported state — never an error.
- Given Bob's personal public repository with secret scanning not enabled,
  When the doctor runs, Then the piece reports **absent** with the enabling
  lever, and where the plan cannot be read the report says
  enablement-undetermined rather than choosing between off and unavailable.
- Given Carol's live ruleset differs from the committed mirror under
  `.abcd/work/rulesets/`, When the doctor runs, Then the piece reports
  **applied but drifted**, naming the rule that moved (iss-277).
- Given a probe that lacks its tool, authentication, or permission (no `gh`,
  unauthenticated, an admin-only read as a non-admin), When the doctor runs,
  Then that piece reports **unverifiable for this caller**, and a summary
  containing any such piece does not render green.
- Given an in-tree rule with no executor (release retention, iss-282), When
  the doctor runs, Then the piece reports **rule without executor** —
  distinct from applied and from absent.
- Given a ruleset-required status check whose workflow does not exist on the
  default branch, When the doctor runs, Then the piece reports **applied**
  together with its stated limit — registration was verified, a living
  executor was not — and never a bare green.

## Open Questions

- **The caller axis needs modelling at planning:** multiple remotes (which is
  "the forge"?), GHES/tenant hosts where `gh` auth and features differ, and
  how a verdict says "unverifiable *for you*, verifiable for an admin".
- **The advisory-admin state:** without `enforce_admins`, protection is
  advisory against the admin's own token, and reading that flag is itself an
  admin-only probe. Where readable it is a stated limit under applied; where
  not, unverifiable — is that honest enough for the solo maintainer, the
  tool's primary persona?
- **Which protection shape is "protected enough" per tier?** Force-push and
  deletion blocks plus required checks as the floor; required reviews
  deadlock solo maintainers; the two-reviewer external rule (iss-281) is
  org-tier only. Per-repo template in `.abcd/config`?
- **Cost advice:** the doctor could say what the user's tier makes free
  (public-repo runner minutes against private-repo multipliers) — worth a
  column, or noise?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
