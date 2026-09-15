---
id: itd-7
slug: rp-workspace-portability
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Lifeboats Carry RepoPrompt Workspace Definitions Forward

## Press Release

> **When the RepoPrompt adapter is configured, abcd packs the RP workspace definition into the lifeboat.** When `dev-sync` runs (and as part of `/abcd:disembark`), abcd reads the project's RepoPrompt workspace definition from `~/Library/Application Support/RepoPrompt/Workspaces/` and writes it into `.abcd/rp/workspace.json` in the repo. On `/abcd:embark`, the workspace is read back and offered for registration with RepoPrompt on the new machine. For personas who drive RepoPrompt, migrating to a fresh user account no longer means rebuilding workspace boundaries by hand for every active project.
>
> "I switched user accounts and dreaded rebuilding RP workspaces for a dozen active projects from memory," said Frank, DevOps engineer. "abcd's workspace pull meant each project's `.abcd/rp/workspace.json` came along with the lifeboat. Embark on the new account asked whether to register each one. Thirty seconds per project instead of half an hour."

## Why This Matters

The RepoPrompt adapter already pulls **review content** from RP's Application Support. That's the lifeboat-valuable *output* of RP. The parallel state that's *also* project-shaped and lifeboat-valuable is the **workspace definition** — which files RP considers part of the project's context, the partition strategy, the project boundaries. Workspace definitions are fiddly to recreate from memory; they encode real curation work.

All of this is scoped to the RepoPrompt adapter: it runs only when that adapter is configured, and every surface degrades gracefully when RP is absent. RP is one optional adapter, never a hard dependency — abcd assumes nothing about it being present.

This intent ships workspace pull only. Presets and routing are global RP state with cross-project complications; they need a richer scoping design before shipping. `--preset` and `abcd rp link` are convenience layers that need workspace pull as a foundation and are out of scope here.

See [`research/legacy-harvest.md`](../../research/legacy-harvest.md) for the broader v0 cleanup context.

## What's In Scope

- **A RepoPrompt workspace-state adapter** — new adapter in the native adapter layer. Reads the workspace whose root path matches the current repo from `~/Library/Application Support/RepoPrompt/Workspaces/`. Workspace identity match: walk all workspaces, parse each `workspace.json`'s root-path field, match against `git rev-parse --show-toplevel`. Multi-match: prefer the most-recently-modified.
- **Pull on `dev-sync`**: writes `.abcd/rp/workspace.json` with `~/`-relative path normalisation (per privacy rules). Visibility-driven gitignore handling: private repos commit; public repos exclude.
- **Pack on `/abcd:disembark`**: dev-sync runs as a pre-pack step (already in brief). Workspace state goes into the lifeboat naturally.
- **Unpack + reconnect on `/abcd:embark`**:
  - `workspace.json` is read; user is asked "register this workspace with RepoPrompt now?" — if yes, abcd writes a workspace entry under `~/Library/Application Support/RepoPrompt/Workspaces/` on the embarking machine.
  - If RP isn't installed: warn gracefully ("RepoPrompt not detected; .abcd/rp/workspace.json preserved for later sync") and continue without failing.
- **Schema validation** for `workspace.json`. The JSON Schema ships with the RP adapter in the Go binary (`internal/core/...`); `abcd dev-sync` validates against it.
- **PII scrub on read**: workspace files may include absolute paths (`/Users/<username>/...`). Pre-commit scrub rewrites to `~/`-relative form. Same approach as session-log schema.

## What's Out of Scope

