---
id: itd-2609151658486398
slug: a-release-cut-publishes-the-security
spec_id: spc-2611241229018160
kind: standalone
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

A maintainer cuts a release that fixes several security advisories. When the release is published, every fixed advisory goes public with it: its fixed version stamped, the credits its reporters accepted, and an advisory whose record was closed as won't-fix is closed with its reason. In a repository whose technical facilitator works in auto mode, nobody is asked anything. abcd checks each advisory text against its fixing record and against the rules for public text, and publishes the ones that pass. Outside auto mode, the facilitator sees all the texts together once, at the cut, and one yes publishes them. The product thinker is never asked to read an advisory. No fixed advisory sits unpublished after its release, waiting for approvals one at a time.

## Why This Matters

Today each fixed advisory waits for its own approval after the release, so a release that fixes many leaves as many approvals queued. That leaves fixed issues undisclosed for days. Users who would upgrade if warned are not warned, and scanners cannot offer the fixed version. The per-text read was also put to a person who cannot judge it: the product thinker owns the brief and what good looks like, not security texts (dcn-2611241109230165).

## Mechanism

We expect publication to keep pace with the fixes, because the only step between a shipped fix and a public advisory becomes either automated (in auto mode) or one batch approval at a moment the facilitator is already present (the cut), rather than a queue of separate acts after it. It is shown wrong if, over the first two releases after this ships, a fixed advisory is still unpublished a day after its release without a recorded reason.

## Scope Conditions

- Repositories on GitHub with private security advisories enabled, whose advisories abcd's embargo store links to their fixing records. <!-- cond: cond-2611241229010906 -->
- Releases cut through `abcd launch ship` and published by the repository's release workflow. <!-- cond: cond-2611241229013962 -->
- The automated check judges consistency and the public-text rule; it does not judge whether a finding is a vulnerability, which triage already decided. <!-- cond: cond-2611241229010644 -->

## Acceptance Criteria

- **Given** a repository whose facilitator mode is auto and a cut whose published release holds the fix commits of three draft advisories, **when** the release is published, **then** each advisory whose text passes the automated check is published with the release's version as its patched version and its accepted credits, with no question asked of anyone.
- **Given** the same cut where one advisory's text fails the automated check (it names a private record, contradicts its fixing record, or breaks the public-text rule), **when** the release is published, **then** that advisory stays a draft, the others publish, and the facilitator is told which one failed and why.
- **Given** a repository whose facilitator mode is not auto, **when** the cut runs, **then** the facilitator is shown every advisory text the release will publish, in one place and in plain words, and one approval publishes all of them after the release is published; a declined text stays a draft.
- **Given** an advisory whose record was closed as won't-fix, **when** the release is published, **then** the advisory is closed with a comment giving the reason, never published.
- **Given** any mode, **when** advisories are published, **then** the product thinker is never asked to read or approve an advisory text.
- **Given** a published advisory judged a vulnerability, **when** the facilitator mode is auto, **then** a CVE is requested for it; outside auto, the facilitator chooses per advisory at the same approval.
- **Given** any publication, **when** it runs, **then** abcd holds no token: every forge call is the person's own `gh`, and the exact request bodies are recorded in the embargo store before they are sent.

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

_All resolved: see Decisions._

## Decisions

- resolved (2026-11-24, technical facilitator under facilitator.mode auto, dcn-2611241109230165): **The ecosystem a fixed advisory declares** is the repository's own toolchain ecosystem with its module or package name. For a Go module this is `go` with the module path, which reaches binary scanning as well as anyone who imports the module. `other` is used only where the repository declares no ecosystem.
- resolved (2026-11-24, technical facilitator under facilitator.mode auto, dcn-2611241109230165): **Where the advisory's id lives:** in the embargo store's state, as it does today. Never on the public record, because a record naming an embargoed advisory would disclose it before the fix ships.
- resolved (2026-11-24, technical facilitator under facilitator.mode auto, dcn-2611241109230165): **When a CVE is requested:** at publication, never at triage. Under auto, one is requested for every published advisory judged a vulnerability. Otherwise, the facilitator chooses per advisory, at the same batch approval.
- resolved (2026-11-24, product thinker): the product thinker neither decides nor reads advisory texts; under auto an automated check publishes at the cut, otherwise the facilitator approves the batch once (dcn-2611241109230165).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
