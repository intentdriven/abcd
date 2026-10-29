# Cross-Document Consistency Review — `.abcd/`

Specialist pass: drift and contradictions BETWEEN document families (brief, ADRs, intents, roadmap, work-state, research). All paths relative to `.abcd/` unless stated. Line numbers from files as reviewed.

---

## 0. The umbrella finding (context for everything below)

### F0. The entire `development/` record describes a different project than `work/` says this repo is — and adr-5 makes that a formal contradiction — **CRITICAL**

- `development/decisions/adrs/adr-5-brief-is-current-state.md:31`:
  > "**The brief is the current state of the project.** One canonical version. Always reflects today."
- `development/brief/README.md:3`:
  > "The brief reflects the project's *current* state — it is not versioned, snapshotted, or archived"
- `development/roadmap/README.md:14`:
  > "**Brief**: lives at `.abcd/development/brief/README.md` (canonical, current)."
- vs `work/CONTEXT.md:30-34`:
  > "The copied `.abcd/development/` record still describes the *old* architecture (flow-next required, overlay/abstraction-boundary, two-repo launch, RP/codex oracles). It is the starting spec, not current truth, until Phase 0.5 reconciles it. **Do not treat it as authoritative before then.**"

Why it's drift: adr-5 (accepted, unrevoked) asserts the brief is always current truth; `work/CONTEXT.md` asserts the opposite. Nothing in the brief, roadmap, or ADR corpus carries any marker of this suspension — a reader entering via `development/` (which calls itself canonical in three places) has no way to discover it is pre-reconciliation material. Either adr-5 needs a superseding/amending ADR, or the brief README needs the Phase 0.5 caveat. The filesystem confirms `work/` is right: the repo root is a Go module (`cmd/`, `internal/`, `go.mod`); there is **no** `.flow/`, no `scripts/`, no `skills/`, no `hooks/`, no `agents/` — yet `development/brief/02-constraints/01-platform.md:7` states:
> "The cwd is the private dev repo where abcd is built. The plugin lives at the repo root (`commands/`, `skills/`, `hooks/`, `scripts/`, `agents/`, `tests/`, `docs/`); spec/task tracking lives at `.flow/`"

Many findings below are instances of this umbrella drift; they are still listed individually because each is a concrete edit target for the Phase 0.5 reconciliation.

---

## 1. ADR ↔ brief

### F1. Eight architecture-shaping decisions in work/DECISIONS.md have no ADR, despite the file's own graduation rule — **HIGH**

- `work/DECISIONS.md:3-4`:
  > "Architecture-shaping decisions graduate to an ADR under `../development/decisions/adrs/`."
- `work/DECISIONS.md:8-23` records (all 2026-07-06): "Rebuild abcd from scratch in Go, no external tools (specstory, RepoPrompt, flow-next, Ralph, codex)"; "Transport-agnostic Go core"; "LLM work host-delegated by default"; "Spec/task layer native-minimal; … flow-next dropped. Autonomous run not a Ralph port"; "Single repo, curated release (no dev→public mirror)"; "Module path github.com/<personal account>/abcd-cli; Cobra approved".
- The newest ADR is `adr-20` dated 2026-07-03 (`development/decisions/adrs/README.md:135`). No adr-21+ exists.

Why it's drift: these are precisely the "hard to reverse, surprising without context, real trade-off" decisions the ADR bar (`adrs/README.md:12-15`) describes — a full-language rewrite, dropping four load-bearing tools, and a repo-topology reversal — yet none is promoted. Until promoted, the ADR corpus (all 20 `status: accepted`) formally asserts the opposite architecture.

### F2. Accepted ADRs whose subject matter was voided by the 2026-07-06 decisions are not marked superseded/deprecated — **HIGH**

All 20 ADRs are `status: accepted` with `superseded_by: null` (verified by frontmatter grep). But:

- `adr-6` (RP review storage), `adr-8` (dual-backend RP + Codex review), `adr-17` (`rp chat-send` override) — voided by `work/DECISIONS.md:8-9` ("no external tools (… RepoPrompt, … codex)").
- `adr-13`/`adr-16` (fn-38 memory writer; Ralph post-iteration autodrain) — voided by "flow-next dropped. Autonomous run not a Ralph port" (`work/DECISIONS.md:16-17`).
- `adr-18` (launch payload / lifeboat gate), `adr-19` (plugin.json version carve-out), `adr-20` (manifest lockstep, dated three days before the rewrite decision) — at minimum need re-affirmation under the single-repo, Go-binary model (`work/DECISIONS.md:18-19`: "Single repo, curated release (no dev→public mirror); the repo is the marketplace").

The `adrs/README.md:42` lifecycle has exactly the right state for this (`deprecated` — "the decision no longer applies but no successor replaces it") and it is used zero times.

### F3. adr-17's claimed supersession — verified consistent, with one caveat — **INFO**

adr-17 supersedes a **spec-level** decision, not an ADR, and says so explicitly (`adr-17:15-18`): "this ADR supersedes a **spec-level** decision (fn-33 Cluster J / AC-J1 …), not a prior ADR — so the frontmatter `supersedes` chain field stays `null`". Frontmatter matches (`supersedes: null`, line 6). Internally consistent. Caveat: the superseded artifact (`.flow/specs/fn-33…`) does not exist in this repo (`.flow/` absent), so the back-link is unverifiable here.

### F4. adr-7 has no YAML frontmatter at all — breaks the ADR format contract — **MEDIUM**

- `adrs/README.md:50-51`: "Every ADR has frontmatter (machine-readable) plus a Markdown body"; `adrs/README.md:100`: "`intent_lint.py` extends to verify these reciprocally."
- `adr-7-grill-skill-and-glossary.md:1-3` opens directly with `# ADR-7: …` / `**Status:** Accepted` — no `id:`, `status:`, `supersedes:`, `related_intents:` fields. It is the only ADR of the 20 in this shape. Any machine pass over ADR status will silently skip adr-7.

### F5. Phase-0 doc cites "adr-02 / ADR-02 (MCPBridge contract)" — no such ADR; adr-2 is something else — **MEDIUM**

- `development/roadmap/phases/phase-0-substrate.md:143-144`: "**ADRs realised:** adr-02 (MCPBridge contract — fn-5's implementation contract)" and `:114`: "(the harness interface, ADR-02)".
- `adrs/README.md:117`: adr-2 is "Three intent kinds (standalone / bundle-member / discipline)". The MCPBridge contract actually lives at `development/research/notes/02-mcpbridge-implementation-contract.md` (a research note, not an ADR).
- Also violates the ID convention: `adrs/README.md:29` "ADR IDs follow the pattern `adr-N` (unpadded…)" — "adr-02" is padded.

### F6. Two files claim ADRs use `NNNN` padded IDs; the ADR README says unpadded `adr-N` — **MEDIUM**

- `development/README.md:23`: "ADRs use sequential `NNNN` (stable cross-reference handles)"
- `development/plans/README.md:4-5`: "unlike ADRs (stable sequential `NNNN` handles)"
- vs `development/decisions/adrs/README.md:29`: "ADR IDs follow the pattern `adr-N` (unpadded, mirrors `itd-N` / `rfc-N` / `fn-N`)" — and every actual file is `adr-N-<slug>.md`.

### F7. notes/README defines the ADR bar as the exact opposite of adrs/README — **LOW**

- `development/decisions/notes/README.md:5-7`: "An **ADR** … records a settled architecture decision … for a choice that is **reversible in principle**." And `:15-16`: "If a note's subject later hardens into a **reversible** architecture choice, promote it to an ADR."
- vs `adrs/README.md:13`: "**Hard to reverse** — the cost of changing the decision later is meaningful."

Almost certainly a typo for "irreversible", but as written the two doors point opposite ways.

---

## 2. Intent lifecycle

### F8. intents/README.md's own directory listings contradict the filesystem wholesale — **HIGH**

