---
id: itd-8
slug: with-code-bundling
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Lifeboats Carry Source Code, Not Just Lessons

## Press Release

> **abcd lets you bundle source code into your lifeboats.** Running `/abcd:disembark pack <repo> <path> --with-code` packs not only the synthesised brief, principles, and reviews, but also a curated copy of the existing implementation. On the receiving end, `/abcd:embark from <path> --with-code` unpacks both the design lessons AND the working code, so the new project starts running immediately rather than rebuilding from scratch.
>
> "We were running disembark for big rewrites where the lessons were the goal, but for some projects we just wanted to fork-and-improve," said Bob, staff engineer. "Now I can choose: rebuild clean, or carry the code over and refactor where it matters."

## Why This Matters

abcd deliberately scopes code-shipping out of the base lifeboat: the lifeboat captures *lessons* (principles, decisions, reviews), and `code-rescuer` extracts patterns from source without ever copying source itself. This is right for greenfield rewrites but wrong for the substantial fraction of cases where users want to fork-and-improve rather than rebuild.

Code bundling needs a thoughtful scope decision (source dirs only? everything tracked? curator-driven? user-picked?) and a corresponding lifeboat shape. The `--with-code` flag is the extension point; the simpler "lessons-only" model ships first, with code bundling added once real disembark/embark cycles provide evidence.

## What's In Scope

- `--with-code` flag on `/abcd:disembark pack <repo> <path>` (pack code into lifeboat)
- `--with-code` flag on `/abcd:embark from <path>` (unpack code from lifeboat)
- Code scope decision: source dirs / everything tracked / curator-driven / user-picked (one wins)
- Updated lifeboat shape with `code/` subdirectory and manifest
- `code-rescuer` agent extended (currently principle-only) to make scope-aware copy decisions

## What's Out of Scope

- Migrating existing lifeboats forward to include code retroactively (covered by itd-9 schema migration)
- Selective code "diff" packing (only changed files vs full snapshot) — possible future extension
- Test-suite carrying as a separate axis (could be future intent)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** a project with a `src/` directory and `/abcd:disembark pack <repo> <path> --with-code` is invoked, **when** disembark completes, **then** the lifeboat at `<path>` contains a `code/` subdirectory holding the curated source AND a `code/_manifest.json` declaring per-file scope decision (`copy_verbatim` / `copy_redacted` / `excluded` with reason).
- **Given** a lessons-only lifeboat (no `code/` subdirectory), **when** the user runs `/abcd:embark from <path> --with-code` against it, **then** the command warns "this lifeboat carries lessons only; nothing to unpack as code" and embarks the brief content normally without erroring.
- **Given** a code-bundled lifeboat, **when** `/abcd:embark from <path> --with-code` runs, **then** the unpacked repo contains the source under the documented destination (e.g., `src/` matching the disembark scope) AND the embark report records the per-file copy outcomes.
- **Given** the user declines code curation at disembark time (or runs `disembark pack <repo> <path>` without `--with-code`), **when** disembark completes, **then** no `code/` directory is written to the lifeboat — the `--with-code` flag is the *only* path to code shipping; default behaviour stays lessons-only.
- **Given** `disembark pack <repo> <path> --with-code` is invoked on a repo where the curator-suggested defaults exclude a file the user wants included, **when** the user inspects the curator output interactively, **then** the user can override the per-file decision before disembark commits the lifeboat.
- **Given** a code-bundled lifeboat, **when** `/abcd:launch ship` runs against it, **then** `code/` is *not* automatically promoted to the public `abcd/` repo — code shipping respects the launch payload manifest's default-deny semantics; explicit allow-list entries are required.

## Open Questions

- Which scope model wins? Likely user-picked at disembark with curator-suggested defaults, but needs confirmation.
- How does `--with-code` interact with `/abcd:launch`'s default-deny payload manifest?
- Do code-bundled lifeboats grow large enough to warrant compression / streaming?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
