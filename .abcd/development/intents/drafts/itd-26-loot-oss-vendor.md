---
id: itd-26
slug: loot-oss-vendor
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-36]
severity: minor
related_issues: [iss-2609211905346507]
builds_on: [itd-25]
related_rfcs: [rfc-1]
---

<!-- 2026-05-08: brought forward to align with README's command shape table.
     The original later-scheduling rationale (gated on dredge usage data + demonstrated demand) is
     preserved as a plan-review consideration: when /abcd:intent plan itd-26 runs, the planner
     should verify dredge has accumulated enough usage to justify the loot surface. -->


# Raid the Open Ocean, Bring the Cargo Home with Papers

## Press Release

> **abcd makes it fun *and* responsible to vendor code from public OSS repositories.** A new `/abcd:loot` command clones selected files from a GitHub repo into the consuming project's `vendor/<source>/` directory, records full provenance — origin URL, licence text, commit SHA, files taken, rationale — into `.abcd/work/loot/<source>.md`, and refuses to proceed when the licence is incompatible with the consuming project's declared licence stance. The pirate verb is intentional: "loot" prompts the licence-check reflex that "import" or "vendor" wouldn't, while the structured provenance trail means future you (or future auditors) can always answer "where did this code come from, why is it ours now, and are we allowed to keep it?".
>
> "I kept finding small, perfect snippets in OSS repos — a 30-line scheduler here, a clever retry-with-jitter there — and the choice was always between adding a heavy dependency for one function or copy-pasting and forgetting where it came from," said Bob, full-stack engineer. "I run `/abcd:loot https://github.com/example/scheduler --files src/jitter.py` and abcd vendors the file, writes a provenance card with the MIT licence text inline, and commits everything as one operation. Six months later I can run `/abcd:loot` and see every external bit I've absorbed, with SHAs and licences. The name made me smile, then made me actually read the LICENSE file."

## Why This Matters

Three patterns repeat across abcd-managed projects:

1. **Useful OSS snippets get copy-pasted without provenance.** A clever helper from a CC0 repo and a slightly-rewritten function from a GPL repo end up looking identical in the consuming codebase. By the time licence questions arise, the trail is cold.
2. **Heavy dependencies are pulled in for tiny needs.** Adding `lodash` for one utility, or `requests` for one HTTP call, because the alternative ("copy 30 lines, attribute properly") feels like more work than `npm install`. abcd's other principles (sovereignty, committed artefacts, structured ledgers) point toward vendoring — but the tooling doesn't make vendoring easy enough to compete with package managers.
3. **The verb shapes the behaviour.** `import` and `vendor` are mechanical verbs that don't prompt any reflexive consideration. `loot` carries a faint "are you sure you have rights to this?" undertone that pulls licence-checking up from "thing I should probably do" to "thing the verb is asking me about". Playfulness is doing real compliance work.

This intent ships the missing surface: a single command that makes the right thing (vendor with full provenance + licence verification) easier than the wrong thing (copy-paste-and-forget or add-a-package-for-one-function). The maritime metaphor pairs cleanly with `/abcd:dredge` (own-corpus salvage) — `loot` is the public-corpus counterpart, deliberately raid-coded rather than salvage-coded because the action genuinely is opportunistic gathering from outside, not rescue from within.

This is also a chance to formalise the `vendor/` convention abcd projects should use: vendored OSS code lives at `vendor/<source>/`, with `.abcd/work/loot/<source>.md` as the provenance card. Both committed; both first-class.

## What's In Scope

