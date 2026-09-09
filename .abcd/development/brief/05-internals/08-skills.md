# Skills — Procedural Workflows That Aren't Commands

abcd's surface namespace (`/abcd:`) is **commands only**. The skill/command distinction is documented here for the boundary rules; **abcd ships zero skills**. The three workflows once shipped as skills — `consult`, `ingest`, and `prepare-this-repo` — are **commands** (they mutate state: the sources corpus, its ledger, and the target repo), and live at `commands/<name>.md` with surface chapters [`13-consult.md`](../04-surfaces/13-consult.md), [`14-ingest.md`](../04-surfaces/14-ingest.md), and [`15-prepare-this-repo.md`](../04-surfaces/15-prepare-this-repo.md).

## What's a skill?

A **skill** is a procedural-workflow surface — a markdown-encoded interview, audit, or guided pass that runs against existing material in the repo. Skills are auto-registered by Claude Code's plugin system from the plugin's `skills/<skill-name>/` directory; abcd does not maintain a separate skill registry.

**abcd ships zero skills.** The three workflows once classified as skills (`consult`, `ingest`, `prepare-this-repo`) were reclassified as commands — each mutates state, which the boundary rule below makes command-shaped. An earlier version of this brief also proposed `/abcd:grill` as a skill; its **design target** is promotion to a sub-verb of `/abcd:intent` (per the round-2 command-structure review, see `04-surfaces/05-intent.md § 2`) — the `intent` parent ships (a binary verb with `commands/intent.md`), but `grill` is not among its sub-verbs. Grill's mid-session glossary writes and per-session report output are command-shaped responsibilities — exactly the trigger for promotion this file describes below.

## What's a command?

A **command** is a stateful operation that creates, modifies, or moves artefacts. Commands have:

- A **brief surface file** (`04-surfaces/NN-<verb>.md`) describing acceptance criteria, interaction flow, and side effects (or a sub-verb row in an existing parent's surface file) — for example `docs`, `history`, and `version` are chapters `10`–`12`. The CLI-only binary verbs `changelog`, `completion`, `hook`, `rules`, and `spec` have a Go verb but no surface chapter of their own; the surfaces index lists all five under [§ Operator-internal verbs](../04-surfaces/README.md#operator-internal-verbs) with the record that delivered each.
- A **per-invocation report subdirectory** under the gitignored `.abcd/.work.local/logs/<verb>/` tier for commands that emit run reports — `memory lint` writes each run to `logs/memory/lint-<timestamp>/`.
- A **status + help** mode when called bare — the bare-status-board convention holds for `ahoy`, `banlist`, `capture`, `identity`, `intent`, `memory`, `reading`, `site`, and `spec` (and bare `abcd`); it is not universal, and the [surfaces index](../04-surfaces/README.md) carries the one enumeration of where it holds and where a parent prints usage instead.

abcd ships **nineteen binary-backed top-level commands**, each with both a Go verb and a `commands/` file: `/abcd:ahoy`, `/abcd:banlist`, `/abcd:capture`, `/abcd:decide`, `/abcd:disembark`, `/abcd:docs`, `/abcd:embark`, `/abcd:guard`, `/abcd:history`, `/abcd:ideate`, `/abcd:identity`, `/abcd:intent`, `/abcd:launch`, `/abcd:lint`, `/abcd:memory`, `/abcd:reading`, `/abcd:site`, `/abcd:update`, and `/abcd:version`. The `disembark` parent carries the largest sub-verb tree (`coverage`, `graveyard`, `review`, `pack`, `plan`, `press-release`, `principles`, `probe`); the `intent` parent's shipped sub-verbs are `link`, `plan`, `ready`, and `audit` (alongside the deprecated `new` alias and the canonical bare quoted create `/abcd:intent "<text>"`), with `refine`, `grill`, `ship`, `consistency`, `shape`, and `reclassify` as design targets. The newer parents carry small trees: `guard` exposes `check` and `hook`, `ideate` exposes `record`, `identity` exposes `init` and `render` (with a bare read-only status render), `site` exposes `build` and `check`, and `reading` exposes `assemble` and `ingest`. `decide` is a leaf: its one operand is the quoted title it mints a record from, so bare `abcd decide` refuses rather than rendering a board. Five further binary verbs — `changelog`, `completion`, `hook`, `rules`, and `spec` — expose a Go verb but no `commands/` file, so they are CLI-only; `hook` is registered but hidden from `--help`, reached from `hooks/hooks.json` rather than typed. See [`04-surfaces/`](../04-surfaces) for per-command detail.

Alongside the nineteen binary-backed verbs, abcd ships **three host-delegated commands** — `/abcd:consult`, `/abcd:ingest`, and `/abcd:prepare-this-repo` (chapters `13`–`15`). They are commands (state-mutating) but have **no Go verb**: the workflow runs in the host agent, so they have no bare-status render and no binary sub-verbs. This is the shape a command takes when its work is host-delegated rather than owned by the transport-agnostic core.

## Skill vs command — decision criteria

A surface is a **skill** when:

- The verb describes a *workflow that runs against existing content* — interview, audit, review, walkthrough, stress-test.
- Output is *findings or suggestions only*, never artefact creation/modification.
- The procedure is *re-runnable on the same input* without different effects (idempotent).
- The surface fits naturally as a workflow markdown file the agent reads and follows.

A surface is a **command** when ANY of:

- The verb describes a *state change* — install, pack, unpack, capture, plan, ship.
- Output includes *new or modified artefacts* (files, directory moves, frontmatter updates) — even alongside findings.
- Re-running has different effects (idempotent or not, but state-mutating).
- The surface needs an `acceptance` block, side-effect documentation, and (often) a checkpoint/resume protocol.
- The surface writes run reports to a `.abcd/.work.local/logs/<verb>/` subdirectory.

When in doubt, ship as a command. The earlier "ship as a skill first; promote on mutation" guidance was overturned by the round-2 review: by the time you discover a skill is mutating state, downstream contracts have hardened around the skill shape and rework is expensive. Better to recognise command-shape up front.

## Skills are not in `04-surfaces/`

`04-surfaces/` documents commands. A skill gets **no surface chapter** there: no `NN-<verb>.md` file carrying acceptance criteria, interaction flow, and side effects. The plugin's `skills/<skill-name>/SKILL.md` is the executable form; the intent file (when one exists) is the canonical user-moment reference.

A skill does need a **row** in the surfaces registry, [`04-surfaces/README.md`](../04-surfaces/README.md). The blocker-severity `surface_coverage` record-lint rule reads `skills_dir` alongside `commands_dir` and counts every immediate subdirectory of `skills/` as a real surface, so the first skill added under `skills/` fails `make record-lint` with `surface '<name>' has no row in the brief surface registry` until the table carries it. The registry is the one place a skill and a command share: presence is machine-checked in both directions, and only the skill-vs-command classification is left to judgement.

## Skill registration

**The registration list is empty — abcd ships no skills.** There is no `skills/`
directory in the tree; the plugin system's `skills/<name>/` auto-registration finds
nothing to register. The three workflows that once lived here (`consult`, `ingest`,
`prepare-this-repo`) were reclassified as commands and moved to `commands/<name>.md`;
because `commands/` is in the release payload (`.abcd/config/launch-payload.json`) and
`skills/` never was, the move also closes iss-61 (a shipped skill silently dropped from
the cut artefact — there are no skills left to drop).

If a later phase introduces a user-facing skill under `/abcd:` (a slash-invokable
workflow that does NOT have a parent command and is findings-only/idempotent per the
boundary rule), it is registered here.

## Future skills

itd-30 (design fictions, a later phase) is a **command extension**, not a new skill — it extends the canonical create `/abcd:intent "<text>"` with `--format=fiction`.

If a later phase introduces skills like `/abcd:plan-stress-test` (cross-intent adversarial review) or `/abcd:walkthrough` (read-aloud orientation pass), each gets:

- An intent file (capturing the user moment)
- A `skills/<skill-name>/{SKILL.md, workflow.md}` directory (the executable form)
- An entry in this file's "Skill registration" section
- A row in the [`04-surfaces/README.md`](../04-surfaces/README.md) registry table, marked `shipped`: the `surface_coverage` gate demands one for every `skills/<name>/` directory
- **No** `04-surfaces/` surface chapter unless the skill is command-shaped (in which case it ships as a command, not a skill)

The skills-vs-commands *classification* is enforced by reviewer judgement, not by lint: a lint reads a directory, not a shape, so nothing mechanical tells a findings-only workflow from a mutating one. The registry row is the half that is machine-checked, as the section above says. The strict rule from the round-2 review: **any logbook output, any artefact mutation, any state change → command, not skill.** Findings-only, idempotent, read-only-against-existing-content → skill.
