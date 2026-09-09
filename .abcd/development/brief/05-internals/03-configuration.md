# Configuration Model

This file holds the configuration schema (`config.json`, including its `meta` setup block), the user-scope and repo-scope stores, the visibility-driven gitignore policy, and the plugin's directory layout.

**Read the config schema as two lists.** The keys the binary reads today are listed first and are shipped behaviour; everything after them is **staged** — a design target a future change builds toward, per the truth rule in [`../00-meta.md`](../00-meta.md#the-truth-rule). The `dev-sync` namespace of § 2, which pumps volatile sources into curated artefacts, is staged in full: no `dev-sync` verb is registered on the binary and no Go source names one.

## Setup metadata — `config.json["meta"]`

Setup metadata is a `meta` block inside `.abcd/config.json`; there is no separate `.abcd/meta.json` at repo scope (spc-16, predecessor store). ahoy stamps and reads it via `config.json["meta"]` (example values shown):

```json
{
  "meta": {
    "schema_version": 1,
    "setup_version": "0.1.0",
    "setup_date": "2026-05-04",
    "project_name": "abcd-cli"
  }
}
```

## `.abcd/config.json`

### The keys the binary reads

Seven keys plus the `meta` block are the whole of the config surface the shipped binary consults. ahoy asks four questions on install — visibility, docs target, oracle backend, and deep scan when visibility is private and TruffleHog is on `PATH` — and writes exactly those four values back. `attribution.hook` is written by `ahoy install --attribution`. `scan.native_secret_scanning` and `rules.force_refresh_every_n` are hand-set: the binary reads them and never writes them.

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
  "oracle": {                           // oracle seam (adr-25)
    "backend": "host-delegated"         // "host-delegated" | "native" | "cli" | "api" | "mcp"
                                        // host-delegated (default): abcd emits a prompt, the host's subagent
                                        //   dispatch runs it — no API keys, no model config
                                        // native | cli | api | mcp: opt-in oracle adapters, selected when an
                                        //   operator wants abcd to reach a model directly
  },
  "scan": {                             // scanner seam — internal/adapter/scanner
    "deep": false,                      // native secret/PII scan is the default; deep adds an opt-in TruffleHog
                                        //   backend, asked only when visibility=private and the tool is on PATH
    "native_secret_scanning": true      // an explicit false opts the repo out of abcd enabling GitHub's own
                                        //   secret scanning through `ahoy remote apply`
  },
  "attribution": {
    "hook": true                        // written by `ahoy install --attribution`: the committed
                                        //   prepare-commit-msg prompt asking each commit to declare assistance
  },
  "rules": {
    "force_refresh_every_n": 15         // prompt-router fixed-N refresh backstop (per itd-3); event-driven
                                        // reset on SessionStart/PreCompact is the primary refresh, so this
                                        // large counter only re-injects always-relevant domains. Default 15.
  }
}
```

The repository's own `.abcd/config.json` carries four of these blocks (`docs`, `meta`, `oracle`, `repo`) and nothing else, which is what an unremarkable managed repo looks like.

### Staged config keys

**Staged (no shipped code reads any of these).** They are the axes the design commits to as the seams and the sync namespace land. None appears in any repository's `config.json` today, and writing one has no effect:

```json
{
  "ai_transparency": {                  // staged — no backing record; no code reads it
    "level": "metadata"                 // "full" | "metadata" | "none" — separate axis from visibility
                                        // full: conversations + plans + tasks + metadata
                                        // metadata: plans + tasks + metadata (no conversation transcripts)
                                        // none: nothing
  },
  "spec":      { "backend": "native" }, // staged spec seam — "native" (directory-as-truth + dependency graph,
                                        //   adr-26) | "ccpm" (the companion harness over conventions, adr-24).
                                        //   The native store ships; the seam that would select between
                                        //   backends does not.
  "run":       { "backend": "native" }, // staged run seam — "native" (thin Go loop, adr-27) | "workflows" |
                                        //   "companion"; each iteration gates on a receipt
  "history":   { "backend": "native" }, // staged history seam — "native" (local redacted transcript store,
                                        //   root-SHA-keyed, adr-29) | "specstory". The native store ships;
                                        //   the backend selector does not.
  "disembark": { "maxAgentTokens": 100000 },  // staged: per-agent context budget; over -> stream + summarise
  "memory":    { "harvest": "native" }, // staged: "native" | "<custom-path>" — vendor memory harvest as an
                                        //   opt-in read-only source; see 04-universal-patterns.md § 7
  "reviews":   { "capture": "oracle" }, // staged: "oracle" (capture over the native review store) | "none"
  "dev_sync": {                         // staged per-source enable flags (§ 2)
    "reviews": { "enabled": true },     // oracle-adapter capture -> .abcd/work/reviews/
    "memory":  { "enabled": true },     // memory harvest -> .abcd/memory/
    "work":    { "enabled": true }      // .abcd/.work.local/ issues + notes -> .abcd/work/
  },
  "intent": {
    "auto_link": true,                  // staged as a KEY only: the behaviour ships unconditionally —
                                        //   `abcd intent plan` mints the spec and writes both sides of the
                                        //   link, and the spec-close hook reconciles planned/ -> shipped/.
    "auto_ship": true                   //   Neither is switchable, so neither key is read.
  },
  "capture": {
    "default_severity": "minor"         // staged: today `abcd capture` takes the severity on the invocation
  }
}
```

Two axes are absent from both lists on purpose. There is no `embark.scan` key, because there is no `embark scan` sub-verb for it to switch: `abcd embark` ships `from` and `probe`. There is no `adapters` registry block, because the wired-adapter registry it indexed has no home — `internal/registry` is not a package in this tree and no code refers to one.

**Owed-review draining (staged — spc-43 predecessor store, itd-53 in `planned/`) is receipt gating in the `run` seam (adr-27).** No `run` seam ships, so nothing drains today, and the shape below is what the seam commits to. Owed fidelity reviews drain at the `run` seam's iteration boundary: each iteration gates on a **receipt** and applies the safety guard, a report-not-block step whichever adapter provides the loop (native Go loop, Claude Workflows, the companion harness). There is no autodrain config knob and no post-iteration edge to hang it on — receipt gating is part of the seam contract, inherited by every adapter loop rather than re-implemented per loop. The full report-vs-block / cost-bound decision record is [`adr-27`](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md); the companion consistency gate's `RC*` codes are registered in [`06-lint.md § 1`](06-lint.md#1-lint-code-namespace).

**Audit-loop mode + budget — per-intent frontmatter (staged: itd-50 in `planned/`, spc-52 predecessor store).** No intent frontmatter carries these keys today, no validator reads them, and no review queue or drainer exists; what follows is the shape the intent commits to. The audit-loop policy is NOT configured in `config.json`; it is elected **per intent** in the intent's own frontmatter, so the choice is portable with the intent (it survives the lifeboat) and one intent can loop while another stays record-only:

```yaml
# intents/<dir>/itd-N-*.md frontmatter
audit_mode: loop-to-acceptance   # "record-only" (default) | "loop-to-acceptance"
audit_budget: 3                  # iteration ceiling for loop-to-acceptance (default 3)
```

- **`audit_mode`** — `record-only` is the default (and the value when the key is **absent**): a `NOT_MET` is recorded to `## Audit Notes`, no re-work. `loop-to-acceptance` makes a `NOT_MET` re-open the linked work and iterate until `MET` or the budget bounds it. Set at plan time.
- **`audit_budget`** — the **declared** iteration ceiling for `loop-to-acceptance`. **Default `3`** when the key is absent but the mode is `loop-to-acceptance` (the intent-grain equivalent of the implementation-grain `MAX_REVIEW_ITERATIONS`). One iteration = one full re-open + re-review cycle that returned `NOT_MET`. A `record-only` intent **ignores** budget entirely (not an error).
- **Budget validation is fail-closed.** A malformed, zero, or negative `audit_budget` on a `loop-to-acceptance` intent is a **policy error**: the loop never starts (recorded fail-closed, issue logged) — it never silently coerces to the default or loops forever.
- **Enqueue-time snapshot.** The review-queue entry **snapshots** the effective `audit_mode` + declared `audit_budget` (plus the `audit_iterations_used` counter and the terminal `audit_outcome`) at enqueue time — the drainer reads the snapshot, never live frontmatter, so a mid-loop edit cannot change an in-flight loop. These four queue-entry fields are additive; a legacy entry without them parses as `record-only`.
- **The loop terminal and the gate belong in the drainer/policy layer**, never in the pure on-close hook. No such layer exists: `internal/core/repolint` is `abcd lint`'s rule set (layout, docs, decisions, privacy, positioning, router) and holds no audit loop. The gated manual-verification **receipt schema** is `{intent_id, machine_rollup, state, justification?, recorded_by_role, ts}`, distinct from the machine verdict; its home is the local ephemeral tier (`.abcd/.work.local/logs/`), since `.abcd/logbook/` is retired (see [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)). For the full state-coverage table and the `UNACHIEVABLE` replan surface, see [`../04-surfaces/05-intent.md` § 7 Role 1 — the audit loop](../04-surfaces/05-intent.md).

