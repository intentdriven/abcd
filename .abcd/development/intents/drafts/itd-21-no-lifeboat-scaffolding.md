---
id: itd-21
slug: no-lifeboat-scaffolding
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-18, itd-1]
severity: minor
---

# Start a New Project With abcd Conventions

## Press Release

> **abcd ships `/abcd:init-project scaffold` for greenfield projects with no lifeboat to embark from.** Run it in an empty directory and abcd scaffolds a complete project skeleton: language-detected layout, abcd conventions baked in (CLAUDE.md with marker block, `.abcd/` namespace, gitignore policy, dev-sync setup, intent capture), and a starter `itd-1` intent capturing the project's mission. You start with abcd's full discipline from day one rather than retrofitting later. Bare `/abcd:init-project` shows status+help.
>
> "I love abcd's conventions but every new project I'd start without them and add abcd later," said Liam, mobile developer. "init-project starts the project with everything in place. The first commit already has the marker block, the first intent already captures what the project is for. I never have to retrofit."

## Why This Matters

abcd has two onboarding paths: `/abcd:ahoy install` (add abcd to an existing project) and `/abcd:embark from <path>` (rebuild from a lifeboat). Neither covers greenfield: "I'm starting a new project from scratch, no prior lifeboat exists, but I want abcd discipline from commit one."

`~/.claude/templates/` from abcd v0 has 24 template files (per-language project scaffolds + generic templates) that suggest the pattern. Adapting those makes init-project a real onboarding path.

## What's In Scope

- `/abcd:init-project scaffold` sub-verb in empty (or near-empty per emptiness rules) directory; bare `/abcd:init-project` shows status+help
- Project-type detection / interview (language, intended use, deployment target)
- Templates for common project shapes: Python, Swift, Go, TypeScript, Rust, Docker, plain
- All abcd conventions installed at scaffold time (no need to re-run ahoy)
- Starter intent (`/abcd:intent` interview kicked off as last step) capturing the project's mission as `itd-1`
- Templates committed and versioned as assets embedded in the Go binary (`internal/core/...`)

## What's Out of Scope

- Replacing every cookiecutter / yeoman / npx-create-* tool (this is abcd-shaped scaffolding, not general-purpose)
- Cross-language polyglot projects (start with one language; add more manually)
- IDE-specific setup (.vscode/, .idea/) — leave to user
- Auto-creating a GitHub repo (separate concern)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-aware terminal, **when** the user runs bare `/abcd:init-project`, **then** the dispatcher detects whether the current directory is empty / a fresh repo / already abcd-shaped and shows status + suggested next actions (per the universal bare-command-as-help convention) — no scaffolding runs without an explicit verb or flag.
- **Given** an empty directory (no files except possibly `.git/`), **when** the user runs `/abcd:init-project scaffold` (or accepts the suggestion offered by bare invocation), **then** the interview captures language + intended use + deployment target, scaffolds the chosen project template (a per-language asset embedded in the Go binary, `internal/core/...`), installs all abcd conventions (CLAUDE.md marker block, `.abcd/` namespace, gitignore policy), and offers to start the `/abcd:intent new` flow as the final step capturing the project's mission as `itd-1`.
- **Given** a directory that fails the brief's emptiness rules (existing source files, vendored code, etc.), **when** `/abcd:init-project scaffold` is invoked, **then** the command refuses with a clear error AND suggests `/abcd:ahoy install` as the correct path for retrofitting abcd into an existing project.
- **Given** the user picks "Python" in the interview, **when** scaffolding completes, **then** the directory contains a `pyproject.toml`, `src/<package>/__init__.py`, `tests/`, a permission template at `.claude/settings.local.json` (per itd-18), and an `.abcd/` namespace with the brief skeleton.
- **Given** the user declines the optional starter intent, **when** init-project completes, **then** `intents/drafts/` is empty AND the ahoy report records "starter intent: declined" so the user can run `/abcd:intent new` later without confusion.
- **Given** the user accepts the starter intent, **when** init-project completes, **then** `intents/drafts/itd-1-<slug>.md` exists with a populated press release AND a populated `## Acceptance Criteria` section — itd-1's gate applies even at scaffold time.
- **Given** a project type unsupported by the shipped templates (e.g., Elixir, Haskell), **when** the user picks "plain", **then** scaffolding falls back to a minimal language-agnostic skeleton (`README.md`, `LICENSE`, `.gitignore`, abcd conventions) AND the report records the absence of a language-specific template as a hint the user could contribute one.

## Open Questions

- How does this differ from running `/abcd:ahoy install` after `git init` — what's the unique value?
- Should `/abcd:init-project scaffold` always require running `/abcd:intent new` immediately after, or offer it as optional?
- What's the conflict policy if directory isn't fully empty (per the brief's emptiness rules)?

## Amendments

Amended by itd-2610030814013772 (2026-10-03, its decisions 1 and 7): a scaffolded project receives AGENTS.md alone, never a CLAUDE.md or any other tool's conventions file; where this draft's text or criteria name a CLAUDE.md, read AGENTS.md.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
