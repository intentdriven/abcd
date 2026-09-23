# `/abcd:prepare-this-repo` — Repo Onboarding Bridge

Take a repository you own and bring it up to abcd's working conventions in one
sitting: it audits first, shows you the gap report, and only then adopts the
three-tier `.abcd/` layout, a working-conventions section in `AGENTS.md`, an
identity block, and the commit gates. What you get back is a repo a fresh agent
session can build and test from `AGENTS.md` alone. What it costs you is a
sign-off at each phase, which is the point: nothing is adopted before you have
seen what it would change.

It is an **interim bridge**. abcd cannot yet manage repositories directly, so
the command does by hand what the CLI will later take over, in a shape the CLI
can adopt without unpicking.

It is **host-delegated**: no dedicated Go verb backs it, and there is no
bare-status render. The workflow runs in the host agent from
[`commands/prepare-this-repo.md`](../../../../commands/prepare-this-repo.md).
It takes no argument, always operating on the current repository.

Because the binary carries no verb of this name, typing `abcd prepare-this-repo`
at a shell is an unknown command rather than a route in. Today that refusal also
blames a stale binary and asks for a rebuild, which is the wrong reading for a
surface that is host-delegated by design: what the reader needs is the plugin
command above.

## What it does

- **Refuses on repos the user does not own.** The first phase checks the origin
  remote and stops entirely: no audit, no writes. Imposing these conventions on
  a third-party repo would interfere with its own development principles.
- **Audits before it touches anything.** It produces a gap report covering
  existing structure, documentation shape, decision and working-state hygiene,
  principles followed or violated, and privacy, and presents it before adopting
  anything.
- **Adopts the conventions.** The three-tier layout; a merged, never
  overwritten, `AGENTS.md` carrying verified repo facts plus a marked
  working-conventions block; a registered identity block; and the commit gates.

## Where the binary does the work

The command is markdown, but three of its steps are the binary's, so the result
does not depend on an agent's memory of what a convention looks like.