- **Preset pull** — `Presets/` directory. Presets are global in RP; copying all into every repo's `.abcd/rp/presets/` risks cross-project leakage. Explicit scoping (tag-based, or per-project preset list) is needed before this is shipped.
- **MCP routing scoping** — `mcp-routing.json` carries 44 KB of entries across many contexts. This intent does not try to extract project-scoped subsets; that depends on a settled scoping rule.
- **`--preset <name>` selection** — runtime preset selection at task time. Depends on preset pull; out of scope with it.
- **`abcd rp link` window helper** — workspace-window linking via RP MCP. Workspace pull covers the migration use case; window-linking is a convenience layer for a follow-up intent.
- **Active workspace monitoring** — watching for changes in RP's Application Support and auto-syncing to `.abcd/rp/`. Dev-sync-triggered only.
- **Auto-attach RP window to current repo session** — beyond `abcd rp link` reporting the window ID, automatic window selection / activation.
- **Per-command preset binding** — declaring in `.abcd/config.json` "this preset for review-collator runs, that preset for press-release-composer runs."
- **Backend switching mid-session** — switching the backend RP routes to (Claude Code vs local model vs Codex) without leaving the abcd command.
- **Cross-tool portability** — pulling equivalent state from another harness's workspace tooling. Couples to [`itd-22-harness-portability`][itd-22].
- **CodeMapCaches / Partitions / windowSessions / Settings** — pure tool state, not project-shaped. Stays in RP's Application Support; not pulled.
- **Workspace partitioning auto-derivation** — abcd inferring an RP partition strategy from `.gitignore` + repo structure if no workspace exists. Heuristic; out of scope.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. This acceptance block uses Given-When-Then directly._

- **Given** an abcd repo with an existing RP workspace defined in `~/Library/Application Support/RepoPrompt/Workspaces/` whose root matches the repo, **when** `abcd dev-sync` runs, **then** `.abcd/rp/workspace.json` exists and matches the source workspace definition (with `~/`-relative path normalisation applied).
- **Given** an abcd repo with no matching RP workspace, **when** `abcd dev-sync` runs, **then** the command logs a one-line note ("no RP workspace matches this repo") and continues without failing; no `.abcd/rp/workspace.json` is created.
- **Given** an abcd repo with multiple matching RP workspaces, **when** `abcd dev-sync` runs, **then** the most-recently-modified workspace is selected and `.abcd/rp/workspace.json` is written from it.
- **Given** a public repo (per brief visibility setting), **when** dev-sync runs, **then** `.abcd/rp/` is added to `.gitignore` and not committed.
- **Given** a private repo, **when** dev-sync runs, **then** `.abcd/rp/` is committed (and visible in `git status`).
- **Given** a lifeboat containing `.abcd/rp/workspace.json`, **when** `/abcd:embark from <path>` runs on a fresh account/machine with RP installed, **then** the user is interactively asked whether to register the workspace; if yes, the workspace is written to RP's Application Support and registered.
- **Given** the embarker has no RP installed, **when** embark encounters `.abcd/rp/`, **then** the command warns gracefully ("RepoPrompt not detected; .abcd/rp/workspace.json preserved for later sync") and continues without failing.
- **Given** an RP workspace whose `workspace.json` contains absolute paths under `/Users/<username>/`, **when** `abcd dev-sync` reads it, **then** the written `.abcd/rp/workspace.json` has those paths rewritten to `~/`-relative form.

## Open Questions

- **Workspace identity disambiguation when multiple match.** Most-recently-modified is the proposed default; should the user be prompted instead? Probably no (keep dev-sync non-interactive), revisit if the heuristic misfires.
- **Provenance field**: should `.abcd/rp/workspace.json` carry a `_source` field noting which RP install / Library version produced it? Helps debugging cross-version issues. Recommended yes; confirm during implementation.
- **Conflict resolution on embark**: if the embarker has a workspace with a clashing UUID (unlikely but possible), what's the policy? Probably: rename the embarked one with a suffix (`<name>-from-lifeboat`).

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

[itd-22]: ../drafts/itd-22-harness-portability.md "itd-22 — harness portability"
[itd-6]: ../superseded/itd-6-rp-mcp-only-integration.md "itd-6 — RP-only integration via MCP"