Schema versioning + cross-version migration **comes in a later phase** (abcd stamps `schema_version: 1` everywhere; migrators added if/when a later phase changes the shape — see itd-9).

## The history store

The history store is a **user-scope** artefact, shared across every abcd-managed repo on the machine, living at `~/.abcd/history/`. ahoy bootstraps it once (first `install` on a fresh machine creates it transparently — see [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)). Layout:

```
~/.abcd/history/
  index.json                  registry — root-SHA → repo entry
  <root-sha>/
    meta.json                 identity + lineage for one repo
```

### The transcript corpus

The transcript corpus is a **sibling** user-scope store, not a sub-tree of the registry, and `internal/core/history` is the only package that may lay out or judge its path:

```
~/.abcd/transcripts/
  <root-sha>/
    records/                  redacted transcript records (root-SHA-keyed; the
                              store is adr-29's, this location and the
                              self-creation are adr-2609090717039680's)
    staging/                  raw transcripts awaiting redaction (0o700, files 0o600)
```

It is **self-creating**: the store bootstraps on first use, so no install step stands between a wired hook and a stored transcript (iss-95). Every level is created individually and re-verified as a real directory on every resolve, so the store never creates or writes through a symlink.

A repo may **pull its transcripts in**, by declaring the checkout in the caller's own home — one absolute path per line in `~/.abcd/local-transcript-roots`, the `path-entry` / `trusted-roots` idiom (home-scoped, abcd-owned, line-oriented, honoured only when it is a regular file this uid owns that no one else can write). A declared checkout keeps its store at `<repo>/.abcd/.work.local/transcripts/<root-sha>/`, in the gitignored per-worktree local tier. A corpus left at the earlier `~/.abcd/history/<root-sha>/transcripts/` path is moved into the store on first resolve, reported, and tombstoned at the old path — see [`../04-surfaces/11-history.md`](../04-surfaces/11-history.md).

### `index.json`

```json
{
  "schema": 1,
  "description": "abcd history/lifeboat registry. Keyed on each repo's root-commit SHA (immutable under rename, GitHub-handle change, or remote move). Names, GitHub URLs, and paths are mutable labels held in each repo's entry and refreshed by ahoy.",
  "repos": [
    {
      "root_commit": "<sha>",          // immutable key — git rev-list --max-parents=0 HEAD
      "name": "<repo-name>",           // mutable label
      "github": "<git-remote-url>",    // mutable label
      "path": "~/...",                 // mutable label — ahoy REFRESHES this every run if the repo moved
      "status": "active",              // "active" | "superseded"
      "supersedes": "<old-sha>",       // present when this repo re-founded an older one
      "superseded_by": "<new-sha>"     // present on the old entry after re-founding
    }
  ]
}
```

`root_commit` is the only immutable field — it survives rename, remote move, and GitHub-handle change. `name`, `github`, and `path` are mutable labels: ahoy refreshes them on every run rather than treating them as write-once.

### `meta.json` (per `<root-sha>/`)

Identity and lineage for one repo. Beyond the obvious identity fields:

- **`aliases`** — array of prior names the repo has had (e.g. a repo renamed on GitHub).
- **`note`** — free-text provenance: why the repo was re-founded, where a backup of the full pre-refounding tree lives, what the old git history carries.
- **`corpus`** — where this repo's captured evidence lives: `{"transcripts": "~/.abcd/transcripts/<root-sha>/records"}`, the store `history.Resolve` returned at install time, home-redacted. ahoy points the block at the store; it does not lay the corpus out (adr-2609090717039680).

