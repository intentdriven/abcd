---
id: itd-11
slug: pass-b-pitfall-mitigation
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# Disembark Survives Noisy Transcripts

## Press Release

> **abcd produces faithful lifeboats even when the source transcripts are messy.** Pass B's chat-distiller reads the native redacted transcript store and applies multiple filtering strategies — time-window, keyword, spec-id, semantic relevance — falling back gracefully when one strategy underperforms. Transcript noise no longer leaks into rationale-fill or unrecorded-decision findings; the lifeboat-oracle's audit catches and quarantines low-confidence Pass B output before it pollutes the brief.
>
> "Half the sessions in my transcript store are tangents and meta-discussion," said Maya, AI/agent researcher. "Earlier disembark gave me beautiful spec extraction but Pass B's findings were obviously hallucinated from off-topic chats. Now it either filters that out or marks it explicitly as low-confidence."

## Why This Matters

abcd's baseline Pass B (chat-distiller) commits to time-windowed transcript filtering as the primary noise-reduction strategy. Phase 0's transcript sampling will tell us if that's enough. If transcripts are noisier than expected (which experience suggests is likely on real-world repos), Pass B becomes a source of confident-sounding-but-wrong findings — exactly what we don't want from a lifeboat.

This intent hardens Pass B *if* Phase 0 sampling reveals the need. It's defensive: scope the work now so when we hit the problem, the solution is queued.

## What's In Scope

- Multiple filtering strategies (time-window, keyword, spec-id, semantic) with a fallback chain
- Confidence scoring per chat-distiller finding
- lifeboat-oracle audit pass that quarantines low-confidence output
- Tuning knobs in `.abcd/config.json` for strategy weights

## What's Out of Scope

- Re-disembarking old lifeboats with the better filters (covered by `--apply-audit`-style re-runs already)
- Building our own transcript classifier model
- Replacing chat-distiller with an entirely different approach (this is hardening, not redesign)

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** the native transcript store holding off-topic transcripts (tangents, meta-discussion, unrelated sessions), **when** Pass B's chat-distiller runs with the multi-strategy filter chain, **then** every emitted finding (rationale-fill, unrecorded-decision, pitfall) carries a confidence score in `[0.0, 1.0]` AND findings below the configured quarantine threshold are written to a separate `low-confidence/` subdirectory rather than promoted into the lifeboat.
- **Given** a single chat-distiller call with multiple filter strategies configured (time-window, keyword, spec-id, semantic), **when** the primary strategy returns a low-coverage result (e.g., < N% of the transcript matched), **then** the fallback chain activates the next strategy in priority order until either coverage exceeds the threshold or all strategies are exhausted.
- **Given** Pass B has produced findings, **when** lifeboat-oracle runs in Pass C, **then** any finding with confidence below the quarantine threshold appears in the oracle's report as "quarantined for low confidence" rather than being absent — the audit trail is explicit.
- **Given** the user runs `/abcd:disembark` with `pass_b.confidence_threshold` set in `.abcd/config.json`, **when** Pass B emits findings, **then** the threshold value (and the per-strategy weights) appear verbatim in the disembark report's "Pass B configuration" section.
- **Given** a finding flagged as low-confidence, **when** the user inspects the lifeboat, **then** they can read the finding alongside its source transcript range and the reason for the low score (e.g., "matched only by semantic similarity, no keyword anchor; off-topic markers present").
- **Given** Phase 0 sampling on the validation corpus showed a noise pattern, **when** the multi-strategy filter chain runs against that same corpus, **then** the precision (true positives / total findings) measurably exceeds the time-windowed baseline by at least a documented threshold — recorded in the disembark report's regression-check section.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [harness, interop and lifeboats SOTA note](../../research/notes/2026-10-23-harness-interop-lifeboats-sota.md), which reuses the local [transcript sampling](../../research/notes/transcript-sampling.md) measurement.

- The local baseline: decision turns were 22.3% of user turns (138/619) and 1.4% of all turns across 14 stratified transcripts from 439 in one project, with stub sessions at zero; the viability ruling against a 15% floor rests on the user-turn denominator the sampling note adopted as a recorded measurement deviation, and on the spec's original all-turn metric (1.4%) the result was below the floor (NEEDS REDESIGN, superseded); the chain is tuned against this corpus or not at all.
- Compaction practice keeps decisions and drops tool output: Anthropic's guidance preserves "architectural decisions, unresolved bugs, and implementation details", discards "redundant tool outputs", starts by maximising recall, and warns that aggressive compaction loses "subtle but critical context whose importance only becomes apparent later".
- Agents "consistently depend on raw experience" and "often disregard or misinterpret condensed experience, even when it is the only experience provided" across 4 frameworks, 13 backbones and 9 environments (January 2026). The study tests self-evolving agents acting on experience, not a summariser reading a transcript, so by analogy it suggests a distiller should read transcript lines rather than a summary of them.
- Verbalised confidence is overconfident; "consistency across multiple sampled responses" and aggregation reduce it, and no method wins everywhere (ICLR 2024). A model-reported `[0.0, 1.0]` is not a calibrated number.
- The best of thirteen models located the step responsible for a hallucination in 41.1% of 693 annotated agent trajectories, 11.6% for tool-use hallucinations (AgentHallu, January 2026); the benchmark measures hallucination attribution, not fault-finding in general, so by analogy it suggests a finding without its transcript range is unreviewable.
- abcd's transcript store is already JSONL and redacted fail-closed (`internal/core/history`), and the hosts' `transcript_path` files are JSONL, so time-window, keyword and record-id filtering is a deterministic pass over lines before any model reads them.

**Path 2 — native structural confidence shaped to the calibration evidence: each finding carries the anchors that selected its range (window, keyword, record id, semantic only) and the range itself; the score is a rule over anchors present plus agreement between two independent distiller runs, never a self-report; semantic-only findings are quarantined, visibly, with the reason; the regression check runs against the sampled corpus.** The independent fit-challenge runs at plan time.

## Open Questions

- Does Phase 0 sampling actually reveal a noise problem, or is the time-windowed design sufficient?
- What's the right confidence threshold for quarantine (vs accept, vs reject)?
- Should low-confidence findings still be visible (in a separate `low-confidence.md` section) or hidden?
- (SOTA 2026-10-23) Is a second distiller run for agreement worth its host cost on every disembark, or only when a finding rests on a semantic anchor alone?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