adr-3 makes directory location the truth, and the README restates it (`intents/README.md:61`: "Directory location is the single source of truth for lifecycle state"). The README's mirror listings then get nearly everything wrong:

- **Planned** (`intents/README.md:321-325`): "One intent currently planned: `itd-6-rp-mcp-only-integration.md`". Reality: `intents/planned/` holds **8** intents — itd-20, itd-24, itd-63, itd-65, itd-66, itd-67, itd-69, itd-72 — and itd-6 is in `intents/shipped/`.
- **Shipped** (`intents/README.md:356-361`): lists only itd-27 and itd-28. Reality: `intents/shipped/` holds **17** files (itd-2, 3, 4, 6, 27, 28, 34, 36, 40, 42, 46, 47, 48, 49, 52, 53, 58).
- **Drafts** (`intents/README.md:261-296`): lists itd-2, itd-3, itd-4 ("drafts") — all three are in `shipped/`; lists itd-20, itd-24 — both in `planned/`; lists itd-34, itd-36, itd-40 — all in `shipped/`; and omits ~15 intents actually in `drafts/` (itd-43, 44, 45, 50, 51, 54, 55, 56, 57, 59, 60, 61, 62, 64, 70).
- Footnote ¹ (`:298`): "itd-2 … sits in `drafts/` with `spec_id: null` until that replan happens" — itd-2 is in `shipped/`.

The **Superseded** listing (`:339-341`, itd-31 + itd-32) and **Disciplines** listing (`:307-311`, itd-1/5/37) do match the filesystem.

### F9. Numbering: gaps 38, 68, 71 — one explained, two unexplained; no duplicates; plus a "never renumbered" rule with an unreconciled recorded exception — **MEDIUM**

- The corpus spans itd-1..itd-72. Missing IDs: **38, 68, 71**. No number appears in two lifecycle folders (verified across all five directories).
- itd-38 is documented as deliberately released: `intents/disciplines/itd-37-modification-grammar.md:78` ("itd-38 ID released, not reserved") and `brief/03-evidence/04-tradeoffs.md:43-47`.
- **itd-68 and itd-71 have zero mentions anywhere under `.abcd/`** (grep across all `*.md`). No release note, no supersession record — unexplained holes in a scheme whose README promises capture-stable IDs.
- `intents/README.md:25`: "IDs are assigned in capture order and **never renumbered**" — vs `roadmap/phases/phase-3-intent.md:125-127`: "Duplicate `itd-45` resolved by fn-31 on 2026-05-30: the drift-detector intent was **renumbered** to `itd-49`". The rule admits no exception and the exception cites no rule amendment.

### F10. Shipped intents not reflected: brief's out-of-scope list still carries five shipped intents as future work, and its "physically in drafts/" claims are all false — **HIGH**

`brief/06-delivery/03-out-of-scope.md` is the brief's canonical "later phase" enumeration:

- `:14-19` (exclusion-list comment): "the phased-in IDs that are STILL physically in drafts/ (lifecycle move pending): itd-2,3,4,7 … and itd-34,36,40,42". Reality: itd-2, 3, 4, 34, 36, 40, 42 are **all in `shipped/`** (only itd-7 remains in `drafts/`).
- `:62` "~~itd-46~~ … Draft retained pending the `drafts/` → `shipped/` lifecycle move" and `:64` same for itd-48 — both files are already in `intents/shipped/`.
- `:63` "itd-47 — fn-12's oracle-backed gates pass honestly … **(not yet shipped)**" — `intents/shipped/itd-47-fn-12-oracle-gates-autonomous-mode.md` exists.
- `:65` itd-49, `:68` itd-52, `:69` itd-53 listed as later-phase draft items — all three are in `intents/shipped/`.
- `:32`: "the bullet list is kept in lockstep with the command output" — demonstrably not.

### F11. Shipped intents belong to no phase, though the model says phase `## Scope` defines what ships together — **MEDIUM**

