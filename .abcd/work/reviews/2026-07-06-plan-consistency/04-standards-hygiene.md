# Documentation-Standards Compliance Audit — `.abcd/` and `docs/`

Specialist pass: house documentation standards (single source of truth, no git-inferable info, personal-info protection, no time estimates, present-state only, directory coverage, directory naming, cross-references, over-explanation, `docs/` skeleton). **Scope:** 213 files (210 `.md`) under `.abcd/` and `docs/`, audited against the umbrella development standards (workspace-level `CLAUDE.md`).

**Essential context found during audit:** `.abcd/work/CONTEXT.md` states the entire `.abcd/development/` record was **copied from the predecessor `abcdDev` repo and still describes the old architecture** ("It is the starting spec, not current truth, until Phase 0.5 reconciles it"). A large fraction of the findings below (stale paths, Python references in a Go repo, `roadmap/intents/` links, phase-status contradictions) are instances of that known, declared debt. They are reported anyway; the Phase 0.5 reconciliation is the natural fix vehicle. A prior in-repo hygiene audit exists at `.abcd/development/decisions/notes/fn-82-r10-docs-hygiene-closure.md` and already adjudicated several patterns (noted inline).

---

## Standard 1 — Single Source of Truth

### 1.1 Grill-history / skills-boundary text triplicated (with contradiction drift)
The "abcd ships zero skills; `/abcd:grill` was originally proposed as a skill, now a sub-verb of `/abcd:intent`" story appears near-verbatim in three files:

- `.abcd/development/brief/04-surfaces/README.md:19`
- `.abcd/development/brief/05-internals/08-skills.md:9`
- `.abcd/development/brief/05-internals/README.md:14`

**The duplication has already caused drift:** `04-surfaces/README.md` (table, lines 5–15) enumerates **nine** commands (`ahoy, disembark, embark, launch, intent, capture, memory, /abcd, reflect`), while `05-internals/08-skills.md` (~line 21) states "abcd ships **six top-level commands**". Two copies of the same enumeration, out of sync.

### 1.2 "Brief is current state" boilerplate duplicated
- `.abcd/development/brief/README.md:3`, `.abcd/development/brief/00-meta.md:3` (near-verbatim), and `.abcd/development/roadmap/README.md` ("Brief" bullet — third adapted copy). Two would be defensible as audience adaptation; three verbatim-ish copies of the same spec is copy-paste.

### 1.3 Persona list copied out of its canonical source
- `.abcd/development/personas.json` (canonical registry, 13 names) vs `.abcd/development/brief/01-product/05-personas.md:3`, which copies the full name roster. Any persona added to the JSON now requires a doc edit. The doc also references `personas.py` — a Python module that does not exist in this Go repo (stale copied detail).

### 1.4 ADR-numbering convention stated inconsistently
- `.abcd/development/README.md:23` — "ADRs use sequential `NNNN`"
- `.abcd/development/plans/README.md:4` — "unlike ADRs (stable sequential `NNNN` handles)"
- `.abcd/development/decisions/adrs/README.md` ("ADR IDs") — "ADR IDs follow the pattern `adr-N` (**unpadded**…)". Actual files are `adr-1` … `adr-20`. Two files assert a four-digit convention the canonical README and the corpus contradict.

### 1.5 Stale duplicated path facts: `roadmap/intents/`
54 occurrences reference `roadmap/intents/` (e.g. `.abcd/development/decisions/README.md:13`; `.abcd/development/decisions/adrs/README.md`; `.abcd/development/research/related-work.md:60`; `.abcd/development/research/notes/README.md`). The directory is actually `.abcd/development/intents/`. A location fact duplicated 54 times has gone stale 54 times.

### 1.6 Glossary README duplicates directory facts, now stale
- `.abcd/development/brief/glossary/README.md` — "Current contexts: `core`, `interview`" but a `distribution/` context directory exists with 3 term files; the Directory Layout block labels the root `terminology/` though the directory is `glossary/`.

