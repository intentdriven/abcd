# Legacy Harvest — v0 Scaffolding to the abcd Plugin

This document is the audit trail for migrating accumulated abcd v0 scaffolding (`~/ABCDevelopment/.claude/`, `~/.abcd/`, the `~/.claude/commands/abcd.md` symlink) into the abcd plugin's defaults. Every decision is recorded with verdict and reasoning. Once the plugin ships and per-repo migration is complete, the listed source paths can be archived.

This is a one-time record, not a living document.

## Scope

Five sources reviewed:

1. **Methodology** — `~/ABCDevelopment/.claude/CLAUDE.md` (978 lines, 16 sections)
2. **Skills** — `~/ABCDevelopment/.claude/skills/` (12 skill packages)
3. **Agents** — `~/ABCDevelopment/.claude/agents/` (7 agents)
4. **Templates** — `~/.claude/templates/` (24 files)
5. **v0 runtime state** — `~/.abcd/` (~1.6 GB; runtime data, scripts, schemas)

Decision interview conducted 2026-05-04 in conversation. Outcomes captured below.

## Pass 1 — Methodology rules

Source: `~/ABCDevelopment/.claude/CLAUDE.md`. Disposition: split into CARL-style domain rules per [`itd-3-modular-rules-loader`](../intents/shipped/itd-3-modular-rules-loader.md), shipped as `scripts/abcd/defaults/rules.json` in the plugin.

| § | Section | Verdict | Domain mapping | Notes |
|---|---|---|---|---|
| 1 | Single Source of Truth | **Keep** | `DOCUMENTATION` | Universal discipline; recall on `docs`, `readme`, `tutorial`, `ADR` |
| 2 | Personal Information Protection | **Keep (high priority)** | `PII` | Couples to plugin's gitleaks/Presidio/TruffleHog stack |
| 3 | Working Files Directory (.work/) | **Keep, retarget** | `ARTEFACTS` | `.work/` for scratch, `.abcd/development/activity/` for curated |
| 4 | Mandatory Issue Recording | **Keep, retarget** | `ISSUES` | Scratch → `.work/issues.md`; curated → `.abcd/development/activity/issues/` via dev-sync. Couples to `itd-4` (capture) and `issue-scout` |
| 5 | Documentation Structure Standards | **Keep decision tree only** | `DOCUMENTATION` | The work-to-do vs work-done framing survives; the prescriptive directory layout (Diátaxis-flavoured) drops |
| 6 | Feature-Centric Roadmap Standard | **Drop** | — | Superseded by intent system (`.abcd/development/intents/`) |
| 7 | No Time Estimates | **Drop** | — | Implicit in press-release intent format; no separate enforcement needed |
| 8 | Milestone Completion Requirements | **Drop file mandates; reframe** | — | Devlogs become flow-next epic completion records (plan-sync output). Retrospectives land in `/abcd:reflect` (a later phase, see [`itd-24`](../intents/planned/itd-24-reflect-command.md)). Time logs dropped entirely |
| 9 | Singular vs Plural Naming | **Keep** | `NAMING` | Low-cost rule; prevents bikeshedding |
| 10 | Complete Directory Coverage (README per dir) | **Drop** | — | Busywork without enforcement layer |
| 11 | Cross-Reference Standards | **Drop** | — | Bidirectional linking decays without tooling |
| 12 | Available Slash Commands | **Drop** | — | All v0 commands superseded by `/abcd:*` plugin |
| 13 | Available Agents (catalogue) | **Drop as section** | — | Agent disposition handled in Pass 3 |
| 14 | Documentation Footer Standards | **Keep** | `DOCUMENTATION` | Folds into Section 1's SSOT discipline |
| 15 | Documentation Workflow | **Drop** | — | Process advice, not rule-shaped |
| 16 | Quick Reference / Configuration Hierarchy / Scope Boundaries | **Drop** | — | Stage-hierarchy (Planning/Autonomous/Collaborative/Validation) is the v0 shape being retired |

**Domains established:** PII (high-priority), COMMITTING, DOCUMENTATION, ARTEFACTS, ISSUES, INTENTS, NAMING, LIFEBOAT.

## Pass 2 — Skills

Source: `~/ABCDevelopment/.claude/skills/` (12 packages). Disposition: 3 skills survive (consolidated to 2 native `SKILL.md` files in `skills/`); rest drop or land in a later phase.

