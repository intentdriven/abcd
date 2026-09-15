---
id: itd-28
slug: rp-reviews-into-flow
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
prd_path: null
prd_grandfathered: true
grandfathered: true
grandfathered_at_phase: phase-3-intent
glossary_terms_used:
  - core/brief
  - core/intent
  - core/lifeboat
  - core/oracle
  - core/persona
  - core/phase
  - core/spec
  - core/transport
builds_on: [itd-1]
severity: major
---

# Spec-Tied Reviews Live Next To The Spec They Reviewed

## Press Release

> **abcd lands every spec-tied review next to the spec it reviewed, in the native review store, pinned to the commit it reviewed, with a sanitisation pass before commit.** When a plan-review or impl-review finishes — whichever oracle adapter produced it — an abcd-side post-processor captures the review receipt via the native receipt contract and lands a canonical JSON sidecar (`review.json`) and rendered Markdown view (`review.md`) in a per-review directory at `.abcd/reviews/<spec-id>/<NNNN>-<slug>-<ref>/`. The sidecar carries the full sanitised review body, structured findings, and a `review_of_commit` SHA pin so future agents can detect when a review's findings have gone stale. Raw transcripts land in a per-review `raw/` subdirectory (gitignored). A two-stage redaction scheme applies: Stage 1 is a write-time sanitiser (AWS, GCP, Azure, Cloudflare, GitHub, Anthropic, OpenAI, Stripe, Slack, JWT, PEM) before any file is written; Stage 2 is a detect-and-block commit gate run through the scanner seam — native patterns by default, `gitleaks protect --staged --redact=100` as the stronger opt-in adapter when the binary is present, the engine always reported — that rejects commits if secrets survive Stage 1. An on-demand `reviews-index --spec <spec-id>` regenerates `INDEX.md` + `INDEX.json`; CI runs `--check` mode to catch drift without ever writing to the working tree.
>
> "Reviews used to die on the laptop they were generated on, in a folder I had to know about," said Frank, SRE. "Spec-tied reviews live where the spec lives. When I clone the repo six months later, every review comes with it — and `staleness: 14_commits` tells me at a glance which ones I should re-run."

## Why This Matters

An oracle review that isn't landed in the native review store lives only where the oracle adapter that produced it happened to write it — an external app's application-support directory, a scratch temp file. Reviews there are not portable, not scoutable, not survivable across `git clone`, and not linked to the spec they reviewed. Three failure modes follow:

1. **Confidence laundering** — future agents read a review with no SHA pin and treat its conclusions as ground truth, even when the code has since changed.
2. **Bias propagation** — re-reviews shown the prior verdict anchor-bias toward it ([arXiv 2603.18740][bias]).
3. **Context-window pollution** — scouts pulling unbounded review history burn 5–30% of their window on stale reviews from completed specs ([Liip][liip]).

This intent commits abcd to a per-spec, SHA-pinned, hybrid commit/ignore review store with a redaction safety net. The post-processor is **oracle-adapter-agnostic**: it consumes the native review receipt contract, so a review lands identically whether the host-delegated default, the RepoPrompt adapter, or any other oracle adapter produced it.

This is the **`press-release`-shaped commitment** behind the native review-store spec, which decomposes into a small set of implementation tasks.

### Carve-out: spec-tied reviews vs unscoped chats

This intent covers **spec-tied reviews only** — the plan-review, impl-review, and (future) spec-completion-review artifacts that have a known spec ID, landed by the native review pipeline. A separate job is the **lifeboat-style storage of unscoped ad-hoc oracle chats** (the "I spent 30 minutes brainstorming the next intent in an oracle" case), swept into `.abcd/work/reviews/` by whichever oracle adapter is configured (e.g. the RepoPrompt adapter, when present). Two stores, two clear jobs:

| Store | Purpose | Cadence | Format | Lifeboat consumes |
|---|---|---|---|---|
| `.abcd/reviews/<spec>/` | Per-spec engineering audit trail | Push at write-time (native review pipeline) | Canonical JSON sidecar (`review.json`) + derived MD render (`review.md`) | Yes |
| `.abcd/work/reviews/` | Lifeboat-style storage of unscoped oracle transports | Pull at sync time (`abcd dev-sync`) | Verbatim MD per transport | Yes |

The unscoped-transport sweep is adapter-scoped and runs only when an oracle adapter is configured; spec-tied reviews are already landed by the native review pipeline.

## What's In Scope

