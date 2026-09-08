# Configuration Model

This file holds the configuration schema (`config.json`, including its `meta` setup block), the visibility-driven gitignore policy, and the `dev-sync` namespace that pumps volatile sources into curated artefacts.

## Setup metadata — `config.json["meta"]`

Setup metadata is a `meta` block inside `.abcd/config.json`; there is no separate `.abcd/meta.json` at repo scope (spc-16). ahoy stamps and reads it via `config.json["meta"]` (example values shown):

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

(example values shown; ahoy populates from prompts on first run, no silent default for `visibility`):

```json
{
  "repo": {
    "visibility": "private"            // "private" | "public" — set by ahoy each run, no silent default
  },
  "ai_transparency": {
    "level": "metadata"                 // "full" | "metadata" | "none" — separate axis from visibility
                                        // full: conversations + plans + tasks + metadata
                                        // metadata: plans + tasks + metadata (no conversation transcripts)
                                        // none: nothing
                                        // Drives what dev-sync captures and what disembark includes.
  },
  "docs": {
    "target": "both"                    // "claude_md" | "agents_md" | "both" | "skip"
  },
  "oracle": {                           // oracle seam — internal/adapter/oracle
    "backend": "host-delegated"         // "host-delegated" | "native" | "cli" | "api" | "mcp"
                                        // host-delegated (default): abcd emits a prompt, the host's subagent
                                        //   dispatch runs it — no API keys, no model config (adr-25)
                                        // native | cli | api | mcp: opt-in oracle adapters, selected when an
                                        //   operator wants abcd to reach a model directly; unreachable → host-delegated
  },
  "spec": {                             // spec seam — internal/adapter/spec
    "backend": "native"                 // "native" (directory-as-truth + dependency graph, adr-26) | "ccpm"
                                        //   (the companion harness over conventions, adr-24)
  },
  "run": {                              // run seam — internal/adapter/run
    "backend": "native"                 // "native" (thin Go loop, adr-27) | "workflows" | "companion"
                                        //   each iteration gates on a receipt and enforces the safety guard
  },
  "history": {                          // history seam — internal/adapter/history
    "backend": "native"                 // "native" (local redacted transcript store, root-SHA-keyed, adr-29)
                                        //   | "specstory" (opt-in capture source over the same store)
  },
  "scan": {                             // scanner seam — internal/adapter/scanner
    "deep": false                       // native secret/PII scan is the default; deep adds an opt-in TruffleHog
                                        //   backend (also gitleaks when wired), asked only if visibility=private
  },
  "disembark": {
    "maxAgentTokens": 100000            // per-agent context budget; over → stream + summarise
  },
  "embark": {
    "scan": true                        // default: include `embark scan` discovery in onboarding suggestions
  },
  "memory": {                           // memory harvest — read-only source over the native .abcd/memory/ substrate
    "harvest": "native"                 // "native" | "claude" | "<custom-path>" — vendor memory harvest is an
                                        //   opt-in read-only source; see 04-universal-patterns.md § 7
  },
  "reviews": {                          // review-artefact capture — written by whichever oracle adapter runs
    "capture": "oracle"                 // "oracle" (capture over the native review store, adr-25) | "none"
  },
  "dev_sync": {                         // per-source enable flags (asked during ahoy); semantic names per 04-universal-patterns.md § 7
    "reviews": { "enabled": true  },    // oracle-adapter capture → .abcd/work/reviews/ — sweeps ad-hoc reviews not tied to a spec only;
                                        // spec-tied reviews land in the native spec review store at write-time (not controlled by this flag)
    "memory":  { "enabled": true  },    // memory harvest → .abcd/memory/
    "work":    { "enabled": true  },    // .abcd/.work.local/ issues + notes/ → .abcd/work/{issues,notes}/
    "rp":      { "enabled": true  }     // RP workspace pull (per itd-7; opt-in RP adapter) → .abcd/rp/
  },
  "intent": {
    "auto_link": true,                  // /abcd:intent plan injects bidirectional link automatically
    "auto_ship": true                   // the native spec-close hook (spc-36) reconciles linked
                                        // intents planned/ → shipped/ on a successful close (spc-28
                                        // intent_lifecycle.reconcile); /abcd:intent "<text>" always lands in
                                        // drafts/ (no auto-trigger)
  },
  "capture": {
    "default_severity": "minor"         // default severity when /abcd:capture omits it (per itd-4)
  },
  "rules": {
    "force_refresh_every_n": 15         // prompt-router fixed-N refresh backstop (per itd-3); event-driven
                                        // reset on SessionStart/PreCompact is the primary refresh, so this
                                        // large counter only re-injects always-relevant domains. Default 15.
  },
  "adapters": {                         // wired-adapter registry (internal/registry), refreshed each ahoy —
                                        // records which optional external backends are present per seam
    "repoprompt": { "detected": true }, // an oracle (mcp) backend
    "ccpm":       { "detected": false } // the spec-seam deeper backend (the companion harness over conventions)
  },
  "scout": {
    "issue_scout": { "enabled": false } // opt-in (asked during ahoy)
  }
}
```