| Skill | Verdict | Destination |
|---|---|---|
| `commit-attribution` | **Keep** | Plugin skill `skills/commit-attribution/SKILL.md`; activates on commit-related triggers |
| `pii-protection` | **Keep, consolidate** | Merged with `secret-scan` into `skills/secrets-and-pii/SKILL.md` |
| `secret-scan` | **Keep, consolidate** | Merged into `skills/secrets-and-pii/SKILL.md` |
| `devlog-reminder` | **Drop** | Devlogs become flow-next epic records (Pass 1 §8) |
| `documentation-standards` | **Drop as skill** | Folded into `DOCUMENTATION` domain rules |
| `governance` | **Later phase** | v0 bounded-autonomy framework not load-bearing now |
| `metrics` | **Drop** | DORA metrics inappropriate for plugin-development context |
| `sig-verify` | **Later phase** | Couples to [`itd-16-hash-chain-merkle-audit`](../intents/drafts/itd-16-hash-chain-merkle-audit.md) |
| `vap-verify` | **Later phase** | Same audit-machinery cluster |
| `spec-export` | **Later phase** | Captured as [`itd-23-spec-kit-interop`](../intents/drafts/itd-23-spec-kit-interop.md) |
| `spec-import` | **Later phase** | Same |
| `test-reminder` | **Drop** | Modern Claude handles test discipline internally; re-emerges as domain rule if needed |

**Net:** 12 skills → 2 plugin `SKILL.md` files (`commit-attribution`, `secrets-and-pii`).

## Pass 3 — Agents

Source: `~/ABCDevelopment/.claude/agents/` (7 agents). Disposition: 1 new plugin agent file; 2 capabilities folded into existing agents; 4 dropped.

| Agent | Verdict | Destination |
|---|---|---|
| `documentation-auditor` | **Keep as 15th plugin agent** | Subagent-only role; invoked by `/abcd:disembark` (audit pre-pack), `/abcd:embark` (audit post-scaffold), `/abcd:launch` (audit pre-public-promotion). Not directly user-invoked. No `/abcd:audit` command yet |
| `documentation-architect` | **Capability folded into embark + launch** | Embark gains user-doc-structure design at scaffolding time; launch gains user-doc update/extension at public-promotion time. No standalone agent file |
| `codebase-security-auditor` | **Capability folded into launch-gatekeeper** | OWASP-checking + vuln scanning becomes part of launch-gatekeeper's pre-promotion checks (alongside PII/secret scanning via tools) |
| `architecture-advisor` | **Drop** | Opus-tier general-purpose; no abcd-specific use case |
| `git-documentation-committer` | **Drop** | Overlaps modern Claude commit flow + retired `/commit` command |
| `task-decomposer` | **Drop** | Superseded by flow-next (`/flow-next:plan` → `/flow-next:work` → plan-sync) |
| `ux-architect` | **Drop; revisit in a later phase** | Niche for now; revisit alongside `itd-22-harness-portability` and a future Typer CLI |

**Net:** 7 agents reviewed → 1 new plugin agent file (`documentation-auditor`); 2 existing agents (`embark-scaffolder`, `launch-gatekeeper`) gain extended responsibilities; 4 dropped. `intent-fidelity-reviewer` also gains a retrospective output mode for major intents (Pass 4 retrospective decision).

Brief amendment: § 12 (agents) updates from 14 to 15. `embark-scaffolder` and `launch-gatekeeper` descriptions extended to reflect doc-architecture and security-auditing capabilities.

## Pass 4 — Templates

Source: `~/.claude/templates/` (24 files). Disposition: most drop; transparency-of-process content retargeted; project bootstrap comes in a later phase.

