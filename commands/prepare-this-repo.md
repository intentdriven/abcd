---
name: prepare-this-repo
description: Prepare the current repository to abcd's working conventions — audit it against the abcd record, then adopt the three-tier .abcd/ layout, an AGENTS.md conventions section, and the commit gates. Interim bridge until abcd manages repos directly. Use when the user asks to prepare, onboard, scaffold, or bootstrap a repo for agent work, or to bring a repo up to current conventions. Owned repos only — refuses everywhere else.
---

# Prepare this repo

An interim bridge: abcd cannot yet manage repositories directly, so this command
reads the abcd record and brings the current repository up to its conventions
in a way the abcd CLI can later take over without unpicking. It supersedes the
older `scaffold-repo` layout (`.work/` at the repo root); Phase 3 migrates that
layout when found.

**Locate the record first.** This command file lives at
`<abcd-root>/commands/prepare-this-repo.md` — the abcd repository root is two
levels up from it, the same root `${CLAUDE_PLUGIN_ROOT}` names (Phase 2 onward
uses it for the binary). Call that `$ABCD` below. Never assume any other
checkout location.

## Phase 0 — Refuse unless the user owns this repo

Check the origin remote (`git remote get-url origin`). If the repository is not
owned by the user (their account or an org they control) — or there is no
remote and the user cannot confirm ownership — **stop entirely**. No audit, no
local layer, no exceptions: imposing these conventions on a third-party repo
interferes with its own development principles.

## Authority ordering

Parts of the abcd record are still being reconciled. When sources conflict,
trust in this order:

1. `$ABCD/AGENTS.md` — the binding conventions.
2. `$ABCD/.abcd/work/CONTEXT.md` — current phase and the **Live constraints /
   sharp edges** section, which names what is not yet authoritative.
3. `$ABCD/.abcd/development/decisions/adrs/` — ratified decisions.
4. Everything else (brief, intents, research) — read for understanding, not as
   gospel.

## Phase 1 — Orient

Read from `$ABCD`, delegating heavy reading to subagents where available:

- `.abcd/README.md` — the three-tier layout (durability × sharing):
  `development/` (durable record, committed), `work/` (shared working state,
  committed), `.work.local/` (local ephemeral, gitignored).
- `.abcd/development/brief/` — the shape of a design brief (product,
  constraints, evidence, surfaces, internals, delivery, glossary).
- `.abcd/development/principles/` — one principle per file, plus the
  three-rung promotion ladder in its README.
- `.abcd/development/decisions/adrs/` — MADR, sequential `NNNN`. Session
  decisions live in `work/DECISIONS.md`; architecture-shaping ones graduate.
- `.abcd/development/intents/` — lifecycle by directory (`drafts/` →
  `planned/` → `shipped/` → `superseded/`).
- `docs/README.md` — strict Diátaxis, four directories, present tense only,
  user-facing only.
- `.abcd/docs-lint.json` and `record-lint.json` — study as patterns
  (banned-token rules with severity, fix message, `allow_context` escapes).
- `.abcd/.work.local/private-names.txt` — **if present**: the banlist for the
  Phase 2 privacy audit. Its contents are read-only context; never reproduce
  them in anything committed or published. Do not read anything else under
  `$ABCD/.abcd/.work.local/`.

## Phase 2 — Audit

