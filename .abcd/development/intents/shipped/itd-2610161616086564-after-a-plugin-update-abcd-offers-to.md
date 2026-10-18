---
id: itd-2610161616086564
slug: after-a-plugin-update-abcd-offers-to
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

# After a plugin update, abcd offers to bring the repository's setup up to date

## Press Release

> After a plugin update, abcd offers to bring the repository's setup up to date. When the abcd plugin has been updated since a repository was last set up and the update brings something the repository needs, the agent's first reply in the next session says so and offers to run setup again, so new settings and new setup questions reach the repository instead of waiting until someone happens to run setup. Alice, a solo founder, updates the plugin in the evening; the next morning the first reply to their first message says: abcd was updated to 0.16.0, two setup questions are waiting here, run setup now? They answer yes, setup asks only the new questions, and their repository picks up the new settings. Nothing changes without that yes, and noticing the update reads only what is on disk, never the internet.

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

- The repository has been set up by `ahoy install` at least once, so <!-- cond: cond-2610170635090059 -->
  `.abcd/config.json` records a release `setup_version`; a repository abcd does
  not manage gets no offer, because adopting it is ahoy's own path.
- Both the recorded setup version and the running binary are release versions; <!-- cond: cond-2610170635091194 -->
  a dev build on either side takes no part, the rule `version.upgrade` already
  keeps (iss-2608241115259170).
- The host runs abcd's session-start hook and its agent can put a question to <!-- cond: cond-2610170635092426 -->
  the person; a host without hooks learns of the waiting setup only from
  `abcd ahoy`.
- The update has reached the running binary: the binary's own version is the <!-- cond: cond-2610170635092640 -->
  "updated to" side, not a plugin root whose binary has not been swapped yet.

## Acceptance Criteria

Seeded by the agent on 2026-10-16 from the code and the two drafts, unconfirmed:
the planning interview walks each one.

- **Given** Alice, a solo founder, whose repository was set up with 0.15.0 and
  whose plugin now runs 0.16.0 and adds a setup question they have never
  answered, **when** they send the first message of their next session in that
  repository, **then** the agent's first reply offers to run setup again in one
  line, naming the version the repository was set up with, the version now
  running, and how many questions and changed defaults wait, with their titles
  on request; nothing is written to the repository before they answer.
- **Given** an update that brings this repository nothing it needs (no question
  it has never answered and no setting whose default changed), **when** the next
  session starts, **then** no offer appears, and `abcd ahoy` stays the place to
  check.
- **Given** Alice answers yes, **when** setup runs, **then** it asks only the
  questions this repository has never answered and the settings whose default
  changed in the update, never re-asks a question answered with a value or
  deferred with "decide later", and records the new setup version when it
  completes.
- **Given** Alice answers "not now", **when** later sessions start in that
  repository before the next update, **then** the offer is not repeated, and
  `abcd ahoy` still lists the waiting setup with the command that runs it.
- **Given** Alice answers the offer either way, **when** the answer is recorded,
  **then** the abcd verb the agent runs with her answer records it, keyed on the
  repository and the release, in abcd's own data directory; the session-start
  hook writes nothing for it.
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
  checked to be exactly a release tag of the form vX.Y.Z (digits only) before it
  reaches the session's context, and any other value is left off that channel.

## Decisions

The product thinker answered four of these on 2026-10-16; the agent decided the
rest on 2026-10-17 during an autonomous run, on the grounds given.

- **How often:** once per update per repository; after that only `abcd ahoy`
  lists it (product thinker).
- **Where it appears:** asked in the agent's first reply, after the person's
  first message, so a session opened for something urgent is not interrupted
  before it starts (product thinker).
- **Which questions count:** every question this repository has never answered,
  leaving out those deferred with "decide later", plus every setting whose
  default changed in the update (product thinker).
- **An update with nothing needed:** offered only when something is required;
  an update that brings no such question or setting offers nothing (product
  thinker).
- **Where the answer is remembered:** in abcd's data directory, keyed on the
  repository's root commit and the release, and written by the verb the agent
  runs with the answer, never by the session-start hook, so the hook's single
  write exception (dcn-2609300000000011) stays single (agent, from the design
  review).
- **One sentence or the list:** one line with the count, the titles on request,
  matching the summary's one-line form (agent).
- **Bundle shape with itd-2610161617096084:** one moment in the first reply:
  the one-line summary of what changed, then the offer (agent).

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-139f1fd28b60 -->
Fidelity review OWED (receipt rcp-139f1fd28b60).
<!-- abcd-review-end receipt=rcp-139f1fd28b60 -->

## Grounds

- pursued: we expect a repository to pick up a release's new questions and changed defaults within one session of the update, because the offer arrives in the first reply and costs one yes; repositories still on an old setup version a week after an update would show it wrong
