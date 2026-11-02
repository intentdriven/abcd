---
id: itd-2609151658486398
slug: a-release-cut-publishes-the-security
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A release cut publishes the security advisories its fixes close: when the release workflow cuts a version, every draft advisory whose fixing record shipped in that cut is published with the fixed version stamped and its opt-in credits, and an advisory whose record was closed as won't-fix is closed as not a vulnerability, so no fixed advisory sits unpublished after its release and no hand step decides which ones publish

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

A release cut publishes the security advisories its fixes close: when the release workflow cuts a version, every draft advisory whose fixing record shipped in that cut is published with the fixed version stamped and its opt-in credits, and an advisory whose record was closed as won't-fix is closed as not a vulnerability, so no fixed advisory sits unpublished after its release and no hand step decides which ones publish

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [install-update-release SOTA note](../../research/notes/2026-10-23-install-update-release-sota.md).
The 2026-08-27 pilot note's F-K, F-P and F-Q are the local evidence this
builds on.

- **The forge's REST surface does everything the title names**: `PATCH
  /repos/{owner}/{repo}/security-advisories/{ghsa_id}` takes `state`
  (`published`, `closed`, `draft`), `vulnerabilities[]` with
  `package.ecosystem` (`go` and `other` among the values),
  `vulnerable_version_range` and `patched_versions`, and `credits[]` of
  `{login, type}`, where "to add a credit, send the whole `credits` array";
  a CVE is requested at `POST .../{ghsa_id}/cve` (public repositories only);
  tokens need `repository_advisories:write`.
- **Credits are opt-in and the fix version must precede publication**: the
  public sees credits "the credited users have accepted"; without a fix
  version Dependabot alerts users "without offering any safe version to
  update to"; GitHub reviews a published advisory for its database within
  "up to 72 hours".
- **Publication is one-way**: a published advisory is deleted only "by
  contacting Support". The publish step therefore sits after the human
  release gate (pilot F-Q), never before it, and shows its exact request
  bodies first.
- **"Not a vulnerability" is a close with a comment first**: the maintainer
  flow offers "Close security advisory" and advises that "Where possible, you
  should add a comment explaining why you don't consider the report a security
  risk before you close the advisory"; no close-reason enumeration
  exists, the API's `state: closed` is the whole of it.
- **OSV is the interchange shape**: the database exports advisories "in the
  Open Source Vulnerability (OSV) format"; OSV's `fixed` event "is strongly
  preferred over `last_affected`", SEMVER ranges carry no leading `v`, and
  OpenSSF recommends publishing a JSON entry in the OSV schema as part of its
  CVE step and calls omitting credit "inappropriate" unless declined.
- **The ledger has no link to the advisory** (pilot F-E): the v0.6.7 cut
  published its advisories by hand after the release. An automated publish
  needs the fixing record to carry the GHSA id before the cut.

**Path 1 — adopt the forge API as it is**, called from `launch ship` after the
Release is published: for each draft advisory whose linked record shipped,
`patched_versions` is the derived version and `state` becomes `published`,
credits as already recorded on the advisory; for each linked to a won't-fix
record, a comment and `state: closed`. "The release workflow" in the title is
read here as the `launch ship` verb, not a CI workflow step, which the note
weighs and rejects (the secret would live in CI and a failed publish be
invisible at the terminal). One record change: the issue record gains an
`advisory:` field holding the GHSA id, set when the report is accepted as a
draft; that is an issue-schema change, a prerequisite for the record-lint and
schema owners. Over the pinned HTTP client the update package already
has, or `gh api`; no new Go dependency. The independent fit-challenge runs at
plan time.

## Open Questions

- (SOTA 2026-10-23) Which `package.ecosystem` does a Go binary whose
  importable packages are all `internal/` declare: `go` with the module path,
  so a reviewed advisory reaches anyone who vendors the module, or `other`?
- (SOTA 2026-10-23) Where does the GHSA id live: an `advisory:` field on the
  issue record, the record id in the advisory's description, or both?
- (SOTA 2026-10-23) Is the CVE requested at triage (reserved privately, "usually
  within 72 hours") or at publication, and who decides that a record warrants
  one?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