The conformance core is **engine-backed**: run the binary's read-only audit and
build the gap report on its result, rather than hand-producing the whole thing.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lint --json
```

Its `findings` (each with a stable `ruleId`, `severity`, `file`, a `line` where
applicable, `message`, `fix`) cover the conventions the binary checks — the three-tier
`.abcd/` layout, an `AGENTS.md` router, durable decisions, docs currency,
privacy hygiene (absolute local paths in committed files), and identity
positioning (rendered surfaces match the canonical identity block). Its
`skipped` list names rules that did not apply (e.g. `docs-currency` where there
is no `docs/`, or `identity-positioning` where there is no `.abcd/positioning.json`).
The exit code is the tri-state (`0` clean, `1` warnings, `2` any error).

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

Then supplement with the judgement the binary does not make:

- **Existing structure (mandatory):** any convention layer already present —
  a legacy `.work/` + `.work.local/` at the root, an existing `AGENTS.md`
  router, CLAUDE.md/GEMINI.md bridges, pre-commit config — and where each
  piece maps in the three-tier layout.
- **Principles:** which abcd principles the repo follows, violates, or has no
  opinion on — cite files as evidence.
- **Privacy beyond absolute paths:** real hostnames/usernames/emails, private
  repo names, or `private-names.txt` matches the `privacy-hygiene` rule's v1
  scope does not yet cover. A finding on a deliberately illustrative line can be
  waived with `abcd-lint:allow` on that line (the earlier `abcd-audit:allow` spelling is honoured too).

Write the report — the `abcd lint` findings plus these supplements — to the
target's `.abcd/.work.local/scratch/` (create the directory via
`.git/info/exclude` if needed) and present it before touching anything.

## Phase 3 — Adopt

1. **Three tiers.** Create `.abcd/development/` (subdirectories only where
   there is content), `.abcd/work/` with a repo-specific `CONTEXT.md` (what the
   repo is, current phase, sharp edges — never copied from abcd) and an empty
   `DECISIONS.md`, and `.abcd/.work.local/` (`NEXT.md`, `scratch/`, `logs/`)
   excluded via `.git/info/exclude`.
2. **Legacy migration — propose, then wait.** If Phase 2 found a legacy
   `.work/` layout, present the mapping (`.work/DECISIONS.md` →
   `.abcd/work/DECISIONS.md`, and so on) and get explicit sign-off before
   moving anything. Moves are content-preserving (`git mv`, then fold), never
   delete-and-recreate. Hard rule: never leave both `.work/` and `.abcd/work/`
   behind — complete the migration or do not start it.
3. **AGENTS.md.** Merge — never overwrite — into the repo's `AGENTS.md`
   (create it if absent, with `CLAUDE.md` as a symlink to it):
   - Repo facts: what the repo is, exact build/test/lint commands (verified by
     running them, including how to run a single test), boundaries, definition
     of done. The done-test: a fresh agent session must be able to build and
     test the repo from `AGENTS.md` alone.
   - The working-conventions section below, between markers. An existing
     attribution policy in the file stays authoritative over the template.
4. **Identity — detect first, interview only if there is nothing to detect.**
   A project's positioning fragments silently: the README strapline, the
   package or plugin manifest description, and the conventions file's opening
   are edited at different moments until three surfaces tell three stories.
   One canonical block stops that.

   **Detect before asking.** Look for an identity block the repo already
   carries — three bullets (`- **Title:**`, `- **Tagline:**`, `- **Pitch:**`)
   under a heading such as "Identity (canonical)", typically in a brief or
   product chapter under `.abcd/development/`. If one exists, register it and
   ask nothing:

   ```bash
   "${CLAUDE_PLUGIN_ROOT}/abcd" identity init --file <path-to-that-file> --heading "<that heading>"
   ```

   `init` adopts the existing block byte-for-byte and writes only the pointer.
   Re-interviewing over an answer the repo already gives is how a project ends
   up with two canons.

   **Only if there is no block**, ask once, in the maintainer's own words:

   - **Title** (required) — what the project is called, as it should read in a
     heading.
   - **Tagline** (required) — one line saying what it is. This is the line
     every surface is held to.
   - **Pitch** (optional — offer to skip) — two or three sentences a newcomer
     could read and know whether this is for them.

   Then record the answers:

   ```bash
   "${CLAUDE_PLUGIN_ROOT}/abcd" identity init --title "…" --tagline "…" [--pitch "…"]
   ```

   This writes the markdown block (markdown stays the source of truth) and
   `.abcd/positioning.json`, which records where the block lives and which
   surfaces render from it. From then on `abcd lint` reports any surface that
   drifts from it, and `abcd identity render` proposes the correction as a
   diff. abcd never rewrites a surface itself — adopting a proposal is always
   the maintainer's move.

5. **Commit gates.** Scaffold them from the binary — every hook it writes is
   embedded in it, so the step applies the same artefacts on a fresh clone as on
   the machine that wrote them:

   ```bash
   "${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install
   git config core.hooksPath .githooks
   ```

   That writes the committed private name guard (`.githooks/pre-commit` and its
   `pre-merge-commit` half), the gitignored local banlist stub, and the
   `.gitattributes` line pinning the hooks to LF. The second command is once per
   clone: a committed hook is not a running hook until git is pointed at it.
   Report the hooks as scaffolded but unarmed if the user declines that config
   change — never silently.
6. **Attribution (opt-in only).** If the user says this repo requires
   AI disclosure, scaffold the prompt hook from the binary and add the
   attribution section to `AGENTS.md`:

   ```bash
   "${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install --attribution
   ```

   That writes `.githooks/prepare-commit-msg`, which seeds a commented
   disclosure prompt into every commit message an editor opens, and records the
   opt-in so a later plain `ahoy install` keeps the hook. It never writes a
   value: which tool assisted, and which version, is a fact only the committer
   has. Otherwise the default is no attribution, and nothing is written.

### The working-conventions section

Write this between `<!-- working-conventions YYYY-MM-DD -->` and
`<!-- /working-conventions -->` markers (today's date; the markers let later
tooling find and replace the block). The section is **self-contained and
nameless**: it never mentions abcd, this command, or any private repository —
the conventions read as the repo's own. Adapt wording to the repo; keep the
substance:

- Three-tier working state: `.abcd/development/` (durable record: ADRs in
  `decisions/adrs/` using MADR + `NNNN`, dated plans and research notes),
  `.abcd/work/` (committed: `CONTEXT.md` orientation, `DECISIONS.md`
  append-only one-line decision log), `.abcd/.work.local/` (gitignored:
  `NEXT.md` handover, `scratch/`, `logs/` — runtime artefacts go here, never
  in tracked directories).
- Decisions: one dated line in `.abcd/work/DECISIONS.md`; promote
  architecture-shaping decisions to ADRs.
- Docs: `docs/` is user-facing only, one Diátaxis type per page, present tense
  only; British English prose, US English in code and commits. No stray root
  markdown beyond the standard set.
- Privacy: no absolute local paths, hostnames, usernames, emails, tokens, or
  private repository names in anything committed; repo-relative paths only.
- Examples and user stories use the personas Alice, Bob, and Carol — never
  other names.
- Refer to the maintainer as they/them in every artefact.
- Never commit or push without being asked; substantive work goes on a branch
  and PR; new dependencies need explicit sign-off first.

### Never commit downstream

Assets that will later be provided by tooling are applied, not copied: no
`personas.json`, no lint-config JSON files, no content copied from the abcd
record. Commit only content that is about this repository.

## Definition of done

- The gap report exists in the target's `.abcd/.work.local/scratch/` and was
  presented to the maintainer.
- The three-tier layout exists with a repo-specific `CONTEXT.md`; any legacy
  `.work/` layout was fully migrated (with sign-off) or fully left alone.
- `AGENTS.md` carries verified repo facts and the marked, nameless
  working-conventions section; the done-test passes.
- One identity block is recorded and registered — adopted where the repo
  already had one, interviewed only where it did not — and `abcd lint identity`
  reports every registered surface as `ok`, or the drift it reports was shown
  to the maintainer with the proposed diff.
- Nothing from `private-names.txt` and no abcd-internal content appears in any
  committed or published artefact.
- Every asset the adopt phase applied resolved from this record or from the
  binary. A step that reaches for a path only one machine has does not fail —
  it silently does nothing, and the adoption degrades against loud-staging, so
  this file carries no home-relative path and the onboarding self-containment
  check refuses one.
