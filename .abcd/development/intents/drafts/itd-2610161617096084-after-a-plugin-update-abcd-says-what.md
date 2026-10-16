---
id: itd-2610161617096084
slug: after-a-plugin-update-abcd-says-what
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-111, itd-105]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# After a plugin update, abcd says what changed for this repository

## Press Release

> After a plugin update, abcd says what changed in words a person can use. When the next session notices that the abcd plugin was updated, it shows a short summary of what the update means for this repository: new features to try, new setup questions waiting, and anything that now works differently, each in one plain sentence with where to read more. Alice, a solo founder, sees: abcd 0.16.0, live brief sessions are new and need setup, the session page now opens on the brief; read more in the release notes. They know what changed without reading the whole changelog. The summary is built from the release's own records on disk, never fetched from the internet.

## Why This Matters

_Proposed by the agent on 2026-10-16 from the code and the ledger, unconfirmed: the planning interview confirms or rewrites it._

After an update, abcd says only "abcd updated from X to Y" (`update.UpdatedFormat`,
shown once by whatever made the swap). What changed is in the changelog, and the
changelog is long: the 0.15.0 section is eight paragraph-length lines, three of
them marked "Breaking:" that ask a person to change what they type (the bare
`abcd decide` no longer mints an ADR; `--outbound-hook` and `--outbound-forge`
are refused). Nobody reads that at the start of a working session, so a person
learns of a breaking change when a command they relied on refuses, and of a new
feature not at all.

The release's own records already say what changed, release by release, each
line citing its record: `CHANGELOG.md`'s dated sections, `RELEASE.md`'s press
release and its archive under `.abcd/development/releases/`, and the shipped
intents' press releases. Today they sit on disk in the plugin root, because the
plugin root carries the repository. Not every line matters to every repository:
the outbound change matters only to a repository that opted into it, and a new
setup question matters only where it has not been answered.

## Mechanism

_Proposed by the agent on 2026-10-16, unconfirmed._

We expect people to act on a breaking change, and to try a new feature, within a
session of updating, because a short summary shown at the first use of the new
version, built from the release's own records, gives each change in one sentence
with where to read more. Today the same facts exist only in a changelog the
person has to think to open.

## Scope Conditions

_Proposed by the agent on 2026-10-16, unconfirmed._

- The plugin root carries the release's records (`CHANGELOG.md`, `RELEASE.md`,
  `.abcd/development/`), as it does while the marketplace installs the whole
  repository. The curated payload itd-108 drafts (`.abcd/config/launch-payload.json`
  names neither `CHANGELOG.md` nor `.abcd/`) would remove them, so under itd-108
  the summary's source has to travel in the payload or in the binary.
- Releases cut by `abcd launch ship`, whose changelog sections are derived from
  records and cite record ids; sections rolled by hand (0.6.0 and earlier) are
  summarised as written, or named as not summarised.
- The version the repository or machine moved from is known on disk: the
  repository's `setup_version` or the binary record's `previous_tag`.
- The plugin root has no git history (an installed root is a plain copy), so the
  summary reads files, never tags or commits.

## Acceptance Criteria

Seeded by the agent on 2026-10-16 from the code and the two drafts, unconfirmed:
the planning interview walks each one.

- **Given** Alice, a solo founder, whose abcd moved from 0.15.0 to 0.16.0, **when**
  the summary is shown, **then** it gives each change in one plain sentence with
  the record id or release section to read more, breaking changes first, within a
  fixed length, ending with how many more changes the full section holds and
  where it is.
- **Given** an update that skipped releases (0.14.2 to 0.16.0), **when** the
  summary is shown, **then** it covers every release in between, and names the
  releases it covers.
- **Given** Alice's repository has setup questions waiting from this update,
  **when** the summary is shown, **then** it keeps what waits on this repository
  (questions not yet answered here, from ahoy's own detection) apart from what
  changed in abcd in general.
- **Given** a plugin root missing a release's records (no section for a release
  in the range, no `CHANGELOG.md`, or a payload that carries none), **when** the
  summary is built, **then** it says which releases it could not describe and
  where their notes are published, and never fills the gap with a guess or
  presents a partial summary as complete.
- **Given** any session, **when** the summary is built, **then** nothing is
  fetched from the network: it reads only files in the plugin root of the
  running release, and the build passes the zero-network test harness.
- **Given** Alice has been shown the summary for 0.16.0, **when** later sessions
  start, **then** it is not shown again, and one abcd command shows it again on
  request.
- **Given** a session with no person to read it (a headless or scripted run),
  **when** the session starts, **then** the summary is not counted as seen, so the
  next session with a person shows it.
- **Given** Maya, an AI/agent researcher, working in a repository whose own
  `CHANGELOG.md` is not abcd's, **when** the summary is built, **then** its text
  comes only from the plugin root the running binary belongs to, never from the
  repository being worked in, and every line is sanitised as every other text
  abcd reads at run time.

## Open Questions

_Seeded by the agent on 2026-10-16, unconfirmed; each lists the defensible
answers, none recommended._

- **Shown on its own, or on request?** (a) shown once at the first session after
  the update; (b) a one-line notice that a summary is waiting, shown in full on
  request; (c) only on request.
- **Who sees it: each repository or each machine?** (a) once per machine per
  update; (b) once per repository per update, since the repository part
  differs; (c) per machine for the general part, per repository for what waits
  here.
- **What makes a change "for this repository"?** (a) only the setup questions
  and settings waiting here, everything else general; (b) records gain a field
  naming what they touch, so the summary can leave out what this repository does
  not use; (c) no filtering beyond breaking changes first.
- **Which record is the source?** (a) the `CHANGELOG.md` sections; (b) the
  `RELEASE.md` press releases, which are shorter but exist only for feature
  releases; (c) a short digest the release cut writes for this purpose, which
  also survives itd-108's curated payload.
- **How long is it?** (a) a fixed number of lines, the rest counted; (b) one line
  per release; (c) breaking changes in full, the rest counted.
- **Which channel?** The session-start hook may put only fixed text into the
  session's context today (iss-2608241115201044); the rest goes to the hook's
  stderr, which the session itself does not read. (a) the agent runs a verb that prints the summary and relays
  it; (b) the notice channel only; (c) a file the status line or a command points
  at.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
