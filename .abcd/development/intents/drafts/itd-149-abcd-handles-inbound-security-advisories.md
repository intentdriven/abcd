---
id: itd-149
slug: abcd-handles-inbound-security-advisories
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# abcd handles inbound security advisories and issues, and cuts the release, for every managed repo. The 2026-08-27 security-advisory pilot proved the loop by hand across two independent operators: advisory and issue intake, recall-dedup triage against the ledger, record-first fixes in isolated worktrees, one integration branch, an independent adversarial gate over the assembled diff, auto-merge armed only after APPROVE, and a derived release cut through the semantic-receipt gates. This intent automates that loop as abcd capability for any managed repo, with findings F-A through F-W of the pilot note as the acceptance-criteria source and F-U (re-run every required pass against the final content commit; never transfer a verdict) and F-Q (the two release gates are human-by-design) load-bearing.

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

abcd handles inbound security advisories and issues, and cuts the release, for every managed repo. The 2026-08-27 security-advisory pilot proved the loop by hand across two independent operators: advisory and issue intake, recall-dedup triage against the ledger, record-first fixes in isolated worktrees, one integration branch, an independent adversarial gate over the assembled diff, auto-merge armed only after APPROVE, and a derived release cut through the semantic-receipt gates. This intent automates that loop as abcd capability for any managed repo, with findings F-A through F-W of the pilot note as the acceptance-criteria source and F-U (re-run every required pass against the final content commit; never transfer a verdict) and F-Q (the two release gates are human-by-design) load-bearing.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).
The 2026-08-27 pilot note (F-A to F-W) is the acceptance-criteria source and
is reused, not redone; the bullets below are what the forge and the
disclosure bodies document that the pilot did by hand.

- **Private vulnerability reporting has a full API surface**: reporters file
  at `POST /repos/{owner}/{repo}/security-advisories/reports`; maintainers
  list by `state=triage`, accept into a draft, open a temporary private fork
  (asynchronous, "up to 5 minutes"), and add collaborators. OpenSSF warns
  that such forks "do not have access to integration like CI systems", so
  tests run locally, which is the pilot's worktree discipline (F-C).
- **The triage guidance looks for an AI-assistance disclosure**: it tells the
  maintainer to review "any disclosure of AI assistance"; no structured field
  is documented, so the disclosure is whatever the reporter volunteers, a
  provenance signal the pilot's F-H intake records when it is there, not one
  it can rely on.
- **The forge offers ten credit types, and three of the pilot's four roles
  have one**: of `analyst`, `finder`, `reporter`, `coordinator`,
  `remediation_developer`, `remediation_reviewer`, `remediation_verifier`,
  `tool`, `sponsor` and `other`, `finder` or `reporter` covers F-I's reporter,
  `remediation_developer` the author of record, and `remediation_reviewer` or
  `remediation_verifier` the independent gate; the authoriser has no direct
  type (`coordinator` or `other` are the candidates), so the advisory carries
  most, not all, of the roles the commit trailers carry.
- **Timelines have reference points** (OpenSSF maintainer guide): acknowledge
  within 1–2 days; embargo defaults of under 7 days (distros), 45 days
  (CERT/CC) and 90 days (Google), "90 days is the longest"; publish the OSV
  entry with the CVE; withholding exploit details helps "for a few days at
  most"; a partial fix that breaks the exploitation chain beats none and is
  documented as partial. `SECURITY.md` promises acknowledgement within a
  week, slower than the guide's 1–2 day target; whether to tighten it is a
  question for the loop's intake stage.
- **Only reviewed advisories drive Dependabot**; a repository advisory becomes
  reviewed through the forge's post-publication review.
- **Attribution still reads the running model** (pilot F-G): no source opened
  records which model assisted a fix; the transcript stays the only authority.

**Path 2 — native loop on the pilot's protocol, shaped to the OpenSSF
maintainer flow and the forge's advisory API, with the forge adapter as the
seam**: intake records an AI-assistance disclosure as provenance where the
report volunteers one, credits are filed by the types that match (the
authoriser's is open below), the adversarial gate over the assembled diff
(F-D) is a host agent, and the two human gates (PROMOTE and publish, F-Q) stay
human. itd-2609151658486398 is this loop's last stage and ships first. No new
Go dependency. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) Does the loop's intake record the report's AI-assistance
  disclosure on the ledger record, beside the external-contributor provenance
  F-H asks for?
- (SOTA 2026-10-23) Which forge credit type the authoriser takes
  (`coordinator`, `other`, or none), which types the loop stamps itself, and
  which only on a person's say-so (credit is opt-in on the credited side, so
  none shows before acceptance)?
- (SOTA 2026-10-23) Does `SECURITY.md`'s promise of an acknowledgement within
  a week tighten to the OpenSSF guide's 1–2 days once the loop has an intake
  stage to keep it?
- (SOTA 2026-10-23) Does the shell guard judge `gh api` advisory bodies the
  way it judges `gh pr` and `gh issue` text, so no unpublished GHSA id leaves
  the machine in a comment?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
