# Development record

The abcd design record — the durable "what / why" the build works from. Kept in the
repo (transparent) and present in every repository checkout, marketplace installs
and release source archives included, but never in the released binaries;
user-facing docs live under [`../../docs/`](../../docs/). Organised **flat by
artefact type**, one canonical home per concept:

| Folder | What it holds |
|--------|---------------|
| [`brief/`](brief) | The living canvas: what abcd IS (product … delivery) + the [glossary](brief/glossary). |
| [`intents/`](intents) | Press-release intents — the WHY of each user-facing change. Lifecycle by directory: `disciplines/` `drafts/` `planned/` `shipped/` `superseded/`. |
| [`specs/`](specs) | Specs (`spc-N`) — the HOW derived from an intent. Lifecycle by directory: `open/` `closed/`. |
| [`principles/`](principles) | Distilled cross-cutting design principles (first-class — the lifeboat packs these). |
| [`decisions/`](decisions) | ADRs (MADR) — ratified architecture decisions, one canonical home; plus `notes/`. |
| [`roadmap/`](roadmap) | Sequencing: `phases/` + `rfcs/` (an accepted RFC produces an ADR). |
| [`plans/`](plans) | Dated design / implementation plans (`YYYY-MM-DD-*`). |
| [`research/`](research) | <!-- index: research-children -->Investigations: `notes/` (dated write-ups) + `prompting/` (prompt R&D) + `data/` (measured corpora) + `abcdev-site/` (the website work cluster).<!-- /index --> |
| [`readings/`](readings) | The readings family: one directory per cold-reading run (`rdg-<yymmddHHMMSS><rrrr>`), holding the manifest of what the assembler passed. Its [charter](readings/README.md) renders the include table. |
| [`release/`](release) | The committed compatibility baseline: `surface.json` (the shipped-surface registry `surface_coverage` reads). |
| [`release-gate/`](release-gate) | The pre-tag runbook and its `manifest.json` — the periodic brief↔surface cross-check `gate_lockstep` pins. |

Also here: `personas.json` — the machine-checked persona roster for press-release
quote attribution (the `persona_registry` lint reads it); it migrates to embedded
Go data once the grill sub-verb lands.

**Conventions.** Durable-vs-working is the `development/` ↔ `../work/` ↔
`../.work.local/` tiering. Issues are not a record folder: the ledger lives in the
working tier at [`../work/issues/`](../work/issues) (adr-32), and a
design-significant issue graduates from it into `intents/` or `principles/`. ADRs
use sequential `NNNN` (stable cross-reference handles); plans and research notes are
date-prefixed (chronological). Present tense only; history lives in git.