- **abcd-side post-processor** that captures the native review receipt and writes a canonical JSON sidecar (`review.json`) + rendered MD (`review.md`) into a per-review directory under `.abcd/reviews/<spec-id>/<NNNN>-<slug>-<ref>/`. Atomic write via staging-dir + `rename(2)`. Idempotent. Never deletes source until target write+verify confirmed.
- **Native review-pipeline trigger**: the post-processor runs when the native review pipeline emits a receipt, regardless of which oracle adapter produced the review. The receipt contract is adapter-agnostic.
- **JSON sidecar schema** (committed under `docs/reference/`): required metadata (`review_of_commit`, `spec_path`, `spec_sha256`, `reviewer_model`, `reviewer_tool`, `verdict`, `generated_at`, `iteration`, `focus`, `review_type`, `target_id`, `reviewed_files`, `backend`, `pinning`, `allow_no_commit`), required content (`summary`, `body_markdown`, `findings`), required provenance (`sanitized_raw_artifact_sha256`), required truncation metadata (`truncated`, `truncation_method`, `omitted_bytes`, `body_max_bytes`, `render_max_bytes`); optional (`superseded_by`, `worktree_sha256`, `dirty`, `chat_id`, `session_id`, `receipt_path`). `backend` records which oracle adapter produced the review.
- **Verdict enum locked**: `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` — the canonical review verdict enum, shared by every oracle adapter.
- **Sequence allocation**: `flock` with 5s timeout; up to 5 retries with 100ms exponential backoff on contention. No fallback filename variants — if all retries fail, exit non-zero with guidance.
- **Directory convention**: `<NNNN>-<slug>-<ref>/` where `<ref>` = 7-char short SHA (when `pinning: "commit"`) or literal `unpinned` (when `pinning: "none"`).
- **Two-stage redaction**: Stage 1 write-time sanitiser strips home-dir paths, AWS/GCP/Azure/Cloudflare/GitHub/Anthropic/OpenAI/Stripe/Slack/JWT/PEM secrets before any file is written. Stage 2 detect-and-block runs through the **scanner seam** (decided 2026-07-24, maintainer grill — see DECISIONS.md): the native pattern engine is the default; `gitleaks protect --staged --redact=100` is the stronger opt-in adapter when the binary is present; the hook reports which engine ran (loud staging — no silent downgrade); CI's gitleaks pass remains the authoritative backstop. Native-vs-gitleaks pattern parity (iss-96) is in scope for the spec. No new hard dependency — adr-22 holds.
- **Size caps**: `body_max_bytes` (default 20 KiB, bounds `body_markdown`); `render_max_bytes` (default 24 KiB, bounds `review.md`). Both caps recorded in `review.json`.
- **`.gitignore`** allow-lists the native review store `.abcd/reviews/`; ignores each per-review `raw/` subdirectory (`.abcd/reviews/**/<NNNN>-*/raw/`).
- **Index discovery**: `reviews-index --spec <spec-id>` generates `INDEX.md` + `INDEX.json` on demand (no new command surface). CI verifier runs `--check` mode on PRs touching `.abcd/reviews/**`. No git hooks — on-demand plus CI only.
- **Brief edits in this repo (abcd-cli)**:
  - `05-internals/02-adapters.md`: clarify the review dispatcher's two stores (spec-tied via the native review pipeline → `.abcd/reviews/`; unscoped via a configured oracle adapter → `.abcd/work/reviews/`).
  - `05-internals/03-configuration.md`: clarify `dev_sync.reviews.enabled` only sweeps unscoped transports.
- **Default**: post-processor ON by default; `ABCD_REVIEW_POSTPROCESS=0` kill switch.
- **Dirty-tree policy**: capture `worktree_sha256` and tag `dirty: true`; do NOT block.
- **No-commit-yet policy**: refuse review on branch with zero commits; emit guidance; respect `--allow-no-commit` override.
- **Acknowledgements**: README "Acknowledgements" cites `gitleaks` (Apache-2.0), `joelparkerhenderson/architecture-decision-record` (CC-0, ADR `NNNN` naming convention), `REPPL/abcdZero` F-075 / F-037 (prior art for abcd-side wrapper architecture).

## What's Out of Scope

