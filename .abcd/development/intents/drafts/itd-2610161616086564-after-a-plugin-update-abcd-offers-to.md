---
id: itd-2610161616086564
slug: after-a-plugin-update-abcd-offers-to
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-111, itd-105]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# After a plugin update, abcd offers to bring the repository's setup up to date

## Press Release

> After a plugin update, abcd offers to bring the repository's setup up to date. When the abcd plugin has been updated since a repository was last set up, the next session opens by saying so and offering to run setup again, so new settings and new setup questions reach the repository instead of waiting until someone happens to run setup. Alice, a solo founder, updates the plugin in the evening; the next morning their session opens with: abcd was updated to 0.16.0, two new setup questions are waiting, run setup now? They answer yes, setup asks only the new questions, and their repository picks up the new settings. Nothing changes without that yes, and noticing the update reads only what is on disk, never the internet.

## Why This Matters

_Proposed by the agent on 2026-10-16 from the code and the ledger, unconfirmed: the planning interview confirms or rewrites it._

Setup (`ahoy install`) is the only step that brings a repository up to what the
installed abcd can do: the settings a new release writes, the questions it adds,
and the records it reshapes. The plugin updates on the host's schedule, one
machine at a time; a repository's setup moves only when someone runs setup, and
nothing tells them to. Release 0.15.0 is the example: `--outbound-hook` and
`--outbound-forge` became `--outbound`, and a repository that recorded only one
half "is reported, and the next `ahoy install` completes it". A repository whose
people never re-run setup never gets that completion. The README question
(itd-2610031348087517) reached existing repositories the same way: only when
someone happened to run setup again.

The fact is already on disk. `ahoy install` stamps `meta.setup_version` into
`.abcd/config.json`, and the bare `abcd ahoy` reports a `version.upgrade` gap when
the running abcd differs from it. But that gap is seen only by someone who
thinks to run `abcd ahoy`. Session start compared the two once and stopped
(iss-2609291942520919, dcn-2609300000000011): that ruling moved the "abcd updated
from X to Y" line to whatever performed the swap, once per machine. It did not
decide whether each repository's setup has caught up, which is the question this
intent asks.

## Mechanism

_Proposed by the agent on 2026-10-16, unconfirmed._

We expect a repository to pick up a release's new settings and questions within
one session of the update, because the offer arrives when the person is already
starting work in that repository, names what is waiting (how many new questions),
and costs one yes. Today the same fact sits behind a verb nobody runs without a
reason, so a repository's setup lags until something breaks.

## Scope Conditions

_Proposed by the agent on 2026-10-16, unconfirmed._

- The repository has been set up by `ahoy install` at least once, so
  `.abcd/config.json` records a release `setup_version`; a repository abcd does
  not manage gets no offer, because adopting it is ahoy's own path.
- Both the recorded setup version and the running binary are release versions;
  a dev build on either side takes no part, the rule `version.upgrade` already
  keeps (iss-2608241115259170).
- The host runs abcd's session-start hook and its agent can put a question to
  the person; a host without hooks learns of the waiting setup only from
  `abcd ahoy`.
- The update has reached the running binary: the binary's own version is the
  "updated to" side, not a plugin root whose binary has not been swapped yet.

## Acceptance Criteria

Seeded by the agent on 2026-10-16 from the code and the two drafts, unconfirmed:
the planning interview walks each one.

- **Given** Alice, a solo founder, whose repository was set up with 0.15.0 and
  whose plugin now runs 0.16.0, **when** their next session starts in that
  repository, **then** the session offers to run setup again, naming the version
  the repository was set up with, the version now running, and how many setup
  questions are new, and nothing is written to the repository before they answer.
- **Given** Alice answers yes, **when** setup runs, **then** it asks only the
  questions this repository has not answered yet, never re-asks one already
  answered with a value, and records the new setup version when it completes.
- **Given** Alice answers "not now", **when** later sessions start in that
  repository before the next update, **then** the offer is not repeated, and
  `abcd ahoy` still lists the waiting setup with the command that runs it.
- **Given** Alice declined the offer for 0.16.0, **when** the plugin moves on to
  0.16.1, **then** the next session in that repository offers once more.
- **Given** a session with no person to answer it (a headless or scripted run),
  **when** the session starts, **then** setup is not run and nothing is written;
  the waiting setup is reported as a notice only, and an `ahoy install --yes` run
  never counts as the person's yes to a new value question.
- **Given** Bob, a staff engineer, whose repository's setup version matches the
  running abcd, or where either side is a dev build, **when** a session starts,
  **then** no offer appears.
- **Given** Carol, a tech lead, whose teammate has already re-run setup and
  committed a setup version equal to or newer than Carol's plugin, **when**
  Carol's session starts, **then** no offer appears, and setup is never offered
  as a way to record an older version.
- **Given** any session start, **when** the offer is worked out, **then** nothing
  is fetched from the network (the versions come from the running binary and the
  repository's `.abcd/config.json`), and no word from the repository's tracked
  files reaches the session's context unchecked: the recorded setup version is
  shape-checked as a release tag or left off that channel.

## Open Questions

_Seeded by the agent on 2026-10-16, unconfirmed; each lists the defensible
answers, none recommended._

- **How often is the offer made?** (a) once per update per repository, then
  only `abcd ahoy` lists it; (b) every session until it is answered yes or no;
  (c) once per update per machine, whichever repository is opened first.
- **Where does the offer appear?** (a) as a question the agent asks at the
  start of the session; (b) as a session-start notice naming the command, with
  no question; (c) as a question asked at the first prompt, so a session opened
  for something urgent is not interrupted before it starts.
- **Where is "offered" or "declined" remembered?** Session start writes one
  file today, by an explicit single exception (dcn-2609300000000011). (a) a
  marker per repository and release in abcd's data directory, a second
  exception; (b) the repository's local tier, per worktree, so each worktree
  asks once; (c) nothing written, which makes (b) of the first question the only
  workable cadence.
- **Which questions count as new?** (a) only questions that did not exist at the
  recorded setup version; (b) every question this repository has no value for,
  including ones answered "decide later" before; (c) (b), but "decide later"
  answers wait until the person runs setup by hand.
- **Is the offer one sentence or the full list?** (a) a count of new questions;
  (b) the new questions' titles; (c) the count, and the titles on request.
- **Does a plugin update that adds no new question still offer setup?** (a)
  yes, because settings and records may still change; (b) no, only when a
  question or a required change waits; (c) offer only when ahoy reports a
  required gap.
- **Bundle shape with the summary draft (itd-2610161617096084).** (a) one
  session-start moment: summary first, then the offer; (b) two separate
  moments; (c) the offer is one line of the summary.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