Phase docs' `## Scope` sections collectively name: itd-1, 2, 3, 4, 5, 6, 7, 27, 28, 34, 36, 37, 40, 42, 43. Shipped intents **itd-46, itd-47, itd-48, itd-49, itd-52, itd-53, itd-58** appear in no phase doc. Per `phases/README.md:77-78` "an intent listed in no phase doc's `## Scope` is implicitly **unscheduled**" — yet these are shipped. `roadmap/README.md:103` promises "Each major capability … gets a corresponding shipped intent **as its phase closes**"; a third of the shipped corpus shipped outside any phase.

### F12. intents/README lifecycle machinery describes dropped tooling — **MEDIUM** (umbrella-F0 instance)

The entire lifecycle (`intents/README.md:75-133`) is defined in terms of `/flow-next:plan`, `/flow-next:work`, `.flow/specs/`, `intent_lifecycle_hook`, `intent_lint.py` — vs `work/DECISIONS.md:16-17` ("flow-next dropped") and the absence of `.flow/` and `scripts/` in the repo. The bidirectional-link contract (`:141-144`, "`.flow/specs/fn-N-<slug>.md` | `intent: itd-N`") points at files that cannot exist here.

---

## 3. Roadmap phases ↔ brief build sequence

### F13. Phase names/order: consistent by design, but three unrelated "Phase N" axes now coexist — **MEDIUM**

- The roadmap's product phases (`phases/README.md:60-65`: 0 Substrate, 1 ahoy, 2 capture, 3 intent, 4 lifeboat, 5 round-trip) and the brief's plumbing phases (`brief/06-delivery/01-build-sequence.md` §2-§10) **are declared deliberately distinct**: `01-build-sequence.md:7`: "Two numberings, deliberately distinct. … They are not the same axis and not meant to align one-to-one." Within that framing, names and order agree — no drift between these two.
- **But** `work/CONTEXT.md:18` introduces a third axis: "**Phase 0 — Foundations.** Scaffolding the repo: … the Go core + CLI + plugin surface skeleton" — which is neither the roadmap's "Phase 0 — Substrate & disciplines" nor the brief's "Phase 0 — Foundation (fn-1)". And `work/CONTEXT.md:23` names "**Phase 0.5** — a full up-front reconciliation" which exists in no phase doc, no roadmap index — only in `work/CONTEXT.md` and the two stub READMEs (`development/plans/README.md:8`, `development/principles/README.md:10`).

### F14. Roadmap "Current State" vs work/CONTEXT "Current phase" — flatly incompatible — **HIGH**

- `development/roadmap/README.md:47-49`:
  > "**Phase progress.** Phases 0–3 are complete … Phase 4 (the lifeboat pipeline) is in progress, opening with the entry-verification spec fn-49."
  and `:38`: "**Plugin v1** — in active design and implementation."
- vs `work/CONTEXT.md:18-21`:
  > "**Phase 0 — Foundations.** Scaffolding the repo … Exit: `make preflight` green and a verb round-tripping CLI → core → JSON."

Two authoritative-sounding "where are we" statements disagree about both the phase number and the phase system. The roadmap's stale-proofing commands make it worse: `roadmap/README.md:59-64` instructs `scripts/ralph/flowctl specs` — `scripts/` does not exist in this repo (and Ralph/flow-next are dropped per `work/DECISIONS.md`), so the "read status live" mechanism the dashboard leans on is dead.

### F15. Roadmap README's intent-bucket command uses a path that doesn't exist — **MEDIUM**

- `development/roadmap/README.md:71-72`: `"$(ls .abcd/development/roadmap/intents/$b/itd-*.md 2>/dev/null | wc -l)"`
- Intents live at `development/intents/` (no `development/roadmap/intents/` exists). The command silently prints 0 for every bucket — a "never hand-kept, derived live" counter that always reads zero. Part of a stale-path family; see F21.

---

## 4. work/CONTEXT.md and work/DECISIONS.md vs everything

Covered substantively by F0, F1, F2, F13, F14. Two additions:

