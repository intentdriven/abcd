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

The list is generated, not hand-edited: `go generate ./internal/surface/cli`
writes the marked region from [`drafts/`](../../intents/drafts/), one line per
record with its id and its own title, in id order. The `index_drift` record-lint
rule fails while the region differs from what `drafts/` derives, so a capture, a
promotion or a supersession is followed by rerunning the generator, and two
branches that each conflict here are reconciled the same way, by rerunning it on
the merged tree. Each line's text is the record's own H1, so a wrong or stale
line is fixed in the record under `drafts/` and regenerated, never edited here.
That is what keeps "not hand-counted" true after the day it was written.

<!-- index: later-phase-intents -->
- `itd-8` — Lifeboats Carry Source Code, Not Just Lessons
- `itd-9` — Old Lifeboats Still Unpack
- `itd-10` — Full Removal When You Mean It
- `itd-11` — Disembark Survives Noisy Transcripts
- `itd-12` — Working Notes Don't Outrank Settled Decisions
- `itd-13` — Reviews and Memory Stay Fresh, Always
- `itd-14` — Prompts Are Versioned Like Code
- `itd-15` — abcd Audits Its Own Prompts on Every Self-Disembark
- `itd-16` — `/abcd:audit` Is the Umbrella; Conversation/Edit Chain Is Its Default Application
- `itd-18` — Sensible Permissions From Day One
- `itd-19` — Defaults That Match Where the Project Lives
- `itd-21` — Start a New Project With abcd Conventions
- `itd-22` — abcd Reaches Any Harness Through an Adaptor
- `itd-23` — Lifeboats Speak To External Tools
- `itd-25` — Patterns Surface Themselves
- `itd-26` — Raid the Open Ocean, Bring the Cargo Home with Papers
- `itd-30` — Design Fictions As An Alternative Capture Format For Intents
- `itd-35` — Lifeboats Carry Their Own Integrity Proof
- `itd-41` — abcd Tells You, Honestly, What a Phasing Choice Costs
- `itd-44` — A Capture-Time `decision` Verdict That Routes Infrastructure Choices Into The Existing ADR Store
- `itd-51` — abcd Knows What "Safe Enough to Adopt" Means Before a New Harness Arrives
- `itd-55` — abcd Can Tell Whether Its Own Reasoning Rests On Bedrock Or On An Unexamined Assumption
- `itd-57` — Every abcd Voyage Ships With A Manual-Hold Sentinel That Blocks A Spec From Autonomous Pickup Until A Human Lifts It
- `itd-59` — Every Autonomous Run Pass Leaves the Same Durable, Queryable Transcript an Interactive Session Does
- `itd-61` — A Human Edit To The Brief Cannot Commit Until Its Implied Intents And Principles Are Drawn Out And Reconciled
- `itd-62` — abcd Routes The Invisible Risks An Amateur Cannot See To A Real, Fail-Closed Gate That Wraps A Trusted Scanner
- `itd-64` — abcd Learns From Its Own Runs Which Configurations Work Best, And Tunes Its Configuration Layer Accordingly
- `itd-70` — Drawn-out intent
- `itd-75` — abcd Proves Your CLI Actually Runs — From Fixtures You Drop In
- `itd-77` — abcd's User-Level Home Lives Where You Want It
- `itd-78` — abcd Knows What to Build First — Even When It's Something Small
- `itd-83` — The Review Bar Fires By Itself, In Every Repo abcd Manages
- `itd-85` — A Read-Only Repo-Conformance Audit
- `itd-86` — abcd Can Read Its Own Design Documents As A Stranger Would
- `itd-87` — A Finding That Comes Back After You Closed It Tells You Something, And abcd Stops Throwing It Away
- `itd-90` — The Product Thinker Is Handed Exactly The Questions Only They Can Answer — And Nothing Else
- `itd-91` — A Project Declares Once How It Wants AI Acknowledged — And Every Commit, PR, And Record Follows
- `itd-92` — abcd tells every repo the truth about its fence
- `itd-97` — The Facilitator Is a Mode, Not a Person
- `itd-98` — Solo vs. Duo Is Measured, Not Debated
- `itd-99` — A Team of Product Thinkers Decides as One
- `itd-106` — abcd Sets Up the CI a Repo Requires, and Reports What It Did
- `itd-107` — Autonomous Routines Assemble From One Versioned Template
- `itd-108` — The plugin installs from the curated release artifact, not the repo, and every cut release reaches users automatically
- `itd-109` — Acceptance Criteria Verify Themselves, And The Manual Rest Renders For A Human
- `itd-113` — The MCP Front Door Opens — abcd's Core Verbs From Any MCP-Capable Harness
- `itd-115` — A Ready PR Merges Without Ever Wedging BEHIND
- `itd-116` — Validated GitHub Issues Become Ledger Entries Without Retyping
- `itd-118` — Merged work leaves no residue: abcd-managed repos delete a PR branch on merge, and abcd tidies the stale local branch, tracking ref, and worktree
- `itd-126` — A Team Shares One Bibliography Without Sharing Anyone's Corpus
- `itd-127` — A Paper Is Reconstructed From the Provenance Ledger
- `itd-128` — One canonical YAML scalar resolver: frontmatter exports the single null/bool/quote resolution helper and the capture, lint, lifeboat, and memory decoders all delegate to it, so a cross-package verdict split on any scalar axis becomes unrepresentable
- `itd-129` — Forge mirror as an opt-in adapter: the ledger mirrors out to forge issues one way, the forge id lives in the record schema, closures without import self-heal, and the whole surface is loudly absent without forge auth
- `itd-134` — Every abcd-managed CLI greets like abcd does: one command scaffolds a banner for a managed repo, rendered from that repo's canonical identity block — name, tagline, version — as a committed, language-neutral artefact (the generator emits the rendered banner, not code), so a CLI in any language opens with its own identity without hand-drawing anything. Split out of itd-112 at its 2026-08-21 grill: itd-112 keeps abcd's own banner and the colour ladder; this intent carries the managed-repo generator and the per-repo accent question.
- `itd-142` — The brief-creation interview: abcd elicits a repository's brief in a staged conversation — narrative first, then frontier rounds that ask only what the record cannot yet settle, options at conjectural questions (at most three candidate construals, a ceiling not a target, with a null answer always available), and a per-item confirmation — distilling committed products into the brief and a ledger while the transcript is discarded. Entry is one door for both modes: a brownfield repository arrives pre-populated from embark probe coverage, provenance-stamped and still requiring confirmation; a greenfield repository is the same interview run from the all-blank coverage state. Command-shaped, because it mutates state — abcd ships zero skills
- `itd-143` — The brief gains a framing chapter under 01-product/ — the macro-why home: one section holds the project's framing, the why behind the why, and receives the brief-creation interview's committed framing products; the brief-lifeboat mapping gains the matching row so a lifeboat carries framing like any other section
- `itd-144` — Every livery mark has a surface and earns it: the duckling, the lifeboat, and the two-by-two signal-flag icon are placed deliberately across abcd's surfaces instead of sitting as generated-but-unused assets. The lifeboat mark renders when disembark finishes packing a lifeboat and mirrors when embark unpacks one, so the direction of travel reads at a glance; the duckling appears as the mascot wherever the harness greets a human; the icon becomes the website and app icon. Settles the forge/web logo question itd-112 deferred as 'the existing logo stays for now'. Follows itd-133, which shipped the marks deliberately unwired, and consumes itd-112's colour ladder and emission discipline rather than minting its own.
- `itd-149` — abcd handles inbound security advisories and issues, and cuts the release, for every managed repo. The 2026-08-27 security-advisory pilot proved the loop by hand across two independent operators: advisory and issue intake, recall-dedup triage against the ledger, record-first fixes in isolated worktrees, one integration branch, an independent adversarial gate over the assembled diff, auto-merge armed only after APPROVE, and a derived release cut through the semantic-receipt gates. This intent automates that loop as abcd capability for any managed repo, with findings F-A through F-W of the pilot note as the acceptance-criteria source and F-U (re-run every required pass against the final content commit; never transfer a verdict) and F-Q (the two release gates are human-by-design) load-bearing.
- `itd-159` — the managed .gitignore visibility table has no mode for a public repo that commits its record: on abcd-cli (public, single-repo-curated-release, .abcd/** deliberately in-tree) ahoy install applied the public policy and fenced /.abcd/ and /memory/, contradicting the repo's own boundary that the record is present in every checkout. The visibility table needs a committed-record declaration (config or marker) that suppresses the fence
- `itd-163` — A cited source always resolves and an adopted inspiration is always acknowledged: the reference-closure and acknowledgements-mirror gate
- `itd-164` — A source is licence-vetted before the record may lean on it: cite refresh records the verdict, the gate reads the committed baseline
- `itd-165` — A failed fidelity verdict becomes work somebody can see
- `itd-166` — A run records what it was actually run with
- `itd-167` — The product thinker answers a stop in a medium they already use, and answers it whenever they get to it
- `itd-168` — The product thinker sets how the system talks to them, and the defaults are plain language and options to choose from
- `itd-169` — An agent loop that stops says who it is asking and what it needs, and never simply halts
- `itd-170` — What the product thinker reports after using the product finds its way back to the promise that predicted it
- `itd-171` — A decision points back to the conversation where it was reached
- `itd-172` — Every record has a short title and a one-line summary a non-engineer can read
- `itd-173` — Verification escalates from the built-in check to an outside audit, and the product thinker picks the rung
- `itd-174` — Each repository configures how far its facilitator is consulted and when it escalates
- `itd-175` — The product thinker writes down how this could be wrong, and what would show it
- `itd-176` — Whatever ships says how hard anyone looked at it
- `itd-2609090746410233` — A lifeboat packs from a worktree, a branch, or an abandoned feature test, experiment or implementation, not only from a whole repository
- `itd-2609090746414083` — A lifeboat packs from a lab session home, the throwaway experiment's intention, harvest and bundle, with the same coverage honesty as a repository
- `itd-2609091416304128` — capture resolve refuses a record the default branch has already closed, judged by the last fetch
- `itd-2609150819439571` — abcd has no supported operation for correcting a factual error inside a durable record, and no documented convention saying what to do instead. The record is deliberately not rewritten, which is right, but "not rewritten" and "wrong" are different states and only the first has a mechanism.
- `itd-2609150819440345` — Which session holds which worktree, branch or record is coordinated entirely by conversation, so every new session repeats a handshake that nothing records. A session joining work in progress has no way to ask what is already claimed: it messages the peers it can see, waits for replies, and rebuilds a picture that the sessions before it had already built and did not write down. One measured encounter cost four messages and about fifteen minutes before any work began, and the picture it produced is not durable, so the session after that pays again. The convention that a diff you did not make is a peer's work depends on knowing who the peers are and what they hold, which is precisely the thing no artefact carries. The repository already records this gap for the narrow case of detecting a peer session before mutating git state; the wider case is claim rather than presence, and the two want the same substrate. Whatever holds it should be as cheap to write as it is to read, because a coordination record nobody updates is worse than the chat it replaced.
- `itd-2609151138388536` — Every Decision Is Its Own Record, and the Log Stops Conflicting
- `itd-2609151516525843` — A Repository Can Ban A Name On Its First Commit, And A Machine Can Ban One Everywhere
- `itd-2609151658486398` — A release cut publishes the security advisories its fixes close: when the release workflow cuts a version, every draft advisory whose fixing record shipped in that cut is published with the fixed version stamped and its opt-in credits, and an advisory whose record was closed as won't-fix is closed as not a vulnerability, so no fixed advisory sits unpublished after its release and no hand step decides which ones publish
- `itd-2609151838312703` — Sessions on one machine or one local network leave each other messages in a shared mailbox abcd owns: a session under another account or on another machine writes a signed record into a shared directory, the recipient's prompt hook injects it and acknowledges it, and every message stays a readable record; no daemon, no dependency, nothing leaves the local network
- `itd-2609151838327688` — An opt-in adapter to a local message broker brings push delivery and cross-machine reach to the session mailbox: with the adapter enabled a session is woken the moment a message arrives, machines on the local network exchange messages without a shared mount, and every delivered message is still mirrored to the mailbox as a record; without it the mailbox works as before
- `itd-2609180517121254` — Every payload a host hands back from a delegated step names the model that produced it and the number of agents that ran, and the ingesting verb refuses one that does not
- `itd-2609231507251267` — Release pages on the project website
- `itd-2609291924469783` — Every lint warning rule shares one baseline file that only shrinks, so a new warning fails the change that introduces it
- `itd-2609292106557115` — A fresh machine installs abcd through one small starter that hands over to a trusted abcd binary
- `itd-2609292107351737` — A development session's token cost is measured, and set against the size class its spec predicted
- `itd-2609292108089653` — Ingesting and consulting sources are abcd verbs, and no corpus entry is a keyword stub
- `itd-2609292108373494` — The grill hands the interview to an installed interviewing skill, and runs its own when there is none
- `itd-2609292109005937` — An opt-in local model checks every prompt for secrets and personal data before it leaves the machine
- `itd-2609292109214516` — Rule injection is a seam: the native loader stays the default, and an opt-in CARL back end can take it over
- `itd-2609292109475690` — Every verb family has a behavioural scenario that drives the built binary end to end
- `itd-2609301020001595` — A pinned-action bump syncs its scaffold template and lands re-authored
- `itd-2609301918174237` — A remote provider takes a file-reading agent only as a fixed, blanked, capped bundle
- `itd-2610021503208208` — Personal git hooks keep running in abcd-managed repositories
- `itd-2610031259176838` — Outside contributors reserve work through a draft pull request that abcd honours
- `itd-2610031325050110` — abcd notices a harness update downloaded but not yet running
- `itd-2610031651058674` — abcd's check reports README parts that have gone stale, and writes nothing
- `itd-2610040740108331` — The dashboard on the home network, without Tailscale
- `itd-2610040740122709` — Acting from the dashboard: rewriting the brief and approving intents
- `itd-2610040740135705` — The dashboard for more people: the facilitator's view and the team's, configurable
- `itd-2610040754440360` — The private dashboard for every abcd-managed project
- `itd-2610040754453237` — The public record site for every abcd-managed project
- `itd-2610040817105016` — Every brief section shows what realises it, and a brief change names what it touches
- `itd-2610040822131032` — A lab runs end to end from a question the product thinker asks
- `itd-2610050548126044` — abcd reminds a session to refresh its handover note only when that session changed the tree
- `itd-2610070544145422` — Someone on Windows installs abcd and it just works: the
- `itd-2610070549060530` — A word in a new idea marks it to revisit, and the board puts it first
- `itd-2610070550516046` — On the dashboard, a product thinker sees the brief and their ideas together, and puts the ideas in order
- `itd-2610071545369520` — A person confirms the assistant's verdict on each warned banned-name line instead of judging it alone
- `itd-2610071545380041` — Work about abcd itself goes to abcd as a report, never into a managed repository's own plans
- `itd-2610100627454580` — An agent's text comes back from abcd's rewriter written for its reader and within its limits
- `itd-2610100704270254` — An abcd run sets how many sub-agents it keeps alive by the models they run on
- `itd-2610110940216682` — Every change is checked in full once, on exactly what lands
- `itd-2610111655362931` — On the dashboard, the product thinker and facilitator review memory notes and promote the ones worth keeping
- `itd-2610111707085098` — Each session takes a persona name, shown in the status line and the session list
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

**Superseded addition (2026-08-28):** itd-145 (the acknowledgement convention arming itself, captured 2026-08-22) has been superseded by itd-163 in the list above, which delivers its mechanically checkable core, and moved to `intents/superseded/`. (See `superseded/itd-145-an-adopted-idea-cannot-ship-uncredited.md`.)

Each intent captures the press-release-shaped scope and acceptance criteria. A later-phase intent enters work by being scoped into a phase, then promoted to `planned/` via `/abcd:intent plan <itd-N>`. It reaches `shipped/` one way only: closing the last of the specs that realise it, which moves the intent as its close-hook — an intent owns one or more specs, so a spec that delivered only part of it closes on its own terms while the intent stays planned. There is no `intent ship` verb, so nothing promotes a record on its own — the close is a manual step run in the change that lands the work.

The brief does not get re-versioned. What has shipped is defined by which phases are complete and which intents are in `shipped/`; this brief stays the canonical current-state design record.