**Owed-review draining (spc-43, itd-53) is receipt gating in the `run` seam (adr-27).** Owed fidelity reviews are drained at the `run` seam's iteration boundary: each iteration gates on a **receipt** and applies the safety guard, a report-not-block step whichever adapter provides the loop (native Go loop, Claude Workflows, the companion harness). There is no autodrain config knob and no post-iteration edge to hang it on — receipt gating is part of the seam contract, inherited by every adapter loop rather than re-implemented per loop. The full report-vs-block / cost-bound decision record is [`adr-27`](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md); the companion consistency gate's `RC*` codes are registered in [`06-lint.md § 1`](06-lint.md#1-lint-code-namespace).

**Audit-loop mode + budget — per-intent frontmatter (itd-50, spc-52).** The audit-loop policy is NOT configured in `config.json`; it is elected **per intent** in the intent's own frontmatter, so the choice is portable with the intent (it survives the lifeboat) and one intent can loop while another stays record-only:

```yaml
# intents/<dir>/itd-N-*.md frontmatter
audit_mode: loop-to-acceptance   # "record-only" (default) | "loop-to-acceptance"
audit_budget: 3                  # iteration ceiling for loop-to-acceptance (default 3)
```

- **`audit_mode`** — `record-only` is the default (and the value when the key is **absent**): a `NOT_MET` is recorded to `## Audit Notes`, no re-work. `loop-to-acceptance` makes a `NOT_MET` re-open the linked work and iterate until `MET` or the budget bounds it. Set at plan time.
- **`audit_budget`** — the **declared** iteration ceiling for `loop-to-acceptance`. **Default `3`** when the key is absent but the mode is `loop-to-acceptance` (the intent-grain equivalent of the implementation-grain `MAX_REVIEW_ITERATIONS`). One iteration = one full re-open + re-review cycle that returned `NOT_MET`. A `record-only` intent **ignores** budget entirely (not an error).
- **Budget validation is fail-closed.** A malformed, zero, or negative `audit_budget` on a `loop-to-acceptance` intent is a **policy error**: the loop never starts (recorded fail-closed, issue logged) — it never silently coerces to the default or loops forever.
- **Enqueue-time snapshot.** The review-queue entry **snapshots** the effective `audit_mode` + declared `audit_budget` (plus the `audit_iterations_used` counter and the terminal `audit_outcome`) at enqueue time — the drainer reads the snapshot, never live frontmatter, so a mid-loop edit cannot change an in-flight loop. These four queue-entry fields are additive; a legacy entry without them parses as `record-only`.
- **The loop terminal + the gate live in the drainer/policy layer** (`internal/core/repolint`), never in the pure on-close hook. For the full state-coverage table, the `UNACHIEVABLE` replan surface, and the gated manual-verification **receipt schema** (`{intent_id, machine_rollup, state, justification?, recorded_by_role, ts}` under `.abcd/logbook/audit/verify-<ts>/`, distinct from the machine verdict), see [`../04-surfaces/05-intent.md` § 7 Role 1 — the audit loop](../04-surfaces/05-intent.md).

Schema versioning + cross-version migration **comes in a later phase** (abcd stamps `schema_version: 1` everywhere; migrators added if/when a later phase changes the shape — see itd-9).

## The history store

The history store is a **user-scope** artefact, shared across every abcd-managed repo on the machine, living at `~/.abcd/history/`. ahoy bootstraps it once (first `install` on a fresh machine creates it transparently — see [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)). Layout:

```
~/.abcd/history/
  index.json                  registry — root-SHA → repo entry
  <root-sha>/
    meta.json                 identity + lineage for one repo
    transcripts/              native local redacted transcript store (root-SHA-keyed, adr-29)
    prompt-exports/           oracle-adapter ad-hoc review exports
```

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
- **`corpus`** — paths (relative to the `<root-sha>/` dir) of the captured evidence: `{"transcripts": "transcripts/", "prompt_exports": "prompt-exports/"}`.

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
    embark/
      provenance.json         source path + manifest hash + timestamp + files written
      from/<timestamp>/       opt-in via embark --archive: verbatim copy of the input lifeboat
```

Two properties follow from the move, and both are load-bearing:

- **`disembark` never writes to the source repository.** `abcd disembark <source-repo> to <dest>` reads the source and writes the lifeboat to an operator-chosen `<dest>`; the operations log lands at the operator level. Mining a dead or archived project must not require `ahoy install` into a repo we only want to read.
- **Voyage records absolute source paths, so it must not be committed anywhere.** abcd's own `privacy-hygiene` audit rule (itd-85) flags `/Users/<name>/` in committed files — an in-tree voyage namespace would have made abcd fail its own audit. Keeping voyage at the operator level dissolves the collision rather than exempting it, which is why **no gitignore rule for voyage appears in § 1 below**: there is nothing in-tree to ignore.

Because the lifeboat lands out-of-tree, `embark` reads it from wherever disembark wrote it. The destination is protected by a **safety gate** rather than adr-4's overwrite-in-place-with-`.bak` model: abcd refuses unless `<dest>` is absent, an empty directory, or one carrying a parseable `_provenance.json` — it **never overwrites a directory abcd did not produce**. See [`../04-surfaces/03-embark.md`](../04-surfaces/03-embark.md) for the surface contract.

## The two `.abcd/` scopes

`.abcd/` is **one namespace pattern instantiated at two scopes**. abcd lives in **one repository** ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)): its design record is **repo-scoped and in-tree**, and the user scope holds only state that is genuinely machine-wide. `/abcd:ahoy` classifies the folder it runs in (see [`../04-surfaces/01-ahoy.md`](../04-surfaces/01-ahoy.md)) and acts on the scope that applies:

| Scope | Location | Holds |
|---|---|---|
| **user** | `~/.abcd/` | one per machine — **machine-local shared state only**: the root-SHA-keyed `history/` store (`index.json` + per-root-SHA transcript corpus, [adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)), the root-SHA-keyed `voyage/` operations namespace ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)), machine `config.json` defaults, the user-scope `memory/` (personal, cross-project knowledge), and the caller-controlled trust declarations `path-entry` (the owned PATH copy) and `trusted-roots` (foreign-uid configuration roots, below). **Never the design record.** |
| **repo** | in-tree `.abcd/` | this repository's record and working files — the three-tier layout below, plus `config.json` (with its `meta` setup block), `rules.json`, and the `memory/`, native spec store, `logbook/`, `rp/` namespaces. **The home for project work.** There is **no in-tree `lifeboat/`**: the lifeboat is out-of-tree output at an operator-chosen destination (adr-35). |

**The repo-scope three-tier working layout** (matching [`../02-constraints/01-platform.md`](../02-constraints/01-platform.md) and [`../01-product/02-context.md`](../01-product/02-context.md)):

| Tier | Path | Committed? | Holds |
|---|---|---|---|
| **record** | `.abcd/development/` | committed — in every repository checkout, never in the released binaries | the durable design record: brief, roadmap, intents, ADRs, research |
| **shared work** | `.abcd/work/` | committed | shared working files — `CONTEXT.md` + `DECISIONS.md` |
| **local ephemeral** | `.abcd/.work.local/` | gitignored | machine-local scratch — `NEXT.md`, `scratch/`, `logs/` |

**The record is repo-scoped and in-tree — no workspace layer holds it, and there is no `workspaces.json`.** abcd is one repository, so the record lives in that repository's tree; there is no dev→public mirror and no workspace registry. The user scope survives only for state that cannot live in any one repo's tree because it is shared across every abcd-managed repo on the machine: the `history/` store keyed on each repo's root-commit SHA, and machine `config.json` defaults. The repo keeps its `config.json` (carrying the `meta` setup block) and `rules.json` in-tree too — the Claude Code hook and the marker-block installer read them from the repo directory deterministically.

**The `~/.claude/` boundary.** `~/.claude/` is the vendor harness directory. abcd keeps it minimal — **only the abcd plugin install lives there.** No abcd-specific material is written under `~/.claude/`; it routes to the scope-appropriate `.abcd/` instead. The one interaction abcd has with `~/.claude/` is *read-only*: `dev-sync memory` harvests `~/.claude/projects/<encoded-cwd>/memory/` as a source (see [`02-adapters.md`](02-adapters.md)). abcd never writes there.

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
| `.abcd/` | gitignored² | **committed** (entire namespace: `development/` (brief, roadmap, research, personas), the native spec store, `memory/`, `logbook/`, `rp/` — visibility is the single switch, no per-subdirectory exceptions) |
| `memory/` (legacy snapshot) | gitignored | **committed** if present¹ |
| `.abcd/.work.local/` | gitignored | gitignored (local-only scratch, per global abcd CLAUDE.md) |

Two artefacts are absent from this table by construction, not by exception:

- The native local transcript store is **always** gitignored (user-scope `~/.abcd/history/`, local working data — adr-29), so it is not a repo directory.
- **`voyage/` and the lifeboat are not repo directories either** (adr-35). Voyage is user-scope (`~/.abcd/voyage/`, § The voyage store) and the lifeboat is out-of-tree output at an operator-chosen destination, so **no gitignore rule applies to either under any visibility** — there is nothing in-tree to switch.

¹ New projects use `.abcd/memory/` (curated by `dev-sync memory`). `memory/` is the legacy `cp -r` snapshot pattern that some existing projects maintain manually — abcd respects it if present, but doesn't write to it.

² On a repo whose `.abcd/` already holds **tracked** files, install narrows the
`.abcd/` entry to the local tier (`.abcd/.work.local/`) and keeps every other
entry, `memory/` included (iss-255). An ignore rule cannot untrack committed
records: the wholesale fence on such a repo only hides every new record file
from `git status` and makes `git add` refuse them, while the committed tiers
stay published regardless. Narrowing needs positive evidence of tracked files
(a repo whose git cannot be asked keeps the declared fence), and the install
receipt states the narrowing out loud.

**No exceptions to the visibility rule.** Earlier drafts of this brief carved out `.abcd/logbook/` as always-gitignored (sensitivity concern). Locked decision: visibility is **one switch**. If sensitivity is a concern, set visibility=public (which gitignores all of `.abcd/` including logbook). Per-subdirectory exceptions create maintenance burden and contradict the transparent-prompts principle ([`04-universal-patterns.md § 1`](04-universal-patterns.md#1-transparent-prompts)). The tracked-tier narrowing (²) is not a per-subdirectory carve-out of that decision: it is the mechanical boundary of what an ignore rule can do at all — the switch stays single, and where git already tracks the namespace the fence covers the one part git can still fence.

**Sensitivity concern still valid for `/abcd:launch` payload**: regardless of visibility, the launch payload manifest ([`../04-surfaces/04-launch.md § 2`](../04-surfaces/04-launch.md#2-curated-release-artefact-default-deny)) excludes `.abcd/` entirely from what ships in the curated release artifact. So a private repo that commits its logbook locally still doesn't leak it on launch.

In private repos, the entire `.abcd/` namespace is reproducible from a fresh clone — a clone carries the committed record, working files, and logbook (useful for diagnosing past command runs). The lifeboat is **not** part of what a clone carries: it is out-of-tree output at an operator-chosen destination, and embark reads it from there (adr-35).

**Memory locations to keep straight:**

abcd's curated memory exists at **two scopes** (per § The two `.abcd/` scopes), and there is one non-abcd memory location alongside it:

1. **`.abcd/memory/`** (repo scope) — the **primary** abcd memory: curated semantic summaries written by `dev-sync memory`, tracked in private repos, the canonical input for `principle-distiller`. Most memory lives here.
2. **`~/.abcd/memory/`** — **user-scope** memory: personal preferences and cross-project principles that have no single repo home.
3. **`memory/`** — legacy `cp -r` snapshot at the repo root that some existing projects maintain. abcd respects if present but doesn't write to it.

Which scope a curated page lands in is a routing decision — see [`07-memory.md`](07-memory.md) § scope routing. Retrieval across the two scopes is **not** a flat union (that would overflow context); it is keyword-recall + budget-bracketed injection per itd-39. When the brief says "memory" without qualification, it means the repo-scope `.abcd/memory/`.

**`.abcd/.work.local/` is local-only everywhere.** Working notes, drafts, status trackers stay gitignored. abcd consumes them via `dev-sync` ([§ 2](#2-abcdwork-namespace-and-dev-sync)) which promotes useful content into tracked `.abcd/work/` artefacts before disembark.

**AI transparency level** (separate axis from visibility — set by ahoy via `ai_transparency.level`):

| Level | Conversations (transcripts) | Plans | Tasks | Metadata (sessions, actions) |
|---|---|---|---|---|
| `full` | yes | yes | yes | yes |
| `metadata` | no | yes | yes | yes |
| `none` | no | no | no | no |

**Why separate from visibility:** a private repo might want `metadata`-only transparency to keep storage tight; a public OSS project might want `full` for credibility. Visibility decides what's *committed*; transparency decides what's *captured at all*.

Drives behaviour:
- **`dev-sync`** — `none` skips capture entirely; `metadata` skips conversation transcripts; `full` captures everything per source enable flags
- **`disembark`** — chat-distiller (Pass B) is no-op on `none`; runs on metadata/full
- **`launch` payload** — `ai_transparency` value carried into the public `marketplace.json` so consumers know what to expect

Sanitised export pattern (lifted from `~/.abcd/`'s `/export-transparency`): launch's pre-flight scrub strips absolute paths and PII regardless of transparency level. The transparency level determines what *exists*; pre-flight ensures what *ships* is sanitised.

**Visibility × transparency interaction (added post-audit 2026-05-07):** the two axes are independent. Any combination is valid: `private × none` (paranoid, no captures, no commit), `private × full` (everything captured locally, nothing public), `public × none` (public repo, no AI captures committed), `public × full` (everything captured AND committed — useful for OSS-credibility OSS projects). The launch payload's exclusion rules (per [`04-surfaces/04-launch.md § 2`](../04-surfaces/04-launch.md)) are unconditional regardless of `ai_transparency.level` — launch always ships the visibility-determined payload, and `ai_transparency` only governs what was captured in the first place.

## 2. `.abcd/work/` namespace and `dev-sync`

`.abcd/work/` is the **curated-from-volatile-sources** namespace. Volatile inputs (gitignored or external) get analysed and promoted into tracked `.abcd/work/` artefacts via `abcd dev-sync`. This solves three problems: noisy sources stay gitignored; curated lessons get tracked; abcd doesn't have to read volatile sources every time.

**Source → target table:**

| Volatile source (gitignored or external) | Curated abcd target (tracked in private repos) | Adapter |
|---|---|---|
| Agent memory (opt-in harvest per [`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching) — Claude Code: `~/.claude/projects/<encoded-cwd>/memory/`) | `.abcd/memory/` | memory harvest (`internal/core/memory`) |
| Ad-hoc reviews not tied to a spec (captured by whichever oracle adapter runs per [`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching) — e.g. RepoPrompt's local chat store; vendor paths in [`02-adapters.md`](02-adapters.md)). **Spec-tied reviews are written directly to the native spec review store at review time — `dev-sync reviews` does NOT sweep those.** | `.abcd/work/reviews/` | oracle-adapter capture (`internal/core/reviews`) |
| `.abcd/.work.local/issues.md` | `.abcd/work/issues/{open,resolved,wontfix}/iss-N-<slug>.md` (per itd-4) | workdir capture (`internal/core/workdir`; migration on first sync after install) |
| `.abcd/.work.local/notes/`, `.abcd/.work.local/<feature>/` | `.abcd/work/notes/` | workdir capture (`internal/core/workdir`) |
| RepoPrompt workspace state (opt-in adapter; vendor paths in [`02-adapters.md`](02-adapters.md)) | `.abcd/rp/workspace.json` (per itd-7) | RP workspace adapter (`internal/adapter/oracle`, opt-in) |

**`abcd dev-sync` triggers:**

- **Implicit:** `/abcd:disembark` Phase 0 runs `dev-sync` automatically (always-fresh-at-disembark)
- **Manual:** `abcd dev-sync` CLI for ad-hoc refresh

> **Open question (adr-35):** the implicit trigger above is stated against the old model, in which disembark packed *this* repo. adr-35 makes disembark **read-only against the source** (`abcd disembark <source-repo> to <dest>`; a test hashes the source tree before and after), while `dev-sync` **writes** into the source repo (`.abcd/memory/`, `.abcd/work/{reviews,issues,notes}/`, `.abcd/rp/`). The two cannot both hold. What must be decided: whether `dev-sync` is dropped from disembark's Phase 0 entirely and becomes a separate operator-run step (with disembark reading whatever curated artefacts happen to be present, and none at all in a repo abcd never touched — the primary case), or whether the implicit trigger survives only in a narrow "source repo is abcd-managed and the operator opted in" mode. adr-35 does not settle it.

Scheduled/cron sync **comes in a later phase** of the plugin (itd-13).

**Per-source on/off:**

- `.abcd/config.json` extends with `dev_sync.{reviews,memory,work,rp}.enabled = true|false`
- ahoy asks per-source enable (transparent prompts)
- defaults: all sources on for private, all sources off for public
- disembark Phase 0 honours per-source flags (subject to the open question above on whether that implicit trigger survives adr-35)

**Per-source provenance and curation rules:**

- **Memory (volatile) → `.abcd/memory/` (curated):** Source is an opt-in memory harvest per [`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching) — under Claude Code: `~/.claude/projects/<encoded-cwd>/memory/`. The repo-local legacy `memory/` snapshot (the `cp -r` pattern) is the workflow `dev-sync memory` replaces. Output is *not verbatim*: distilled summaries grouped by domain, written as actionable suggestions for future agents (e.g., "When implementing UI hit areas, always use `.contentShape(Rectangle())` — source: `feedback_hit_target_full_box`"). Why curated: raw memories grow unbounded and contain personal phrasing ("user got annoyed when X"). Inputs to `principle-distiller` (Pass C).

- **Reviews (volatile) → `.abcd/work/reviews/` (curated):** Reviews are captured by whichever oracle adapter runs per [`04-universal-patterns.md § 7`](04-universal-patterns.md#7-vendor-agnostic-adapters-with-environment-branching) — host-delegated by default ([adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)), with **RepoPrompt** as one opt-in adapter. When the RepoPrompt adapter is wired, `dev-sync reviews` harvests **ad-hoc oracle reviews not tied to a spec** from RepoPrompt's local chat store; **spec-tied reviews are NOT swept here** — the native spec review store captures them directly at review time. See [`02-adapters.md`](02-adapters.md) for the adapter's harvesting detail (vendor paths, the prompt-exports redirect, workspace matching, and the stability/privacy safeguards). `dev-sync reviews` renders its sources → `.abcd/work/reviews/oracle-{review,chat}-<timestamp>-<description>-<hash>.md` (the format `review-collator` consumes). Dedup by content hash; idempotent. Inputs to `review-collator` (Pass A).

- **`.abcd/.work.local/` (volatile, local-only) → `.abcd/work/issues/`, `.abcd/work/notes/` (curated):** `.abcd/.work.local/issues.md` (the abcd CLAUDE.md mandatory issue log) gets parsed entry-by-entry; each entry promoted to `.abcd/work/issues/open/iss-N-<slug>.md` (per itd-4 ledger structure). `.abcd/.work.local/notes/`, `.abcd/.work.local/<feature>/` get distilled into `.abcd/work/notes/`. Files in `.abcd/.work.local/` are never moved or deleted — `dev-sync work` is read-and-curate, source stays put. Inputs to `principle-distiller` (Pass C) and `chat-distiller` (Pass B, as auxiliary context).

- **RP workspace state (volatile) → `.abcd/rp/workspace.json` (curated, per itd-7):** The opt-in RepoPrompt adapter pulls RepoPrompt's own workspace state (the workspace whose root path matches the current repo) into `.abcd/rp/workspace.json`. The vendor filesystem layout and the match/normalisation mechanics live with the adapter — see [`02-adapters.md`](02-adapters.md). Workspace.json only for now; presets, mcp-routing scoping, `--preset <name>` flag, and `abcd rp link` window helper come in a later phase.

**Reviews as a first-class pitfall source:**

Plan/implementation/completion reviews (the ones in `.abcd/work/reviews/`) are *exceptionally* useful for spotting issues. The `review-collator` agent must extract every "P0 / P1 / watch out for X / found bug" finding as a candidate pitfall — **even when the original issue was fixed**, the lesson survives. Output:

- `reviews-consolidated.json` — full review summaries (existing)
- `candidate-pitfalls.json` — extracted findings ready for distiller dedup

`principle-distiller` (Pass C) has four pitfall sources to dedupe by topic-hash or canonical phrasing: source `memory/pitfalls.md` (or `.abcd/memory/pitfalls.md` after curation), `candidate-pitfalls.json`, Pass B chat-distiller deltas, and code-rescuer's `code-principles.json`.

**Distinct from `principles.json`:** `.abcd/memory/`, `.abcd/work/reviews/`, `.abcd/work/issues/`, `.abcd/work/notes/` are **persistent rolling artefacts** in the source repo, refreshed by `dev-sync`, used as ongoing input to future agents. `principles.json` is **per-disembark synthesis** written into the lifeboat at `<dest>/principles.json` — out-of-tree, at the operator-chosen destination, never back into the source repo (adr-35). The lifeboat consumes `.abcd/work/`; `.abcd/work/` is not the lifeboat.

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
│   │   └── …                           #   intent, capture, memory, lint, reflect, docfidelity,
│   │                                   #   render, schema, provenance, … — each returns structured results
│   ├── adapter/                        # the five seams — each: interface + native default + optional plug-in (adr-22)
│   │   ├── oracle/                     # host-delegated default; native | cli | api | mcp backends (adr-25)
│   │   ├── history/                    # native transcript store; specstory import (adr-29)
│   │   ├── spec/                       # native minimal store; the companion harness ccpm over conventions (adr-26)
│   │   ├── run/                        # thin native loop; Claude Workflows / the companion harness's loop (adr-27)
│   │   └── scanner/                    # native secret/PII scan; gitleaks / TruffleHog
│   ├── registry/                       # wired-adapter registry — resolves <seam>.backend to an implementation
│   └── surface/
│       ├── cli/                        # Cobra front door (ships in the MVP)
│       └── mcp/                        # MCP front door (later)
├── commands/                      # markdown command surfaces that shell to the binary — one file per verb, the
│   └── <verb>.md                       #   gated list in ../04-surfaces/README.md (abcd.md is the bare /abcd board)
│   # NOTE: `uninstall` is a sub-verb of /abcd:ahoy (not a standalone command). The ahoy command
│   # markdown dispatches the install/uninstall/dry-run/doctor/remote sub-verbs internally.
├── agents/                             # 15 agent prompts — see 01-agents.md (markdown, host-delegated)
│   ├── cold-reading-widening.md / cold-reading-entailment.md / cold-reading-comparative.md
│   ├── cold-reading-detection.md / docs-currency-reviewer.md / graveyard-interpreter.md
│   ├── intent-auditor.md / lifeboat-reviewer.md / press-release-composer.md
│   ├── principle-distiller.md / release-changelog-composer.md / ruthless-reviewer.md
│   ├── scribe.md / security-reviewer.md
│   └── sota-researcher.md              # plus per-agent fixtures/ dirs, README.md, CHANGELOG.md
└── hooks/                              # Claude Code event hooks — every event command runs through a self-provisioning shim
    ├── bootstrap.sh                    # builds/refreshes the plugin-root binary; referenced by every event command
    └── hooks.json                      # UserPromptSubmit → hook prompt-router; SessionStart → ONE chained command:
                                        #   bootstrap.sh, then session-start + prompt-router-reset, each fed a copy of the
                                        #   payload (siblings would run in parallel and share one stdin);
                                        # PreToolUse (matcher Bash) → guard hook; PreCompact → prompt-router-reset; SessionEnd → session-end.
                                        # The four non-SessionStart event shims also self-provision: when $CLAUDE_PLUGIN_ROOT/abcd
                                        # is missing they attempt hooks/bootstrap.sh (throttled by a .bootstrap.attempt marker
                                        # within a 10-minute window), then fall back to a PATH-resolved abcd — absolute, outside the
                                        # working directory, not world-writable, else ignored with a reason — before failing loudly
```

The core is organised one package per capability under `internal/core/`, and the
five adapter seams under `internal/adapter/`. [`02-adapters.md`](02-adapters.md)
owns the seam catalogue — each seam's interface, native default, and optional
external plug-in — so this brief does not restate it here.

**Plugin-internal development namespace** (committed in private repos, gitignored in public):

```
.abcd/
├── config.json                         # config + the `meta` setup block (schema_version, setup_version, ...)
├── config/                             # per-surface machine records: identity.json, launch-payload.json, version-location.json
├── corpus.json                         # validation-corpus manifest — a design target (itd-25); not yet in the tree
├── rules.json                          # per-repo override of plugin-bundled rule defaults (per itd-3)
├── development/                        # durable design record — flat by artefact type (per adr-30)
│   ├── personas.json                   # placeholder personas (Alice, Bob, Carol, ...)
│   ├── brief/
│   │   └── README.md                   # canonical, current-state (no archive — per adr-5)
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
│   └── research/
│       ├── notes/                      # dated investigation write-ups
│       └── prompting/                  # prompt R&D
│           ├── README.md
│           └── agents/<name>.md        # per-agent SOTA research (task #1 of each agent spec)
│   # NOTE: there is NO `development/voyage/` here. Voyage is user-scope and never committed —
│   #       ~/.abcd/voyage/<source-root-sha>/{disembark/history.jsonl, embark/provenance.json,
│   #       embark/from/<timestamp>/} (adr-35; see § The voyage store above).
├── work/                               # curated-from-volatile-sources (see § 2)
│   ├── reviews/                        # captured by the oracle adapter (RepoPrompt / codex / future) per 04-universal-patterns.md § 7
│   ├── issues/{open,resolved,wontfix}/ # iss-N-<slug>.md ledger entries (per itd-4)
│   └── notes/                          # distilled from .abcd/.work.local/notes/
├── memory/                             # curated memory artefact (memory harvest → .abcd/memory/, per 04-universal-patterns.md § 7)
├── logbook/                            # per-command / per-phase run logs (design target — no automatic session-log hook ships)
└── rp/                                 # RP workspace pull (per itd-7; opt-in RP adapter); workspace.json only for now
# NOTE: there is NO `.abcd/lifeboat/` here. The lifeboat is out-of-tree output at an operator-chosen
#       destination (`abcd disembark <source-repo> to <dest>`) — still the latest snapshot rather than an
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
└── explanation/                        # conceptual: lifeboats, dev-sync, intents, capture, etc.
```

**Doc framework note**: the native spec store, memory, and config live under `.abcd/`. Plugin-internal design docs live under `.abcd/development/`. User-facing docs live under `docs/`. We use the *shape* of a planning-vs-roadmap-vs-process split, not any one tool's *location* convention.