Re-founding (clean-history rebuild — the case in [`../../research/notes/ahoy-history-store-manual-scaffolding.md`](../../research/notes/ahoy-history-store-manual-scaffolding.md)) produces a *new* root SHA. ahoy registers the new entry with `supersedes` → old SHA, marks the old entry `superseded_by` → new SHA, and leaves the old corpus in place under its own `<root-sha>/` dir for lifeboat review.

> **Legacy:** `~/ABCDevelopment/.abcd/changelog.md` is a hand-maintained toolchain changelog that predates abcd's `.abcd/` namespace. It is **not** an ahoy-managed artefact and is not part of the `.abcd/` namespace defined below.

The history-store `index.json` is the sole user-scope registry — it records each managed repo's identity and lineage keyed on the immutable `root_commit`; `name`, `github`, and `path` are mutable labels ahoy refreshes on every run. There is no separate `workspaces.json`: abcd manages one repository per tree (see § The two `.abcd/` scopes below), so there is no workspace↔repo grouping to register.

## The voyage store

`voyage/` is **operations** (verb — what we did) as distinct from the lifeboat, which is the **artefact** (noun — what gets carried). It is a **user-scope** artefact keyed on the source repo's root-commit SHA, exactly as the history store is, and it is **never committed** ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md), superseding adr-4's in-tree `.abcd/development/voyage/`). Layout:

```
~/.abcd/voyage/
  <source-root-sha>/
    disembark/
      history.jsonl           append-only manifest log of every disembark run
```

The voyage writer in `internal/core/lifeboat` appends to that ledger and creates nothing else under the store. An **embark side** of the store — `embark/provenance.json` recording the source path, manifest hash, timestamp and files written, and an `embark/from/<timestamp>/` verbatim copy of the input lifeboat behind an opt-in flag — is **staged**: no code path writes anywhere under a voyage `embark/` subtree, and `abcd embark from` carries no flags at all.

Two properties follow from the move, and both are load-bearing:

- **`disembark` never writes to the source repository.** `abcd disembark pack <source-repo> <dest>` reads the source and writes the lifeboat to an operator-chosen `<dest>`; the operations log lands at the operator level. Mining a dead or archived project must not require `ahoy install` into a repo we only want to read.
- **Voyage records absolute source paths, so it must not be committed anywhere.** abcd's own `privacy-hygiene` audit rule (itd-85) flags `/Users/<name>/` in committed files — an in-tree voyage namespace would have made abcd fail its own audit. Keeping voyage at the operator level dissolves the collision rather than exempting it, which is why **no gitignore rule for voyage appears in § 1 below**: there is nothing in-tree to ignore.

Because the lifeboat lands out-of-tree, `embark` reads it from wherever disembark wrote it. The destination is protected by a **safety gate** rather than adr-4's overwrite-in-place-with-`.bak` model: abcd refuses unless `<dest>` is absent, an empty directory, or one carrying a parseable `_provenance.json` — it **never overwrites a directory abcd did not produce**. See [`../04-surfaces/03-embark.md`](../04-surfaces/03-embark.md) for the surface contract.

## The worktree store

**Design target (itd-2609091014076309, `intents/drafts/`; unbuilt).** No `worktree` verb exists in the shipped binary, and nothing in it creates or reads this store. What follows is the layout the intent commits to, on the rule [adr-2609091014087993](../../decisions/adrs/2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md) records: a tool never creates directories in user-owned project space, so a session's or an agent's worktree is machine-scoped rather than a sibling of the checkout. It is a **user-scope** artefact keyed on the repository's root-commit SHA, exactly as the history, transcript and voyage stores are, and it is **never committed**:

```
~/.abcd/worktrees/
  <root-sha>/
    <name>/                   one git worktree of that repository, on its own branch
```

Three properties are load-bearing, and each is the intent's to deliver:

- **Git is the registry.** `git worktree list --porcelain` on the checkout already names every worktree wherever it sits, so the store keeps no index of its own; the verb reads git's answer and says which entries are in the lane, which are outside it, and whether each is clean and merged. A cross-repository walk labels each lane through the history store's `index.json` where the root commit is registered and by SHA where it is not; the worktree's own `.git` file, not the registry's mutable `path` label, says which checkout a lane belongs to.
- **It creates itself through one seam, never through a symlink**, on the transcript store's discipline (§ The transcript corpus): each level made individually and re-verified as a real directory.
- **Reclaim is explicit and provable.** `prune` removes a worktree only when its branch is merged into the default branch and its tree is clean, names everything it declined and why, and never deletes a directory git does not recognise as a worktree of the repository — the same stance embark's destination gate takes at its destination.

Worktrees that already sit beside a checkout are outside the store by definition: listed as such, never moved, and retired by the user's own `git worktree remove`.

## The two `.abcd/` scopes

`.abcd/` is **one namespace pattern instantiated at two scopes**. abcd lives in **one repository** ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)): its design record is **repo-scoped and in-tree**, and the user scope holds only state that is genuinely machine-wide. `/abcd:ahoy` classifies the folder it runs in (see [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)) and acts on the scope that applies:

