# `/abcd:prepare-this-repo` — Repo Onboarding Bridge

`/abcd:prepare-this-repo` brings the current repository up to abcd's working
conventions: it reads the abcd record, audits the repo against it, then adopts
the three-tier `.abcd/` layout, a nameless working-conventions section in
`AGENTS.md`, and the commit gates. It is an **interim bridge** — abcd cannot yet
manage repositories directly, so the command does by hand what the CLI will
later take over, in a shape the CLI can adopt without unpicking.

It is a **host-delegated command**: no dedicated Go verb backs it and there is
no bare-status render. The workflow runs in the host agent from the markdown in
[`commands/prepare-this-repo.md`](../../../../commands/prepare-this-repo.md),
invoking the binary's read-only `abcd lint --json` for the engine-backed
conformance core and, in the adopt phase, two writing verbs. `abcd identity
init` writes both halves of the identity record: the block itself, as markdown
in `.abcd/development/IDENTITY.md` or the `--file` target (markdown stays the
source of truth, appended as a section to an existing file or created under a
`# Identity` heading), and `.abcd/positioning.json`, the pointer recording where
that block lives and which surfaces render from it. `abcd ahoy install` writes
the commit gates, and runs a second time with `--attribution` where the user
opts in. `abcd identity render` is the follow-on surface and writes nothing: it
proposes a correction as a diff, and adopting it is always the maintainer's
move.
It takes no argument — it always operates on the current repository.

## What it does

- **Refuses on repos the user does not own.** Phase 0 checks the origin remote
  and stops entirely — no audit, no writes — unless the repository is the user's
  or an org they control. Imposing these conventions on a third-party repo would
  interfere with its own development principles.
- **Audits before it touches anything.** It produces a gap report (existing
  structure, documentation shape, decision and working-state hygiene,
  principles followed or violated, privacy) and presents it before adopting
  anything.
- **Adopts the conventions.** The three-tier layout; a merged (never
  overwritten) `AGENTS.md` with verified repo facts plus the marked
  working-conventions block; and, where absent, the commit gates the binary
  embeds: the committed private name guard (`.githooks/pre-commit` and its
  `pre-merge-commit` half), the gitignored local banlist stub, and the
  `.gitattributes` line pinning the hooks to LF. No hook it scaffolds carries a
  secret-pattern set or an absolute-path check. Absolute-path detection is the
  binary's own `privacy-hygiene` lint rule, which phase 2 already runs through
  `abcd lint`. AI-attribution hooks are opt-in only.

## Flow

Four phases, each gated on the one before:

0. **Refuse unless owned** — origin-remote ownership check; stop if it fails.
1. **Orient** — read the abcd record from `$ABCD` (the abcd checkout root —
   the directory above `commands/`, where the flat command file lives, the
   same root `${CLAUDE_PLUGIN_ROOT}` names): the three-tier README, the brief,
   principles, ADRs, intents, `docs/` Diátaxis rules, and the lint configs as
   patterns.
2. **Conformance lint** — run `abcd lint --json` for the engine-backed conformance core
   (the six convention rules), supplement it with the structure/principles
   judgement the binary does not make, write the gap report to the target's
   `.abcd/.work.local/scratch/`, and present it before any change.
3. **Adopt** — create the three tiers with a repo-specific `CONTEXT.md`, migrate
   any historical `.work/` layout to the new tiers (propose then wait for sign-off;
   never leave a repo with both the old and new working-state homes), merge into
   `AGENTS.md`, scaffold the commit gates with `abcd ahoy install` (a committed
   hook is not a running hook until `git config core.hooksPath .githooks` points
   git at it, once per clone, and a declined config change is reported as
   scaffolded-but-unarmed rather than passed over), register the repo's
   **identity block** (detect an existing block and adopt it via `abcd identity
   init --file`, interview only where none exists; `abcd identity render` then
   holds every surface to it), and — only where the user says the repo requires
   AI disclosure — install the attribution hook with `abcd ahoy install
   --attribution` (opt-in). Three binary writes in this phase, not one:
   `identity init`, `ahoy install`, and the attribution install.

When abcd's own record has conflicting sources, the command trusts a fixed
authority order: `AGENTS.md`, then `work/CONTEXT.md`'s live-constraints section,
then ratified ADRs, then everything else read for understanding only.

## Boundaries

- **Nameless, self-contained output.** The working-conventions block written
  into `AGENTS.md` never mentions abcd, this command, or any private repository
  — the conventions read as the repo's own, between dated markers so later
  tooling can find and replace them.
- **Never commit downstream assets.** Anything tooling will later provide
  (`personas.json`, lint-config JSON, content copied from the abcd record) is
  applied, not copied. Only content about the target repository is committed.
- **Privacy.** `private-names.txt`, if present, is read-only context for the
  audit and never reproduced in any committed or published artefact.

## Acceptance

- **Given** a repo the user does not own, **when** they run
  `/abcd:prepare-this-repo`, **then** it stops at Phase 0 with no audit and no
  writes.
- **Given** an owned repo, **when** the command runs, **then** a gap report
  exists under `.abcd/.work.local/scratch/` and was presented before anything
  was adopted.
- **Given** sign-off, **when** it adopts, **then** the three-tier layout exists
  with a repo-specific `CONTEXT.md`, `AGENTS.md` carries verified repo facts and
  the marked nameless working-conventions section (the done-test — a fresh agent
  can build and test from `AGENTS.md` alone — passes), and any historical `.work/`
  layout is fully migrated or fully left alone.
- **Given** sign-off, **when** it adopts, **then** one identity block is
  recorded and registered — adopted where the repo already had one,
  interviewed only where it did not — and `abcd identity` reports every
  rendered surface against it.
- **Given** the adoption completes, **then** nothing from `private-names.txt`
  and no abcd-internal content appears in any committed artefact.

## Composition

`/abcd:prepare-this-repo` is one of the three host-delegated user-facing
commands under `/abcd:` that carry no Go verb; `/abcd:consult` and `/abcd:ingest`
(over the `~/.abcd/sources` corpus) are the others. It supersedes the older
`scaffold-repo` layout (historical `.work/` at the repo root) and migrates it on sight. Its
end state is the `.abcd/` layout the shipped abcd surfaces
(`/abcd:capture`, `/abcd:docs`, `/abcd:ahoy`) then operate over.

## References

- Plugin command: [`commands/prepare-this-repo.md`](../../../../commands/prepare-this-repo.md)
- The three-tier layout it adopts: [`../01-product`](../01-product) and the abcd `.abcd/README.md`
- The invariants the working-conventions block encodes: [`../02-constraints`](../02-constraints)