The JSON form of `abcd lint` supplies the engine-backed conformance core, read-only,
which the command then supplements with the structural and principles judgement
the binary does not make. The identity verb's initialiser records the identity block: the
markdown itself, which stays the source of truth, plus `.abcd/positioning.json`,
the pointer recording where that block lives and which surfaces render from it.
The ahoy installer is the adopt phase's workhorse, and it does more than write
the commit gates. In one run it writes the repo's settings file with its
visibility, oracle backend and scan depth, writes its rule-loader overrides file,
installs a copy of the binary on `PATH`, records the repo in the machine's own
store, and offers to pin the git commit identity. It leaves `CLAUDE.md` and
`AGENTS.md` as the repo wrote them: abcd's own managed rule-loader block names
the tool, so it lands in either file, or both, only where the project chooses a
docs target, and the repo classifies as managed on its registry entry without
it. It runs a second time, with the installer's attribution flag, where the user
opts in: that run installs the committed `prepare-commit-msg` prompt asking every
commit to declare whether a tool assisted it, and the choice is recorded, so a
later install without the flag keeps the hook. The flag's spelling, like the docs
target's, is ahoy's shape, so it lives in the generated appendix of
[`01-ahoy.md`](01-ahoy.md#appendix-the-shipped-surface) and is not repeated here.

The identity verb's render is the follow-on surface and writes nothing: it proposes
a correction as a diff, and adopting it is always the maintainer's move.

## Flow

Four phases, each gated on the one before.

0. **Refuse unless owned.** Origin-remote ownership check; stop if it fails.
1. **Orient.** Read the abcd record from the plugin root: the three-tier
   README, the brief, principles, ADRs, intents, the `docs/` Diátaxis rules, and
   the lint configs as patterns.
2. **Conformance lint.** Run `abcd lint` in its JSON form, supplement it, write the gap
   report to the target's `.abcd/.work.local/scratch/`, and present it before
   any change.
3. **Adopt.** Create the three tiers with a repo-specific `CONTEXT.md`; migrate
   a historical `.work/` layout at the repo root if one is found, proposing the
   mapping and waiting for sign-off, and never leaving a repo with both homes;
   merge into `AGENTS.md`; scaffold the commit gates; register the identity
   block, adopting one the repo already carries rather than re-interviewing; and,
   only where the user says the repo requires AI disclosure, install the
   attribution hook.

A committed hook is not a running hook until `git config core.hooksPath
.githooks` points git at it, once per clone. A declined config change is
reported as scaffolded-but-unarmed rather than passed over silently.

When abcd's own record has conflicting sources, the command trusts a fixed
authority order: `AGENTS.md`, then `work/CONTEXT.md`'s live-constraints section,
then ratified ADRs, then everything else read for understanding only.

## Boundaries

- **Nameless, self-contained output.** The working-conventions block written
  into `AGENTS.md` never mentions abcd, this command, or any private repository:
  the conventions read as the repo's own, between dated markers so later tooling
  can find and replace them. The paths it states into the `.abcd/` layout are
  the one trace of the tool, and the adopter accepts that namespace by adopting
  the layout. The install leaves the conventions files nameless as well. The
  name-guard hooks and the `.gitignore` fence it commits are the one sanctioned
  mention outside that namespace: the hooks run the binary, the fence warns
  against hand-editing, and both keep the markers detection finds an adopted
  repository by (the product thinker's ruling of 2026-09-23).
- **Never commit downstream assets.** Anything tooling will later provide
  (persona data, lint-config JSON, content copied from the abcd record) is
  applied, not copied. Only content about the target repository is committed.
- **Privacy.** A `private-names.txt`, if present, is read-only context for the
  audit and never reproduced in any committed or published artefact.
- **No secret-pattern hooks.** No hook this command scaffolds carries a
  secret-pattern set or an absolute-path check. Absolute-path detection is the
  binary's own `privacy-hygiene` lint rule, which the audit phase already runs.

## Acceptance

- **Given** a repo the user does not own, **when** they run
  `/abcd:prepare-this-repo`, **then** it stops at the ownership phase with no
  audit and no writes.
- **Given** an owned repo, **when** the command runs, **then** a gap report
  exists under `.abcd/.work.local/scratch/` and was presented before anything
  was adopted.
- **Given** sign-off, **when** it adopts, **then** the three-tier layout exists
  with a repo-specific `CONTEXT.md`, `AGENTS.md` carries verified repo facts and
  the marked nameless working-conventions section, the done-test passes (a fresh
  agent can build and test from `AGENTS.md` alone), and any historical `.work/`
  layout is fully migrated or fully left alone.
- **Given** sign-off, **when** it adopts, **then** one identity block is
  recorded and registered, adopted where the repo already had one and
  interviewed only where it did not, and `abcd identity` reports every rendered
  surface against it.
- **Given** the adoption completes, **then** nothing from `private-names.txt`
  and no abcd-internal content appears in any committed artefact, with one
  sanctioned exception (ruled 2026-09-23): the name-guard hooks
  (`.githooks/pre-commit`, `.githooks/pre-merge-commit`) and the `.gitignore`
  fence name abcd, and cite none of its record ids.

## Composition

`/abcd:prepare-this-repo` is one of the three host-delegated user-facing
commands that carry no Go verb; [`/abcd:consult`](13-consult.md) and
[`/abcd:ingest`](14-ingest.md) are the others. Its end state is the `.abcd/`
layout the shipped abcd surfaces then operate over.

## References

- Plugin command: [`commands/prepare-this-repo.md`](../../../../commands/prepare-this-repo.md)
- The three-tier layout it adopts: [`../02-constraints/01-platform.md`](../02-constraints/01-platform.md) and the abcd `.abcd/README.md`
- The invariants the working-conventions block encodes: [`../02-constraints`](../02-constraints)

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

There is no shipped surface: the command tree registers no `abcd prepare-this-repo` verb, so there are no flags and no sub-verbs to list.

<!-- surface-appendix:end -->
