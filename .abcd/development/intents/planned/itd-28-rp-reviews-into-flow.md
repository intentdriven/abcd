---
id: itd-28
slug: rp-reviews-into-flow
spec_id: spc-2609211854150455
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
impact: additive
---

# Spec-Tied Reviews Live Next To The Spec They Reviewed

> **Re-scoped on 2026-09-21** by the product thinker: the record keeps the pin and the staleness view and drops its own review store, its two-stage redaction and its pre-commit verifier. The record already holds dated review folders under `.abcd/work/reviews/` and gate receipts keyed by the commit they gate; what is missing is that every review names the commit it read and the status board says how stale each has become. The scrub rides the scanner that already exists. The press release and scope below are read through this paragraph and the Decisions section.


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

## Mechanism

We expect a visible staleness count to make a stale review get re-run before a release rather than trusted, because the count turns "is this still valid" from a question nobody asks into a row on the board everyone sees; shown wrong if a release cut still cites reviews past the threshold with nobody re-running them.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a review folder under `.abcd/work/reviews/` is filed by any of abcd's own review paths, **when** it is written, **then** its summary carries `review_of_commit: <full sha>` written by the tool, and the record lint refuses a review folder without one (a folder that predates the rule is named as legacy, not refused).
- **Given** the bare `abcd` status board renders, **when** review folders exist, **then** each is listed with the spec or scope it reviewed and the number of commits the default branch has moved since its `review_of_commit`, and one past twenty is flagged for a re-run.
- **Given** a review of a spec, **when** it is filed, **then** its folder name carries the spec's id, so a reader finds it from the spec.
- **Given** a review body carries a secret, **when** it is committed, **then** the scanner the repository already runs refuses it; this record builds no second scrubber.
- **Given** `--json`, **when** the board renders, **then** the staleness rows and the threshold are in the payload.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **Pin and staleness only.** The review store, the two-stage redaction and the pre-commit verifier this record first described are dropped: the dated review folders and the gate receipts are the store, and the scanner is the scrub.
2. **The staleness view is on the status board**, flagged past twenty commits.

## Open Questions

_None open; the hook-coverage and the danger-threshold questions this record carried fall away with the store, and the threshold is decision 2._

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

## Grounds

- pursued: the release gate now demands review receipts, and a receipt with no commit named cannot be judged fresh; we expect the next cut to read the staleness rows before it trusts a receipt; shown wrong if the next cut never asks how old a receipt is