| Template | Verdict | Destination |
|---|---|---|
| `ai-contributions.md.template` | **Auto-generate, no template** | `dev_sync.py` reads `.abcd/logbook/*.jsonl` and emits `AI-CONTRIBUTIONS.md` at lifeboat-creation time. Always-current, lifeboat-portable |
| `devlog.md.template` | **Drop file; harvest prompt structure** | Per Pass 1 §8: devlogs become flow-next plan-sync output. Template's narrative/challenges/highlights structure folds into plan-sync's output prompt |
| `retrospective.md.template` | **Later phase** | Captured as [`itd-24-reflect-command`](../intents/planned/itd-24-reflect-command.md) — `/abcd:reflect` for major-milestone retrospectives. `intent-fidelity-reviewer` also gains retrospective output mode for major shipped intents |
| `pre-implementation-checklist.md` | **Drop** | flow-next task structure + intent acceptance criteria (`itd-1`) replace |
| `post-implementation-checklist.md` | **Drop** | Same |
| `manual-test-script.md.template` | **Drop** | Not load-bearing now; corpus tests + golden-test fixtures cover plugin's own testing |
| `lineage/README.md` | **Drop file; concept lives in lifeboat** | Lifeboat *is* the lineage artefact |
| `feature-spec.md.template` | **Drop** | Superseded by intent format |
| `use-case.md.template` | **Harvest user-story prompt** | Folded into `/abcd:intent new` interview: "as a {persona}, I want to {action}, so that {benefit}" feeds the press release. Persona drawn from `.abcd/development/personas.json` |
| `milestone.md.template` | **Drop** | abcd uses hand-written phase docs (`roadmap/phases/phase-N-*.md`), each ending in a milestone; template-driven flow not needed |
| `project-CLAUDE.md.template` | **Heavily rewrite** | Becomes `scripts/abcd/defaults/claude-md-marker-block.md` — ~30-line marker block per `itd-3`, not the legacy 5305-byte template. Written by `/abcd:ahoy` |
| `project-README.md.template` | **Drop** | Project READMEs are project-specific; generic template rarely fits |
| `config.json.template` / `inherits.json.template` / `settings.local.json.template` | **Drop** | v0 abcd config schema; superseded by the `.abcd/config.json` schema in plugin |
| `python-project/`, `typescript-project/`, `go-project/`, `rust-project/`, `docker-project/`, `docker-sandboxed-project/`, `cloud-sandbox/` | **Drop entirely** | ahoy works on existing code only; empty-repo language scaffolding is out of scope. Modern toolchains (`uv init`, `cargo init`, `npm create`, `go mod init`) handle this better than abcd reinventing it |
| `roadmap-structure/` | **Drop** | v0 directory layout |
| `decisions/` (empty) | **Drop** | Empty |
| `personal/` (empty) | **Drop** | Empty |
| `version-sync-hook.py` | **Drop** | Versioning-scheme-specific to v0 |
| `README.md` | **Drop** | Directory goes away |

**Net:** 24 files → 0 templates ship in plugin defaults. Three transparency-of-process artefacts re-emerge as runtime-generated outputs (`AI-CONTRIBUTIONS.md` via dev-sync, devlog content via plan-sync, retrospective content via auditor and future `/abcd:reflect`).

## Pass 5 — v0 runtime state

Source: `~/.abcd/` (~1.6 GB). Disposition: drop runtime data; harvest a few patterns already lifted into the brief; archive historical content.

### Runtime data — drop with v0

| Item | Size | Verdict |
|---|---|---|
| `logging.db` | 1.3 GB | **Drop** — replaced by per-repo `.abcd/logbook/*.jsonl` |
| `logs.db` | 256 MB | **Drop** — same |
| `project/` (638 entries) | 5.2 MB | **Drop** — per-project state replaced by per-repo `<repo>/.abcd/` |
| `segments/` | 2.5 MB | **Drop** — F-031 segmentation superseded by Claude's native session storage |
| `conversations.yaml`, `projects.json` | 144K | **Drop** — superseded by Claude's native projects index |
| `hook-debug.log`, `segment-errors.log` | 16K | **Drop** — v0 debugging output |
| `sessions/`, `v3/`, `hashes/`, `keys/`, `ai-transparency/`, `config.yaml`, `session.yaml`, `.current-session` | small | **Drop** — v0 runtime state |

### Already harvested as patterns (no further action)

| Source | Where it lives now |
|---|---|
| `~/.abcd/schemas/session-log.schema.json` | Brief § "concrete session-log schema" — JSONL format, `outcome` enum, `token_usage`, categorised `actions`, `~/`-relative paths, `agent_model` field |
| `~/.abcd/scripts/rp-review-to-json` | Brief § "RP review parsing tolerance" — `review-collator` parser absorbs format variation |
| `~/.abcd/scripts/process-transcript.py` | Pattern reused in plugin's `hooks/session_log_hook.py` |

### Lands in a later phase

| Source | Coupled later-phase intent |
|---|---|
| `audit-export.py`, `audit-query.py`, `hash-chain.py`, `sign.py`, `keygen.py`, `uuid7.py`, `jcs.py` | [`itd-16-hash-chain-merkle-audit`](../intents/drafts/itd-16-hash-chain-merkle-audit.md) |
| `audit.schema.json`, `vap-record-types.json` | Same |
| `spec-export.py`, `spec-import.py` | [`itd-23-spec-kit-interop`](../intents/drafts/itd-23-spec-kit-interop.md) |
| `model-effectiveness.json`, `model-scorecard.jsonl` | [`itd-17-model-effectiveness-tracking`](../intents/superseded/itd-17-model-effectiveness-tracking.md) — referenced as v0 empirical seed; the tracker resets and rebuilds |