### 1.7 Sanctioned exception (report only)
- `.abcd/development/research/_references.md:3` explicitly institutionalises copying: "copy the relevant entry from here into the citing document's `## References` block… the copy is accepted by convention." A deliberate, documented transclusion workaround with a declared canonical source — arguably compliant in spirit, but it is a standing licence to duplicate that the umbrella standard does not obviously permit.

---

## Standard 2 — No Git-Inferable Information

### 2.1 `created:` / `updated:` frontmatter dates — systemic (≈70 files)
Every intent file carries `created: YYYY-MM-DD` and most carry `updated: YYYY-MM-DD` in frontmatter, and the template **mandates** it:

- `.abcd/development/intents/README.md:160-161` (template block)
- All files under `intents/{drafts,planned,shipped,disciplines,superseded}/` (70 `updated:` hits corpus-wide).

`created:` is pure git-inferable metadata; `updated:` is worse — staleness-prone by construction (hand-bumped on every edit). If tooling consumes these fields, that is a design decision worth an explicit carve-out; as written it violates the umbrella rule.

- `.abcd/development/brief/04-surfaces/05-intent.md:56,227` and `04-surfaces/06-capture.md:45` embed `created: YYYY-MM-DD` in their canonical schemas — the violation is specified, not incidental.

### 2.2 ADR `date:` frontmatter — judgment: legitimate
All 20 ADRs plus the template. A decision record recording when the decision happened is legitimate historical content. Not flagged.

### 2.3 Status lines
- ADR/RFC/glossary `status:` frontmatter — lifecycle state, allowed.
- `.abcd/development/roadmap/README.md` "Current State" — "**Plugin v1** — in active design and implementation" plus "**Phase progress.** Phases 0–3 are complete… Phase 4 … is in progress". Exactly the staleness the rule predicts: it directly contradicts `.abcd/work/CONTEXT.md` ("**Phase 0 — Foundations.**" for the Go rebuild). The same file even refuses to hand-maintain counts "they drift the moment work ships" — then hand-maintains a phase-progress paragraph two lines later.
- `.abcd/development/research/notes/fn-34-flowctl-divergence-audit.md:74` — forward-looking status line, allowed.

### 2.4 Hardcoded current version numbers
- `.abcd/development/brief/glossary/_template.md:9` — `introduced_in: v0.0` (template default bakes a version literal).
- Version strings elsewhere (`04-surfaces/04-launch.md:65,123,133-135`; `glossary/distribution/version.md:49`) are *examples of the versioning mechanism* — legitimate.

### 2.5 Clean
- No `Last Updated`, no `Author:`, no `Maintained by`, no embedded licence text (the one-line adapted-from attributions in `glossary/README.md:3` and `_template.md:1` are fine).

---

## Standard 3 — Personal Information

### 3.1 Emails, keys, tokens, `$id` — CLEAN
- Email regex: zero hits. Credential patterns: zero real hits. The single AWS-shaped hit is the canonical public example key used as a test fixture (`intents/shipped/itd-28-rp-reviews-into-flow.md:97`, EXAMPLE-allowlisted). Legitimate. No `$id` fields anywhere.

### 3.2 Home-directory paths — all 9 absolute-path hits are meta-references, not real paths
Every hit is either a regex/example describing the PII *scanner* or a prior audit's record of the same (`04-launch.md:16` scanner spec, adjudicated LEGITIMATE by the prior fn-82 audit; `related-work.md:76`; `fn-82-r10-docs-hygiene-closure.md:50,52,86`; `adr-6:108`; `itd-7:37,64`; `itd-65:42`). No violation.

### 3.3 Username (the product thinker's login) — 13 hits (GitHub org/repo references)
- `work/DECISIONS.md:22` (module path), `research/notes/publish-implementation-spec.md:269,271,321`, `adr-6:37,38,187,188` (prior-art repo paths and relative links), `itd-28:75,102,120`, `itd-67:33,47,62` (marketplace coordinates).

Judgment: functional GitHub coordinates (module path, marketplace source, prior-art citations), not gratuitous authorship attribution — legitimate-but-noteworthy. The adr-6 relative links (`../../../../<personal account>/abcdZero/...`) additionally assume a specific sibling checkout layout and are broken links in any other clone.

