---
id: spc-13
slug: disembark-reads-a-repo-s-naming-and-internals-conventions-in
intent: itd-96
---
# disembark-reads-a-repo-s-naming-and-internals-conventions-in

## Summary

spc-13 delivers itd-96: two conventions-tier `Source` adapters that ground
`constraints/naming` and `internals` on the files a team naturally writes — a
`NAMING.md` or naming page under `docs/` for naming, an `ARCHITECTURE.md` or
architecture/explanation tree plus the package layout for internals. A
repository with conventional docs but no `.abcd/` record now carries both
sections in its lifeboat instead of two blanks. `glossary` is untouched:
`convGlossarySource` already owns it, and this spec adds no second adapter for
it.

## Scope

- **Adapters** (`internal/core/lifeboat/sources_conventions.go`):
  `convNamingSource` (`constraints/naming`) and `convInternalsSource`
  (`internals`), both `TierConventions`, both siblings of `convGlossarySource`.
- **Registration** (`conventionSources()` in `probe.go`): one line each. With
  spc-12's `convOpenQuestionsSource` this brings the conventions tier to
  fourteen adapters.
- **Tier gate** (`hasConventions` in `probe.go`): the new adapters' own name
  lists are appended to the candidate union, per the comment that already
  forbids narrowing the gate below what the adapters read — otherwise a repo
  carrying *only* a `NAMING.md` or `ARCHITECTURE.md` would have the whole Tier-1
  set skipped and blank falsely.
- **Depends on spc-12** for `(*SourceContext).WalkFiles`, which
  `convInternalsSource` uses for its layout scan. spc-13 stacks on spc-12; it
  does not re-land the primitive.
- **No mapping change, no new dependency.**

## Approach

### `convNamingSource` — `constraints/naming`

Preference order, first match wins:

1. A dedicated naming document — `ctx.FindFirst("NAMING.md", "NAMING",
   "naming.md")`, then the first `docs/` entry whose lower-cased name starts
   with `naming` (the same `ListDir`-prefix idiom `convGlossarySource` uses for
   `docs/glossary*`). Cited alone, `ConfidenceMedium`.
2. **Fallback: the glossary document.** A project that never wrote a naming
   registry usually encodes its reserved vocabulary in its glossary, so the
   glossary is real evidence for naming — but weaker, because a glossary
   defines terms rather than ruling on what may be renamed. Cited as
   `"<path> (glossary fallback — no dedicated naming document)"`,
   `ConfidenceLow`.
3. Neither → a blank naming both search sets in `Searched` and asking the human
   question.

**Status is `StatusPartial` in both non-blank cases** — the ceiling the mapping
row predicts for this section, and the honest one: neither a naming page nor a
glossary enumerates a project's full reserved vocabulary. The strength of the
signal is carried by `Confidence` (medium for a dedicated document, low for the
glossary fallback), not by inflating the status.

**Distinctness from `glossary` is structural, not incidental.** The two
adapters declare different `Section()` values, so the orchestrator indexes them
into different coverage rows and neither can displace the other. On a
glossary-only repository the report shows `glossary` partial/medium cited to
`GLOSSARY.md`, and `constraints/naming` partial/low cited to the same file
*with the fallback qualifier* — visibly a weaker, derived reading rather than a
duplicate row. A test asserts the two rows differ in confidence and in cited
string on that fixture.

### `convInternalsSource` — `internals`

Two independent signals, combined:

- **Architecture prose.** `ctx.FindFirst("ARCHITECTURE.md", "ARCHITECTURE",
  "architecture.md", "docs/architecture.md")`, then the directories
  `docs/architecture`, `docs/design`, `docs/explanation` (Diátaxis), in that
  preference order. A file is read and measured with the existing
  `convProseBytes` / `convGroundedProseBytes` threshold — the same measure
  `convContextSource` uses for a README, reused rather than re-invented. A
  directory counts as prose evidence when it holds at least one Markdown file.
