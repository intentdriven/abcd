# Recording agent sessions — SOTA survey

**Date:** 2026-07-30
**Scope:** state of the art in persisting AI coding-assistant session transcripts — SpecStory and its alternatives, how the field's consensus compares with abcd's native transcript store (ADR-29), and the guardrails any optional SpecStory plug-in must carry.
**Backs:** [iss-217](../../../work/issues/open/iss-217-add-a-cross-agent-capture-source-as-an.md) (add a cross-agent capture source as an optional plugin); informs any intent decomposed from it.
**Drawn from:** [ADR-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md), [ADR-29](../../decisions/adrs/0029-native-transcript-corpus.md), [itd-89](../../intents/shipped/itd-89-start-the-transcript-clock.md), [itd-59 (draft)](../../intents/drafts/itd-59-autonomous-worker-transcript-capture.md).

> **Status of citations:** gathered 2026-07-30 by a web-research agent. SpecStory's privacy docs, CLI README, and company page, the Claude Code hooks and monitoring docs, and the ghosttype write-up were opened directly; the smaller GitHub projects and the HN/dev.to threads were read at summary level. Verify before quoting in an ADR.

---

## 1. SpecStory: what it actually is

Two products from SpecStory Inc. (seed-stage, founded 2024): a **CLI wrapper** (`specstory run <agent>`, Apache-2.0) that reads an agent's native store (JSONL/SQLite) and writes one markdown file per session to `.specstory/history/` **inside the repo**, and **closed-source IDE extensions** (Cursor, VS Code/Copilot) doing the same from the IDE's SQLite. Host coverage as of 2026: Claude Code, Codex CLI, Cursor (CLI+IDE), Gemini CLI, Droid CLI, DeepSeek TUI, Antigravity CLI — actively maintained (~1.3k stars, multi-agent support still expanding). Capture is wrapper-based, not hook-based: nothing is recorded unless the agent was launched through `specstory run`. Secret redaction (credential patterns) is on by default; **path/identity redaction is not offered**. Anonymous PostHog telemetry is **on by default** (`--no-usage-analytics` to opt out). The Cloud/Shares/Derived-Rules tier transmits prompts and code to SpecStory's servers, and Shares mint public links. Longevity risk is moderate (seed-stage, closed extensions, early monetisation) but lock-in is low: output is plain markdown in-repo and the CLI is forkable.

**Consequence for abcd:** what SpecStory would add is **cross-agent normalisation** — one tool that parses Codex, Cursor, and Gemini stores as well as Claude Code's — plus a maintained reader for `.specstory/`-era archives. The Cloud tier and default-on telemetry are wiring-time configuration points under the nothing-leaves-the-machine policy (§4).

## 2. The alternatives the field converged on

Ranked for a solo, privacy-strict context:

1. **Native host JSONL as source of truth + on-demand rendering.** Claude Code already persists every session under its projects directory; renderers (Willison's `claude-code-transcripts`, `claude-code-log`, `claude-conversation-extractor`) turn it into HTML/markdown when needed. Zero capture infrastructure.
2. **Hook-based capture.** Every Claude Code hook event carries `transcript_path`; a `SessionEnd` hook copies/converts deterministically — no wrapper to forget. Per-response `Stop` hooks are measurably slow; `SessionEnd`-only risks losing a session killed abruptly.
3. **Git-native linking, transcripts out of repo** (`gammons/ai-session` pattern): a separate local store, linked from commits by trailer. Tiny project, but the pattern — *traceability without repo pollution* — is the consensus position of the 2026 "should the session be part of the commit?" debate. The counter-position (Larson, Willison) argues for *deliberate, curated* publishing of selected transcripts, not commit-everything.
4. **OpenTelemetry pipelines** (Claude Code exports natively; prompt content redacted by default). Enterprise fleet monitoring and cost/policy audit — a poor fit at solo scale: infrastructure-heavy, and traces are not a readable durable record.

**Consequence for abcd:** abcd's shipped design composes options 1–3: native store fed by a `SessionEnd` hook (itd-89), out of repo, keyed on the root-commit SHA. The open edges are coverage (other agents' sessions — SpecStory's niche; autonomous-run passes — itd-59) and rendering.

## 3. What SpecStory would add, and what the import seam preserves

**What it adds** (none of which the native store attempts):

- **Host coverage**: capture from 7+ agents whose stores abcd cannot read — the parsers for Codex, Cursor, and Gemini's undocumented native formats, maintained as those formats change.
- **One normalised format**: every host's sessions rendered to the same per-session markdown.
- **A maintained reader for legacy `.specstory/` archives** — the Phase-0 corpus of 439 transcripts becomes importable.

**What importing through the seam preserves** (the native store's posture, applied to imported material):

- Storage stays out of the repo (`~/.abcd/history/<root-sha>/transcripts/`), whatever the source tool's own layout.
- The two-stage fail-closed redaction — scan → redact → re-scan, refusal on any surviving `hard_fail`, home-path/identity floors, the literal `$HOME` backstop, and the degraded-scanner refusal — runs on import exactly as on native capture.
- Provenance hashing (`source_sha256`, idempotency on the (sha256, session, kind) triple).
- The `SessionEnd` hook remains the capture path for Claude Code sessions; the external tool is wired for the hosts the hook cannot reach.

**Wiring-time configuration** the adopter owns: telemetry is default-on (PostHog; `--no-usage-analytics`), the opt-in Cloud/Shares tier transmits prompts and mints public links, and the tool's own history directory lives in-repo by default — under this repo's privacy policy those become: opt out, out of scope, and never tracked, respectively (§4).

**Consequence for abcd:** the plug-in is a **coverage extension** — worth building when sessions from non-Claude agents or the legacy archives need to reach the corpus, wired so imported material flows through the same redaction door as native capture.

## 4. Guardrail consensus (2025–2026) and where abcd stands

- **Assume transcripts contain secrets.** Documented rates (5 distinct secrets across 34 session files in 30 days of normal use); scanners now explicitly hunt AI-tool directories in public repos, including deleted-but-archived commits. abcd: redact-on-write, fail-closed — compliant, with known pattern gaps already in the ledger (iss-96 unanchored high-entropy secrets, iss-125 hostnames, iss-157 network identifiers).
- **Transcripts stay out of the code repo by default;** commit the reasoning, link the session. abcd: compliant by construction; `.specstory` is already in the launch bundle's default-deny set.
- **Absolute-path and identity leakage** is the under-served risk — no off-the-shelf tool scrubs paths. abcd's scanner already covers it (home-path/identity `hard_fail` floors), so imports must route through that scanner.
- **Telemetry opt-out is the adopter's job.** SpecStory's is default-on.
- **Imported transcripts are untrusted input** (indirect prompt injection) — the `chat-distiller` threat model already mandates spotlighting and an injection-canary fixture.
- **Size/noise:** prefer `SessionEnd` capture + on-demand rendering over continuous per-response writes. abcd: compliant.

**Consequence for abcd:** the guardrails for a SpecStory plug-in are not new policy — they are the existing invariants applied at the import boundary (§5).

## 5. Verdict — adversary-checked fit (prefer-sota, sota-per-intent)

The record has already litigated this fork: ADR-22 demoted SpecStory from bundled dependency to optional adapter; ADR-29 rejected "keep SpecStory + the external history store" and "capture raw, redact later"; the seam is pre-declared (`--kind specstory-import`, `history.backend = "specstory"` in the brief's configuration and adapter catalogues). Under sota-per-intent this is **path 2 being exercised** (native floor + real seam), not a path-1 adoption — no new required dependency, so no hard stop; if SpecStory ever became *required*, the dependency gate would apply.

**Recommended shape — import-only, over the same store:**

1. **One door.** SpecStory output enters only through `history.Capture(..., "specstory-import")`, so the full two-stage redaction (secrets + paths + identity, fail-closed) applies to imported material exactly as to native captures. Never a second store, never a bypass.
2. **The real work is the specified-but-unbuilt merge**: the verification matrix promises "an optional specstory import merges by timestamp/content hash over the same store", but `Capture` today treats `specstory-import` as a plain kind value with no merge logic. That, plus a parser for `.specstory/history/*.md`, is the plug-in.
3. **`.specstory/` never becomes tracked state**: gitignored in managed repos, already launch-excluded; the in-repo carve-out stays limited to `.specstory/cli/config.toml` as the naming constraints record.
4. **Telemetry off by default in any wiring** (`--no-usage-analytics`); Cloud/Shares are out of scope entirely — public share links and prompt upload are incompatible with the privacy invariants.
5. **The `SessionEnd` hook remains the Claude Code capture path** — deterministic and already wired; SpecStory's contribution is the hosts the hook cannot reach, plus the legacy `.specstory/` archives (the Phase-0 corpus of 439 transcripts is an immediate import candidate).
6. **Imported transcripts are untrusted** — the chat-distiller spotlighting rules apply to `specstory-import` records the same as any transcript.
7. **User-facing docs stay host-agnostic** (the `harness/specstory` docs-lint blocker). Flag: the `--kind specstory-import` flag string already leaks the name into the generated CLI reference — pre-existing, worth its own ledger entry.

A note here informs the decision; it does not make it. Whether the coverage gain justifies building the import adapter is the maintainer's call at intent-planning time (verifier selects, gates decide).

## 6. Addendum (2026-08-15) — adoption trigger, target harness, and where the names live

The maintainer's decision (DECISIONS.md 2026-07-30, recorded 2026-08-15) adopts §5's shape and names its trigger: a cross-agent capture source becomes worth building **when compatibility beyond Claude Code becomes a concern**, and the harness the maintainer has named as the target is **opencode**. This note is the record that carries the names; the ledger entry (iss-217) and the DECISIONS line refer here, per the convention that commitment records name no external tool before its adoption is decided.

Facts verified 2026-08-15: SpecStory's documented `specstory run` providers are Claude Code, Cursor (CLI + IDE), Codex CLI, Droid CLI, and Gemini CLI (deprecated in favour of Antigravity CLI). **opencode is not among them.** The CLI exposes a Provider SPI for adding new agent providers, so opencode support is buildable without waiting on the vendor. Both facts are re-verified at the adopting intent, alongside the SOTA verdict itself.

## References

- [SpecStory privacy docs][specstory-privacy]
- [SpecStory CLI (specstoryai/getspecstory)][specstory-repo]
- [SpecStory — Claude Code page][specstory-cc]
- [ghosttype — secrets in AI conversation history][ghosttype]
- [Claude Code hooks reference][cc-hooks]
- [Claude Code monitoring (OTel)][cc-otel]
- [simonw/claude-code-transcripts][cct]
- [Will Larson — sharing Claude transcripts][lethain]
- [daaain/claude-code-log][ccl]
- [markmatsu/claude-logkeeper][logkeeper]
- [gammons/ai-session][ai-session]
- [HN — if AI writes code, should the session be part of the commit?][hn-commit]
- [Knostic — AI assistants leaking secrets in your IDE][knostic]

[specstory-privacy]: https://docs.specstory.com/privacy "SpecStory — privacy and telemetry documentation"
[specstory-repo]: https://github.com/specstoryai/getspecstory "SpecStory CLI and Lore, Apache-2.0"
[specstory-cc]: https://specstory.com/claude-code "SpecStory for Claude Code"
[ghosttype]: https://betheadversary.com/posts/ghosttype/ "ghosttype — credential scanning across AI conversation histories, May 2026"
[cc-hooks]: https://code.claude.com/docs/en/hooks "Claude Code hooks reference"
[cc-otel]: https://code.claude.com/docs/en/monitoring-usage "Claude Code monitoring and OpenTelemetry"
[cct]: https://github.com/simonw/claude-code-transcripts "claude-code-transcripts — render session JSONL to HTML"
[lethain]: https://lethain.com/sharing-claude-transcripts/ "Will Larson — sharing Claude transcripts, Jan 2026"
[ccl]: https://github.com/daaain/claude-code-log "claude-code-log — projects tree to HTML/Markdown"
[logkeeper]: https://github.com/markmatsu/claude-logkeeper "claude-logkeeper — hook-based, age-encrypted transcript capture"
[ai-session]: https://github.com/gammons/ai-session "ai-session — trailer-linked out-of-repo transcript store"
[hn-commit]: https://news.ycombinator.com/item?id=47212355 "HN thread, Feb 2026"
[knostic]: https://www.knostic.ai/blog/ai-coding-assistants-leaking-secrets "Knostic — AI coding assistants leaking secrets"