- **`/abcd:loot` command** with subverbs:
  - `loot <github-url> [--files <glob>...] [--branch <ref>]` — clone the source repo (sparse if `--files` given), display licence + a manifest of what would be copied, ask for confirmation (transparent prompt per § 6.1), copy files into `vendor/<source>/<sub-path>/`, write provenance card to `.abcd/work/loot/<source>.md`, stage both for commit. Single atomic operation.
  - `loot search <query> [--license <permissive|copyleft|any>] [--max <N>]` — search GitHub for repos matching the query; prefer `flow-next:github-scout` if installed, internal fallback otherwise; rank by stars × recency × licence-permissiveness; offer to loot the top result interactively.
  - `loot list [--source <source>]` — show all looted entries in the current project (driven by reading `.abcd/work/loot/`).
  - `loot update <source>` — refresh a looted source to a newer commit; computes diff, asks for confirmation if local modifications exist, appends a "Refreshed at SHA → SHA on YYYY-MM-DD" line to the provenance card. Re-runs the licence check against the *current* policy at refresh time (catches the case where policy tightened between original loot and refresh).
  - `loot remove <source>` — delete `vendor/<source>/` and move the provenance card to `.abcd/work/loot/archived/<source>.md` (don't delete the historical record).
  - `loot verify [--source <source>]` — re-fetch licence text from the source repo at the recorded SHA and assert it still matches the provenance card's recorded licence (catches retroactive licence changes in the upstream).
  - `loot policy add <SPDX-id> --rationale <text>` — amend `.abcd/config.json` → `loot.acceptable_licences`. The *only* pathway to make a previously-refused loot pass; see "Licence compatibility check" below.
  - `loot policy list` — show current acceptable-licence list and amendment history.
- **Provenance card schema** (owned by the `abcd loot` verb in `internal/core/loot`, frontmatter + Markdown body):
  - Frontmatter: `source` (URL), `vendor_path`, `commit_sha`, `branch_or_tag`, `licence_spdx`, `licence_path_in_source`, `looted_at` (date), `rationale` (one-line), `files` (list of source-paths with byte-counts), `refreshed_at` (list of date+SHA pairs), `local_modifications` (boolean — true once any vendored file is touched in-tree).
  - Body: full licence text inlined, then the looter's notes ("why we wanted this", "what we changed", "what we left out").
- **Licence compatibility check (non-circumventable)**:
  - Project declares its licence stance in `.abcd/config.json` → `loot.acceptable_licences = ["MIT", "Apache-2.0", "BSD-3-Clause", "CC0", ...]` (set by ahoy with permissive defaults; user can edit).
  - `loot` refuses to proceed when source licence isn't on the list. **There is no per-loot bypass.** No `--override`, no `--force`, no global mode (e.g. a future "pirate mode" — see RFC-1) circumvents this gate. The check is part of the loot operation, not a confirmation around it.
  - The only pathway past a refusal is to **amend the policy**: `loot policy add <SPDX-id> --rationale <text>` updates `.abcd/config.json` → `loot.acceptable_licences` with a structured rationale recorded in `.abcd/work/loot/_policy-amendments.md` (one entry per amendment: SPDX-id added, date, rationale, who-added). Subsequent loots of code under that licence then pass the unchanged gate against the amended policy.
  - **Why no escape hatch:** per-loot overrides get used reflexively after the third or fourth time a developer hits the refusal; per-policy amendments require editing a file the developer will re-read every time the policy is consulted. Friction is *placed* deliberately at the policy layer (where it generates real deliberation about what licences fit the project's stance), not at the per-loot layer (where it would just become the path of least resistance).
  - **Hard principle**, lifted to brief § 6 as a universal pattern when this intent is `plan`ned: *abcd's licence checks are non-circumventable; the only pathway past a refusal is to amend the policy explicitly.*
- **`vendor/` convention codified**:
  - `vendor/<source>/` is the canonical landing zone for looted code (regardless of language).
  - Per-language adapters for "where the rest of the project is" remain unchanged; loot doesn't try to merge into existing source trees, it lands code in `vendor/` and lets the consumer wire imports.
  - `.gitignore` semantics: `vendor/` is **always committed** (loot's whole point is committed provenance). No visibility-driven exception.
- **`loot-quartermaster` agent** (16th agent — provisional name; the loot-equivalent of `lifeboat-oracle`):
  - Audits the proposed loot operation before commit: confirms licence is correctly identified, manifest matches what was copied, no unwanted files (e.g., the source repo's `.git/`, build artefacts, secrets) crept into `vendor/<source>/`.
  - Oracle backend chain: host-delegated by default, with the opt-in RepoPrompt adapter and Codex CLI as configured alternatives.
  - Failure aborts the loot operation (atomic; nothing is left half-vendored).
- **Brief § 5 reserved-commands table update** (already landed): row for `/abcd:loot` reflecting raid-frame + provenance scope.
- **Provenance/licence substrate reuse from itd-36** (added 2026-05-08). The licence-detection / citation-generation / source-hash-registry / launch-gate-licence-gate machinery itd-26 needs is the **same** machinery memory ingest needs. itd-36 ships the substrate as a separable spec at [`05-internals/09-provenance-substrate.md`](../../brief/05-internals/09-provenance-substrate.md); itd-26 sits on top — no re-implementation, no fragile bespoke licence layer. Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge); one shared substrate. The cross-consumer registry (`.abcd/memory/.sources_index.json`) is shared; the `consumer` field disambiguates which verb produced each entry.

## What's Out of Scope

- **Replacing package managers.** Loot is for vendoring small focused snippets and single-purpose utilities, not for managing graphs of versioned dependencies. `npm` / `pip` / `cargo` continue to handle that.
- **Cross-host support beyond GitHub** (GitLab, Bitbucket, Codeberg, sourcehut, etc.). The initial implementation is GitHub-only; other hosts are itd-N-future when demand emerges.
- **Looting from private repos**. Authentication, rate-limiting, and per-org policy are large surfaces. The initial implementation is public-only; the verb's rationale (loot the open ocean) reinforces the scope.
- **Automatic security scanning of looted code.** gitleaks / Presidio scanning of the *consumer* project at launch time (per the native launch scan) covers the looted code transitively. No per-loot security scan is added; that's a possible itd-N extension.
- **Dependency resolution / transitive looting.** If the looted code itself depends on another OSS snippet, loot doesn't follow the chain. The user decides what to vendor.
- **Auto-updating loot when upstream changes.** No watcher, no scheduled refresh. `loot update` is explicit and manual. Drift between looted SHA and upstream HEAD is intentional — that's the point of vendoring.
- **A treasure-themed alias.** `/abcd:treasure` was deliberately *not* reserved in the reserved-commands work. Loot is the canonical OSS-vendor verb; no alias.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-installed repo, **when** the user runs bare `/abcd:loot`, **then** the dispatcher shows recently vendored sources, current `loot.acceptable_licences` from `.abcd/config.json`, and suggested next actions (per the universal bare-command-as-help convention) — no vendoring runs without an explicit URL argument or sub-verb.
- **Given** a project with `loot.acceptable_licences = ["MIT", "Apache-2.0", "BSD-3-Clause", "CC0"]` in `.abcd/config.json`, **when** the user runs `/abcd:loot https://github.com/example/scheduler --files src/jitter.py` against an MIT-licensed source, **then** the file is copied to `vendor/scheduler/src/jitter.py`, a provenance card is written to `.abcd/work/loot/scheduler.md` with full licence text inlined, and both are staged for commit as one atomic operation.
- **Given** the same project, **when** the user attempts to loot from a GPL-3.0 source NOT on the acceptable list, **then** the operation refuses BEFORE copying any file, displays the offending licence + the project's current acceptable list, and points to `loot policy add` as the only pathway — no `--force`, no `--override`, no per-loot bypass exists.
- **Given** a refused loot, **when** the user runs `loot policy add GPL-3.0 --rationale "needed for embedded copyleft component"`, **then** `.abcd/config.json` is updated, an entry is appended to `.abcd/work/loot/_policy-amendments.md` with date + rationale + actor, and re-running the original loot command now succeeds.
- **Given** an existing looted source, **when** the user runs `loot update <source>` with local modifications present in `vendor/<source>/`, **then** the command surfaces the local diff vs. upstream, requires explicit confirmation, re-runs the licence check against the *current* policy, and appends a "Refreshed at SHA → SHA on YYYY-MM-DD" line to the provenance card.
- **Given** the user runs `loot verify --source scheduler`, **when** the command executes, **then** it re-fetches the licence text from the recorded source URL at the recorded SHA and asserts byte-identity with the licence inlined in the provenance card — any mismatch is reported as a "retroactive licence change" finding.
- **Given** a `loot-quartermaster` agent is invoked before commit, **when** the audit runs, **then** the agent confirms the licence is correctly identified, the manifest matches what was copied, no `.git/` or build artefacts crept in, and any irregularity aborts the loot atomically (nothing left half-vendored). Note: the brief's agent count of 15 is preserved; `loot-quartermaster` bumps the count to 16, recorded in the brief changelog when this intent ships.
- **Given** two looted sources collide on default vendor path (`vendor/bar/`), **when** the second loot is attempted, **then** the command applies the org-namespace strategy (`vendor/foo/bar/` and `vendor/baz/bar/`) automatically OR — if the user prefers — accepts an explicit subpath via `--vendor-path <subpath>` to override.
- **Given** `/abcd:launch ship` runs against a project with recent `loot policy add` amendments, **when** the gatekeeper inspects the launch payload, **then** the policy amendment history is surfaced as a weak warning ("recent licence-stance expansion") so the public sibling repo's release notes can reflect the change.

## Open Questions

- **Default `--files` behaviour.** When the user runs `loot <url>` without `--files`, do we default to (a) cloning the whole repo into `vendor/<source>/` (simple, but often pulls 95% irrelevant code), (b) refusing without `--files` (forces explicit selection but adds friction), or (c) interactively offering a file tree for selection (best UX, most implementation cost)? Probably (c), with (b) as fallback when no TTY.
- **Licence detection accuracy.** Many repos have a clear `LICENSE` / `LICENSE.md` file; many don't, and rely on a `License: MIT` line in `package.json` or `Cargo.toml`. Need a layered detector: file → package metadata → SPDX-classifier on README → fail-closed and ask the user. The initial implementation ships layers 1–2; layer 3 is itd-N.
- **What about CC-licenced *content* (docs, schemas, prompts)?** abcd itself might want to loot prompt patterns or schema definitions. Same command, or a cousin like `/abcd:loot --kind docs`? Probably same command — provenance card schema is the same — but a `kind: code|docs|schema|prompt` frontmatter field categorises the loot for future querying.
- ~~**Licence override audit trail.**~~ Resolved by 2026-05-04 design lock: there is no per-loot override. Policy amendments via `loot policy add` are the only pathway; rationales are mandatory and recorded in `.abcd/work/loot/_policy-amendments.md`. Open sub-question: should `/abcd:launch`'s gatekeeper surface recent policy amendments as a warning before public release? Probably yes, weakly — the public sibling repo's prospective community deserves visibility into "this project recently expanded its licence stance".
- **Interaction with `code-rescuer` agent.** code-rescuer (narrowed to principle extraction) and loot-quartermaster (proposed) overlap conceptually — both deal with "extract from source code". Distinction: code-rescuer pulls *principles* (no code lifted), loot pulls *code* (with provenance). Worth a §-level note in the brief.
- **Vendor-path collision strategy.** What if the user loots `https://github.com/foo/bar` then later loots `https://github.com/baz/bar`? Default vendor path `vendor/bar/` collides. Options: (a) namespace by org → `vendor/foo/bar/` and `vendor/baz/bar/`, (b) collision detected → prompt for explicit subpath, (c) refuse second loot until first is removed/renamed. Probably (a) as default with (b) as override.
- ~~**Does loot deserve to be a later command, or fold in sooner?**~~ **Resolved 2026-05-08** — brought forward to align with README's command shape. Plan-review verifies dredge has accumulated enough usage data before starting work on loot.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