### F16. Brief dependency constraints directly contradict recorded decisions — **HIGH** (the sharpest single ADR/brief-vs-work quote pair)

- `development/brief/02-constraints/02-dependencies.md:28-29`:
  > "**`flow-next`** — preferred provider for `/flow-next:plan`, `/flow-next:work` … abcd never reimplements that surface." / "**`RepoPrompt`** (RP) — preferred oracle backend"
- vs `work/DECISIONS.md:8-9`: "Rebuild abcd from scratch in Go, **no external tools (specstory, RepoPrompt, flow-next, Ralph, codex)**" and `:16-17`: "Spec/task layer **native-minimal** … **flow-next dropped**."

Also `02-dependencies.md:34` bans "Direct API integrations … the only transports are RP MCP, Codex CLI subprocess, and in-session subagent" — vs `work/DECISIONS.md:14-15`: "LLM work host-delegated by default; native/CLI/**API**/MCP oracles are opt-in adapters" (the brief's banned list is the new decision's adapter list).

### F17. Launch surface promotes to a public sibling repo; decisions say single repo, no mirror — **HIGH**

- `development/brief/04-surfaces/README.md:10`: "`/abcd:launch` | Promote `*Dev` → public sibling repo"
- `development/roadmap/phases/phase-5-roundtrip.md:8-9`: "run `/abcd:launch` to promote a `*Dev` repo to its public sibling"; `:21-22`: "promotes `abcdDev` → public `abcd`"
- `development/brief/02-constraints/01-platform.md:7`: "Curated snapshots ship to the public `abcd/` repo via `/abcd:launch`."
- vs `work/DECISIONS.md:18-19`: "**Single repo, curated release (no dev→public mirror)**; the repo is the marketplace."

---

## 5. personas.json vs brief personas

### F18. Persona roster matches; the picker implementation reference is stale — **LOW**

- Names: identical 13 (Alice…Maya) in `development/personas.json:6-18` and `development/brief/01-product/05-personas.md:3`. Role hints in the brief's parenthetical are a consistent subset. The PII convention matches in both. **No roster drift.**
- But `05-personas.md:3`: "**`personas.py`** picks at random" (also `brief/06-delivery/02-verification-matrix.md:40`) — no `scripts/` or Python exists in this repo, and `development/README.md:18-19` already carries the updated framing: "`personas.json` — … (**migrates to embedded Go data** when the intent surface is built)". The brief was not updated to match.

---

## 6. README accuracy vs actual directory structure

### F19. plans/ and principles/ are README-only stubs — acknowledged in the stubs, not in the parent index — **LOW**

- `development/README.md:12-15` lists `principles/` and `plans/` as if populated. Reality: each contains only `README.md`. The stubs do self-acknowledge (both defer to "the Phase 0.5 content reconciliation") but the parent table gives no hint. Notably these two stubs are the only `development/` files already written for the *new* architecture ("transport-agnostic core", "host-delegated by default") — they agree with `work/DECISIONS.md` while their parent README's other rows describe the old world.

### F20. .abcd/README and development/README structural claims — mostly accurate — **INFO**

- `README.md:7-13` three tiers: `development/` ✓, `work/` ✓, `.work.local/` absent on disk but declared gitignored, so absence is legal. `../AGENTS.md` exists ✓.
- `development/README.md` table rows all point at existing folders. Main inaccuracies are F6 (NNNN) and F19 (stub status).

### F21. Stale path family: `roadmap/intents/` (and `../phases/`) — the intents tree moved but half the cross-references didn't — **MEDIUM**

Intents live at `development/intents/`; there is no `development/roadmap/intents/`. Stale references:

- `development/decisions/adrs/README.md:23`: "User-facing capability — those are intents (`../roadmap/intents/`)" — resolves to nonexistent path. Also `:22` RFCs at "(`../roadmap/rfcs/`)" — resolves to nonexistent `decisions/roadmap/rfcs/` (real: `../../roadmap/rfcs/`).
- `development/roadmap/rfcs/README.md:48`: "Intents live at `.abcd/development/roadmap/intents/`".
- `development/roadmap/README.md:71-72` (the `ls` command, F15).
- `development/brief/06-delivery/03-out-of-scope.md:3` and `:21` (runnable command returns nothing).
- `development/brief/06-delivery/01-build-sequence.md:29`; `:3` and `:11` link-text says "`roadmap/intents/README.md`" (href correct — label stale).
- `development/decisions/README.md:12`: link text "`../roadmap/intents/`" with href `../intents` (correct target, stale label).
- `development/intents/README.md:25` and `:236`: text says "the phase docs at [`../phases/`]" — from `intents/` that is nonexistent; the href `../roadmap/phases` is correct.

### F22. roadmap/README links a nonexistent `activity/` directory — **LOW**

- `development/roadmap/README.md:123`: "[activity/](../activity/) — curated-from-volatile-sources artefacts" — no `development/activity/` exists. (The brief describes `activity/` as created at runtime by dev-sync, but linking it as an existing sibling is wrong today.)

### F23. brief/README's tree annotation says 00-meta holds an "archive policy" that adr-5 abolished — **LOW**

- `development/brief/README.md:24`: "`00-meta.md` # naming convention, **archive policy**, structure rationale"
- `development/brief/00-meta.md:30-32` contains only the negation: "## No archive directory — … Per adr-5". The annotation invites a reader to expect the mechanism adr-5 deleted.

---

## 7. Surfaces ↔ phases ↔ intents

### F24. Two of nine surfaces (`/abcd`, `/abcd:reflect`) are delivered by no phase and no plumbing phase — while the brief says one of them already shipped — **HIGH**

- Surface inventory: `brief/04-surfaces/README.md:5-15` lists 9 surfaces. Phase coverage from `## Scope`/`## Milestone` sections: ahoy → phase-1; capture → phase-2; intent → phase-3; disembark + memory → phase-4; embark + launch → phase-5. **`/abcd` (08) and `/abcd:reflect` (09) appear in no phase doc** and in no plumbing section of `brief/06-delivery/01-build-sequence.md`.
- Their intents sit in `planned/` (itd-20, itd-24), i.e. specs in flight — contradicting `phases/README.md:77-78` (the model equates unphased with a `drafts/` bench item, not with in-flight planned work). The same applies to the other unphased planned intents itd-63, itd-65, itd-66, itd-67, itd-69, itd-72.
- Sharper still: the brief already records reflect as **shipped** — `brief/05-internals/01-agents.md:22`: "`reflection-composer` | reflect | **shipped — fn-83 (thin V1)**" and `:32`: "The roster **reached 16** when fn-83 added `reflection-composer`". So a surface exists in the brief as shipped, its intent (itd-24) is only `planned/`, and no phase anywhere owns it. Three families, three different answers for one surface.

### F25. Phase-3 still claims itd-27/itd-28 sit in `planned/`; phase-0 in the same folder says the reconciliation completed — **LOW**

- `roadmap/phases/phase-3-intent.md:110-111`: "itd-27 and itd-28 **currently sit in `intents/planned/`** with plan-reviewed specs (`fn-3`, `fn-2`)."
- vs filesystem (both in `shipped/`) and `phase-0-substrate.md:93-95`: "… fn-48 backfilled the lifecycle state across closed specs; **the directories are now reconciled**."

---

## Summary table

