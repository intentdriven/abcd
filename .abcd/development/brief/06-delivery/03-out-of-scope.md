# Out of Phase Scope

This brief describes the work bundled into the eight planned phases (see [`roadmap/phases/README.md`](../../roadmap/phases/README.md)). **Later-phase items live as press-release intents**: the uncommitted bench in `.abcd/development/intents/drafts/` (enumerated by the commands below), and the committed-but-unscheduled intents in `planned/` — valid per [adr-34](../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md), listed in [intents/README.md](../../intents/README.md) § Planned, and scheduled when a phase doc's `## Scope` names them.

**In a later phase.** The bench is the live `drafts/` corpus — the
uncommitted bench. Per
[adr-34](../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md) no
phase-scoped intent lives in `drafts/` (scheduled ⇒ `planned/`), so there is
nothing to subtract: the filesystem is the list, and it is **not**
hand-counted —

```sh
# Live later-phase (uncommitted bench) IDs = the drafts/ corpus, no exclusions.
ls .abcd/development/intents/drafts/itd-*.md \
  | sed -E 's#.*/(itd-[0-9]+).*#\1#' | sort -V -u
```

Intents that have left `drafts/` (moved to `planned/`, `shipped/`, `superseded/`,
or `disciplines/`) are NOT in this list at all — the enumeration command cannot
emit them. (itd-31, itd-32 and itd-145, all superseded and moved to
`superseded/`, are recorded only in the historical notes at the end of this
section, not here.)

Each id with the record's own title, its H1, comes from the records the same
way:

```sh
grep -m1 '^# ' .abcd/development/intents/drafts/itd-*.md \
  | sed -E 's#^.*/(itd-[0-9]+)[^:]*:\# #\1  #' | sort -V
```

The list is not copied into this chapter. A committed copy, generated from
`drafts/` and checked byte for byte by record-lint, made every records pull
request that captured, planned or moved a draft rewrite neighbouring lines of
one sorted list, so two such pull requests open at once conflicted as soon as
the first merged, and a forge merge runs no generator to settle it
(iss-2610280107422462). The bench is read where it lives, so a capture, a
promotion or a supersession changes `drafts/` and nothing here, and a line can
never go stale. That is what keeps "not hand-counted" true after the day it
was written.

**Later-phase items with no intent id.** These four were written into the brief
before they were captured as intents, so no `itd-N` derives them and they sit
outside `drafts/`. Each is either superseded by a decision or waiting
for a capture pass:

- `.abcd/work/issues/` ledger cleanup bundle (sweep the workshop before a later phase)
- abcd warns when you reach past it into a tool it was built to hide — **obsolete under no-hard-deps ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md))**: with native defaults there is no wrapped foreign surface to reach past; the abstraction boundary is retired
- abcd's largest source files become navigable packages without changing behaviour
- One command re-vendors upstream and restores the abcd overlay in a single guarded step — **obsolete under no-hard-deps ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md))**: no external tool re-vendors itself onto abcd's state, so there is no overlay to re-apply

**Phased-in additions captured post-brief (2026-05-07):** itd-27 (`/abcd:intent grill` sub-verb + glossary), itd-28 (spec-tied reviews in the native spec review store), and itd-34 (three intent kinds with three lifecycle paths) were captured after this brief was written and are scoped into the planned phases. They are listed in `intents/README.md` and the relevant phase docs; this section is canonical for **later-phase** items only and does not enumerate phased-in intents.

**Later-phase additions captured post-brief (2026-05-07):** itd-30, itd-31, itd-32, and itd-33 were captured in the same audit pass. itd-30 remains on the `drafts/` bench, and itd-33 has since moved to `superseded/` too; itd-31 and itd-32 have since been superseded (itd-31 absorbed by itd-48; itd-32 superseded by itd-31) and moved to `intents/superseded/`, so they are no longer on the bench — this note records their capture timing and supersession for the brief's history. (See `superseded/itd-31-cross-document-fidelity-reviewer.md` and `superseded/itd-32-audit-role-taxonomy.md`.)

**Superseded addition (2026-08-28):** itd-145 (the acknowledgement convention arming itself, captured 2026-08-22) has been superseded by itd-163 on the `drafts/` bench, which delivers its mechanically checkable core, and moved to `intents/superseded/`. (See `superseded/itd-145-an-adopted-idea-cannot-ship-uncredited.md`.)

Each intent captures the press-release-shaped scope and acceptance criteria. A later-phase intent enters work by being scoped into a phase, then promoted to `planned/` via `/abcd:intent plan <itd-N>`. It reaches `shipped/` one way only: closing the last of the specs that realise it, which moves the intent as its close-hook — an intent owns one or more specs, so a spec that delivered only part of it closes on its own terms while the intent stays planned. There is no `intent ship` verb, so nothing promotes a record on its own — the close is a manual step run in the change that lands the work.

The brief does not get re-versioned. What has shipped is defined by which phases are complete and which intents are in `shipped/`; this brief stays the canonical current-state design record.