### 3.4 `~/`-form workspace paths — ~15 hits, letter-compliant, layout-leaking
All use `~/` form (compliant), but hardcode a personal directory taxonomy:
- `research/legacy-harvest.md:3,11,12,13,21,46,67,219` (historical migration records — legitimate historical artifact)
- `research/related-work.md:60`; `research/notes/transcript-sampling.md:25`
- `brief/04-surfaces/01-ahoy.md:65` and `brief/05-internals/03-configuration.md:163,176,182,210` — though two of these explicitly argue the workspace dir "is just where a user happens to keep repos — abcd does not privilege it". The JSON examples at `03-configuration.md:176,182` would read better with a neutral example path. The prior fn-82 audit deferred one related fix; grep confirms that line no longer matches — appears since fixed.

---

## Standard 4 — No Time Estimates

### 4.1 Violations
- `.abcd/development/brief/05-internals/01-agents.md:75` — "**~5 min/agent** at v1.0.0 lock; itd-5 stays cheap."
- `.abcd/development/brief/03-evidence/04-tradeoffs.md:51` — second copy of the "~5 min" tradeoff; note `intents/drafts/itd-45-phase-1-cleanup-before-phase-2.md:66` *already records this exact violation as a known issue* — the cleanup intent exists but has not been executed (it sits in `drafts/`).
- `.abcd/development/research/notes/publish-implementation-spec.md:352-360` — "## Estimated effort for **next session**" … "a focused, **single-session** deliverable." Session-count language is duration language in a plan-shaped note. (The line-count estimates in the same block are fine.)

### 4.2 Checked and cleared (duration words describing behaviour, budgets, or history — not estimates)
- `adr-13:18,35` "killed by the 2-hour worker budget" — externally-imposed runtime budget + actual event; allowed.
- `brief/04-surfaces/02-disembark.md:82` (wake-up interval); `08-abcd.md:94` (staleness threshold); `itd-13:49`; `itd-20:34,60`; `itd-29:89` (trigger criterion); `chat-distiller.md:51` (scenario) — all product behaviour or historical record, compliant.
- Roadmap `phases/` and `plans/`: no duration language found ("day one" hits are ordering language — sequence, not duration; borderline idiom but not an estimate).

---

## Standard 5 — Present-State Only

(Exempt by design and not flagged: `brief/03-evidence/**` (explicitly retrospective), `decisions/adrs/**` + `decisions/notes/**` (decision records), `research/**` (research notes), `intents/superseded/**`.)

### 5.1 Historical framing in the brief (current-state document by its own charter — adr-5)
- `brief/04-surfaces/README.md:19` — "`/abcd:grill` **was originally proposed** as a user-facing skill but is now…"
- `brief/05-internals/08-skills.md:9` — "**An earlier version of this brief proposed**…" and (~line 40) "The **earlier** … guidance **was overturned** by the round-2 review…"
- `brief/05-internals/README.md:13` — "(Slot 7 **was previously reserved** for `07-audits.md` — retired when itd-32 was superseded by itd-31; reused for memory on 2026-05-08.)" and `:14` "**was originally proposed** as one"; `:12` "(added 2026-05-07 post-audit)".
- `brief/05-internals/06-lint.md:5` — "This section **was added post-audit (2026-05-07)** because…"; `:34` "(… **formerly pending here** — landed with…)"; pervasive "(added 2026-05-08, itd-36)" annotations and delivery narration per code (`:21-26,46,48,92,94,95,99`).
- `brief/05-internals/03-configuration.md:210` — "Anything that **was previously** 'development-wide' is **user-scoped**"; `:266` "(added post-audit 2026-05-07)".
- `brief/01-product/04-scope.md:23` — capture chronology in the scope doc; git covers this.
- `brief/06-delivery/03-out-of-scope.md:74-76` — two full paragraphs of capture/supersession history; "for the brief's history" is precisely what adr-5 and the umbrella standard say the brief must not carry.
- `brief/04-surfaces/05-intent.md:319` — "The **historical** `status: …` field … **has been retired**; templates and existing files **were stripped in the 2026-05-08 sweep**" — plus repeated "per the 2026-05-08 directive" framing (`:222,316,319`).
- `roadmap/phases/phase-0-substrate.md:178` — "itd-34 **was previously folded into** this phase…"; `phases/phase-1-ahoy.md:70` — "The capture surface (itd-4) **was previously**…". Phase docs are sequencing docs, not decision records; borderline — "was previously folded" is change-narration.

