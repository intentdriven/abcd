# `/abcd:ahoy` — Install / Update

Built on a **detect → contract → render/apply**
architecture: a single detection pass produces a versioned, machine-readable
state contract; every sub-verb is a thin consumer of it. Detection logic lives
in exactly one place so bare invocation, `dry-run`, `doctor`, and `install`
cannot drift apart.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `doctor` | — | shipped |
| `dry-run` | — | shipped |
| `identity-check` | — | shipped |
| `install` | — | shipped |
| `remote` | audit | shipped |
| `remote apply` | gate | shipped |
| `uninstall` | — | shipped |


Bare `/abcd:ahoy` shows read-only status only — never mutates state. The
`/abcd:ahoy` slash command dispatches every sub-verb (`status | install |
uninstall | doctor | dry-run | remote`), the write verbs included — each
announces that it writes before it runs — and each also ships on the CLI as
`abcd ahoy <sub-verb>`. Current sub-verbs:

- **`/abcd:ahoy install`** — install or update the plugin in this repo
  (idempotent; covers first-install and upgrade). Runs the detection pass, then
  the apply pass over the resulting gaps.
- **`/abcd:ahoy uninstall`** — reversible removal: removes the marker block,
  abcd's own `PATH` entry (if owned by this plugin) and the provenance record
  that proves that ownership. `--bin-dir` names the directory holding the entry,
  needed only when `install --bin-dir` put it somewhere not on `PATH`.
  NEVER mutates `hooks/hooks.json` (plugin-static per spc-14 T7 + spc-16 T1).
  Leaves `.abcd/` intact. Re-running `install` re-installs cleanly.
- **`/abcd:ahoy dry-run`** — run the detection pass and render the
  `DetectionResult` envelope as JSON (per spc-16 T1 — JSON ONLY; no
  unified-diff renderer). Never mutates state.
- **`/abcd:ahoy doctor`** — run the **full** detection pass plus a read-only
  audit pass, and report the gap counts for each. The text render shows the
  detection-gap and audit-gap counts; the JSON envelope carries full per-gap
  detail, including the user-scope state (history store, `index.json`, symlink,
  hook) that the shared detection pass validates on every sub-verb — doctor's
  distinct contribution is the read-only audit pass layered on top.
  Distinct from bare invocation, which shows the status board (folder kind,
  plugin-root status, root SHA, binary vintage/staleness, citations on a
  managed repo, gap count, guard health, and the banlist block).
- **`/abcd:ahoy remote`** — report the GitHub-native secret-scanning toggles on
  the repository this checkout's own origin remote names, and the changes an
  apply would make. Read-only: it writes nothing on the remote and nothing in
  the tree. A toggle it could not read reports `unknown`, never `disabled`.
- **`/abcd:ahoy remote apply`** — the one abcd verb that mutates state **outside
  this machine**: it enables GitHub's native secret scanning and then
  secret-scanning push protection (that order, because GitHub refuses push
  protection on a repository whose secret scanning is off), and mirrors the
  desired state into `.abcd/work/rulesets/repo-settings.json`. It answers to
  adr-44 and invariant 10 — no uninvited remote mutation, through a verb the user
  invokes AND confirms — with four gates that refuse rather than guess: the folder
  must be a repo abcd manages, the repository must be the one this checkout's
  origin names, the repo's config must not set `scan.native_secret_scanning` to
  `false`, and the caller must confirm the specific toggles named (an unanswered
  run declines; `--yes` is the explicit advance answer, and a run that changed
  nothing exits non-zero). Every request pins the API host explicitly, so an
  ambient `GH_HOST` cannot send the write to an endpoint the origin never named.
  The call goes through `gh`, so the
  write is made by the caller's own authenticated identity and abcd never holds
  a token. Idempotent: a repository already in the desired state takes no write,
  and a re-run rewrites nothing.
- **`abcd ahoy identity-check`** — exit non-zero if the git commit identity
  does not match `.abcd/config/identity.json` (the commit-identity gate).
  Read-only; a CLI-only operator/CI check with no slash-command surface.
- **Later phase: `/abcd:ahoy destroy`** — nuclear uninstall (per itd-10): removes
  `.abcd/` namespace too. Distinct from `uninstall`'s reversible behaviour.

## What abcd manages — repos and `~/.abcd/`

abcd manages exactly one kind of folder — a **repository** — and keeps one
user-scope directory for machine-local state:

```
~/.abcd/                       USER SCOPE — one per machine (machine-local state only)
  history/                       shared history store + index.json (identity/lineage,
                                 keyed on root-commit SHA — adr-29)
  config.json                    machine config.json defaults (a later phase)
  memory/                        user-scope memory (personal, cross-project — a later
                                 phase; the shipped store is repo-scope .abcd/memory/)

<anywhere>/<repo>/             REPO — a single repository (the only install target)
  .abcd/                         repo-scope record + config.json + rules.json
  CLAUDE.md                      marker block (stands alone)
```

`/abcd:ahoy`'s detection pass **classifies `cwd`** into one of three kinds
(`managed-repo` / `unmanaged-repo` / `unmanaged-folder` — see detection step 0
and itd-40), then `install` acts on the matching kind:

| Folder kind | What `install` does | `.abcd/` written |
|---|---|---|
| **`managed-repo`** | a git repo abcd already manages — run the repo install flow below (idempotent update) | repo-scope `.abcd/config.json` (with the `meta` setup block) + `.abcd/rules.json` per spc-16 T1 (no `meta.json` at repo scope; setup metadata lives at `config.json["meta"]`). Plus `.abcd/config/identity.json` when the git-identity pin is adopted (per iss-62 — a `config-change` write) and the CLAUDE.md/AGENTS.md marker block. See [`05-internals/03-configuration.md` § The two `.abcd/` scopes](../05-internals/03-configuration.md#the-two-abcd-scopes). |
| **`unmanaged-repo`** | a git repo with no abcd yet — bare `/abcd:ahoy` offers `install` to adopt it; `install` runs the same repo flow | same as `managed-repo` |
| **`unmanaged-folder`** | not a git repo — nothing to act on; reports and stops | none |

There is **no workspace, host, or development-environment layer**. A folder
like `~/ABCDevelopment/` is just where a user keeps repos — abcd does not
privilege it, and it groups nothing. abcd lives in **one repository**
([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)): the design
record is repo-scoped and in-tree. Everything genuinely machine-wide — the
`history/` store, plus machine config and user memory in a later phase — lives
under **`~/.abcd/`**, which abcd bootstraps once. The `.abcd/` namespace is scoped at two levels —
**user** (`~/.abcd/`) and **repo** (in-tree `.abcd/`). See
[`05-internals/03-configuration.md` § The two `.abcd/` scopes](../05-internals/03-configuration.md#the-two-abcd-scopes).

The history store is shared across every repo on the machine; its `index.json`
is the **sole user-scope registry** (identity and lineage keyed on each repo's
immutable root-commit SHA). An `install` that finds no `~/.abcd/` **bootstraps
it transparently** before registering — the user is never blocked by missing
user-scope state.

Each repo's CLAUDE.md/AGENTS.md marker block **stands alone** — there is no
repo→workspace inheritance chain.

## Architecture: detection pass + apply pass

`install`, `dry-run`, `doctor`, and bare `/abcd:ahoy` all run the **same
detection pass** and differ only in what they do with its output:

| Sub-verb | Detection | Then |
|---|---|---|
| bare `/abcd:ahoy` | full | render the status board: folder kind, plugin-root status, root SHA, vintage, staleness, citations (managed repo), gap count, guard health, banlist block — no gap detail |
| `doctor` | full | render detection- and audit-gap counts (full per-gap detail in the JSON envelope) |
| `dry-run` | full | render the canonical `DetectionResult` JSON envelope (per spc-16 T1 — no unified-diff) |
| `install` | full | run the apply pass over the gaps |

This means **idempotency is a property of detection, not of a version stamp.**
A check compares *actual state* (`.gitignore` block compare, `readlink`, file
presence, hook-config grep, `index.json` lookup) — never `setup_version` alone.
If a user hand-deletes the marker block while `setup_version` is current,
detection still reports it `missing` and `install` repairs it.

### The detection pass

Probes the folder, the repo, and `~/.abcd/`, produces an in-memory state contract (the
`ahoy-state.json` shape — see [`05-internals/03-configuration.md`](../05-internals/03-configuration.md)).
Steps, run in parallel where independent:

0. **Folder classification** (per itd-40) — classify `cwd` into one of three
   kinds. Classification keys on a **signal hierarchy** — abcd-owned markers
   decide managed-vs-unmanaged; the `.git/` signal only disambiguates shape:
   - **Strong (managed):** an entry for `cwd`'s root-commit SHA in
     `~/.abcd/history/index.json`, an in-tree `.abcd/` directory, or a
     CLAUDE.md/AGENTS.md abcd marker block.
   - **Weak:** a `.git/` directory means `cwd` is *a* repo — **not** that it is
     *managed*. Necessary, not sufficient.

   The three kinds (bare `/abcd:ahoy` offers `install` on an `unmanaged-repo`
   to adopt it, but reports-and-stops on an `unmanaged-folder`, so the two need
   distinct kind tokens):

   | Kind | Strong marker? | `.git/`? | `install` acts as |
   |---|---|---|---|
   | `managed-repo` | yes | yes | `repo` (idempotent update) |
   | `unmanaged-repo` | no | yes | `repo` (after `install` adopts it) |
   | `unmanaged-folder` | no | no | none — nothing to act on |

   Bare `/abcd:ahoy` **reports the kind and stops** — it never adopts an
   `unmanaged-repo`; it names `/abcd:ahoy install` as the way to do that.
   Identity resolution is a `~/.abcd/history/index.json` lookup keyed on the
   root-commit SHA — no hardcoded path, no parent-directory walk. The detection
   checks run on a repo kind; an `unmanaged-folder` short-circuits.
1. Resolve `${ABCD_PLUGIN_ROOT:-${CLAUDE_PLUGIN_ROOT}}`.
2. **Adapter probe** — the native secret + PII scan needs no external tool;
   this step only reports which **opt-in** scanners are on PATH: `gitleaks`
   (deeper secret scan) and `trufflehog` if `scan.deep` is (or would be)
   enabled.
3. **`.abcd/` skeleton** — presence of `config.json` (which carries the `meta` setup block) and `rules.json`.
4. **Identity** — `git rev-list --max-parents=0 HEAD` → root SHA. Cross-check
   against `~/.abcd/history/index.json`: is this SHA registered?
   Is there a sibling entry with a matching name/github but a *different* root
   SHA (the re-founding signal — see below)? Separately, compare the git author
   identity a commit would use against the committed `.abcd/config/identity.json`
   pin (iss-62), emitting a `git_identity.unpinned` / `.mismatch` / `.unset` /
   `.uncheckable` `config-change` gap.
5. **History-store wiring** — does the `~/.abcd/history/` store (abcd's native
   local redacted transcript corpus, per
   [adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)) exist at all
   (bootstrap gap if not)? Does the `<root-sha>/transcripts/` directory exist? Is
   the registered entry's `path` still accurate (mutable label — refresh if the
   repo moved)?
6. **Visibility state** — compare current `.gitignore` allowlist entries
   against the visibility table in
   [`05-internals/03-configuration.md § 1`](../05-internals/03-configuration.md#1-visibility-driven-gitignore-policy).
7. **Marker drift** — diff the existing CLAUDE.md/AGENTS.md marker block
   against the current template; classify `current` / `outdated` / `missing`.
   The marker block stands alone — there is no parent-`CLAUDE.md` reference to
   verify.
8. **PATH entry** — scan `PATH` for `abcd`, resolving symlinks, and classify each
   hit as this plugin's own entry, our dev shim, or a foreign binary. An
   abcd-owned entry anywhere on `PATH` is the install. With none, the default
   location `~/.local/bin/abcd` answers the same question — present, pointing at
   this plugin, at a different binary, or at nothing. An owned entry whose target
   has gone is `symlink.dangling`; an install directory absent from `PATH` is
   `path.bin_dir_not_on_path`, required but not resolvable — abcd prints the
   one-line `export` fix and never edits a shell profile; and any `abcd` that
   comes BEFORE abcd's own entry on `PATH` is `symlink.shadowed`, because an
   entry that is correct and never reached is not an install (iss-171). Install
   carries the two non-resolvable ones on its own result as notes: a fresh user
   cannot run `doctor` by name on a machine where abcd is not yet on `PATH`.
9. **Hook manifest verification** (verify-only per spc-16 T1) — VERIFY that
   `hooks/hooks.json` is present in the plugin install AND contains the three
   required event entries (`UserPromptSubmit`, `SessionStart`, `PreCompact`)
   each referencing the expected prompt-router hook commands. The shipped
   manifest also wires `abcd hook session-start` (chained after the bootstrap
   and ahead of `prompt-router-reset` inside the ONE `SessionStart` command —
   the harness runs sibling hooks in parallel, so the event carries a single
   entry), `abcd hook session-end` (a `SessionEnd` event), and `abcd guard
   hook` (a `PreToolUse` event, matcher `Bash`, that checks a shell command
   against the hazard registry before it runs) — five event types in all;
   verification covers only the three prompt-router commands above. Every
   event command is a self-provisioning shim, not a plain binary call: the
   non-SessionStart shims attempt `hooks/bootstrap.sh` when the plugin-root
   binary is missing (throttled by a `.bootstrap.attempt` marker within a
   10-minute window), then fall back to a PATH-resolved `abcd` before failing
   loudly. The PATH rung is OWNED-ONLY (GHSA-gx3m-3224-qqcv, CWE-426): it
   accepts only an absolute resolution out of a directory that is neither
   under the shim's working directory nor world-writable — the shapes the
   documented install never produces (iss-2609012039117381) — and only when
   `~/.abcd/path-entry` records that exact path as this machine's installed
   binary. The record is a `path=` string comparison and no hashing, because
   adr-46 keeps the fast path at one file test; the install one-liners and
   `ahoy install` both write it. Anything else is ignored with one line
   naming the binary and the reason, and the shim degrades — for the
   `PreToolUse` guard that is the UNGUARDED line and exit 1, never the exit 0
   the harness reads as an approval. SessionStart carries no PATH rung at all
   and fails closed. A missing or
   malformed manifest surfaces as a non-resolvable `plugin-owned` diagnostic
   gap. Neither install nor uninstall ever mutates `hooks.json` — the manifest
   is plugin-static per spc-14 T7.
10. **Version** — read `config.json["meta"].setup_version`; classify `first-time` /
    `upgrade` / `current`. One input among many — never the sole gate.
11. **Name-banlist scaffolding** (itd-74 / spc-20) — are the committed guard
    hooks (`.githooks/pre-commit` and `.githooks/pre-merge-commit`) present and
    abcd's own, does `.abcd/docs-lint.json` carry a banned-names family CI can
    actually enforce, and does the private stub exist in the gitignored local
    tier? Each absence is a `safe-autocreate` gap; every other state is a
    non-resolvable diagnostic, because abcd writes what is missing and never
    replaces what a maintainer put there. A hook present WITHOUT the
    `# abcd-name-guard: v1` line is foreign, and is reported rather than
    claimed as installed. A docs-lint config with no usable `banned_tokens`
    array, one that cannot be read, and one git IGNORES (so CI never sees it —
    the state `visibility: public` puts every repo in) are three distinct
    diagnostics with three distinct remedies. The private-stub gap is
    resolvable only when **git itself** reports the store's path as ignored
    (`git check-ignore`, not a comparison of `.gitignore` text): a stub git
    would track is the hazard the layer exists to prevent, so a gap apply would
    refuse to close is never advertised as resolvable. The same pass reports the
    layers' state (`banlist` in the detection envelope), and that report always
    carries the private layer's reach — CI cannot enforce it, and neither hook sees
    a fast-forward `git pull`, a rebase, a `git am`, a `git revert`, a cherry-pick,
    or a `--no-verify` commit. See
    [`20-banlist.md`](20-banlist.md).

Each detected discrepancy becomes a **gap** with a stable `id`, a `category`,
a `scope`, a `title`, `detail`, and a `fix_hint`. Gaps are grouped by category:

| `category` | Examples | Apply behaviour |
|---|---|---|
| `safe-autocreate` | `.abcd/` skeleton, history-store dirs, name-banlist artefacts (both guard hooks, the `.gitattributes` LF pin, public family, private stub) (`.abcd/bin/` scripts in a later phase) | apply once category approved, no per-item prompt; the banlist artefacts are create-if-absent and never overwrite |
| `config-change` | visibility, oracle adapter, PATH symlink, git-identity pin | transparent confirm; skip-if-set with "current value" notice |
| `plugin-owned` | CLAUDE.md/AGENTS.md marker block (per itd-3); `hooks/hooks.json` manifest verification (verify-only per spc-16 T1) | silent overwrite on marker drift; non-resolvable diagnostic for malformed/missing manifest |
| `dependency` | opt-in scanners: `gitleaks`, `trufflehog` install | offer brew with ONE category-level approval covering the opt-in scanners (per spc-16 T1 — per-CATEGORY, not per-tool) |
| `user-state` | `index.json` entry, re-founding, stale/duplicate entries | guided; never auto-edit user-scope app-state, report extras read-only |

### Templates as files

The marker block `install` writes comes from a canonical file under
`internal/core/ahoy/defaults/` (`claude-md-marker-block.md`), never from inline
prose in this surface. If the template is stale, edit the template file. Drift
detection (step 7) is only meaningful because the marker block has one
canonical source. The name guard `install` writes comes from the same directory
(`pre-commit`, `pre-merge-commit`) — the generalised form of the prototype this
repo runs on itself, with the repo-specific gates dropped, embedded so the binary
is self-contained. The `prepare-commit-msg` prompt that `install --attribution`
opts a repo into — asking every commit to declare whether a tool assisted it —
is the fourth file in that directory.
The `.abcd/rules.json` skeleton is written inline by the apply pass
(`stepRules`); a later phase moves it — and `.abcd/usage.md`, once that artefact
ships — to canonical files under `defaults/` too.

## Install flow (`/abcd:ahoy install`)

`install` = detection pass, then apply pass. The apply pass walks the gaps
grouped by category, asks **one** category-level approval question per
category present, and applies the approved categories' gaps.

Non-interactive override flags pre-answer the prompts: `--yes` approves every
resolvable category without prompting, `--adopt` / `--refuse-adopt` decide the
unmanaged-repo adoption question, `--docs-target` (`claude_md` | `agents_md` |
`both` | `skip`) sets the marker target, `--oracle-backend`
(`host-delegated` | `native` | `cli` | `api` | `mcp`) sets the oracle,
`--scan-deep` (`true` | `false`) toggles the deep scan, `--visibility`
(`private` | `public`) sets repo visibility, `--dev` selects track-latest
dogfood mode (the PATH entry rebuilds from the source tip on every call instead
of pinning the built binary), `--allow-stale-binary` proceeds even when the
running binary is stale against its source tip or of undeterminable vintage
(the default refuses before any write and names the rebuild fix), `--bin-dir`
names the directory for the `PATH` entry (default `~/.local/bin`, or an existing
abcd install adopted in place; a directory abcd cannot write refuses, because
abcd never escalates privileges), and `--attribution` opts the repo into the
committed `prepare-commit-msg` prompt asking every commit to declare whether a
tool assisted it — the choice is recorded, so a later install without the flag
keeps the hook. `--yes` does not adopt an
unmanaged repo or pin an unset git identity — those still need `--adopt` and an
answered prompt. The identity-pin exclusion is stated, never assumed: `--yes`
names it in its own help, the install envelope carries it as `optional_skipped`,
and the completion output prints it with the way to apply it (iss-166).

**Answers arrive from stdin whether or not stdin is a terminal** (iss-167).
The prompts are the same prompts; only the reader differs. At a terminal a human
types them; off one, a caller pipes them — `yes | abcd ahoy install` is how a
host agent drives the identity pin, the one approval no flag covers. Off a
terminal each answer is echoed to the diagnostic stream, so a piped run leaves a
transcript rather than a column of questions with no visible reply.

**The questions come in a fixed order**, held by `categoryPromptOrder`:
`dependency`, `safe-autocreate`, `config-change`, `user-state`, `plugin-owned` —
the order the apply pass acts in, with any later-added category appended sorted.
The order is a contract, not a presentation choice: answers are positional, so
without it the Nth piped answer approves a different category on every run — a
wrong answer that exits 0 and reads as a clean install. **One line answers one
question**, so a caller must supply one per category present; `yes` is the
reliable form precisely because it never runs out.

Answers that run out read as EOF, and EOF declines every confirm and takes the
default for every prompt: an unattended run with nothing on stdin still adopts
nothing it was not told to adopt. The cost of reading a non-terminal stdin is
that a stdin held OPEN and silent makes a prompt wait rather than decline — the
same contract every prompting CLI has. A run that must neither block nor prompt
closes stdin and pre-answers: `abcd ahoy install --yes --refuse-adopt
< /dev/null`.

1. Run the detection pass (above).
2. **Dependency gaps** (`dependency`) — surface brew/pip commands for missing
   tools with ONE category-level approval (per spc-16 T1 — per-category, not
   per-tool). spc-16 NEVER auto-executes package managers; the user runs the
   install commands manually.
3. **Skeleton gaps** (`safe-autocreate`) — the apply pass creates `.abcd/` and
   writes `config.json` (idempotent merge; the `meta` setup block is a key
   within it). Create history-store dirs (see step 6). Later phase: create
   `.abcd/bin/` (only with `--with-scripts`; scripts copied with `chmod +x`)
   and write `.abcd/usage.md`.
4. **Config gaps** (`config-change`) — transparent prompts; each skips with a
   "current value" notice if the key is already set:
   - **Visibility** (private/public) — always re-confirms, shows current state
     and consequences (which directories will be tracked/untracked).
   - **If private:** `scan.deep` — probe `trufflehog`, offer install if missing.
   - **Docs target** (`CLAUDE.md` / `AGENTS.md` / Both / Skip).
   - **Oracle** — host-delegated by default (abcd hands prompts to the host's
     subagent dispatch); an opt-in oracle adapter (native / CLI / API / MCP)
     can be wired instead (per
     [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)).
5. **Apply visibility** — add/remove `.gitignore` allowlist entries to match
   the visibility table in
   [`05-internals/03-configuration.md § 1`](../05-internals/03-configuration.md#1-visibility-driven-gitignore-policy).
   Show the resulting tracked-vs-ignored list before confirming.
6. **Name-banlist scaffolding** (`safe-autocreate`, itd-74 / spc-20) — write the
   five artefacts: the committed guard hooks (`.githooks/pre-commit` and
   `.githooks/pre-merge-commit`, because git runs no pre-commit for a merge), the
   appended `.gitattributes` line keeping them at LF, a `.abcd/docs-lint.json`
   carrying an EMPTY public banned-names family, and the documented private stub in
   the gitignored local tier. Every write is create-if-absent AND contained: paths
   resolve through an `os.Root` opened at the repo, so a symlink committed at
   `.githooks` or at the local tier cannot land an artefact outside it. A hook, a
   CI-gating config, and above all a populated private store are the maintainer's.
   The stub runs AFTER step 5 and only where `git check-ignore` reports the path
   as ignored, so a stub git would track is never created. Its examples are
   commented out and every illustrative value is a reserved documentation value or
   a persona-derived fixture host (`examples-use-reserved-identifiers`). A clone
   arms the hooks once with `git config core.hooksPath .githooks`; abcd never sets
   it, and no surface reports a committed hook as a running one.
7. **History-store registry** (`user-state`) — if the `~/.abcd/history/` store
   does not exist, bootstrap it: create the directory and write an initial
   `index.json` with its `schema` + `description` header (see
   [`05-internals/03-configuration.md`](../05-internals/03-configuration.md)
   for the schema). Then create the `~/.abcd/history/<root-sha>/transcripts/`
   transcript directory (abcd's native local redacted transcript corpus, per
   [adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)), write the
   per-repo `<root-sha>/meta.json` (`root_commit`, `name`, `github`, and a
   `corpus` block pointing at `transcripts/`), and register the repo in
   `index.json` by its immutable `root_commit`, refreshing the entry's mutable
   `path` if the repo moved. A remote URL recorded in either file carries no
   credential: it is scrubbed where the identity is derived, the index is
   scrubbed again as it is LOADED — so every rewrite drops a credential from
   every entry, not only from the one being registered — and a `meta.json`,
   which is otherwise written once and never revisited, is rewritten in place
   when it holds one. A store that already holds a credential raises
   `history.credential_at_rest`, so an otherwise up-to-date repo does not
   short-circuit past the heal. The history `index.json` is the
   **sole user-scope registry** — there is no `workspaces.json`. Transcript
   capture is native — no external tool and no per-repo redirect shim.
8. **Marker block** (`plugin-owned`) — inject/refresh the block between
   `<!-- BEGIN ABCD -->` / `<!-- END ABCD -->` in the target docs. **Silent
   overwrite on drift, per itd-3 — no prompt for hand-edits inside the block;
   users edit outside the markers.** Content comes from
   `internal/core/ahoy/defaults/claude-md-marker-block.md`. Write the minimal
   `.abcd/rules.json` skeleton if missing.
9. **PATH entry** (`config-change`) — transparent prompt: "Install `abcd`
   symlink to `~/.local/bin/abcd`? Default: yes for private repos, no for
   public." An abcd-owned entry already on `PATH` is adopted where it stands
   rather than duplicated; `--bin-dir <dir>` names a different directory (the
   only route to a system-wide one) and fails loudly when it is not writable.
   abcd NEVER escalates privileges. If accepted AND the target is absent or
   already points at this plugin → write it, provided the binary it would point
   at exists; a link to a missing target is refused, because a dangling `abcd`
   early on `PATH` shadows every working one behind it. If a different `abcd`
   binary exists → refuse, show what it points to, suggest manual resolution.
10. **Hook registration** (`plugin-owned`, VERIFY-ONLY per spc-16 T1) — install
    verifies that `hooks/hooks.json` is present (the manifest is plugin-static
    per spc-14 T7). Install NEVER writes `hooks.json`; uninstall NEVER mutates
    `hooks.json`. A missing manifest surfaces as a `plugin-owned` non-resolvable
    diagnostic gap.
11. Stamp `setup_version` + `setup_date` in `config.json["meta"]`.
12. Print summary: installed files, config table, symlink status, hook status,
    notes (re-run hint, uninstall hint).

**Same-version re-install:** if the detection pass reports zero actionable
gaps (gaps both required and resolvable), `install` prints "already up to
date" and exits without writing. This falls out of
detection naturally — it is not a version-stamp short-circuit.

**Mid-run changes (a later phase):** accept `abcd config set <key> <value>`
inline; re-run only the affected detection check and its apply step.

## Re-founding (the `supersedes` flow)

When a repo is re-created with clean history — typically to strip in-repo
transcripts before sharing — it is genuinely a new repo with a new root SHA.
The detection pass (step 4) flags this as a `user-state` gap when the current
root SHA is absent from `index.json` **and** a sibling entry has a matching
name/github under a different SHA.

ahoy never auto-decides this. It surfaces the candidate predecessor and asks
the user to confirm. On confirmation, the apply pass:

1. Registers the new root SHA in `index.json`.
2. Sets the new entry's `supersedes` → old SHA.
3. Marks the old entry's `superseded_by` → new SHA.
4. Leaves the old repo's corpus in place under its own `<root-sha>/` dir for
   lifeboat review — nothing is moved or deleted.

If the user declines, ahoy registers the new SHA with no lineage link and
notes the orphaned-predecessor possibility in the summary.

## Sub-verb semantics

**Uninstall (`/abcd:ahoy uninstall`):** removes the BEGIN/END marker block from
CLAUDE.md/AGENTS.md, abcd's own `PATH` entry (`~/.local/bin/abcd`, or wherever
on `PATH` it sits) **if it points at this plugin** (otherwise leave it alone),
and the provenance record by which that ownership is proven; `--bin-dir` names
the directory holding the entry when `install --bin-dir` put it somewhere not on
`PATH`. `hooks/hooks.json` is plugin-static per
spc-14 T7 — uninstall NEVER mutates it (per spc-16 T1 brief amendment).
**Leaves the entire `.abcd/` namespace intact** (`config/`, `config.json`,
`rules.json`, `development/`, `work/`, `memory/`) and the history store. Deeper
removal (`/abcd:ahoy destroy`) comes in a later phase as itd-10.

**Uninstall ↔ install is a tested round-trip invariant:** after `uninstall`
then `install`, the detection pass must report zero actionable gaps, and the resulting
state must be byte-identical to a fresh install modulo `setup_date`. This is an
acceptance criterion, not just prose — see § Acceptance.

**Dry-run (`/abcd:ahoy dry-run`):** runs the detection pass, then renders the
canonical `DetectionResult` envelope as JSON (per spc-16 T1 — JSON ONLY; no
unified-diff renderer in this command surface). Exits without writing. The
JSON envelope shape is `{folder_kind, adopted, root_sha,
plugin_root_status, repo_identity, signals, guard, banlist, gaps}` (the `guard` key
reports the guard-hook health: `plugin_root_resolved`, `hook_installed`,
`binary_reachable`, `registry_loadable`, `disabled`, `detail`, and `entries`) so the plugin command
(`commands/ahoy.md`) summarises state off `folder_kind` + `gaps` and
names `abcd ahoy install` for anything actionable.

**Doctor (`/abcd:ahoy doctor`):** runs the detection pass plus a read-only
audit pass. The text render shows two counts — detection gaps and audit gaps —
and then names, one line each, every required gap that is NOT resolvable: those
are the ones no later `install` will clear (a `config.json` abcd refuses to touch
until a human repairs it is the case that matters), so a bare count of them
would be a number the reader cannot act on. The JSON envelope (`{detection, audit_gaps}`) carries full per-gap detail: the
detection gaps cover user-state checks (the history store exists and is
writable, the `index.json` entry matches this root SHA, the PATH symlink and
hook manifest are intact), while the audit pass reconciles the registered
history-store `path` against `index.json` read-only (a stale registered path).
The check users reach for after a repo rename, a machine migration, or "why
aren't my transcripts showing up." Never mutates state, never auto-fixes
user-scope app-state.

## Acceptance

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:ahoy`,
  **then** the dispatcher runs the detection pass and shows the status board —
  folder kind, plugin-root status, root SHA, vintage, staleness, citations (on
  a managed repo), gap count, guard health, and the banlist block — plus a
  next-step line on the unmanaged kinds (naming `/abcd:ahoy install` on an
  `unmanaged-repo`, "nothing to act on" on an `unmanaged-folder`); a
  `managed-repo` prints the status board with no next-step line — and never
  mutates state.
- **Given** a fresh repo with no `.abcd/` directory, **when** `/abcd:ahoy
  install` runs to completion, **then** the two-file repo carve-out
  (`.abcd/config.json` + `.abcd/rules.json` per spc-16 T1) is written,
  `.abcd/config/identity.json` is pinned when the git-identity gate is adopted
  (per iss-62), the
  visibility-driven `.gitignore` allowlist entries are present, the
  history-store dirs + `index.json` entry are present, the
  CLAUDE.md/AGENTS.md marker block from
  `internal/core/ahoy/defaults/claude-md-marker-block.md` is installed, and the
  `hooks/hooks.json` check runs verify-only (never mutated) — the manifest is
  plugin-static per spc-14 T7, and a missing or malformed manifest surfaces as
  a non-resolvable `plugin-owned` diagnostic gap.
- **Given** a repo with `install` already run, **when** `/abcd:ahoy install`
  runs again with no state changes, **then** the detection pass reports zero
  actionable gaps, the message reads "already up to date", and nothing is written.
- **Given** a repo where the marker block was hand-deleted but `setup_version`
  is current, **when** `/abcd:ahoy install` runs, **then** detection reports the
  marker `missing` and the apply pass restores it — idempotency keys off state,
  not the version stamp.
- **Given** a repo with `install` run at an older `setup_version`, **when**
  `/abcd:ahoy install` runs, **then** `setup_version` is updated, the marker
  block is refreshed, and existing config keys are preserved (skip-if-set).
- **Given** install is running and an opt-in scanner (`gitleaks` / `trufflehog`)
  is not on PATH, **when** the dependency category is approved, **then** the user
  is shown the brew install commands with ONE category-level approval (per spc-16
  T1 — per-category, not per-tool); spc-16 NEVER auto-executes package managers.
- **Given** no oracle adapter is wired, **when** the detection pass resolves the
  oracle, **then** `oracle` stays **host-delegated** (abcd needs no API keys or
  model config — it emits prompts the host runs, per adr-25), and an opt-in
  adapter (native / CLI / API / MCP) can be configured later.
- **Given** a repo whose root SHA is absent from `index.json` while a sibling
  entry matches its name/github, **when** `/abcd:ahoy install` runs, **then**
  detection flags a re-founding candidate, ahoy asks before linking, and on
  confirmation sets `supersedes` / `superseded_by` and leaves both corpora in
  place.
- **Given** the user runs `/abcd:ahoy uninstall` then `/abcd:ahoy install`,
  **when** both complete, **then** the detection pass reports zero actionable
  gaps and the resulting state is byte-identical to a fresh install modulo
  `setup_date`.
- **Given** the user runs `/abcd:ahoy dry-run`, **when** the command completes,
  **then** the detection pass runs, the canonical `DetectionResult` JSON
  envelope (`{folder_kind, adopted, root_sha,
  plugin_root_status, repo_identity, signals, guard, banlist, gaps}` per spc-16 T1) is printed
  to stdout, and no files are modified.
- **Given** the user runs `/abcd:ahoy doctor` on an installed repo whose
  registered history-store `path` no longer matches `index.json`, **then** an
  audit gap citing both paths appears under `audit_gaps` in the JSON envelope,
  reported read-only, and no files are modified.
- **Given** a fresh machine with no `~/.abcd/`, **when** `/abcd:ahoy install`
  runs in a repo, **then** `~/.abcd/` is bootstrapped — the `history/` store
  (directory + `index.json` with `schema`/`description` header) — before the
  repo is registered, so the user is not blocked by missing user-scope state.
- **Given** a registered repo that has been moved on disk, **when**
  `/abcd:ahoy install` or `doctor` runs, **then** detection notices the stale
  `index.json` `path` and `install` refreshes it (the root SHA is unchanged, so
  the entry is updated, not duplicated).
- **Given** a git repository with no abcd markers and no `~/.abcd/history/index.json`
  entry for its root-commit SHA, **when** the user runs bare `/abcd:ahoy`,
  **then** it reports `unmanaged-repo`, names `/abcd:ahoy install` as the way to
  adopt it, and mutates nothing — bare invocation never adopts.
- **Given** a folder that is not a git repository and has no abcd markers,
  **when** the user runs bare `/abcd:ahoy`, **then** it reports
  `unmanaged-folder`, reports there is nothing to act on, and mutates nothing.