| # | Severity | Finding | Primary files |
|---|---|---|---|
| F0 | Critical | adr-5 "brief is current state" vs CONTEXT.md "not authoritative until Phase 0.5"; brief describes a repo layout that doesn't exist | adr-5:31, brief/README.md:3, work/CONTEXT.md:30-34, brief/02-constraints/01-platform.md:7 |
| F1 | High | Eight 2026-07-06 architecture decisions never graduated to ADRs | work/DECISIONS.md:3-23, adrs/README.md:135 |
| F2 | High | ADRs voided by the rewrite (adr-6/8/13/16/17/18/19/20) all still `accepted`, `deprecated` state unused | adrs/*, work/DECISIONS.md:8-19 |
| F3 | Info | adr-17 spec-level supersession internally consistent | adr-17:6,15-18 |
| F4 | Medium | adr-7 has no frontmatter | adr-7:1-3, adrs/README.md:50 |
| F5 | Medium | phase-0 cites nonexistent "adr-02 (MCPBridge)" | phase-0-substrate.md:114,143 |
| F6 | Medium | "NNNN" ADR-ID claim vs unpadded `adr-N` | development/README.md:23, plans/README.md:4, adrs/README.md:29 |
| F7 | Low | notes/README says ADRs are for "reversible" choices; adrs/README says "hard to reverse" | notes/README.md:5-16, adrs/README.md:13 |
| F8 | High | intents/README planned/shipped/drafts listings contradict filesystem (1 vs 8 planned; 2 vs 17 shipped) | intents/README.md:261-361 |
| F9 | Medium | Gaps itd-68/71 unexplained (itd-38 explained); itd-45→itd-49 renumbering vs "never renumbered" | intents/README.md:25, phase-3-intent.md:125, itd-37:78 |
| F10 | High | out-of-scope lists 5 shipped intents as future; all "still in drafts/" claims false | 03-out-of-scope.md:14-69 |
| F11 | Medium | 7 shipped intents in no phase `## Scope` | phases/*, phases/README.md:77 |
| F12 | Medium | Intent lifecycle defined on dropped flow-next tooling | intents/README.md:75-144, work/DECISIONS.md:16 |
| F13 | Medium | Three coexisting "Phase N" axes; "Phase 0.5" absent from roadmap | build-sequence.md:7, work/CONTEXT.md:18-23 |
| F14 | High | Roadmap "Phase 4 in progress" vs CONTEXT "Phase 0 Foundations"; dead `flowctl` status commands | roadmap/README.md:38-64, work/CONTEXT.md:18 |
| F15 | Medium | Roadmap bucket-count command globs nonexistent path | roadmap/README.md:71 |
| F16 | High | Brief mandates flow-next/RP; DECISIONS drops both; banned-transports list inverted | 02-dependencies.md:28-34, work/DECISIONS.md:8-17 |
| F17 | High | Launch = dev→public mirror vs "single repo, no mirror" | 04-surfaces/README.md:10, phase-5:8-22, work/DECISIONS.md:18 |
| F18 | Low | Personas match; `personas.py` reference stale vs Go-migration note | 05-personas.md:3, development/README.md:18 |
| F19 | Low | plans/ and principles/ are stubs; parent index doesn't say so (stubs do) | development/README.md:12-15 |
| F20 | Info | Top-level READMEs otherwise structurally accurate | README.md, development/README.md |
| F21 | Medium | Stale `roadmap/intents/` path family across 7 files | adrs/README.md:22-23, rfcs/README.md:48, out-of-scope:3,21, build-sequence:29 |
| F22 | Low | roadmap links nonexistent `activity/` | roadmap/README.md:123 |
| F23 | Low | brief/README advertises an "archive policy" adr-5 abolished | brief/README.md:24, 00-meta.md:30 |
| F24 | High | `/abcd` + `/abcd:reflect` surfaces in no phase; brief says reflect shipped while itd-24 is planned; 8 planned intents unphased | 04-surfaces/README.md:14-15, 05-internals/01-agents.md:22,32, phases/README.md:77 |
| F25 | Low | phase-3 says itd-27/28 in planned/; they're shipped and phase-0 says reconciled | phase-3-intent.md:110, phase-0-substrate.md:93 |

**Recommended reconciliation order:** F0/F1/F2 first (either write the Go-rebuild ADR set and mark the voided ADRs `deprecated`, or stamp the brief/roadmap with the Phase 0.5 not-yet-authoritative banner CONTEXT.md already implies); then the mechanical truth-restorations F8/F10/F14 (regenerate listings from the filesystem); then the path sweep F21/F15; the rest are single-line edits.