- **Package layout.** `WalkFiles` from `.`, keeping the top-two path segments
  of every file under a recognised source root (`internal`, `pkg`, `src`,
  `lib`, `cmd`, `app`) — the packages a rescuer must navigate. Cited as
  `"<root>/<pkg>/"` entries, capped at `maxLayoutCitations` (50) with the
  overflow reported as a count. `WalkFiles`'s `truncated` flag, when set, is
  reported in the citation so a capped scan never reads as a complete one.

Outcomes:

| Signals present | Status | Confidence |
|---|---|---|
| Architecture prose above the threshold (± layout) | `StatusPartial` | `ConfidenceHigh` |
| Architecture doc present but thin, or a doc directory with no prose measured | `StatusPartial` | `ConfidenceMedium` |
| Layout only, no architecture doc | `StatusPartial` | `ConfidenceLow` |
| Neither | `StatusBlank` with `Searched` + `Question` | — |

As with naming, `StatusPartial` is the ceiling: an `ARCHITECTURE.md` plus a
package listing describes the shape of a system, not its internals chapters.
This matches the mapping row's conventions prediction, which stays unedited.

### Resolved open questions (itd-96 § Open Questions)

| Question | Decision |
|---|---|
| Naming vs glossary — what is naming's distinct source? | A **dedicated** naming document (`NAMING.md`, `docs/naming*`) is naming's primary source; the glossary is an explicitly-qualified **fallback** at lower confidence. The two sections are distinct rows with distinct evidence strings; no second `glossary` adapter is added. |
| Which paths map to `internals`? | `ARCHITECTURE.md` (and spellings) → `docs/architecture.md` → `docs/architecture/` → `docs/design/` → `docs/explanation/`, in that preference order, plus the package layout as an independent second signal. |
| What counts as "package layout", and the missing primitive? | Top-two path segments under `internal`, `pkg`, `src`, `lib`, `cmd`, `app`, gathered with spc-12's `WalkFiles`. A bounded recursive walk is needed (a top-level `ListDir` cannot see `internal/core/lifeboat`), and it is the shared primitive both intents asked for — landed once, in spc-12. |
| Extraction heuristics and status thresholds | The existing `convProseBytes` ≥ `convGroundedProseBytes` threshold separates real architecture prose from a stub. Status ceilings at `StatusPartial` for both sections (the mapping contract's conventions prediction); the distinction lives in `Confidence`. |
| Confidence levels | Naming: medium (dedicated doc) / low (glossary fallback). Internals: high (real prose) / medium (thin doc) / low (layout only). |
| Reserved-vocabulary spellings | `NAMING.md`, `NAMING`, `naming.md`, and any `docs/` entry whose name starts with `naming` (case-insensitive). Heading-shape sniffing inside a glossary is *not* attempted — it would guess at a convention that has no standard. |

## Acceptance-criteria satisfaction

- **Naming docs → non-blank, cites the file** — a fixture with `NAMING.md`
  asserts `constraints/naming` partial, `TierConventions`, citing `NAMING.md`.
- **ARCHITECTURE.md + layout → non-blank, cites both** — a fixture with
  `ARCHITECTURE.md` and an `internal/<pkg>/` tree asserts `internals` non-blank
  citing the doc and the layout entries.
- **Neither → honest blanks** — a bare fixture asserts both sections blank with
  populated `Searched` and a non-empty `Question`.
- **Read-only** — the byte-for-byte tree-invariance test from spc-12 covers the
  whole probe, both new adapters included.
- **No duplicate `glossary` adapter** — a glossary-only fixture asserts
  `glossary` is still grounded by `convGlossarySource`, that
  `constraints/naming` is partial with the fallback qualifier and lower
  confidence, and that `conventionSources()` contains exactly one adapter whose
  `Section()` is `glossary`.
- **Both tiers present → one deterministic result** — the unchanged
  `beats`/`tierRank` reduction; the native adapters keep winning where a record
  exists, and the report names the winning tier.
