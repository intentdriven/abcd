# Out of Phase Scope

This brief describes the work bundled into the eight planned phases (see [`roadmap/phases/README.md`](../../roadmap/phases/README.md)). **Later-phase items live as press-release intents**: the uncommitted bench in `.abcd/development/intents/drafts/` (enumerated below), and the committed-but-unscheduled intents in `planned/` — valid per [adr-34](../../decisions/adrs/0034-lifecycle-and-scheduling-orthogonal.md), listed in [intents/README.md](../../intents/README.md) § Planned, and scheduled when a phase doc's `## Scope` names them.

**In a later phase.** The set below is the live `drafts/` corpus — the
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

The list is gated rather than trusted: the `index_drift` record-lint rule holds
the marked region to the ids in [`drafts/`](../../intents/drafts/), so a capture,
a promotion, or a supersession that does not edit this list fails the record
gate. That is what keeps "not hand-counted" true after the day it was written.

<!-- index: later-phase-intents -->
- `itd-8` — `--with-code` bundling (lifeboat carries source code)
- `itd-9` — Cross-version lifeboat schema migration
- `itd-10` — `/abcd:ahoy destroy` deeper uninstall
- `itd-11` — Pass B transcript-noise mitigation
- `itd-12` — `.abcd/work/notes/` distiller weighting
- `itd-13` — Scheduled `dev-sync` (cron / launchd)
- `itd-14` — Prompt registry + versioning (heavier successor to itd-5)
- `itd-15` — Self-dogfooded SOTA audit (recurring per-disembark sibling to itd-5)
- `itd-16` — `/abcd:audit` umbrella + chain substrate (default application: hash-chain over conversation/edit history; reframed as umbrella on 2026-05-08, lifeboat-integrity application extracted to itd-35)
- `itd-18` — `.claude/settings.local.json` permission templates
- `itd-19` — ABCDevelopment stage-aware defaults
- `itd-21` — `/abcd:init-project` empty-repo scaffolding
- `itd-22` — Harness portability — the shared adaptor machinery for multi-harness support (host profile seam, adaptor ladder, parity suite) under the host-tier policy ([adr-39](../../decisions/adrs/0039-host-tier-policy.md), mechanism per [adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)); per-host adoption files as its own intent at an explicit decision; awaiting the planning interview
- `itd-23` — Spec Kit interop
- `itd-25` — `/abcd:dredge` cross-corpus synthesist (split from itd-4 capture)
- `itd-26` — `/abcd:loot` OSS-vendor with provenance (pulled to an earlier phase on 2026-05-08)
- `itd-30` — Design fictions as an alternative intent capture format (`--format=fiction`)
- `itd-35` — `/abcd:audit lifeboat <path>` lifeboat-integrity verification (sibling sub-verb under itd-16's umbrella; captured 2026-05-08)
- `itd-39` — Scope-aware memory retrieval (extends itd-3's recall hook to the memory store)
- `itd-41` — Phase negotiator — Socratic phase-proposer (per [adr-10](../../decisions/adrs/0010-phase-negotiator-grounded-tradeoffs.md))
- `itd-44` — A fourth intent kind for infrastructure choices the product thinker wants to record
- `itd-51` — Harness-adoption-readiness rubric ("safe enough to adopt" before a new harness arrives)
- `itd-55` — abcd can tell whether its own reasoning rests on bedrock or an unexamined assumption
- `itd-57` — Manual-hold sentinel blocking a spec from autonomous pickup until a human lifts it
- `itd-59` — Autonomous-run passes leave the same durable, queryable transcript an interactive session does
- `itd-61` — Brief-change derivation: a human brief edit reconciles its implied intents and principles before commit
- `itd-62` — Pluggable fail-closed safety gate wrapping a trusted scanner
- `itd-64` — Benchmark-driven configuration optimisation from abcd's own runs
- `itd-70` — Launch release retention (newest-per-line prune of superseded releases)
- `itd-75` — CLI eval harness: fixture-driven proof the CLI actually runs
- `itd-77` — Relocatable user-level home
- `itd-126` — Team bibliography share/ingest: citation data travels the repo, corpora never do
- `itd-127` — Paper reconstruction from the provenance ledger
- `itd-128` — One canonical YAML scalar resolver: every decoder delegates to one exported frontmatter helper
- `itd-129` — Forge mirror as an opt-in adapter: one-way mirror-out, schema'd forge id, self-healing closures
- `itd-78` — Intent-dependency graph: what to build first, even when it is something small
- `itd-83` — The review bar fires by itself, in every repo abcd manages
- `itd-85` — Read-only repo-conformance audit
- `itd-86` — Cold-reading surface: abcd reads its own design documents as a stranger would
- `itd-87` — Recurrence escalation in capture: a finding that returns after closure is kept, not discarded
- `itd-90` — Brief interview for the blanks: the product thinker is handed only the questions they alone can answer
- `itd-91` — AI-attribution preference declared once and followed by every commit, PR, and record
- `itd-92` — Branch-protection verification on managed repos, gating the launch
- `itd-97` — The facilitator is a mode, not a person
- `itd-98` — Solo vs duo is measured, not debated
- `itd-99` — A team of product thinkers decides as one
- `itd-106` — abcd sets up the CI a repo requires, and reports what it did
- `itd-107` — Autonomous routines assemble from one versioned template
- `itd-108` — The plugin installs from the curated release artifact, not the repo, and every cut release reaches users automatically
- `itd-109` — Acceptance criteria verify themselves; the manual rest renders for a human (`abcd verify`, sha-keyed receipts)
- `itd-113` — The MCP front door opens — abcd's core verbs from any MCP-capable harness (the [adr-39](../../decisions/adrs/0039-host-tier-policy.md) universal floor)
- `itd-115` — A ready PR merges without ever wedging BEHIND (managed-repo merge queue by default, rung-1 auto-update fallback, strict preserved)
- `itd-116` — Validated GitHub issues become ledger entries without retyping (capture extension adopts externally filed findings with provenance; mint stays capture-only)
- `itd-118` — Merged work leaves no residue (post-merge complement of itd-115: delete the PR branch on merge, tidy the stale local branch, tracking ref, and worktree)
- `itd-134` — Managed-repo banner generator: a managed CLI in any language opens with its own identity, rendered from its identity block (split from itd-112)
- `itd-142` — The brief-creation interview: staged elicitation into the brief and a ledger (frontier rounds, options at conjectural questions, hold register, two-output rule per adr-50); spec waits on the collaborating prototype's first run
- `itd-143` — The framing chapter under 01-product/: the macro-why home, with its brief↔lifeboat mapping row; receives itd-142's committed framing products
- `itd-144` — Every livery mark has a surface: the lifeboat on disembark and mirrored on embark, the duckling as the harness mascot, the flag icon for the website (settles itd-112's deferred forge/web logo question)
- `itd-149` — abcd handles inbound security advisories and issues, and cuts the release, for every managed repo (the 2026-08-27 pilot's loop automated; findings F-A…F-W are the acceptance-criteria source, F-U and F-Q load-bearing; filed after the itd-84 SPLIT, awaiting the planning interview)
- `itd-163` — Reference-closure and acknowledgements-mirror gate: every citation resolves to the CSL references, the references and acknowledgements mirror both ways, and a committed influence registry backs the Inspirations list (supersedes itd-145; filed from the 2026-08-28 attribution review with the backfill issue iss-2608280824478819)
- `itd-164` — Licence vetting at source admission: `docs cite refresh` records each source's licence verdict into the committed baseline, and the zero-network gate refuses a new entry without one (builds on itd-163)
- `itd-159` — the repo visibility model has a committed-record mode between private and public, with the matching fence-suppression (graduated from iss-223)
- `itd-165` — A failed fidelity verdict becomes work somebody can see (Phase 8 adjacent; the ratchet half is split out as a seed)
- `itd-166` — A run records what it was actually run with (facilitator-tier diagnostics)
- `itd-167` — The product thinker answers a stop in a medium they already use (Phase 8; adr-2609151528057131 gives abcd that surface)
- `itd-168` — The product thinker sets how the system talks to them (Phase 7, the legible surface)
- `itd-169` — An agent loop that stops says who it is asking and what it needs (Phase 8)
- `itd-170` — What the product thinker reports after using the product finds its way back to the promise that predicted it (Phase 8)
- `itd-171` — A decision points back to the conversation where it was reached (Phase 8)
- `itd-172` — Every record has a short title and a one-line summary a non-engineer can read
- `itd-173` — Verification escalates from the built-in check to an outside audit (security is the first rung built out)
- `itd-174` — Each repository configures how far its facilitator is consulted and when it escalates (sequenced after itd-169)
- `itd-175` — The product thinker writes down how this could be wrong, and what would show it (Phase 8; the defeater list an acceptance rests on)
- `itd-176` — Whatever ships says how hard anyone looked at it (Phase 7)
- `itd-2609151838312703` — sessions on one machine or one local network leave each other messages in a shared mailbox abcd owns (the built-in basic; nothing leaves the local network)
- `itd-2609151838327688` — an opt-in adapter to a local message broker brings push delivery and cross-machine reach to the session mailbox (sequenced after the mailbox)
- `itd-2609151658486398` — a release cut publishes the security advisories its fixes close, and closes those resolved as won't-fix (the publication step the 2026-08-27 advisory-handling pilot named as its target)
- `itd-2609091416304128` — `capture resolve` and `capture wontfix` refuse a record already terminal at the local `origin/main` ref as last fetched, stating the ref's age and performing no fetch; the same judgement rendered read-only on `abcd <record-id>` (split from itd-2609091034175565 on the same ruling; the third clause of iss-2609020716570699's remedy, RS001's answer moved earlier)
- `itd-2609150819439571` — errata as a fourth terminal disposition on a durable record, appended rather than edited, so a correction is distinguishable from the error it corrects (promoted from iss-2609100505146979)
- `itd-2609150819440345` — a claim record beside the machine-scoped worktree store says which session holds which worktree, branch or record, replacing the per-session handshake (promoted from iss-2609100519122086)
- `itd-2609151138388536` — the decisions log becomes a folder of individually minted decision records with an assembled index, `DECISIONS.md` a symlink to it, in abcd and in every managed repository; the shape retires the decisions-append gate (the rule is adr-2609151138420062; promoted from iss-2609100507439414)
- `itd-2609151516525843` — a committed declaration lifts the public visibility fence so a fresh public repository can create its committed banned-names layer on day one, and a machine-global private banned-names list in the user-level home bans a name in every repository on the machine; CI never reads the home list and no pattern from it reaches a committed file (`builds_on` itd-74, `refines` adr-56; promoted from iss-2609100506269348)
- `itd-2609090746410233` — A lifeboat packs from a worktree, a branch, or an abandoned feature test, experiment or implementation, not only from a whole repository (refines itd-88 and adr-35; realises the press release's not-yet-real widening, git-source half)
- `itd-2609090746414083` — A lifeboat packs from a lab session home, the throwaway experiment's intention, harvest and bundle, with the same coverage honesty as a repository (refines itd-88 and adr-35; the non-git half, sequenced after the lab verb family)
- `itd-2609180517121254` — every payload a host hands back from a delegated step names the model that produced it and the number of agents that ran, and the ingesting verb refuses one that does not
- `itd-2609231507251267` — Release pages on the project website, rendered from `RELEASE.md` and the release archive (builds on itd-2609231013154443; next cycle)
- `itd-2609291924469783` — Every lint warning rule shares one baseline file that only shrinks, so a new warning fails the change that introduces it (from iss-46; ruling BT3)
- `itd-2609292106557115` — A fresh machine installs abcd through one small starter that hands over to a trusted abcd binary (from iss-377; ruling J2)
- `itd-2609292107351737` — A development session's token cost is measured, and set against the size class its spec predicted (from iss-2608301744251874, iss-2608301856299268 and iss-2608220150157508; rulings J3 and J4)
- `itd-2609292108089653` — Ingesting and consulting sources are abcd verbs, and no corpus entry is a keyword stub (from iss-27 and iss-55; rulings J5 and J6)
- `itd-2609292108373494` — The grill hands the interview to an installed interviewing skill, and runs its own when there is none (from iss-165; ruling J11)
- `itd-2609292109005937` — An opt-in local model checks every prompt for secrets and personal data before it leaves the machine (from iss-2608261543489261; ruling J14)
- `itd-2609292109214516` — Rule injection is a seam: the native loader stays the default, and an opt-in CARL back end can take it over (from iss-64; ruling J22)
- `itd-2609292109475690` — Every verb family has a behavioural scenario that drives the built binary end to end (from iss-48; ruling J24)
- `itd-2609301020001595` — A pinned-action bump syncs its scaffold template and lands re-authored, the pin half of the dependency re-authoring (builds on itd-2609221842494980; from iss-209; ruling M3)
- `itd-2609301918174237` — A provider off this machine takes a file-reading agent only as a fixed, blanked, capped bundle (builds on itd-2609081951381895; rulings DR5b-1 to DR5b-4; back to draft for two reviews and a planning interview under ruling DR5c-0)
- `itd-2610021503208208` — Personal git hooks keep running in abcd-managed repositories (draft)
- `itd-2610031325050110` — abcd notices a harness update downloaded but not yet running (draft; builds on itd-2610031026190632; filed separately at that draft's interview)
- `itd-2610031651058674` — abcd's check reports README parts that have gone stale, and writes nothing (draft; builds on itd-2610031348087517)
- `itd-2610031259176838` — Outside contributors reserve work through a draft pull request that abcd honours (draft; builds on itd-2609150819440345; not urgent)
- `itd-2610040740108331` — The dashboard on the home network, without Tailscale (draft; builds on itd-2610032150577708; later, by the product thinker's choice of Tailscale first)
- `itd-2610040740122709` — Acting from the dashboard: rewriting the brief and approving intents (draft; builds on itd-2610032150577708)
- `itd-2610040740135705` — The dashboard for more people: the facilitator's view and the team's, configurable (draft; builds on itd-2610032150577708)
- `itd-2610040754440360` — The private dashboard for every abcd-managed project (draft; builds on itd-2610032150577708; proven on a second, sparse sample project; later)
- `itd-2610040754453237` — The public record site for every abcd-managed project (draft; later, by the product thinker's ruling)
- `itd-2610040817105016` — Every brief section shows what realises it, and a brief change names what it touches (draft; refines itd-61; revisited after the downstream lab)
- `itd-2610040822131032` — A lab runs end to end from a question the product thinker asks (draft; builds on itd-2609212137128014; shaped after the downstream brief lab)
- `itd-2610050548126044` — abcd reminds a session to refresh its handover note only when that session changed the tree (draft; takes over a personal stop hook)
- `itd-2610052000411162` — Anyone opens, edits on their own device and submits a public project's brief on abcdesign.app (draft; builds on itd-2610040754453237)
- `itd-2610052000424067` — A group works on one public brief together, adds notes and exports the combined work (draft; builds on itd-2610052000411162)
- `itd-2610070544145422` — Someone on Windows installs abcd and it just works, the tool in PowerShell and the plugin in Claude Code, with no WSL (draft)
- `itd-2610070549060530` — A word in a new idea marks it to revisit, and the board puts it first (draft)
- `itd-2610070550516046` — On the dashboard, a product thinker sees the brief and their ideas together, and puts the ideas in order (draft)
- `itd-2610071545369520` — A person confirms the assistant's verdict on each warned banned-name line instead of judging it alone (draft)
- `itd-2610071545380041` — Work about abcd itself goes to abcd as a report, never into a managed repository's own plans (draft)
- `itd-2610090831227812` — A person's /abcd: list shows only the commands people use (draft)
<!-- /index -->

**Later-phase items with no intent id.** These four were written into the brief
before they were captured as intents, so no `itd-N` derives them and they sit
outside the gated list above. Each is either superseded by a decision or waiting
for a capture pass:

- `.abcd/work/issues/` ledger cleanup bundle (sweep the workshop before a later phase)
- abcd warns when you reach past it into a tool it was built to hide — **obsolete under no-hard-deps ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md))**: with native defaults there is no wrapped foreign surface to reach past; the abstraction boundary is retired
- abcd's largest source files become navigable packages without changing behaviour
- One command re-vendors upstream and restores the abcd overlay in a single guarded step — **obsolete under no-hard-deps ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md))**: no external tool re-vendors itself onto abcd's state, so there is no overlay to re-apply

**Phased-in additions captured post-brief (2026-05-07):** itd-27 (`/abcd:intent grill` sub-verb + glossary), itd-28 (spec-tied reviews in the native spec review store), and itd-34 (three intent kinds with three lifecycle paths) were captured after this brief was written and are scoped into the planned phases. They are listed in `intents/README.md` and the relevant phase docs; this section is canonical for **later-phase** items only and does not enumerate phased-in intents.

**Later-phase additions captured post-brief (2026-05-07):** itd-30, itd-31, itd-32, and itd-33 were captured in the same audit pass. itd-30 and itd-33 remain in the later-phase list above; itd-31 and itd-32 have since been superseded (itd-31 absorbed by itd-48; itd-32 superseded by itd-31) and moved to `intents/superseded/`, so they are no longer in the canonical later-phase set above — this note records their capture timing and supersession for the brief's history. (See `superseded/itd-31-cross-document-fidelity-reviewer.md` and `superseded/itd-32-audit-role-taxonomy.md`.)

**Superseded addition (2026-08-28):** itd-145 (the acknowledgement convention arming itself, captured 2026-08-22) has been superseded by itd-163 in the list above, which delivers its mechanically checkable core, and moved to `intents/superseded/`. (See `superseded/itd-145-an-adopted-idea-cannot-ship-uncredited-abcd-enforces-its-own.md`.)

Each intent captures the press-release-shaped scope and acceptance criteria. A later-phase intent enters work by being scoped into a phase, then promoted to `planned/` via `/abcd:intent plan <itd-N>`. It reaches `shipped/` one way only: closing the last of the specs that realise it, which moves the intent as its close-hook — an intent owns one or more specs, so a spec that delivered only part of it closes on its own terms while the intent stays planned. There is no `intent ship` verb, so nothing promotes a record on its own — the close is a manual step run in the change that lands the work.

The brief does not get re-versioned. What has shipped is defined by which phases are complete and which intents are in `shipped/`; this brief stays the canonical current-state design record.
