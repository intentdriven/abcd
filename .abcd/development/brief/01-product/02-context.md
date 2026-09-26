# Context

You did a manual lifeboat from iDelphiZero → iDelphi (`idelphiDev/.work/lifeboat/rescue/extraction.md` (historical) is the gold standard). Before rebuilding iDelphi from scratch, you want to automate: (a) **packing** lessons from an in-progress repo into a portable lifeboat artefact, and (b) **unpacking** that artefact to bootstrap a clean rebuild.

abcd ships these user-facing commands:

- **`/abcd:ahoy install`** — install / update abcd in any project (bootstraps configuration, gitignore, marker blocks, PATH symlink). Bare `/abcd:ahoy` shows status+help.
- **`/abcd:disembark <source-repo> to <dest>`** — *pack* a lifeboat by **reading** `<source-repo>` — any repository, including one abcd has never touched — and writing the artefact **out-of-tree** to the operator-chosen `<dest>`. The source is never written to; the destination must be absent, an empty directory, or one carrying a parseable `_provenance.json` ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)). Bare `/abcd:disembark` shows status+help; `probe <source-repo>` and `dry-run` sub-verbs preview without writing — `probe` reports **coverage** (which brief sections a repository can ground, and which come back blank).
- **`/abcd:embark from <dest>`** — *unpack* the lifeboat at `<dest>` (wherever a disembark wrote it) into a (typically empty) target project. Bare `/abcd:embark` shows status+help; `scan` and `probe <path>` sub-verbs discover/inspect without unpacking.
- **`/abcd:launch ship`** — cut a curated release from the single repo (the packaging filter denies `.abcd/**`, so the released binaries do not carry it), scrub for PII/secrets, stamp the version, update the marketplace entry. Bare `/abcd:launch` shows status+help; `dry-run` sub-verb runs the full pre-flight gate suite without writing the release artifact.
- **`/abcd:intent`** — bare quoted `/abcd:intent "<text>"` is the canonical create (spc-30/itd-46), plus the `plan` / `ready` / `audit` / `link` / `consistency` sub-verbs that ship (alongside the deprecated `new` alias; `consistency` per itd-48, which superseded itd-31), with `refine` / `grill` / `ship` / `shape` / `reclassify` remaining design targets. There is no plain `list` sub-verb — it is folded into the bare render per SD001. Manages **intents** (press-release-format intent docs at `.abcd/development/intents/{drafts,planned,shipped,disciplines,superseded}/`). `plan` promotes an intent to `planned/` and plans the work as a spec on the native spec store ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md); the companion harness `ccpm` as the deeper backend); `ship` drives that spec to completion (or the full pipeline if from drafts/); a spec-close hook reconciles standalone/bundle intents planned → shipped automatically on a successful close (spc-28 `intent_lifecycle.reconcile`); disciplines move from drafts/ to disciplines/ on plan and stay there. Bare `/abcd:intent` shows status+help.
- **`/abcd:capture`** — capture / list / promote / resolve / wontfix issues (the structured `.abcd/work/issues/` ledger). Issues live at `.abcd/work/issues/{open,resolved,wontfix}/iss-N-<slug>.md`. See itd-4. The cross-corpus synthesist (`/abcd:dredge`) comes in a later phase as itd-25.
- **`/abcd:memory`** — curate a queryable knowledge substrate (`ingest` external sources / `ask` queries / `lint` health-checks) from specs, ADRs, reviews, and memory. See itd-36 and [`05-internals/07-memory.md`](../05-internals/07-memory.md).

**Plus the `intent grill` sub-verb** (sibling of `refine` under `/abcd:intent`):

- **`/abcd:intent grill <itd-N>`** — Socratic-questioning sub-verb that stress-tests an intent (or brief section, via `--brief-section <id>`) before planning, surfacing unstated assumptions and emerging glossary terms. Sibling of `/abcd:intent refine` (gentle / user-driven). See itd-27.

**The pack/unpack model:**

```
[source repo: specs, .abcd/memory/ (or legacy memory/), ADRs, docs, code, transcripts]
        │                            READ-ONLY — the source tree is never written to
        │ /abcd:disembark <source-repo> to <dest>   (PACK)
        ▼
[lifeboat artefact at <dest>: a portable directory, out-of-tree]
   ├── README.md, press-release.md, principles.md   ← synthesised
   ├── coverage.json, coverage.md                   ← what could NOT be grounded (a blank is a result)
   ├── rescue/specs/, docs/adrs/                    ← verbatim copies
   ├── graveyard/                                   ← what was tried and failed (every interpreted entry cites evidence)
   ├── research/, audit/                            ← passes B/C outputs + audits
   └── _provenance.json                             ← (full shape in [04-surfaces/02-disembark.md § 5](../04-surfaces/02-disembark.md#5-output-shape))
        │
        │ /abcd:embark from <dest>   (UNPACK)
        ▼
[target repo: files placed at canonical locations]

  operations log (never in either repo): ~/.abcd/voyage/<source-root-sha>/
```

Lifeboats are portable directories; share by copy/tar/git, not by global archive. **The lifeboat is always *output*** — written out-of-tree to an operator-chosen `<dest>`, never back into the repo being read (adr-35); it is the latest disembark snapshot, regenerable from current state. Embark/disembark provenance and history live separately at the operator level, `~/.abcd/voyage/<source-root-sha>/` (see [`02-constraints/01-platform.md`](../02-constraints/01-platform.md) and [`04-surfaces/03-embark.md § 7`](../04-surfaces/03-embark.md#7-voyage-layout--embarkdisembark-provenance-and-history)); the lifeboat itself never accumulates past versions.

**Repo:**

- abcd lives in **one repository** ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)) — this directory. The Go binary, its user documentation, and the design record share one tree; `.abcd/**` stays in-tree and is present in every repository checkout — marketplace installs and release source archives included — never in the released binaries. `/abcd:launch ship` cuts that release from this repo, and the repo is the marketplace. There is no dev→public mirror. `abcdSubZero/` (an earlier CLI) is skimmed in Phase 0 for patterns to learn from — reference only, not a port target.

**Validation corpus:** user-maintained list in `.abcd/corpus.json`, seeded with:

- `idelphiDev/` (primary) — a mature spec corpus, a large transcript history, manual lifeboat present
- `abcdSubZero/` — a lightweight codebase with sparse specs and lighter docs (exercises adapter graceful-degradation)
- `idelphiSubZero/` — likely sparse on transcripts and specs (exercises Pass A with thin inputs, code-rescuer leaning on source)

Per-phase acceptance runs against the corpus with documented exemptions where a feature genuinely doesn't apply (e.g., the spec-essence pass on a repo with no spec store emits an empty `spec-essence.json` — the oracle should note and pass).
