---
schema_version: 1
id: "iss-2610121542385455"
slug: "the-claude-runner-internal-core-runner"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "the build loop's stored-text sweep on fix/sec-build-next-series, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/claude.go"
remedy: "Hold the reported model to one plain line (termsafe.Sanitize equality, or mask it) where the runner records it, as receipts now hold their model (iss-2610121532247004), with a test."
resolution: "The claude runner already refused a reported model off modelRe (a plain ASCII id, since 70d49494f), now pinned for bidi, C1, zero-width and tag runes by a test; the remaining route, a stored route model reaching implement record --json from a forged state.json, is masked where the state is stored and read by the fix this names."
impact: fix
resolved_by:
  commit: "ffee62d99aa0f14396606ccc0ad5622d11c6f3d3"
---

The claude runner (internal/core/runner/claude.go) copies the external tool's reported model into the route record, which reaches abcd implement record --json raw, so a tool reporting a model name with a control or bidi rune puts it on the stream.

Security scope (abcd-fixes-what-it-adds): the route does not exist without abcd. The text is echoed by abcd's own --json surface, so abcd added the route and fixes it at the boundary the text enters by; a fix changes nothing about how the repository works.