### Archive (move outside active tree, don't delete, don't harvest into plugin)

Move to `~/Archive/abcd-v0-archive-2026-05-04/`:

- `~/.abcd/archive/` (642 entries, 10 MB) — historical session archives
- `~/.abcd/reviews/` (227 entries, 900K) — past code review outputs
- `~/.abcd/ai-transparency/` — per-project transparency logs
- `~/.abcd/model-effectiveness.json`, `~/.abcd/model-scorecard.jsonl` — referenced by `itd-17`

Reason: low probability of need; non-zero probability of utility. Archiving (not deleting) preserves the option without cluttering active tooling.

### Drop entirely (no harvest, no archive)

`init-db.sql`, `install-hooks.py`, `metrics.py`, `mlx-compare`, `mlx-start`, `multi-judge`, `review_locally`, `synthesize-review`, `test-qwen235b-judge`, `compare_reviews`, `test-segment-capture.py`, `test-stop-hook.py`, `current-task.schema.json`, `segment.schema.json`, `ai-recording.schema.json`, `log-database.sql`, `metrics.sql`.

## Synthesis: plugin-defaults structure

Final shape of harvested content in the plugin:

```
abcdDev/
├── scripts/abcd/
│   ├── defaults/
│   │   ├── rules.json                         # 8 domains harvested from Pass 1
│   │   └── claude-md-marker-block.md          # Pass 4 — heavily rewritten from project-CLAUDE.md.template
│   └── schemas/
│       ├── rules.schema.json                  # validates per-repo rules.json
│       ├── config.schema.json                 # already in brief
│       └── session-log.schema.json            # Pass 5 — lifted from ~/.abcd/schemas/
│
├── skills/
│   ├── commit-attribution/SKILL.md            # Pass 2 — harvested
│   ├── secrets-and-pii/SKILL.md               # Pass 2 — consolidated from pii-protection + secret-scan
│   └── (5 plugin-internal: abcd-ahoy, abcd-disembark, abcd-embark, abcd-launch, abcd-intent — already in brief)
│
├── agents/                                    # 15 agents (was 14 in brief)
│   └── documentation-auditor.md               # Pass 3 — new; subagent-only
│   # documentation-architect capabilities folded into embark-scaffolder + launch-gatekeeper
│   # codebase-security-auditor capabilities folded into launch-gatekeeper
│   # intent-fidelity-reviewer gains retrospective output mode (Pass 4)
│
└── hooks/
    └── prompt_router_hook.py                  # NEW — CARL-style rule injector per itd-3
```

## Per-repo override mechanism

`<repo>/.abcd/rules.json`:

```json
{
  "version": 1,
  "extends": "plugin-defaults",
  "overrides": {
    "<DOMAIN>": {
      "state": "active|dormant",
      "rules_append": ["..."],
      "rules_replace": ["..."],
      "recall_append": ["..."]
    }
  },
  "custom_domains": {
    "<NEW_DOMAIN>": { "state": "...", "recall": ["..."], "rules": ["..."] }
  }
}
```

Merge semantics, conflict policy, and diagnostic surface (`abcd rules show`, `abcd rules diff`, `abcd rules lint`) defined in [`itd-3-modular-rules-loader`](../intents/shipped/itd-3-modular-rules-loader.md).

## Migration sequence

After the plugin ships:

1. `/abcd:ahoy` in each existing project writes `<repo>/.abcd/rules.json` skeleton + CLAUDE.md marker block. Plugin defaults pick up the work the legacy CLAUDE.md was doing.
2. Verify rules-loader behaves correctly across the validation corpus (`idelphiDev`, `abcdSubZero`, `idelphiSubZero`).
3. Move `~/ABCDevelopment/.claude/` → `~/Archive/abcd-v0-archive-2026-05-04/abcd-claude/`.
4. Move `~/.abcd/archive/`, `~/.abcd/reviews/`, `~/.abcd/ai-transparency/`, `~/.abcd/model-effectiveness.json`, `~/.abcd/model-scorecard.jsonl` → `~/Archive/abcd-v0-archive-2026-05-04/`.
5. Delete `~/.abcd/` (excluding the moved content) and `~/.claude/commands/abcd.md` symlink.
6. Reclaim ~1.6 GB.

No `/abcd:cull` command needed — this is one-shot manual work that doesn't need re-running.

