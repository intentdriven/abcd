# Configuration Model

Almost nothing here is a decision anyone has to make. Installing abcd asks four
questions, records the answers, and gets on with it; two further keys can be
hand-set and are read but never written. That is the whole of the configuration
surface the shipped binary consults. Everything else on this page is either a
store abcd lays out for itself, a policy that follows from one of those four
answers, or an axis the design commits to and has not built.

**Read the schema as two lists.** The keys the binary reads come first and are
shipped behaviour. Everything after them is **staged**, per the truth rule in
[`../00-meta.md`](../00-meta.md#the-truth-rule).

## `.abcd/config.json`

### The keys the binary reads

Seven keys plus a `meta` block. Install asks about visibility, the docs target,
the oracle backend, and the deep scan (the last only when visibility is private
and the deep scanner is on `PATH`), and writes exactly those four values back.
`attribution.hook` is written by `ahoy install --attribution`. The remaining two —
whether abcd should enable the forge's own secret scanning, and the rules
loader's refresh backstop — are hand-set: the binary reads them and never writes
them.

```json
{
  "meta": {                             // setup block — stamped and read by ahoy
    "schema_version": 1,
    "setup_version": "0.1.0",
    "setup_date": "2026-05-04",
    "project_name": "abcd-cli"
  },
  "repo": {
    "visibility": "private"             // "private" | "public" — set by ahoy each run, no silent default
  },
  "docs": {
    "target": "both"                    // "claude_md" | "agents_md" | "both" | "skip" — which conventions
                                        //   router carries the marker block
  },
  "oracle": {
    "backend": "host-delegated"         // "host-delegated" (default: abcd emits a prompt, the host runs it —
                                        //   no API keys, no model config) | "native" | "cli" | "api" | "mcp"
  },
  "scan": {
    "deep": false,                      // the native secret/PII scan is the default; deep adds an opt-in
                                        //   external backend, asked only when it is available
    "native_secret_scanning": true      // an explicit false opts the repo out of abcd enabling the forge's
                                        //   own secret scanning through `ahoy remote apply`
  },
  "attribution": {
    "hook": true                        // the committed prepare-commit-msg prompt asking each commit to
                                        //   declare whether a tool assisted it
  },
  "rules": {
    "force_refresh_every_n": 15         // prompt-router refresh backstop (itd-3). The primary refresh is
                                        //   event-driven, on SessionStart and PreCompact, so this large
                                        //   counter only re-injects always-relevant domains
  }
}
```

There is no separate `.abcd/meta.json` at repo scope: setup metadata is the `meta`
block. This repository's own config carries four of these blocks — `docs`, `meta`,
`oracle` and `repo` — which is what an unremarkable managed repo looks like.

### Staged config keys

No shipped code reads any of the keys below. None appears in any repository's
config today, and writing one has no effect. They are the axes the design commits
to as the seams and the sync namespace land:

- **`ai_transparency.level`**: a second axis, independent of visibility, deciding
  what is captured at all rather than what is committed. Described below.
- **`spec.backend`, `run.backend`, `history.backend`**: the selectors for three
  seams. The native spec store and the native history store both ship; what does
  not ship is anything that would choose between backends.
- **`disembark.maxAgentTokens`**: a per-agent context budget, over which an input
  would stream and summarise.
- **`memory.harvest`**: vendor memory harvest as an opt-in read-only source.
- **`reviews.capture`** and the **`dev_sync.*`** per-source enable flags: the
  sync machinery of § 2, staged in full.
- **`intent.auto_link` / `intent.auto_ship`**: staged as keys only. The behaviour
  ships unconditionally: `abcd intent plan` mints the spec and writes both sides of
  the link, and the spec-close hook moves the intent. Neither is switchable, so
  neither key is read.
- **`capture.default_severity`**: today the severity rides on the invocation.

Two axes are absent from both lists on purpose. There is no `embark.scan` key,
because there is no `embark scan` sub-verb for it to switch. There is no
`adapters` registry block, because the wired-adapter registry it indexed has no
home: no registry package exists and no code refers to one.

**Owed-review draining (staged)** is receipt gating in the `run` seam
([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)). No `run`
seam ships, so nothing drains today. When it does, owed fidelity reviews drain at
the seam's iteration boundary: each iteration gates on a receipt and applies the
safety guard, report-not-block, whichever adapter provides the loop. There is
deliberately no autodrain config knob — receipt gating is part of the seam
contract, inherited by every adapter loop rather than re-implemented per loop.

**Audit-loop mode and budget (staged: itd-50)** are elected **per intent**, in the
intent's own frontmatter rather than in `config.json`, so the choice is portable
with the intent and one intent can loop while another stays record-only. No intent
carries these keys today and no validator reads them. The shape:
`audit_mode` is `record-only` by default, meaning a failed criterion is recorded
and nothing re-opens; `loop-to-acceptance` re-opens the linked work and iterates
until the criterion is met or `audit_budget` (default 3) bounds it. Budget
validation is **fail-closed**: a malformed, zero, or negative budget on a looping
intent never starts the loop and never silently coerces to the default. The
review-queue entry snapshots the effective mode and budget at enqueue time, so a
mid-loop edit cannot change an in-flight loop. The loop terminal and the gate
belong in a drainer layer that does not exist.

Schema versioning and cross-version migration come in a later phase: a
configuration record carries `schema_version: 1`, and migrators are added if a
later phase changes the shape (itd-9). The stamp is a convention rather than a
held rule, and the tree is not uniform — eight of the thirteen committed records
carry it, and five do not, the record-lint and docs-lint configuration among
them. Nothing refuses an unstamped record, so a migration that arrives before the
stamps do has no version to read on those five.

## The history store

The history store is a **user-scope** artefact, shared across every abcd-managed
repo on the machine, living at `~/.abcd/history/`. ahoy bootstraps it once, on the
first install on a fresh machine. It holds a registry keyed on each repo's
root-commit SHA, and one identity-and-lineage file per repo under that key.

The root commit is the only immutable identity a repo has: it survives a rename, a
remote move, and a change of forge handle. Everything else the registry records —
the repo's name, its remote URL, its path on this machine — is a mutable label
ahoy refreshes on every run rather than treating as write-once. A repo entry also
carries its lifecycle status and, where a repo was re-founded, the cross-references
to and from the entry it supersedes.

Re-founding a repository (a clean-history rebuild) produces a *new* root SHA. ahoy
registers the new entry, marks the old one superseded, and leaves the old corpus
in place under its own key for lifeboat review. A per-repo entry can also carry
prior names, free-text provenance about why the repo was re-founded, and a pointer
to where this repo's captured evidence lives.

This registry is the **sole user-scope registry**. abcd manages one repository per
tree, so there is no workspace-to-repo grouping to register and no second registry
file.

### The transcript corpus

The transcript corpus is a **sibling** user-scope store rather than a sub-tree of
the registry, at `~/.abcd/transcripts/<root-sha>/`, holding redacted records and a
staging area for raw transcripts awaiting redaction
([adr-2609091248201071](../../decisions/adrs/2609091248201071-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)).
One package owns its layout: `internal/core/history` declares both the user-scope
default and the opt-in per-repo location, and every resolver goes through it. The
rule is a convention with nothing behind it, and it already has one exception —
the cold-reading assembler repeats the per-repo path as a literal in its
exclusion rows rather than reading the constant, so moving that store would leave
the exclusion pointing at the old place.

It is **self-creating**: the store bootstraps on first use, so no install step
stands between a wired hook and a stored transcript (iss-95). Every level is
created individually and re-verified as a real directory on every resolve, so the
store never creates or writes through a symlink.

A repo may **pull its transcripts in**, by declaring the checkout in the caller's
own home — one absolute path per line in `~/.abcd/local-transcript-roots`, the
same home-scoped, abcd-owned, line-oriented idiom as the other declarations, and
honoured only when the file is a regular file this uid owns that no one else can
write. A declared checkout keeps its store in the gitignored per-worktree local
tier. A corpus left at an earlier path is moved into the store on first resolve,
reported, and tombstoned where it was.

## The voyage store

`voyage/` is **operations** — the verb, what we did — as distinct from the
lifeboat, which is the **artefact**, the noun that gets carried. It is user-scope
and keyed on the source repo's root-commit SHA exactly as the history store is, and
it is **never committed**
([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md),
superseding adr-4's in-tree home). What ships is an append-only manifest log of
every disembark run; nothing else is written anywhere under the store, and an
embark side recording what a write did is **staged**.

Two properties follow from keeping it out of the tree, and both are load-bearing:

- **`disembark` never writes to the source repository.** It reads the source and
  writes the lifeboat to an operator-chosen destination, and the operations log
  lands at the operator level. Mining a dead or archived project must not require
  installing abcd into a repo we only want to read.
- **Voyage records absolute source paths, so it must not be committed anywhere.**
  abcd's own privacy rule flags a home path in a committed file, so an in-tree
  voyage namespace would have made abcd fail its own audit. Keeping it at the
  operator level dissolves the collision rather than exempting it, which is why no
  gitignore rule for voyage appears in § 1: there is nothing in-tree to ignore.

Because the lifeboat lands out of tree, `embark` reads it from wherever disembark
wrote it. The destination is protected by a **safety gate** rather than an
overwrite-with-backup model: abcd refuses unless the destination is absent, an
empty directory, or one carrying a provenance file abcd itself wrote. See
[`../04-surfaces/03-embark.md`](../04-surfaces/03-embark.md).

## The worktree store

**Design target (itd-2609091014076309, `intents/drafts/`; unbuilt).** No
`worktree` verb exists in the shipped binary, and nothing in it creates or reads
this store. What follows is the layout the intent commits to, on the rule
[adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md)
records: a tool never creates directories in user-owned project space, so a
session's or an agent's worktree is machine-scoped rather than a sibling of the
checkout. It would be user-scope and keyed on the repository's root-commit SHA,
exactly as the history, transcript and voyage stores are, and never committed:

```
~/.abcd/worktrees/
  <root-sha>/
    <name>/                   one git worktree of that repository, on its own branch
```

Three properties are load-bearing, and each is the intent's to deliver:

- **Git is the registry.** `git worktree list --porcelain` already names every
  worktree wherever it sits, so the store keeps no index of its own. The verb reads
  git's answer and says which entries are in the lane, which are outside it, and
  whether each is clean and merged. The worktree's own `.git` file, not a mutable
  path label, says which checkout a lane belongs to.
- **It creates itself through one seam, never through a symlink**, on the
  transcript store's discipline: each level made individually and re-verified as a
  real directory.
- **Reclaim is explicit and provable.** `prune` removes a worktree only when its
  branch is merged into the default branch and its tree is clean, names everything
  it declined and why, and never deletes a directory git does not recognise as a
  worktree of the repository.

Worktrees already sitting beside a checkout are outside the store by definition:
listed as such, never moved, and retired by the user's own `git worktree remove`.

## The two `.abcd/` scopes

`.abcd/` is **one namespace pattern instantiated at two scopes**. abcd lives in
one repository ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)):
its design record is repo-scoped and in-tree, and the user scope holds only state
that is genuinely machine-wide. `/abcd:ahoy` classifies the folder it runs in and
acts on the scope that applies.

