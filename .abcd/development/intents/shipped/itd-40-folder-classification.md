---
id: itd-40
slug: folder-classification
spec_id: spc-5
kind: standalone
suggested_kind: standalone
reclassification_history: []
severity: major
impact: additive
---

# abcd Knows What Kind of Folder You're In

## Press Release

> **abcd recognises what kind of folder you ran it in and tracks every repo it manages in one place.** Run `/abcd:ahoy` anywhere and it tells you plainly: this is a repo abcd manages, a git repo abcd hasn't adopted yet, or just a folder. You never declare a "central location" or wire up paths — abcd keeps a single machine-wide registry, the history store's `index.json` keyed on each repo's root-commit SHA, and bootstraps it silently the first time you install.
>
> "I used to keep a mental map of which folders had abcd set up and which didn't," said Bob, staff engineer. "Now I just run `/abcd:ahoy` and it tells me. When I cloned a new repo into my apps folder it said 'unmanaged git repo — run install to adopt,' and when I ran it in a plain folder it just said there was nothing to manage. I never typed a path."

## Why This Matters

`/abcd:ahoy` sets up abcd in a folder — but "a folder" is not one shape. It might be a repo abcd already manages, a git repo it hasn't adopted, or a plain folder that is nothing to act on. Earlier drafts asked the persona to understand a multi-scope shape (machine-wide / workspace / repo) with a layer that grouped several repos together. abcd lives in **one repository** (adr-28): there is no such grouping layer. The only scopes are **user** (`~/.abcd/`) and **repo** (in-tree `.abcd/`), and abcd bootstraps the user scope silently.

This intent makes `/abcd:ahoy` classify the folder reliably, report it plainly, and track every managed repo in one machine-wide registry — the history store's `index.json`, keyed on each repo's immutable root-commit SHA — so abcd never has to guess or ask the persona for a path.

## What's In Scope

- **Folder classification** — bare `/abcd:ahoy` classifies `cwd` into one of three kinds and **reports only** (never mutates):
  - `managed-repo` — an abcd-managed repository
  - `unmanaged-repo` — a git repository with no abcd markers; reachable via `/abcd:ahoy install`
  - `unmanaged-folder` — not a git repository; nothing to act on
- **Signal hierarchy** — classification keys on abcd-owned markers first, never on weak signals alone:
  - **Strong:** a CLAUDE.md/AGENTS.md marker block, an in-tree `.abcd/` directory, or an entry for `cwd`'s root-commit SHA in `~/.abcd/history/index.json`.
  - **Weak:** a `.git/` directory tells abcd it's *a* repo, not that it's *managed* — necessary, not sufficient.
- **The history-store registry** — `~/.abcd/history/index.json` is the sole user-scope registry: it records every repo abcd manages, keyed on the immutable root-commit SHA, with `name`, `github`, and `path` as mutable labels ahoy refreshes each run. Central-location discovery becomes a lookup in this file — no hardcoded path, no env var, no parent-directory walk.
- **First-run bootstrap** — if `~/.abcd/history/` does not exist, the first `/abcd:ahoy install` creates it (directory + `index.json`) before registering the repo.

## What's Out of Scope

- **Auto-adoption** — bare `/abcd:ahoy` never adopts an unmanaged repo. It classifies and reports; adoption is always the explicit `install` sub-verb.
- **A grouping layer** — abcd is one repository (adr-28); it does not group repos, and there is no `workspaces.json`. A folder that merely holds several repos is an `unmanaged-folder`, not a managed grouping.
- **Duplicating identity** — the history `index.json` is the single registry; root-SHA identity and lineage live there directly. See `05-internals/03-configuration.md`.
- **Classifying non-local folders** — remote repos, URLs, and not-yet-cloned projects are out of scope; classification is of a local `cwd`.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** a folder with an abcd CLAUDE.md marker block and an `.abcd/` directory, **when** the persona runs bare `/abcd:ahoy`, **then** it reports `managed-repo` and mutates nothing.
- **Given** a git repository with no abcd markers, **when** the persona runs bare `/abcd:ahoy`, **then** it reports `unmanaged-repo` and names `/abcd:ahoy install` as the way to adopt it — without adopting it.
- **Given** a folder that is not a git repository and has no abcd markers, **when** the persona runs bare `/abcd:ahoy`, **then** it reports `unmanaged-folder`, reports there is nothing to act on, and mutates nothing.
- **Given** no `~/.abcd/history/` store exists, **when** the first `/abcd:ahoy install` runs, **then** the store is bootstrapped (directory + `index.json`) and the repo is registered in it by its root-commit SHA.
- **Given** a managed repo registered in the history `index.json`, **when** abcd needs the central/host location, **then** it is resolved by reading `index.json` — not by a hardcoded path or directory walk.

## Open Questions

- Should `/abcd:ahoy doctor` reconcile the history `index.json` against the filesystem (flagging registered-but-missing or present-but-unregistered repos)?
- When a classified `unmanaged-repo` is discovered, should the report suggest `install` inline, or stay purely descriptive?

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-a3791f7dde2e -->
Fidelity review — receipt rcp-a3791f7dde2e (verifier intent-fidelity-reviewer claude-opus-4-8).

