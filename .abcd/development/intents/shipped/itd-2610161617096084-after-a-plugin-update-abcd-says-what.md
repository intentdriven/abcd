---
id: itd-2610161617096084
slug: after-a-plugin-update-abcd-says-what
spec_id: spc-2610170635097509
kind: bundle-member
suggested_kind: null
reclassification_history: []
builds_on: [itd-111, itd-105]
severity: minor
origin: researcher-authored
production_mode: hand-written
bundle: after-update
impact: additive
---

# After a plugin update, abcd says what changed for this repository

## Press Release

> After a plugin update, abcd says what changed in words a person can use. When the next session notices that the abcd plugin was updated, the agent's first reply gives one line per release saying what changed, with the full summary one command away: what waits on this repository, and anything that now needs the person to act. Alice, a solo founder, sees: abcd 0.16.0, live brief sessions are new and need setup here; ask for details. They know what changed without reading the whole changelog. The summary comes from a short digest each release carries inside abcd itself, never fetched from the internet.

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

- The summary's source travels inside the abcd binary: a digest the release cut <!-- cond: cond-2610170635095619 -->
  writes, one line per shipped record, embedded at build time. The plugin root
  cannot be relied on, because the curated payload itd-108 drafts
  (`.abcd/config/launch-payload.json`) carries none of `CHANGELOG.md`,
  `RELEASE.md` or `.abcd/`.
- Releases cut by `abcd launch ship`, whose changelog sections are derived from <!-- cond: cond-2610170635090370 -->
  records and cite record ids; sections rolled by hand (0.6.0 and earlier) are
  summarised as written, or named as not summarised.
- The version the repository or machine moved from is known on disk: the <!-- cond: cond-2610170635095099 -->
  repository's `setup_version` or the binary record's `previous_tag`.
- Each digest line is the record's own changelog line as the release cut <!-- cond: cond-2610170635096007 -->
  composed it (the line `launch ship` checks against the record id it cites),
  so the "one plain sentence" has one source and is never rewritten for the
  summary.

## Acceptance Criteria

Seeded by the agent on 2026-10-16 from the code and the two drafts, unconfirmed:
the planning interview walks each one.

- **Given** Alice, a solo founder, whose abcd moved from 0.15.0 to 0.16.0, **when**
  the agent's first reply after the update is written, **then** it carries one
  line for the release, naming what waits on this repository and anything that
  needs Alice to act, and one abcd command shows the full summary: each change
  as its digest line with the record id to read more, changes that need action
  first.
- **Given** an update that skipped releases (0.14.2 to 0.16.0), **when** the
  summary is shown, **then** it covers every release in between, and names the
  releases it covers.
- **Given** Alice's repository has setup questions waiting from this update,
  **when** the summary is shown, **then** it keeps what waits on this repository
  (questions not yet answered here, from ahoy's own detection) apart from what
  changed in abcd in general; the general part is shown once per computer and
  what waits here once per repository.
- **Given** a binary whose digest has no entry for a release in the range (a
  release cut before digests existed), **when** the summary is built, **then** it says which releases it could not describe and
  where their notes are published, and never fills the gap with a guess or
  presents a partial summary as complete.
- **Given** any session, **when** the summary is built, **then** nothing is
  fetched from the network: it reads only the digest inside the running binary,
  and the build passes the zero-network test harness.
- **Given** Alice has been shown the summary for 0.16.0, **when** later sessions
  start, **then** it is not shown again, and one abcd command shows it again on
  request.
- **Given** a session with no person to read it (a headless or scripted run),
  **when** the session starts, **then** the summary is not counted as seen, so the
  next session with a person shows it.
- **Given** Maya, an AI/agent researcher, working in a repository whose own
  `CHANGELOG.md` is not abcd's, **when** the summary is built, **then** its text
  comes only from the digest inside the running binary, never from the
  repository being worked in, and every line is sanitised as every other text
  abcd reads at run time.

## Decisions

The product thinker answered four of these on 2026-10-16; the agent decided the
rest on 2026-10-17 during an autonomous run, on the grounds given.

- **Shown on its own or on request:** one line in the first reply, the full
  summary on request (product thinker).
- **Who sees it:** the general news once per computer, what waits here once per
  repository (product thinker).
- **How long:** one line per release (product thinker).
- **What counts as "for this repository":** only what waits here, plus anything
  that needs the person to act; everything else is in the full summary (product
  thinker).
- **Which record is the source:** a digest the release cut writes, one line per
  shipped record taken from its changelog line, embedded in the binary, because
  the curated payload carries no changelog or records and the binary is the one
  thing every install has (agent, from the design review).
- **Which channel:** the session-start hook adds only a fixed line saying a
  summary waits; the agent runs an abcd verb that prints the summary and relays
  it in its first reply, which keeps the hook's fixed-text rule
  (iss-2608241115201044) (agent).

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-272fb9bcc5f0 -->
Fidelity review OWED (receipt rcp-272fb9bcc5f0).
<!-- abcd-review-end receipt=rcp-272fb9bcc5f0 -->

## Grounds

- pursued: we expect people to act on a change that needs them within a session of updating, because one line per release in the first reply names it; a breaking change first noticed through a refused command after the summary was shown would show it wrong