**User scope, `~/.abcd/`** — one per machine, machine-local shared state only: the
history registry, the transcript corpus, the voyage operations namespace, the
staged worktree store, the run state an autonomous run's sessions share
([`../04-surfaces/27-implement.md`](../04-surfaces/27-implement.md)), the inbox of reports managed repositories file back to abcd
([`../04-surfaces/28-report.md`](../04-surfaces/28-report.md)), machine config defaults (a later phase: every config read
in the binary resolves the repo-scope `.abcd/config.json`, and no home-scope one
is read at all), user-scope memory for personal cross-project knowledge (a later
phase too: the shipped memory store is repo-scope), and the `sources/` corpus `/abcd:ingest` and
`/abcd:consult` read (abcd never creates that one, and both verbs say so and stop
when it is absent). It also holds the caller-controlled declarations: the owned
PATH entry, the trusted configuration roots, and the checkouts whose transcripts
are pulled in. **Never the design record.** The same inventory is drawn as a tree
in [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md#what-abcd-manages--repos-and-abcd);
the two are one list and must agree.

**Repo scope, in-tree `.abcd/`** — this repository's record and working files: the
three-tier layout below, the config file with its `meta` block, the rules
overrides, the per-surface machine records under `config/`, the lint and site
configuration records, the identity and positioning registry, the citation
baseline, and the native spec store. A `memory/` namespace is written
where memory is curated. **The home for project work.** There is no in-tree
lifeboat directory: the lifeboat is out-of-tree output at an operator-chosen
destination. Two namespaces are not part of it either: `logbook/`, a retired name,
and `rp/`, staged with its adapter.

**The repo-scope three-tier working layout** (matching
[`../02-constraints/01-platform.md`](../02-constraints/01-platform.md)):

| Tier | Path | Committed? | Holds |
|---|---|---|---|
| **record** | `.abcd/development/` | committed — in every repository checkout, never in the released binaries | the durable design record: brief, roadmap, intents, ADRs, research |
| **shared work** | `.abcd/work/` | committed | current orientation, the append-only decision log, the external-contribution runbook, the issue ledger, the reviews charter, and the branch-ruleset mirror |
| **local ephemeral** | `.abcd/.work.local/` | gitignored | machine-local scratch: handover notes, scratch, logs, intent-audit receipts, the per-machine banlist layer. Per-worktree, so it never merge-conflicts |

**The record is repo-scoped and in-tree.** No workspace layer holds it and there is
no workspace registry: abcd is one repository, so the record lives in that
repository's tree, and there is no dev-to-public mirror. The user scope survives
only for state that cannot live in any one repo's tree. The repo keeps its config
and rules files in-tree too, so the hooks and the marker-block installer read them
from the repo directory deterministically.

**The vendor harness directory is not abcd's.** abcd writes nothing into it beyond
the plugin install itself; everything else routes to the scope-appropriate
`.abcd/`. The one interaction the design gives abcd with it is read-only: the
staged memory harvest of § 2 would read from it as a source.

## The rules root — which `.abcd/` governs a session

The rules, the hazard registry and the per-repo config are read from ONE resolved
root, so a session's injected rules and its hazard registry can never come from two
different directories. The resolution is bounded at the git working tree
(GHSA-vvqc-3mv2-5p49): the toplevel git reports for the working directory is
resolved first, the walk runs upward from the working directory and stops at that
toplevel inclusive, so the nearest `.abcd/` inside the tree wins and one above the
tree is never reached.

**When git will not answer.** abcd runs every git command under an isolated
environment so an inherited `GIT_DIR` or an injected config cannot redirect it.
That isolation also discards the operator's safe-directory exceptions, so the
toplevel query fails inside a checkout owned by another uid, and it fails outright
when the host launches a hook with git off its `PATH`. "Not a repository" and "a
repository git will not answer for" are different outcomes: only the first resolves
to the working directory with no walk, because collapsing the second onto it
dropped the repository's own configuration layer with no error anywhere — the
guard's hazards became allows, and the rules kill switch stopped applying. The
second falls back to the `.git` marker, under two bounds:

| Bound | What it refuses | What it costs a real checkout |
|---|---|---|
| **shape** | a `.git`-named entry that is not a repository: an empty file, a HEAD-less directory, a dangling symlink. The marker walk is a name check that runs to the filesystem root, so without this an unprivileged empty `.git` beside a planted `.abcd` in a world-writable directory governs every session under it | nothing: the two shapes git itself reads both pass |
| **ownership** | a marker root whose owner is not the caller. Shape alone is not a trust boundary: `git init` in a shared world-writable directory produces a genuine repository, and git's refusal on ownership is the same signal in that attack as in the legitimate foreign-uid case (iss-2609020259564193) | a declaration, once, per foreign-uid checkout |

A refused root is refused **loudly and fail-closed**: the session resolves to its
own working directory with no walk, the bundled rule defaults and bundled hazard
registry stand in for the repository's, and every front door prints one line naming
the refused directory, the two uids, and the exact command that re-admits it
([`../../principles/loud-staging.md`](../../principles/loud-staging.md)). The
ownership bound applies only to the git-refused fallback: where git answers, the
toplevel it named stands whoever owns it, because that is a repository git itself
vouched for.

**The opt-in.** A foreign-uid checkout, a container bind mount and a shared CI
checkout are all legitimate, so the refusal has a supported route back: one
absolute path per line in `~/.abcd/trusted-roots`, matched exactly both as written
and symlink-resolved, with `#` starting a comment. The file is read through the
guarded bounded read, and only when it is a regular file this uid owns that no one
else can write — a declaration anyone could have written is not the caller's word.

**Why the home scope, and not an environment variable.** The bar the opt-in must
clear is that the untrusted tree cannot assert it: a file inside the foreign root
declaring itself trusted would be circular. A home-scoped file clears that
structurally, because writing it needs write access to the caller's own home, the
same authority the caller already holds over everything abcd trusts. An
environment variable clears the letter of the bar and not its spirit: a repository
can ship the shell or task-runner configuration that sets it, and an operator whose
shell auto-loads that has the tree declaring itself trusted one indirection out.
abcd has ruled on this shape once already, in refusing an env-supplied data
directory, and
[adr-46](../../decisions/adrs/0046-persistence-never-weakens-the-verification-posture.md)
treats home write as the ownership root.

One residual stays open and recorded rather than assumed shut:
**iss-2609020219198779**, the user scope when the home directory is itself a git
working tree. The toplevel for a session in a non-repo directory beneath such a
home is the home, so the user-scope `.abcd` governs it; closing it needs a decision
on whether a home-directory toplevel is a legitimate configuration scope.

## 1. Visibility-driven gitignore policy

One question at install decides what a repo commits. Set by ahoy:

| Directory | Public default | Private default |
|---|---|---|
| `.abcd/` | gitignored² | **committed** (the entire namespace: the record, the working tier, the machine records, and `memory/` where it exists — visibility is the single switch, no per-subdirectory exceptions) |
| `memory/` (legacy snapshot) | gitignored | **committed** if present¹ |
| `.abcd/.work.local/` | gitignored | gitignored (local-only scratch) |

Two artefacts are absent from this table by construction rather than by exception:

- The native transcript store is **always** gitignored. By default it is
  user-scope, so it is not a repo directory at all; where a checkout has pulled it
  in, it sits under the local tier, which the row above already gitignores.
- **The voyage store and the lifeboat are not repo directories either.** Voyage is
  user-scope and the lifeboat is out-of-tree output, so no gitignore rule applies
  to either under any visibility.

¹ New projects use `.abcd/memory/`, the substrate `abcd memory` curates. The
root-level `memory/` is the legacy snapshot pattern some existing projects
maintain by hand; abcd respects it if present and never writes to it.

² On a repo whose `.abcd/` already holds **tracked** files, install narrows the
`.abcd/` entry to the local tier and keeps every other entry (iss-255). An ignore
rule cannot untrack committed records: the wholesale fence on such a repo only
hides every new record file from `git status` and makes `git add` refuse them,
while the committed tiers stay published regardless. Narrowing needs positive
evidence of tracked files, so a repo whose git cannot be asked keeps the declared
fence, and the install receipt states the narrowing out loud.

**No exceptions to the visibility rule.** Locked decision: visibility is one
switch, with no always-gitignored carve-out for any namespace on sensitivity
grounds. If sensitivity is a concern, set visibility to public, which gitignores
all of `.abcd/`. Per-subdirectory exceptions create maintenance burden and
contradict the transparent-prompts principle. The tracked-tier narrowing above is
not a carve-out of that decision: it is the mechanical boundary of what an ignore
rule can do at all.

**The launch payload is unconditional regardless.** Whatever the visibility, the
release artefact excludes `.abcd/` entirely, so a private repo that commits its
whole record locally still leaks none of it on launch. In a private repo the whole
namespace is reproducible from a fresh clone; the lifeboat is not part of what a
clone carries, because it is out-of-tree output.

**Memory locations to keep straight.** Curated memory exists at both scopes, and
there is one non-abcd location alongside them: the repo-scope `.abcd/memory/` is
the **primary** store, holding the curated summaries `abcd memory ingest` writes
and the canonical input for principle distillation; the user-scope `~/.abcd/memory/` is a
**later phase**, which will hold personal preferences and cross-project
principles with no single repo home (nothing in the binary resolves it today);
and a root-level `memory/` is the legacy snapshot abcd respects and never writes
to. Which scope a curated page lands in is a routing decision — see
[`07-memory.md`](07-memory.md). Retrieval across the two is not a flat union, which
would overflow context, but keyword recall with a budget bracket (itd-39). When
the brief says "memory" unqualified, it means the repo-scope store.

**AI transparency level (staged).** A second axis, independent of visibility. It
has no backing intent or ADR, no shipped code reads it, and ahoy neither prompts
for it nor writes it. The shape the design commits to is three levels: `full`
captures conversations, plans, tasks and metadata; `metadata` drops the
conversation transcripts; `none` captures nothing. Visibility decides what is
*committed*; transparency would decide what is *captured at all*, which is why the
two are separate — a private repo might want metadata only to keep storage tight,
and a public project might want full capture for credibility. Any combination is
valid. Launch's pre-flight scrub strips absolute paths and PII whatever the level:
the level would determine what exists, the pre-flight decides what ships.

## 2. `.abcd/work/` namespace and `dev-sync`

**This whole section is staged.** No `dev-sync` verb is registered on the binary,
no Go source names one, and none of the adapters below is built. `.abcd/work/`
itself is real and committed; what is staged is the machinery that would populate
it by sweeping volatile sources.

The one thing that already works this way got there without a sync: `abcd capture`
writes the issue ledger directly, so the migration this design was drawn around has
nothing left to migrate.

The idea is a curated-from-volatile-sources namespace. Noisy inputs stay
gitignored or external; the lessons drawn from them get tracked; and abcd does not
have to re-read volatile sources every time. Four source-to-target pairs are
drawn: an opt-in agent-memory harvest into `.abcd/memory/`; ad-hoc oracle reviews
not tied to a spec into `.abcd/work/reviews/`, where the reviews charter already
governs the shape any sweep would have to meet; the local tier's notes into a
curated notes target; and an external tool's workspace state into its own
namespace (itd-7). Only the memory package exists today, and its shipped front
door takes a URL rather than a harvested directory.

Two triggers are drawn: an implicit one at disembark's first phase, and a manual
CLI refresh. Per-source enable flags would default to on for a private repo and off
for a public one. Scheduled sync comes in a later phase (itd-13).

> **Open question (adr-35):** the implicit trigger is stated against the old model,
> in which disembark packed *this* repo. adr-35 makes disembark read-only against
> the source, while a sweep **writes** into the source repo. The two cannot both
> hold. What must be decided: whether the sweep is dropped from disembark entirely
> and becomes a separate operator-run step, with disembark reading whatever curated
> artefacts happen to be present and none at all in a repo abcd never touched — the
> primary case — or whether the implicit trigger survives only where the source repo
> is abcd-managed and the operator opted in. adr-35 does not settle it. The question
> is not urgent while neither half ships, and it is the first thing to settle when
> either does.

Two curation rules are worth stating now, because they bound what a sweep may do.
Output is **not verbatim**: raw memories grow unbounded and carry personal
phrasing, so what lands is distilled and grouped by domain. And files in the local
tier are never moved or deleted: the sweep reads and curates, and the source stays
put.

**Reviews are a first-class pitfall source.** Plan, implementation and completion
reviews are exceptionally good at spotting issues, and the lesson survives even
where the original issue was fixed. A collator would extract every such finding as
a candidate pitfall for the distiller to dedupe against its other sources.

**Distinct from the lifeboat's own synthesis:** the curated tiers are persistent
rolling artefacts in the source repo, used as ongoing input to future agents. The
lifeboat's principles file is per-disembark synthesis written out of tree at the
operator-chosen destination, never back into the source. The lifeboat consumes
`.abcd/work/`; `.abcd/work/` is not the lifeboat.

## 3. Plugin shape — directory layout

A Go binary plus the markdown plugin surface that shells to it:

```
abcd/
├── .claude-plugin/                     # plugin.json + marketplace.json
├── cmd/                                # the shipped entrypoint plus four build-time binaries
│   ├── abcd/main.go                    #   entrypoint — wires the CLI front door to the core
│   ├── record-lint/                    #   the record gate `make preflight` runs (06-lint.md)
│   ├── scaffold-sync/                  #   keeps the scaffolded release workflows in step
│   ├── abcd-gen-surface/               #   writes the committed command-surface snapshot
│   └── abcd-gen-cli-ref/               #   writes the generated CLI reference page
│                                       #   The four are developer tooling, not user surface: they run
│                                       #   from the Makefile or `go generate`, and ship in no release
├── internal/
│   ├── core/                           # transport-agnostic core, one package per capability
│   │                                   #   (adr-23); each returns structured results
│   ├── adapter/                        # the seam packages (adr-22): interface + native default
│   │   ├── scanner/                    #   + optional plug-in. The scanner is the one seam with a
│   │   └── gitleaks/                   #   package today; 02-adapters.md is the catalogue
│   └── surface/cli/                    # the only front door that ships (an mcp/ door is adr-23's third)
├── commands/<verb>.md                  # markdown command surfaces, flat; the gated list lives in
│                                       #   ../04-surfaces/README.md (abcd.md is the bare /abcd board)
├── agents/<name>.md                    # host-delegated agent prompts, plus per-agent fixtures/,
│                                       #   README.md and CHANGELOG.md — see 01-agents.md
└── hooks/                              # host event hooks; every event command runs through a
    ├── bootstrap.sh                    #   resolving shim. bootstrap.sh PROVISIONS the plugin-root
    └── hooks.json                      #   binary and never builds one
```

The seams that are planned rather than present are gated rather than trusted: the
`index_drift` record-lint rule holds every path in the planned-seams region of
[`internal/README.md`](../../../../internal/README.md) to being absent from the
tree, so a seam that ships cannot go on being described as planned.

**The hook wiring, and why it is shaped this way.** `bootstrap.sh` either copies a
re-verified artefact out of the persistent download cache, or resolves the latest
release tag and downloads the pinned asset, verified against that release's own
checksums; its build strings are printed instructions to the operator for the cases
it cannot provision. `SessionStart` runs one chained command rather than siblings,
because siblings would run in parallel and share one stdin.

`SessionStart`, `UserPromptSubmit`, `PreToolUse` and `PreCompact` are the events
that reach `bootstrap.sh` at all. The last three self-provision only when the
plugin-root binary is missing, throttled by a `.bootstrap.attempt` marker within a
ten-minute window, and then fall back to a PATH-resolved abcd that must be
absolute, outside the working directory, not world-writable, and recorded as this
machine's own, before failing loudly. `SessionEnd` and `SubagentStop` are the two
exceptions and download nothing at all: each fires where the harness cancels a slow
hook rather than wait — one as the session exits, the other inside a live session as
a sub-agent finishes — so a fetch there races that cancellation and loses the
transcript the hook exists to capture (iss-2608210934566223). Both resolve the
plugin root, then PATH, then say the transcript was not captured. `SubagentStop`
additionally never exits non-zero beyond that refusal, because exit 2 is the host's
BLOCKING code on that event.

`SessionEnd` carries `hook session-end`, which stages the session's own transcript;
`SubagentStop` carries `hook subagent-stop`, which stages a finished sub-agent's
transcript beside it with the lineage the harness payload and its per-agent sidecar
supply. `SessionStart` drains both through the fail-closed redaction path.

**The plugin-internal development namespace** (committed in private repos,
gitignored in public) holds, at its root, the config file and the per-surface
machine records under `config/`, the rules overrides, the lint and site
configuration records, the identity and positioning registry the surfaces are
held to, and the citation baseline `docs cite` maintains. Under `development/`
sit the durable record's families, flat by artefact type (adr-30): the chaptered
brief with its glossary, the intents
store with directory-as-status, the principles, the decisions, the roadmap with its
phases and RFCs, dated plans, the native spec store, the cold-reading ledger, the
release surface declaration, the release gate's manifest, research, and the
persona roster a press-release quote must attribute to. Under
`work/` sit the shared working files. The local tier is the third and is
gitignored under every visibility.

Three namespaces are deliberately absent from every tree: `voyage/`, which is
user-scope and never committed; `logbook/`, retired in favour of the local tier's
logs; and `lifeboat/`, because the lifeboat is out-of-tree output at an
operator-chosen destination.

**User-facing docs** live under `docs/` and are packaged into the release
artefact, split by Diátaxis type with a shared assets directory and the pinned
site-build toolchain. The native spec store, memory and config live under
`.abcd/`; plugin-internal design docs under `.abcd/development/`. The split is a
shape rather than any one tool's location convention.
