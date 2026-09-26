---
schema_version: 1
id: "iss-242"
slug: "abcd-intent-json-reports-bucket-counts-and-spec-links-only-n"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: "internal/surface/cli"
resolution: "abcd intent --json carries an intents array: per intent its id, title, bucket, ac_state real|seeded and the filing date a timestamp id encodes (null for an ordinal id)."
impact: additive
resolved_by:
  commit: "22505a099f8dc127ad495036d25f4f3886234d72"
---

abcd intent --json reports bucket counts and spec links only — no per-intent listing. A planning sweep over drafts/ (which intents are plannable vs seeded-placeholder AC, filed when) required shell-grepping 55 files and git history. The verb wants a list mode with per-intent id, title, bucket, ac_state (seeded|real), and filing date; the seeded placeholder is a stable string, so ac_state is cheap.

## Grounds

- pursued: a planning sweep learns which drafts are plannable from one call; a seeded draft reported real, or an intent missing from the listing, would show it wrong