- **Reaching into an oracle adapter's internals** — the native review receipt contract is sufficient; abcd never patches an adapter's own storage or code.
- **One-shot import tool** for reviews sitting in an external adapter's own storage — explicitly out of scope.
- **Cross-spec review aggregation, sigstore signing, IDE integration, scoring/trends** — out of scope.
- **Modifying ephemeral build artifacts** (`/tmp/review-prompt.md`, `/tmp/re-review.md`) — keep these in `/tmp/`.
- **Hand-rolling a new redaction engine for this feature** — Stage 2 reuses the scanner seam's canonical patterns as its native default; gitleaks provides the deeper ruleset as the opt-in adapter.
- **Replacing the review pipeline** — this intent adds storage + governance, not a new oracle adapter.
- **Automatic git-hook regeneration of INDEX.md** — practice-scout established this is the canonical anti-pattern (pre-commit#2240 re-stage loop, post-commit feedback loops). On-demand only + CI verifier.
- **Unscoped oracle transport storage** — covered by the adapter-scoped sweep into `.abcd/work/reviews/`.
- **`Reviewed-by:` git trailer auto-injection on implementation commits** — out of scope (nice-to-have bidirectional linkage).

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a persona runs a plan-review for `spc-X` end-to-end via any oracle adapter, **when** the review completes, **then** a per-review directory lands at `.abcd/reviews/spc-X/<NNNN>-<slug>-<ref>/` containing `review.json` (all required fields populated, `verdict` ∈ `{SHIP, NEEDS_WORK, MAJOR_RETHINK}`, non-empty `body_markdown` and `reviewed_files`) and `review.md` (mechanically rendered from `review.json`).
- **Given** the post-processor runs twice on the same receipt, **when** both invocations complete, **then** there is exactly one per-review directory in `.abcd/reviews/spc-X/` (idempotent; the second invocation is a no-op).
- **Given** the post-processor is killed mid-write (`kill -9`), **when** the persona inspects the working tree, **then** no `.tmp` or partial files are visible to git.
- **Given** 5 concurrent post-processor invocations on the same spec, **when** they complete, **then** 5 distinct sequence numbers exist (no collisions).
- **Given** the persona sets `ABCD_REVIEW_POSTPROCESS=0` and runs a plan-review, **when** the review completes, **then** the post-processor exits 0 with no side effects and the review remains only in the producing oracle adapter's raw output.
- **Given** a staged `.abcd/reviews/**` file containing a multi-cloud secret (AWS access key, fine-grained GitHub PAT, Anthropic key, JWT, or PEM private key) that was NOT caught by Stage 1, **when** the pre-commit hook runs the Stage-2 scan (native engine, or gitleaks when present), **then** the commit is blocked with the finding path/line/rule reported (never the raw secret value).
- **Given** the gitleaks binary is absent, **when** the pre-commit hook runs the Stage-2 scan, **then** the native engine runs and the hook output names the engine that ran — a downgrade is never silent.
- **Given** a committed `.abcd/reviews/**/*.md` file containing `AKIAIOSFODNN7EXAMPLE`, **when** the pre-commit hook runs, **then** the EXAMPLE-allowlisted value is NOT redacted.
- **Given** a review file with a `review_of_commit` SHA that fails `git rev-parse --verify`, **when** the pre-commit hook runs, **then** the commit is rejected with a clear error message.
- **Given** a `review.json` with `body_markdown` exceeding `body_max_bytes`, **when** the pre-commit verifier runs, **then** the commit is rejected with guidance (the post-processor truncates automatically; this acceptance captures the case where someone manually edits the sidecar to violate the cap).
- **Given** a CI run on a PR touching `.abcd/reviews/**`, **when** `reviews-index --check --all` runs, **then** the workflow fails on drift with the exact remediation command and never writes back to the branch.
- **Given** the brief is updated, **when** a contributor reads `05-internals/02-adapters.md` and `05-internals/03-configuration.md`, **then** they find explicit text describing the two-store carve-out (spec-tied via the native review pipeline; unscoped via a configured oracle adapter).
- **Given** the README is updated, **when** a contributor reads the Acknowledgements section, **then** they find explicit citations of `gitleaks` (Apache-2.0) and `REPPL/abcdZero` F-075 / F-037 prior art.

## Open Questions

- **Hook coverage for hosts without a Stop-hook equivalent**: standalone invocation of the post-processor with `--from-receipt <path> --spec <id>` is the documented fallback. Should a per-host wrapper also ship?
- **Stale review threshold for `staleness` column**: currently `<N>_commits` since `review_of_commit`. Should we add a "danger" threshold (e.g., `>20_commits` shown red)? Deferred polish.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Linked spec: the native review-store spec (in this repo's native spec layer).
- Depends on: `itd-1` (acceptance gates).
- Coordinates with: `itd-7` (RP workspace portability) — both are RepoPrompt-adapter-scoped but pull different artifact types.
- Coordinates with: `itd-13` (scheduled dev-sync) — unscoped review pull would benefit from scheduled sync.
- Prior art: `REPPL/abcdZero` F-075 (planned, evaluated upstream-vs-wrapper, chose wrapper) and F-037 (completed v2.5.1, shipped patch-based wrapper).

[bias]: https://arxiv.org/html/2603.18740v1 "Confirmation Bias in `LLM`-Assisted Security Code Review"
[liip]: https://www.liip.ch/en/blog/preventing-context-pollution-for-%61i-agents "Liip — Preventing Context Pollution for `AI` Agents"