## Decisions captured during interview

| Question | Decision |
|---|---|
| Methodology rules cluster | Per-section verdicts (Pass 1 table) |
| Issue Recording rule | Keep, retarget to `.abcd/development/activity/issues/` |
| Documentation Structure | Decision tree only |
| Time Estimates | Drop (implicit in intent format) |
| Milestone Completion | Devlogs as flow-next records; retrospectives in a later phase |
| Audit/provenance machinery (governance, sig-verify, vap-verify) | entirely in a later phase |
| Spec Kit interop | Later phase, with placeholder intent (itd-23) |
| test-reminder skill | Drop |
| codebase-security-auditor | Fold into launch-gatekeeper |
| documentation-architect | Fold into both embark and launch |
| documentation-auditor | Keep as 15th agent, subagent-only |
| ai-contributions.md | Auto-generated from session logs |
| retrospective.md | Future /abcd:reflect command (a later phase) |
| use-case template | Harvest user-story prompt into /abcd:intent new |
| Per-language project scaffolds | Dropped entirely; ahoy works on existing code only |
| RP workspace pull (workspace.json only — presets and MCP routing in a later phase) | plumbing via [itd-7]; richer pull and monitoring in a later phase |

## Intents drafted from this harvest

All five intents below were drafted during the 2026-05-04 interview session:

- [itd-3] — modular rules loader
- [itd-1] — acceptance gates
- [itd-23] — spec-kit interop (a later phase)
- [itd-24] — `/abcd:reflect` command (a later phase)
- [itd-7] — RP workspace portability

[itd-17] (model-effectiveness-tracking, already in drafts, a later phase) should be updated separately to reference the v0 seed data preserved in `~/Archive/abcd-v0-archive-2026-05-04/Home-dot-abcd/model-effectiveness.json` and `model-scorecard.jsonl`.

## Surface-area check

This harvest added **three new intents**: `itd-3` (rules loader), `itd-1` (acceptance gates), `itd-7` (RP workspace pull). Combined with the brief's pre-existing surface (now 6 commands after itd-4 capture lands, 14 → 15 agents, 7 skills, the lifeboat pack/unpack model, dev-sync, intent system), the footprint has grown deliberately.

Worth re-evaluating before the first phase is declared complete:

- `itd-3` (rules loader) — load-bearing for the "remove scaffolding" goal. Cannot move to a later phase without losing the migration's core value.
- `itd-1` (acceptance gates) — small schema bump, large quality return. Cheap to ship; sets the acceptance-criteria discipline every later intent inherits.
- `itd-7` (RP workspace pull) — load-bearing for the user-account migration. The first cut is deliberately narrow (workspace.json pull only); presets, `mcp-routing.json` scoping, `--preset` selection, and `abcd rp link` explicitly land in a later phase.

If the first phase starts feeling too heavy, the candidates for moving to a later phase (in order) are:

1. `itd-7` even further — move workspace pull entirely to a later phase if the user-account migration use case turns out to be one-shot rather than recurring.
2. `itd-1` lint enforcement — keep the `## Acceptance Criteria` section requirement, move the `/abcd:intent plan` blocking lint to a later phase.
3. `itd-3` star-command bypass — keep keyword-recall injection, move `*<DOMAIN>` syntax to a later phase.

`itd-20` (top-level `/abcd` dispatcher) and `itd-21` (no-lifeboat-scaffolding / `/abcd:init-project`) remain in a later phase per the user's "ahoy already covers init-project" decision; they were considered for the first phase and explicitly moved later.

## References

[itd-1]: ../intents/disciplines/itd-1-acceptance-gates.md "itd-1 — Acceptance gates"
[itd-3]: ../intents/shipped/itd-3-modular-rules-loader.md "itd-3 — Modular rules loader"
[itd-7]: ../intents/drafts/itd-7-rp-workspace-portability.md "itd-7 — RP workspace portability"
[itd-17]: ../intents/superseded/itd-17-model-effectiveness-tracking.md "itd-17 — Model effectiveness tracking (a later phase)"
[itd-23]: ../intents/drafts/itd-23-spec-kit-interop.md "itd-23 — Spec Kit interop (a later phase)"
[itd-24]: ../intents/planned/itd-24-reflect-command.md "itd-24 — /abcd:reflect command (a later phase)"
[carl]: https://github.com/ChristopherKahler/carl "CARL — Context Augmentation & Reinforcement Layer"
[paul]: https://github.com/ChristopherKahler/paul "PAUL — Plan-Apply-Unify Loop"
