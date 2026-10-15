---
id: dcn-2607110000000004
slug: itd-3-rules-loader-design-signed-off
date: 2026-07-11
---
2026-07-11 — itd-3 rules-loader **design signed off** (plan
`2026-07-11-itd-3-rules-loader.md`, prefer-sota verdict). Surviving shape: a
transport-agnostic `internal/core/rules` capability with two front doors
(`abcd rules [domain]` verb + `abcd hook prompt-router`), **not** an adapter
seam. Four intent deltas approved: **D1** event-driven refresh on
`SessionStart(compact)` (fixed-N demoted to a ~15–20 backstop, not primary);
**D2** keep the shipped `{schema_version,disabled,domains{}}` shape (legacy
`extends`/`overrides` sketch superseded); **D3** zero model-facing tokens on
no-match + out-of-band diagnostic log (supersedes the "<200-token header"
acceptance criterion); **D4** `.abcdignore` rejected for v1. Build proceeds
phased/TDD from Phase 1 (`internal/core/rules`).