| Scope | Location | Holds |
|---|---|---|
| **user** | `~/.abcd/` | one per machine — **machine-local shared state only**: the root-SHA-keyed `history/` registry (`index.json` + per-root-SHA `meta.json`) and the root-SHA-keyed `transcripts/` corpus ([adr-2609090717039680](../../decisions/adrs/2609090717039680-the-transcript-corpus-is-a-sibling-store-that-creates-itself.md)), the root-SHA-keyed `voyage/` operations namespace ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)), the root-SHA-keyed `worktrees/` store for session and agent checkouts (design target — [itd-2609091014076309](../../intents/drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md), `intents/drafts/`; the rule is [adr-2609091014087993](../../decisions/adrs/2609091014087993-a-tool-never-creates-directories-in-user-owned-project-space.md)), machine `config.json` defaults, the user-scope `memory/` (personal, cross-project knowledge), the `sources/` corpus `/abcd:ingest` and `/abcd:consult` read — **abcd never creates it**, and both verbs say so and stop when it is absent — and the caller-controlled declarations `path-entry` (the owned PATH copy), `trusted-roots` (foreign-uid configuration roots, below) and `local-transcript-roots` (checkouts whose transcripts are pulled into the checkout). **Never the design record.** The same inventory is drawn as a tree in [`../04-surfaces/01-ahoy.md § What abcd manages`](../04-surfaces/01-ahoy.md#what-abcd-manages--repos-and-abcd); the two are one list and must agree. |
| **repo** | in-tree `.abcd/` | this repository's record and working files — the three-tier layout below, plus `config.json` (with its `meta` setup block), `rules.json`, the `config/` per-surface machine records, the lint and site configuration records (`docs-lint.json`, `record-lint.json`, `citations-baseline.json`, `positioning.json`, `site.json`, `site-baseline.json`) and the native spec store under `development/specs/`. A `memory/` namespace is written when memory is curated. **The home for project work.** There is **no in-tree `lifeboat/`**: the lifeboat is out-of-tree output at an operator-chosen destination (adr-35). Two namespaces are not part of it: there is no `logbook/`, a retired name (see [`../02-constraints/04-naming.md`](../02-constraints/04-naming.md)), and `rp/` is staged with the RP adapter (§ 2). |

**The repo-scope three-tier working layout** (matching [`../02-constraints/01-platform.md`](../02-constraints/01-platform.md) and [`../01-product/02-context.md`](../01-product/02-context.md)):

| Tier | Path | Committed? | Holds |
|---|---|---|---|
| **record** | `.abcd/development/` | committed — in every repository checkout, never in the released binaries | the durable design record: brief, roadmap, intents, ADRs, research |
| **shared work** | `.abcd/work/` | committed | shared working files: `CONTEXT.md` (current orientation), `DECISIONS.md` (the append-only decision log), `intake.md` (the external-contribution runbook), the issue ledger `issues/` (working-tier data per adr-32), the reviews charter `reviews/`, and the branch-ruleset mirror `rulesets/` |
| **local ephemeral** | `.abcd/.work.local/` | gitignored | machine-local scratch: `NEXT.md` (handover), `scratch/`, `logs/`, `reviews/` (intent-audit receipts), `private-names.txt` (the per-machine banlist layer). Per-worktree, so it never merge-conflicts |

**The record is repo-scoped and in-tree — no workspace layer holds it, and there is no `workspaces.json`.** abcd is one repository, so the record lives in that repository's tree; there is no dev→public mirror and no workspace registry. The user scope survives only for state that cannot live in any one repo's tree because it is shared across every abcd-managed repo on the machine: the `history/` store keyed on each repo's root-commit SHA, and machine `config.json` defaults. The repo keeps its `config.json` (carrying the `meta` setup block) and `rules.json` in-tree too — the Claude Code hook and the marker-block installer read them from the repo directory deterministically.

**The `~/.claude/` boundary.** `~/.claude/` is the vendor harness directory. abcd keeps it minimal — **only the abcd plugin install lives there.** No abcd-specific material is written under `~/.claude/`; it routes to the scope-appropriate `.abcd/` instead. The one interaction the design gives abcd with `~/.claude/` is *read-only*: the staged memory harvest (§ 2) would read `~/.claude/projects/<encoded-cwd>/memory/` as a source (see [`02-adapters.md`](02-adapters.md)). abcd never writes there.

## The rules root — which `.abcd/` governs a session

`rules.json`, `guard.json` and the per-repo `config.json` are read from ONE resolved root (`internal/core/rules.Resolve`), so a session's injected rules and its hazard registry can never come from two different directories. The resolution is bounded at the git working tree (GHSA-vvqc-3mv2-5p49): the toplevel git reports for the working directory is resolved first, the walk runs from the working directory upward and stops at that toplevel inclusive, so the nearest `.abcd/` INSIDE the tree wins and a `.abcd/` above the tree is never reached.

**When git will not answer.** abcd runs every git command under an isolated environment (`GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`) so an inherited `GIT_DIR` or an injected config cannot redirect it. That isolation also discards the operator's `safe.directory` exceptions, so `rev-parse --show-toplevel` fails inside a checkout owned by another uid — and it fails outright when the host launches a hook with git off its `PATH`. "Not a repository" and "a repository git will not answer for" are different outcomes: only the first resolves to the working directory with no walk, because collapsing the second onto it dropped the repository's own configuration layer with no error anywhere (the guard's hazards became allows, the rules kill switch stopped applying). The second falls back to the `.git` marker, under two bounds:

| Bound | What it refuses | What it costs a real checkout |
|---|---|---|
| **shape** | a `.git`-NAMED entry that is not a repository — an empty file, a HEAD-less directory, a dangling symlink. The marker walk is a name check that runs to the filesystem root, so without this an unprivileged `: > /tmp/.git` beside a planted `/tmp/.abcd` governs every session in a plain directory under `/tmp`. | nothing: the two shapes git itself reads (a `.git` directory carrying `HEAD`, a `.git` file beginning `gitdir: `) both pass. |
| **ownership** | a marker root whose owner is not the caller. Shape alone is not a trust boundary: `git init` in a shared directory an unprivileged user can write — a root-owned mode-1777 `/tmp` or `/Users/Shared` — produces a genuine repository, and git's refusal on ownership is the SAME signal in that attack as in the legitimate foreign-uid case. Nothing about the tree separates them (iss-2609020259564193). | a declaration, once, per foreign-uid checkout — see below. |

A refused root is refused **loudly and fail-closed**: the session resolves to its own working directory with no walk, the bundled rule defaults and bundled hazard registry stand in for the repository's, and every front door prints one line naming the refused directory, the two uids, and the exact command that re-admits it ([`../../principles/loud-staging.md`](../../principles/loud-staging.md)). The ownership bound applies ONLY to the git-refused fallback: where git answers, the toplevel it named stands whoever owns it — that is a repository git itself vouched for.

**The opt-in — `~/.abcd/trusted-roots`.** A foreign-uid checkout, a container bind mount and a shared CI checkout are legitimate, so the refusal has a supported route back:

```sh
mkdir -p ~/.abcd && printf '%s\n' '/path/to/checkout' >> ~/.abcd/trusted-roots
```

One absolute path per line, matched exactly (both as written and symlink-resolved); `#` starts a comment; blank lines are ignored. The file is read through the guarded bounded read, and only when it is a regular file this uid owns and no one else can write — a declaration anyone could have written is not the caller's word, and is ignored with its own diagnostic line.

**Why the home scope, and not an environment variable.** The bar the opt-in must clear is that the untrusted tree cannot assert it: a `.abcd/` file inside the foreign root declaring itself trusted would be circular. A home-scoped file clears that structurally — writing it needs write access to the caller's own home, the same authority the caller already holds over everything abcd trusts, so the opt-in grants an attacker nothing they did not already have. An environment variable clears the letter of the bar and not its spirit: a repository can ship the shell, `direnv` or task-runner configuration that sets it, and an operator whose shell auto-loads that has the tree declaring itself trusted one indirection out. abcd has ruled on this shape once already — `dataDirHazard` refuses an env-supplied data directory for the same reason, and GHSA-4q78-ccfv-f374's recorded remedy is to move the trust floor "from env to home write", which [adr-46](../../decisions/adrs/0046-persistence-never-weakens-the-verification-posture.md) decision 4 treats as the ownership root. `trusted-roots` follows the `path-entry` idiom (home-scoped, abcd-owned, line-oriented, absent means it vouches for nothing) as a sibling file rather than a section of it: `path-entry` records exactly one thing and is parsed by ahoy, and a second record family in it would couple the rules resolver to that parser.

One residual stays open and recorded rather than assumed shut: **iss-2609020219198779**, the user-scope `~/.abcd` when the home directory is itself a git working tree (dotfiles-in-home). The toplevel for a session in a non-repo directory beneath such a home is the home, so `~/.abcd` governs it; closing it needs a decision on whether a home-directory toplevel is a legitimate configuration scope (spc-23 plans a user layer that would make it one).

## 1. Visibility-driven gitignore policy

Set by ahoy:

| Directory | Public default | Private default |
|---|---|---|
| `.abcd/` | gitignored² | **committed** (entire namespace: `development/` (brief, roadmap, research, personas, the native spec store), `work/`, the machine records at the root and under `config/`, and `memory/` where it exists — visibility is the single switch, no per-subdirectory exceptions) |
| `memory/` (legacy snapshot) | gitignored | **committed** if present¹ |
| `.abcd/.work.local/` | gitignored | gitignored (local-only scratch, per global abcd CLAUDE.md) |

Two artefacts are absent from this table by construction, not by exception:

- The native local transcript store is **always** gitignored: user-scope `~/.abcd/transcripts/` by default (local working data — adr-29, relocated there by adr-2609090717039680), so it is not a repo directory; and when a checkout is declared in `~/.abcd/local-transcript-roots`, the pulled-in store sits under `.abcd/.work.local/`, which the row above already gitignores.
- **`voyage/` and the lifeboat are not repo directories either** (adr-35). Voyage is user-scope (`~/.abcd/voyage/`, § The voyage store) and the lifeboat is out-of-tree output at an operator-chosen destination, so **no gitignore rule applies to either under any visibility** — there is nothing in-tree to switch.

¹ New projects use `.abcd/memory/`, the substrate `abcd memory` curates. `memory/` is the legacy `cp -r` snapshot pattern that some existing projects maintain manually — abcd respects it if present, but doesn't write to it.

² On a repo whose `.abcd/` already holds **tracked** files, install narrows the
`.abcd/` entry to the local tier (`.abcd/.work.local/`) and keeps every other
entry, `memory/` included (iss-255). An ignore rule cannot untrack committed
records: the wholesale fence on such a repo only hides every new record file
from `git status` and makes `git add` refuse them, while the committed tiers
stay published regardless. Narrowing needs positive evidence of tracked files
(a repo whose git cannot be asked keeps the declared fence), and the install
receipt states the narrowing out loud.

**No exceptions to the visibility rule.** Locked decision: visibility is **one switch**, with no always-gitignored carve-out for any single namespace on sensitivity grounds. If sensitivity is a concern, set visibility=public, which gitignores all of `.abcd/`. Per-subdirectory exceptions create maintenance burden and contradict the transparent-prompts principle ([`04-universal-patterns.md § 1`](04-universal-patterns.md#1-transparent-prompts)). The tracked-tier narrowing (²) is not a per-subdirectory carve-out of that decision: it is the mechanical boundary of what an ignore rule can do at all — the switch stays single, and where git already tracks the namespace the fence covers the one part git can still fence.

**Sensitivity concern still valid for `/abcd:launch` payload**: regardless of visibility, the launch payload manifest ([`../04-surfaces/04-launch.md § 2`](../04-surfaces/04-launch.md#2-curated-release-artefact-default-deny)) excludes `.abcd/` entirely from what ships in the curated release artifact. So a private repo that commits its whole record locally still doesn't leak any of it on launch.

In private repos, the entire `.abcd/` namespace is reproducible from a fresh clone — a clone carries the committed record and working files. The lifeboat is **not** part of what a clone carries: it is out-of-tree output at an operator-chosen destination, and embark reads it from there (adr-35).

**Memory locations to keep straight:**

abcd's curated memory exists at **two scopes** (per § The two `.abcd/` scopes), and there is one non-abcd memory location alongside it:

1. **`.abcd/memory/`** (repo scope) — the **primary** abcd memory: curated semantic summaries written by `abcd memory ingest`, tracked in private repos, the canonical input for `principle-distiller`. Most memory lives here.
2. **`~/.abcd/memory/`** — **user-scope** memory: personal preferences and cross-project principles that have no single repo home.
3. **`memory/`** — legacy `cp -r` snapshot at the repo root that some existing projects maintain. abcd respects if present but doesn't write to it.

Which scope a curated page lands in is a routing decision — see [`07-memory.md`](07-memory.md) § scope routing. Retrieval across the two scopes is **not** a flat union (that would overflow context); it is keyword-recall + budget-bracketed injection per itd-39. When the brief says "memory" without qualification, it means the repo-scope `.abcd/memory/`.

**`.abcd/.work.local/` is local-only everywhere.** Working notes, drafts and status trackers stay gitignored. Promotion of useful content into the tracked `.abcd/work/` tier is a deliberate act today; the `dev-sync` machinery that would sweep it ([§ 2](#2-abcdwork-namespace-and-dev-sync)) is staged.

**AI transparency level (staged).** A second axis, independent of visibility, elected through `ai_transparency.level`. It has no backing intent or ADR and no shipped code reads it; ahoy neither prompts for it nor writes it. The shape the design commits to:

| Level | Conversations (transcripts) | Plans | Tasks | Metadata (sessions, actions) |
|---|---|---|---|---|
| `full` | yes | yes | yes | yes |
| `metadata` | no | yes | yes | yes |
| `none` | no | no | no | no |

**Why separate from visibility:** a private repo might want `metadata`-only transparency to keep storage tight; a public OSS project might want `full` for credibility. Visibility decides what's *committed*; transparency decides what's *captured at all*.

What it would drive, each half staged with the surface it names:

- **`dev-sync`:** `none` skips capture entirely; `metadata` skips conversation transcripts; `full` captures everything per source enable flag.
- **`disembark`:** the chat-distiller pass is a no-op on `none` and runs on `metadata` or `full`. That pass is itself staged: `chat-distiller` is a Phase-6 design target, not a shipped agent.
- **`launch` payload:** the value would be carried into the public `marketplace.json` so consumers know what to expect. `.claude-plugin/marketplace.json` carries no such field today, and `internal/core/launch` reads that file only for the version lockstep check.

Independently of the axis, launch's pre-flight scrub strips absolute paths and PII whatever the transparency level. The level would determine what *exists*; the pre-flight decides what *ships*.

**Visibility × transparency interaction (added post-audit 2026-05-07):** the two axes are independent. Any combination is valid: `private × none` (paranoid, no captures, no commit), `private × full` (everything captured locally, nothing public), `public × none` (public repo, no AI captures committed), `public × full` (everything captured AND committed — useful for OSS-credibility OSS projects). The launch payload's exclusion rules (per [`04-surfaces/04-launch.md § 2`](../04-surfaces/04-launch.md)) are unconditional regardless of `ai_transparency.level` — launch always ships the visibility-determined payload, and `ai_transparency` only governs what was captured in the first place.

## 2. `.abcd/work/` namespace and `dev-sync`

**This whole section is staged.** No `dev-sync` verb is registered on the binary, no Go source names one, and none of the adapters below is built. `.abcd/work/` itself is real and committed (§ The two `.abcd/` scopes); what is staged is the machinery that would populate it by sweeping volatile sources. The one thing that already works this way is the issue ledger, and it got there without a sync: `abcd capture` writes `.abcd/work/issues/{open,resolved,wontfix}/` directly, so the "parse `.work.local/issues.md` on first sync" migration below has nothing left to migrate.

`.abcd/work/` is the **curated-from-volatile-sources** namespace. Volatile inputs (gitignored or external) would be analysed and promoted into tracked `.abcd/work/` artefacts by `abcd dev-sync`. This addresses three problems: noisy sources stay gitignored; curated lessons get tracked; abcd does not have to read volatile sources every time.

**Source → target table.** The Package column names where the work would live; only `internal/core/memory` exists today.

| Volatile source (gitignored or external) | Curated abcd target (tracked in private repos) | Package |
|---|---|---|
| Agent memory (opt-in harvest per [`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching)) | `.abcd/memory/` | `internal/core/memory` (exists; `abcd memory ingest` is its shipped front door, and it takes an https URL rather than a harvested directory) |
| Ad-hoc reviews not tied to a spec, captured by whichever oracle adapter runs (vendor paths in [`02-adapters.md`](02-adapters.md)). **Spec-tied reviews are written directly to the native spec review store at review time — a sweep does NOT collect those.** | `.abcd/work/reviews/` | oracle-adapter capture: unbuilt, and the target directory is already governed by the reviews charter at [`../../../work/reviews/README.md`](../../../work/reviews/README.md), whose shape (`<YYYY-MM-DD>-<scope>/00-summary.md`, append-only, `RD001`–`RD003`) any sweep must meet |
| `.abcd/.work.local/notes/`, `.abcd/.work.local/<feature>/` | a curated notes target under `.abcd/work/` | workdir capture: unbuilt. No `.abcd/work/notes/` directory exists, and nothing creates one |
| RP workspace state (opt-in adapter; vendor paths in [`02-adapters.md`](02-adapters.md)) | `.abcd/rp/workspace.json` (per itd-7) | RP workspace adapter: unbuilt, and no `.abcd/rp/` namespace exists in any tree |

**Triggers, as designed:**

- **Implicit:** `/abcd:disembark` Phase 0 would run `dev-sync` (always-fresh-at-disembark)
- **Manual:** an `abcd dev-sync` CLI for ad-hoc refresh

> **Open question (adr-35):** the implicit trigger above is stated against the old model, in which disembark packed *this* repo. adr-35 makes disembark **read-only against the source** (`abcd disembark pack <source-repo> <dest>`; a test hashes the source tree before and after), while `dev-sync` **writes** into the source repo (`.abcd/memory/`, `.abcd/work/`, `.abcd/rp/`). The two cannot both hold. What must be decided: whether `dev-sync` is dropped from disembark's Phase 0 entirely and becomes a separate operator-run step (with disembark reading whatever curated artefacts happen to be present, and none at all in a repo abcd never touched — the primary case), or whether the implicit trigger survives only in a narrow "source repo is abcd-managed and the operator opted in" mode. adr-35 does not settle it. The question is not urgent while neither half ships, but it is the first thing to settle when either does.

Scheduled/cron sync **comes in a later phase** of the plugin (itd-13).

**Per-source on/off**, the `dev_sync.*` keys in the staged config list above:

- ahoy would ask per-source enable (transparent prompts); today it asks four questions and none of them is this
- defaults: all sources on for private, all sources off for public
- disembark Phase 0 would honour per-source flags (subject to the open question above)

**Per-source provenance and curation rules:**

- **Memory (volatile) → `.abcd/memory/` (curated):** the source is an opt-in memory harvest per [`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching). The repo-local legacy `memory/` snapshot (the `cp -r` pattern) is the workflow this replaces. Output is *not verbatim*: distilled summaries grouped by domain, written as actionable suggestions for future agents. Why curated: raw memories grow unbounded and carry personal phrasing. Inputs to `principle-distiller` (Pass C).

- **Reviews (volatile) → `.abcd/work/reviews/` (curated):** reviews are captured by whichever oracle adapter runs — host-delegated by default ([adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)), with an opt-in adapter as the alternative. The sweep would harvest **ad-hoc oracle reviews not tied to a spec** from that adapter's local store; **spec-tied reviews are NOT swept** — the native spec review store captures them directly at review time. See [`02-adapters.md`](02-adapters.md) for the adapter's harvesting detail (vendor paths, workspace matching, the stability and privacy safeguards). Dedup by content hash; idempotent. Inputs to `review-collator` (Pass A), itself a Phase-6 design target rather than a shipped agent.

- **`.abcd/.work.local/` (volatile, local-only) → the curated tier:** the local tier's notes and per-feature directories would be distilled into `.abcd/work/`. Files in `.abcd/.work.local/` are never moved or deleted — the sweep is read-and-curate, and the source stays put. Inputs to `principle-distiller` (Pass C) and `chat-distiller` (Pass B, as auxiliary context).

- **RP workspace state (volatile) → `.abcd/rp/workspace.json` (per itd-7):** the opt-in adapter would pull its own workspace state (the workspace whose root path matches the current repo) into `.abcd/rp/workspace.json`. The vendor filesystem layout and the match/normalisation mechanics live with the adapter — see [`02-adapters.md`](02-adapters.md). Workspace state only; presets, routing scope and a window helper come in a later phase.

**Reviews as a first-class pitfall source (staged with `review-collator`, a Phase-6 design target):**

Plan, implementation and completion reviews are *exceptionally* useful for spotting issues. The `review-collator` agent must extract every "P0 / P1 / watch out for X / found bug" finding as a candidate pitfall: **even when the original issue was fixed**, the lesson survives. Output:

- `reviews-consolidated.json` — full review summaries
- `candidate-pitfalls.json` — extracted findings ready for distiller dedup

`principle-distiller` (Pass C) has four pitfall sources to dedupe by topic-hash or canonical phrasing: a source `pitfalls.md`, `candidate-pitfalls.json`, Pass B chat-distiller deltas, and code-rescuer's `code-principles.json`.

**Distinct from `principles.json`:** `.abcd/memory/`, `.abcd/work/reviews/` and `.abcd/work/issues/` are **persistent rolling artefacts** in the source repo, used as ongoing input to future agents. `principles.json` is **per-disembark synthesis** written into the lifeboat at `<dest>/principles.json` — out-of-tree, at the operator-chosen destination, never back into the source repo (adr-35). The lifeboat consumes `.abcd/work/`; `.abcd/work/` is not the lifeboat.

## 3. Plugin shape — directory layout

**Repository layout** (a Go binary plus the markdown plugin surface that shells to it):

```
abcd/
├── .claude-plugin/plugin.json
├── .claude-plugin/marketplace.json
├── README.md
├── go.mod / go.sum
├── cmd/
│   └── abcd/main.go                    # entrypoint — wires the CLI front door to the core
├── internal/
│   ├── core/                           # transport-agnostic core — one package per capability (adr-23)
│   │   └── …                           #   ahoy, banlist, capture, changelog, cite, decide, frontmatter,
│   │                                   #   glossary, grounds, guard, history, ideate, identity, intent,
│   │                                   #   issueschema, launch, lifeboat, lint, mdrecord, memory,
│   │                                   #   positioning, provenance, reading, record, recordid, release,
│   │                                   #   repolint, rules, site, spec, surface, update, vintage —
│   │                                   #   each returns structured results
│   ├── adapter/                        # the seam packages (adr-22): interface + native default + optional plug-in
│   │   ├── scanner/                    # native secret/PII scan; TruffleHog as the opt-in deep backend
│   │   └── gitleaks/                   # the gitleaks backend behind that seam
│   │                                   # STAGED: the oracle (adr-25), history (adr-29), spec (adr-26) and run
│   │                                   #   (adr-27) seams have no adapter package. Their native behaviour lives
│   │                                   #   in internal/core (history, spec) or is host-delegated (oracle);
│   │                                   #   nothing selects between backends, and there is no adapter registry.
│   └── surface/
│       └── cli/                        # Cobra front door — the only front door that ships
│                                       # STAGED: an mcp/ front door (adr-23's third door)
├── commands/                      # markdown command surfaces — <verb>.md, the gated list in
│   └── <verb>.md                       #   ../04-surfaces/README.md (abcd.md is the bare /abcd board).
│   # NOTE: the mapping to binary verbs is not one-to-one in either direction. `changelog`, `completion`,
│   # `hook`, `rules` and `spec` are binary verbs with no command page (`hook` registered but hidden from
│   # --help, reached from hooks.json); `consult`, `ingest` and `prepare-this-repo`
│   # are command pages that invoke no binary verb (prepare-this-repo is an interim bridge until abcd
│   # manages repositories directly). `uninstall` is a sub-verb of /abcd:ahoy, not a standalone
│   # command: the ahoy page dispatches install, uninstall, dry-run, doctor, remote and
│   # identity-check internally.
├── agents/                             # 15 agent prompts — see 01-agents.md (markdown, host-delegated)
│   ├── cold-reading-widening.md / cold-reading-entailment.md / cold-reading-comparative.md
│   ├── cold-reading-detection.md / docs-currency-reviewer.md / graveyard-interpreter.md
│   ├── intent-auditor.md / lifeboat-reviewer.md / press-release-composer.md
│   ├── principle-distiller.md / release-changelog-composer.md / ruthless-reviewer.md
│   ├── scribe.md / security-reviewer.md
│   └── sota-researcher.md              # plus per-agent fixtures/ dirs, README.md, CHANGELOG.md
└── hooks/                              # Claude Code event hooks — every event command runs through a resolving shim
    ├── bootstrap.sh                    # PROVISIONS the plugin-root binary; it never builds one. Either it copies a
    │                                   #   re-verified artefact out of the persistent $CLAUDE_PLUGIN_DATA download cache,
    │                                   #   or it resolves the latest release tag off a 302 and downloads the pinned asset,
    │                                   #   verified against that release's own checksums.txt. Its two `go build` strings are
    │                                   #   printed INSTRUCTIONS to the operator for the cases it cannot provision.
    │                                   #   Referenced by four of the five event commands; SessionEnd is the exception below
    └── hooks.json                      # UserPromptSubmit → hook prompt-router; SessionStart → ONE chained command:
                                        #   bootstrap.sh, then session-start + prompt-router-reset, each fed a copy of the
                                        #   payload (siblings would run in parallel and share one stdin);
                                        # PreToolUse (matcher Bash) → guard hook; PreCompact → prompt-router-reset; SessionEnd → session-end.
                                        # UserPromptSubmit, PreToolUse and PreCompact also self-provision: when $CLAUDE_PLUGIN_ROOT/abcd
                                        # is missing they attempt hooks/bootstrap.sh (throttled by a .bootstrap.attempt marker
                                        # within a 10-minute window), then fall back to a PATH-resolved abcd — absolute, outside the
                                        # working directory, not world-writable, and recorded in ~/.abcd/path-entry as this machine's
                                        # own; else ignored with a reason — before failing loudly (SessionStart has no PATH rung).
                                        # SessionEnd downloads nothing: a fetch as the session exits races the harness's hook
                                        # cancellation and loses the transcript the hook exists to capture (iss-2608210934566223),
                                        # so it resolves the plugin root, then PATH, then says the transcript was not captured
```

The core is organised one package per capability under `internal/core/`.
[`02-adapters.md`](02-adapters.md) owns the seam catalogue — each seam's
interface, native default, and optional external plug-in — so this brief does not
restate it here. The catalogue names five seams; one of them, the scanner, has a
package under `internal/adapter/` today.

**Plugin-internal development namespace** (committed in private repos, gitignored in public):

```
.abcd/
├── config.json                         # config + the `meta` setup block (schema_version, setup_version, ...)
├── config/                             # per-surface machine records: identity.json, launch-payload.json,
│                                       #   reading-presets.json, version-location.json
├── corpus.json                         # validation-corpus manifest — a design target (itd-25); not yet in the tree
├── rules.json                          # per-repo override of plugin-bundled rule defaults (per itd-3)
├── README.md                           # the namespace's own index
├── docs-lint.json                      # docs-lint rule configuration (also the public banlist layer)
├── record-lint.json                    # record-lint rule configuration
├── citations-baseline.json             # the citation baseline the citation_* rules enforce offline
├── positioning.json                    # the identity block every rendered surface is held to
├── site.json                           # site render declaration
├── site-baseline.json                  # the site check's admitted-reference baseline
├── development/                        # durable design record — flat by artefact type (per adr-30)
│   ├── personas.json                   # placeholder personas (Alice, Bob, Carol, ...)
│   ├── brief/                          # canonical, current-state (no archive — per adr-5); chaptered
│   │   ├── README.md                   #   index
│   │   ├── 00-meta.md                  #   brief conventions and the truth rule
│   │   ├── 01-product/ 02-constraints/ 03-evidence/
│   │   ├── 04-surfaces/ 05-internals/ 06-delivery/
│   │   └── glossary/                   #   one file per term per bounded context
│   ├── intents/                        # press-release intents — directory-as-status (adr-30)
│   │   ├── README.md                   # intent format + lifecycle + index
│   │   ├── disciplines/                # itd-N-<slug>.md (active cross-cutting rules)
│   │   ├── drafts/                     # captured intents, no plan yet
│   │   ├── planned/                    # has linked native spec, work pending or in flight
│   │   ├── shipped/                    # populated as linked specs close + intent-auditor runs
│   │   └── superseded/                 # retired intents (moved by hand)
│   ├── principles/                     # distilled cross-cutting design principles (the lifeboat packs these)
│   ├── decisions/                      # ratified architecture decisions (MADR)
│   │   ├── adrs/                        # <N>-<slug>.md — the ordinals 0001-0058, then minted stamps (`abcd decide`)
│   │   └── notes/
│   ├── roadmap/
│   │   ├── README.md                   # status dashboard
│   │   ├── phases/
│   │   │   ├── README.md               # phase index
│   │   │   └── phase-N-<slug>.md       # ordered build plan; each ends in a milestone (per adr-9)
│   │   └── rfcs/
│   │       ├── README.md
│   │       └── rfc-N-<slug>.md         # community discussion artefacts (an accepted RFC produces an ADR)
│   ├── plans/                          # dated design / implementation plans (YYYY-MM-DD-*)
│   ├── specs/                          # native minimal spec store — directory-as-truth (adr-26): open/ + closed/
│   ├── readings/                       # the cold-reading ledger — charter, reading records, dispositions
│   ├── release/                        # release surface declaration (README.md + surface.json)
│   ├── release-gate/                   # the release gate's manifest and its brief-surface cross-check
│   └── research/
│       ├── data/                       # dated machine-readable run outputs
│       ├── notes/                      # dated investigation write-ups
│       └── prompting/                  # prompt R&D
│           ├── README.md
│           ├── 01-general-best-practices.md
│           └── agents/<name>.md        # per-agent SOTA research (a design target — see 05-prompt-quality.md)
│   # NOTE: there is NO `development/voyage/` here. Voyage is user-scope and never committed —
│   #       ~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl, with the embark side staged
│   #       (adr-35; see § The voyage store above).
└── work/                               # shared working tier (committed)
    ├── CONTEXT.md                      # current orientation
    ├── DECISIONS.md                    # append-only decision log; architecture-shaping entries graduate to ADRs
    ├── intake.md                       # the external-contribution runbook
    ├── issues/{open,resolved,wontfix}/ # iss-N-<slug>.md ledger entries, written by `abcd capture` (per itd-4)
    ├── reviews/                        # the reviews charter and its dated <YYYY-MM-DD>-<scope>/ directories
    └── rulesets/                       # the branch-ruleset mirror
# `.work.local/` is the third tier and is gitignored under every visibility, so it is not drawn here.
# A `memory/` namespace joins these where memory is curated (see § Memory locations to keep straight).
# STAGED, and absent from every tree today: `logbook/` (retired — run output goes to
#   .abcd/.work.local/logs/) and `rp/` (the RP workspace pull, per itd-7 with its opt-in adapter).
# NOTE: there is NO `.abcd/lifeboat/` here. The lifeboat is out-of-tree output at an operator-chosen
#       destination (`abcd disembark pack <source-repo> <dest>`) — still the latest snapshot rather than an
#       accumulating archive, but the source repo is never written to (adr-35, ../04-surfaces/03-embark.md).
```

**User-facing docs** (packaged into the curated release artifact):

```
docs/
├── README.md
├── assets/                            # shared images and static assets (adr-47)
├── tutorials/                          # learning guides
├── how-to/                             # task-oriented how-tos
├── reference/                          # command reference, config schemas
├── explanation/                        # conceptual: lifeboats, intents, capture, etc.
└── requirements.txt                    # the pinned site-build toolchain the docs render through
```

**Doc framework note**: the native spec store, memory, and config live under `.abcd/`. Plugin-internal design docs live under `.abcd/development/`. User-facing docs live under `docs/`. We use the *shape* of a planning-vs-roadmap-vs-process split, not any one tool's *location* convention.
