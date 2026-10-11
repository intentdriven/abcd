# Platform & Repo

This file holds locked decisions about *where* abcd runs and *how it ships*. These are non-negotiable starting conditions for any agent working on abcd — the agent designs architecture inside these rails, not around them.

## One repository

abcd lives in **one repository** and ships a **curated release artifact** cut from it — there is no dev→public mirror ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)). The Go binary is the product ([adr-21](../../decisions/adrs/0021-rebuild-in-go.md)); its source, its user-facing documentation, and the design record all share this one tree. The design record (this brief, the roadmap, intents, ADRs, research) lives in-tree at `.abcd/development/`; shared working material is committed under `.abcd/work/`, and `.abcd/.work.local/` is the gitignored local-only surface. See [`../../../../AGENTS.md`](../../../../AGENTS.md) for the canonical "where things live" map.

**`.abcd/**` stays in-tree and is present in every repository checkout — marketplace installs and release source archives included — never in the released binaries** — exclusion is a build-time filter over the one tree, not a copy between two repos; the launch bundler implements it as a structural namespace deny, which no cut release has run yet. `/abcd:launch` cuts a curated GitHub Release from this repo; **the repo is the marketplace**, so discovery, install, and the design record share one location.

## One conventions file

**`AGENTS.md` is the one conventions file abcd writes, in its own project and in every project it sets up** ([adr-2610030814023326](../../decisions/adrs/2610030814023326-agents-md-is-the-one-conventions-file.md)). abcd writes no `CLAUDE.md` and no other tool's own conventions file, as a copy, a link or a pointer. It still reads such a file where it finds one, for detection and uninstall, and finds one for the lifeboat packer; it takes abcd's own block back out of one where an earlier setup put it, and otherwise leaves a file holding the owner's words as it is.

## Front doors

The core is a transport-agnostic Go package ([adr-23](../../decisions/adrs/0023-transport-agnostic-core.md)) behind thin front doors. The **CLI (Cobra)** is the first front door and ships in the MVP. An **MCP server** and a **markdown plugin surface that shells to the Go binary** are later front doors — each is a new adapter over the unchanged core, not the substrate abcd is built on.

## Lifeboat path

**Out-of-tree, at an operator-chosen destination** — `disembark pack <source-repo> <dest>`. There is no in-tree lifeboat home and nothing to gitignore in the source, because **disembark never writes to the source repo** (a test hashes its tree before and after). Per [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md), superseding adr-4's `.abcd/lifeboat/`: mining a dead or archived project must not require installing abcd into a repo we only want to read.

**Lifeboat is always *output*.** `<dest>` holds the latest snapshot only — produced by `disembark`, regenerable from current state; there is no `lifeboat-v1/` / `lifeboat-v2/` proliferation. Re-running is governed by a **destination safety gate**, not adr-4's `.bak` overwrite: refuse unless the destination is absent, an empty directory, or one carrying a parseable `_provenance.json`. **abcd never overwrites a directory it did not produce.** Past disembarks are recorded as manifests (hash + file list + label) at the operator level, not as preserved snapshots — see [`04-surfaces/03-embark.md § 7`](../04-surfaces/03-embark.md#7-voyage-layout--embarkdisembark-provenance-and-history).

## Embark sources

**Input lifeboats are external by default.** Embark reads from `embark from <lifeboat-dir> [target-dir]`: the lifeboat at whatever destination a disembark wrote it to. The unpack takes no modifiers of its own. Two operator-level additions are a design target ([`03-embark.md` § The design-target surface](../04-surfaces/03-embark.md#the-design-target-surface)): an `embark/provenance.json` under `~/.abcd.noindex/voyage/<source-root-sha>/` holding the source path and manifest hash (never committed, because voyage records absolute source paths, per adr-35), and an opt-in archive modifier that copies the input lifeboat verbatim into `~/.abcd.noindex/voyage/<source-root-sha>/embark/from/<timestamp>/` for the rare case where the source repo will disappear.

**Embark sources, in order (post bare-as-help refactor — see [`04-surfaces/03-embark.md`](../04-surfaces/03-embark.md)):**

1. `embark from <lifeboat-dir>` (any explicit path to a lifeboat destination a disembark wrote) — **there is no `home` shorthand**, because there is no in-tree lifeboat home to expand it to (adr-35). The round-trip / self-test case is just `disembark pack <repo> <dest>` followed by `embark from <dest>`.
2. *Design target, not built:* an `embark scan` discovery sub-verb, with a deep mode, that walks sibling directories (`../`), lists **lifeboat destinations** — directories carrying a parseable `_provenance.json`, the same marker the destination safety gate keys on — ranked by mtime; does not unpack; pass the chosen path to `embark from <path>`
3. *Design target, not built:* free-text path input via the embark interview; `from` requires the lifeboat path

> **Open question (adr-35):** where `scan` searches. Walking `../` made sense when a lifeboat lived inside its producing repo, so siblings-of-cwd *were* the candidate set. Destinations are now operator-chosen and need not sit beside the repo being embarked into. Either the sibling walk is kept as a cheap heuristic, or scan is given explicit roots (an argument, a configured search path, or the voyage records under `~/.abcd.noindex/voyage/`). adr-35 does not settle this; it must be decided before `scan` is specified.

**No global lifeboat archive (`~/.abcd.noindex/archive/`).** A lifeboat lives at the destination its operator chose and abcd keeps no registry of them; the only operator-level state is voyage (`~/.abcd.noindex/voyage/<source-root-sha>/`), which records what was done, not the artefacts themselves. Share externally by copy.

## Validation corpus

See [`01-product/02-context.md`](../01-product/02-context.md) for the canonical validation-corpus list (SSOT). Summary: `idelphiDev/` (primary), `abcdSubZero/`, `idelphiSubZero/`. Acceptance runs against the corpus with documented exemptions where a feature genuinely doesn't apply.
