---
schema_version: 1
id: "iss-2608261533033587"
slug: "docs-lint-renderer-prints-config"
severity: "nitpick"
category: "security"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "internal/surface/cli/cli.go"
resolution: "The renderer sanitises Severity and RuleID alongside File and Message and the enum-constrained comment is corrected; watched-fail ESC-in-token-id test"
impact: fix
resolved_by:
  commit: "b215c8c6"
---

docs lint's findings renderer asserts 'Severity/RuleID are enum-constrained' and prints both unsanitised, but neither is constrained: Severity is verbatim committed-config text and RuleID for the banned_tokens family is the token's configured id. The config file is adjudicated a trust boundary in internal/core/lint/config.go (cross-repo-clonable), and the same surface sanitises the enum-validated banlist severity — so the one unvalidated pair on the line is the unsanitised one. ToUpper does not neutralise OSC/CSI escapes (verified), and RuleID gets no ToUpper at all. Sibling scope: the recorded cmd/record-lint File/Message gap asserts the abcd CLI renderer sanitises — true for File/Message only. Acceptance: the renderer sanitises every config-derived field or the loader makes the comment true by validation.