Provenance: intent-fidelity-reviewer@claude-opus-4-8 · rubric_hash sha256:95792472ae74ca0469f69a51c618946e0d33cb1380032460099ed4b469d67e86 · prompt_hash sha256:f16ea4bd3b8d426558e846d9c6f19c445890c4b1bbe91b765fcc11f2efd4fe2a
Input attestations: diff:7933dc25ca501b9935b9fd22135894f47aea8ae8@-;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: A strong marker (.abcd/ dir or CLAUDE.md/AGENTS.md block or index registration) classifies as managed-repo, and bare ahoy runs the read-only DryRun path, so it mutates nothing; both halves are covered by engine tests.
  evidence: internal/core/ahoy/detect.go:106 — "return ManagedRepo, signals"
  evidence: internal/core/ahoy/detect_test.go:68 — "func TestClassifyManagedRepoByAbcdDir(t *testing.T) {"
  evidence: internal/core/ahoy/detect.go:78 — "// DryRun runs Detect and returns the envelope (Adopted=nil). Zero writes."
- ac-2 — MET: The bare-ahoy render for UnmanagedRepo now names the adoption verb, and the watched-fail test asserts both the `/abcd:ahoy install` hint and that no .abcd/ or CLAUDE.md is written (no adoption).
  evidence: internal/surface/cli/cli.go:1295 — "unmanaged git repo — run `/abcd:ahoy install` to adopt it."
  evidence: internal/surface/cli/cli_test.go:561 — "if !strings.Contains(out, \"/abcd:ahoy install\") {"
  evidence: internal/surface/cli/cli_test.go:566 — "bare ahoy mutated the repo (.abcd/ appeared)"
- ac-3 — MET: The render for UnmanagedFolder emits the nothing-to-act-on line, the CLI test asserts it, and the engine short-circuits a non-git folder to zero gaps via the read-only DryRun (mutates nothing).
  evidence: internal/surface/cli/cli.go:1297 — "not a git repository — nothing to act on."
  evidence: internal/surface/cli/cli_test.go:585 — "if !strings.Contains(out, \"nothing to act on\") {"
  evidence: internal/core/ahoy/detect_test.go:125 — "func TestDetectUnmanagedFolderShortCircuits(t *testing.T) {"
- ac-4 — MET: The characterization test starts with no ~/.abcd/history/ store, runs the first install, then reads the bootstrapped index.json and confirms the repo is registered under its real root-commit SHA; the test passes at HEAD.
  evidence: internal/surface/cli/cli_test.go:625 — "if _, err := os.Stat(indexPath); !os.IsNotExist(err) {"
  evidence: internal/surface/cli/cli_test.go:638 — "if r.RootCommit == rootSHA {"
  evidence: internal/surface/cli/cli_test.go:647 — "repo not registered by root-commit SHA %q in index.json"
- ac-5 — MET: The doctor test corrupts only the registered `path` in index.json and then observes doctor raise history.path_stale quoting that exact value, proving the central/host location is resolved by reading index.json rather than a hardcoded path or directory walk.
  evidence: internal/surface/cli/cli_test.go:682 — "repos[0].(map[string]any)[\"path\"] = \"/somewhere/relocated\""
  evidence: internal/surface/cli/cli_test.go:700 — "if g.ID == \"history.path_stale\" {"
  evidence: internal/surface/cli/cli_test.go:702 — "if !strings.Contains(g.Detail, \"/somewhere/relocated\") {"

Gap audit:
- honoured:
  - Bare /abcd:ahoy classifies cwd into managed-repo / unmanaged-repo / unmanaged-folder on a strong-signal-first hierarchy and reports only (read-only).
    evidence: internal/core/ahoy/detect.go:103 — "strong := registered || abcdDir || markerFired"
    evidence: internal/surface/cli/cli.go:1282 — "res, err := ahoy.DryRun(cwd)"
  - The report now names the path forward per kind: install hint for an unmanaged repo, nothing-to-act-on for a plain folder.
    evidence: internal/surface/cli/cli.go:1291 — "switch res.FolderKind {"
  - The history store index.json is bootstrapped on first install and is the registry keyed on root-commit SHA, and is the source doctor reads for the central/host location.
    evidence: internal/surface/cli/cli_test.go:634 — "if err := os.ReadFile(indexPath)"
    evidence: internal/surface/cli/cli_test.go:708 — "doctor did not resolve the central location from index.json (no history.path_stale)"
- diverged:
  - Provenance-only deviation: spc-5 is record catch-up (engine predates the spec) and the report cut was delegated to a sub-agent worker with the orchestrator re-running the gate; this is a signed-off process note, not a behaviour divergence from the ACs.
    evidence: .abcd/development/specs/closed/spc-5-folder-classification.md:44 — "Deviation: none in behaviour, one in provenance"
- missing: (none)
