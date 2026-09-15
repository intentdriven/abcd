---
id: itd-152
slug: autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib
spec_id: spc-45
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
promoted_from: iss-178
---

# Autonomous cloud runs in sibling repos leaked harness attribution footers and a live session URL into public GitHub artifacts: the harness auto-appends a 'Generated with' footer plus a session link when a PR or issue is created, overriding the repos' Assisted-by-only attribution policy (commit messages stayed clean; the leak surface was PR bodies and issue comments, plus GitHub's public edit history retaining the pre-scrub revision). Leak shape only - no session ids reproduced here. Two remedies needed: (a) every autonomous routine prompt must ban session URLs and harness footers in public text AND mandate a post-create re-read-and-strip of every PR/issue/comment the loop creates, because the append happens outside the model's own text; (b) abcd should detect the class - session-URL and harness-footer patterns belong with the shared privacy pattern set (iss-154 family / itd-74 banlist territory) so audit and docs-lint flag them in any committed or posted text.

## Press Release

> _Seeded by promotion from iss-178. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-178`: Autonomous cloud runs in sibling repos leaked harness attribution footers and a live session URL into public GitHub artifacts: the harness auto-appends a 'Generated with' footer plus a session link when a PR or issue is created, overriding the repos' Assisted-by-only attribution policy (commit messages stayed clean; the leak surface was PR bodies and issue comments, plus GitHub's public edit history retaining the pre-scrub revision). Leak shape only - no session ids reproduced here. Two remedies needed: (a) every autonomous routine prompt must ban session URLs and harness footers in public text AND mandate a post-create re-read-and-strip of every PR/issue/comment the loop creates, because the append happens outside the model's own text; (b) abcd should detect the class - session-URL and harness-footer patterns belong with the shared privacy pattern set (iss-154 family / itd-74 banlist territory) so audit and docs-lint flag them in any committed or posted text.. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an outbound PR body that carries a harness-appended attribution footer, **when** the scanner evaluates the artefact before it is posted, **then** the footer is caught and stripped so the posted text carries the repo's `Assisted-by:`-only attribution.
- **Given** an outbound issue comment containing a live session URL, **when** the scanner evaluates it, **then** the session URL is caught, whether the model authored it or the harness appended it.
- **Given** an outbound artefact whose text is already clean, **when** the scanner evaluates it, **then** it passes unchanged with no finding.
- **Given** the shared privacy pattern set, **when** it is applied by `abcd audit` and `docs-lint`, **then** the session-URL and harness-footer patterns are flagged in any committed or posted text, not only in freshly created PR bodies.
- **Given** an autonomous routine prompt assembled for a managed repo, **when** the prompt is composed, **then** it carries the policy that bans session URLs and harness footers in public text and mandates a post-create re-read-and-strip of every PR, issue and comment the loop creates.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