### 5.2 Systemic (known, declared)
The whole `development/` tree describes the predecessor Python/flow-next/RepoPrompt architecture while the repo is the Go rebuild. `.abcd/work/CONTEXT.md` declares this openly. Not re-itemised per file.

---

## Standard 6 — Directory Coverage (missing `README.md`)

17 directories lack a README:

| Directory | Files | Assessment |
|---|---|---|
| `.abcd/work/` | 2 | **Gap.** Non-trivial tier; purpose only described one level up and in `AGENTS.md`. |
| `.abcd/development/research/` | 3 root files + 3 subdirs | **Gap.** The tree's largest, most heterogeneous area; `research/notes/` has an excellent README but the parent has none. `notes/README.md` even points at sibling dirs (`phase/`, `adr/`) that don't exist here. |
| `.abcd/development/research/prompting/` | 1 + `agents/` | **Gap** (non-trivial substructure). |
| `.abcd/development/research/spikes/` | 1 | **Gap**, low priority. |
| `.abcd/development/research/notes/fn-25-closeout/` | 2 | Marginal — parent README explains the subdirectory convention; acceptable. |
| `brief/01-product/`, `02-constraints/`, `03-evidence/`, `06-delivery/` | 5/4/4/3 | **Gap.** Siblings `04-surfaces/` and `05-internals/` both have index READMEs. `brief/README.md:20` openly hedges: "each folder's `README.md` — **where present** — indexes its files". Inconsistent within one document set. |
| `intents/{drafts,planned,shipped,disciplines,superseded}/` | 39/8/17/3/3 | Acceptable — `intents/README.md` has a "Lifecycle Directories" table; per-bucket READMEs would duplicate it (standard 1 wins). |
| `glossary/{core,distribution,interview}/` | 10/3/2 | Acceptable — `glossary/README.md` defines the bounded-context-per-directory rule (though it fails to list `distribution`). |

`docs/` is fully covered (all five directories have READMEs).

---

## Standard 7 — Directory Naming

No clear violations. Singular-entity dirs are singular (`brief/`, `roadmap/`, `glossary/`, `research/`, `work/`); collections are plural (`decisions/`, `adrs/`, `notes/`, `intents/`, `plans/`, `principles/`, `phases/`, `rfcs/`, `spikes/`, lifecycle buckets). `docs/` uses Diátaxis-canonical names — upstream framework convention, not a violation. Only note: `research/prompting/` (gerund) is neither pattern but reads as a single conceptual area — acceptable.

---

## Standard 8 — "Related Documentation" Cross-References

**Pattern: 16 of 210 markdown files (~8%) have a "Related Documentation" section.** The convention is essentially not practised; where footers exist they follow *local* conventions:

- **Intents** end with `## References` (well-linked) — a parallel convention doing the same job under a different heading.
- **ADRs**: only 4 of 20 (`adr-6`, `adr-13`, `adr-16`, `adr-17`) have Related Documentation.
- **The entire `brief/` tree**: 1 of ~40 files (`05-internals/10-in-session-dispatch.md`). Brief files cross-link heavily inline instead.
- **`docs/`**: 0 of 6 (stubs; light inline links only).
- **Top-level READMEs**: none.
- Consistent adopters: `roadmap/README.md`, `roadmap/{phases,rfcs}/README.md`, newer research notes, `research/prompting/agents/*`.

**Bidirectionality spot-checks:** `roadmap/README.md` ↔ `phases/README.md` — bidirectional, good. `research/notes/design-principles-sota.md` → `related-work.md` — **one-way**. `docs/README.md` → `.abcd/development/` — effectively bidirectional via prose. **Worst offenders:** the `brief/` tree and 16 of the 20 ADRs.

---

## Standard 9 — Over-Explanation (top offenders only)

1. `brief/04-surfaces/05-intent.md` (458 lines) — the same worked example (the `intent-capture-discipline` bundle retirement) is told in full **three times** (lines 43, 316, plus `06-lint.md:46`); the `IL013` no-status rationale restated at `:222` and `:319`.
2. `brief/05-internals/06-lint.md` (236 lines, single 1,000+-char paragraphs) — each lint code carries full reservation-history, delivery narration, and deferral rationale inline; a registry table plus one-line rationale would do.
3. `intents/README.md` (361 lines) — dedicated "Why press releases instead of feature specs", "Why unpadded", and "why no `intents/decisions/` directory" digressions. Classic long "why not X" material.
4. `brief/00-meta.md` — four-point numbered rationale for folders-not-one-file; "keep rationale to a sentence" applies.
5. `brief/05-internals/08-skills.md` — the closing "the earlier guidance was overturned because…" paragraph is rationale-narration of a superseded position (doubles as a standard-5 hit).
6. `decisions/adrs/adr-9-phase-as-product-layer.md` (261 lines) — long even for an ADR, amendment-upon-amendment delivery notes.

(`research/related-work.md` at 300+ lines is genuinely comparative research — appropriate depth, not flagged.)

---

## Standard 10 — `docs/` Diátaxis Skeleton

**Verdict: the strongest area of the corpus. Internally consistent, mutually consistent, and accurate.**

- The four stubs share a uniform template: one-line Diátaxis-type definition + boundary statement pointing at the sibling that owns the excluded content. Matches `docs/README.md`'s table exactly.
- `docs/README.md` ↔ `.abcd/README.md` agree on the split.
- The generated-CLI warning is consistent between `reference/README.md` and `reference/cli/README.md` (adapted, not copy-pasted). The referenced source path `internal/surface/cli/` **exists** (verified), and "Cobra command tree" matches `work/DECISIONS.md`.
- Terminology vs the `.abcd` brief: "transport-agnostic core" and "host-delegated design" match `work/CONTEXT.md`/`DECISIONS.md` (current architecture) *and* `principles/README.md`. **One watch-item:** `explanation/README.md` also names "the abstraction boundary", which `work/CONTEXT.md:31` lists among the *old*-architecture concepts pending Phase 0.5 reconciliation. Flag for that pass.
- Minor: none of the docs/ stubs carries a "Related Documentation" footer, and `explanation/README.md`'s deep relative link into `.abcd/development/decisions/adrs/` will 404 on any docs-site render that excludes `.abcd/` (which, per `.abcd/README.md`, is exactly what the release artifact does).

---

## Mechanical check summary

| Check | Result |
|---|---|
| Emails | **0 hits** |
| Absolute home-dir paths | 9 hits, all scanner-spec/audit-record meta-references (§3.2) |
| `Last Updated` / `Author:` / `Maintained by` | **0 hits** |
| `created:` frontmatter | ~70 files (intents corpus + templates) (§2.1) |
| `updated:` frontmatter | 70 hits (§2.1) |
| Duration regexes | 3 real violations (§4.1); remainder behaviour-spec false positives (§4.2) |
| Credentials / `$id` | 0 real hits (1 allowlisted AWS example fixture) |
| Licence text | 0 hits |

## Highest-leverage fixes (by repair value)

1. Execute the already-drafted cleanup intent `intents/drafts/itd-45-phase-1-cleanup-before-phase-2.md` — it self-identifies the time-estimate and stale-path violations.
2. Reconcile the 9-vs-6 command-count contradiction and de-triplicate the grill history to one canonical home.
3. Global sweep of `roadmap/intents/` → `intents/` (54 refs) alongside the Phase 0.5 reconciliation.
4. Decide the `created:`/`updated:` frontmatter question explicitly (tooling carve-out or removal) — the largest single class of standard-2 hits, template-mandated, so it grows with every new intent.
5. Add the four missing `brief/` section READMEs (or drop the "where present" hedge) and a `research/README.md`.
